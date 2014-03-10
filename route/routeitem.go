// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package route

import (
	"errors"
	"fmt"
	//beecontext "github.com/smithfox/beego/context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type ControllerInterface interface{}

/*
type ControllerInterface interface {
	Init(*beecontext.Context)
	Prepare()
	Finish()
	Get()
	Post()
	Delete()
	Put()
	Head()
	Patch()
	Options()
}
*/

type FilterFunc func(http.ResponseWriter, *http.Request) bool

func (f FilterFunc) FilterHTTP(w http.ResponseWriter, r *http.Request) bool {
	return f(w, r)
}

type Filter interface {
	FilterHTTP(http.ResponseWriter, *http.Request) bool
}

// RouteItem stores information to match a request and build URLs.
type RouteItem struct {
	// Request handler for the route.
	handler http.Handler

	//controller
	cType      reflect.Type
	controller ControllerInterface

	// List of matchers.
	matchers []matcher
	// Manager for the variables from host and path.
	regexp *routeRegexpGroup
	// If true, when the path pattern is "/path/", accessing "/path" will
	// redirect to the former and vice versa.
	strictSlash bool
	// If true, this route never matches: it is only used to build URLs.
	buildOnly bool
	// The name used to build URLs.
	name string
	// Error resulted from building a route.
	err error

	// onlyscheme=http,  https will force redirect to http
	// onlyscheme=https, http will force redirect to https
	// otherwise, ignore
	onlyscheme string
}

// Match matches the route against the request.
func (r *RouteItem) Match(req *http.Request, match *RouteMatch) bool {
	if r.buildOnly || r.err != nil {
		return false
	}
	// Match everything.
	for _, m := range r.matchers {
		if matched := m.Match(req, match); !matched {
			return false
		}
	}
	// Yay, we have a match. Let's collect some info about it.
	if match.RouteItem == nil {
		match.RouteItem = r
	}
	if match.Controller == nil {
		match.Controller = r.controller
		match.CType = r.cType
	}
	if match.Handler == nil {
		match.Handler = r.handler
	}
	if match.Vars == nil {
		match.Vars = make(map[string]string)
	}
	// Set variables.
	if r.regexp != nil {
		r.regexp.setMatch(req, match, r)
	}

	match.OnlyScheme = r.onlyscheme
	return true
}

// ----------------------------------------------------------------------------
// RouteItem attributes
// ----------------------------------------------------------------------------

// GetError returns an error resulted from building the route, if any.
func (r *RouteItem) GetError() error {
	return r.err
}

// BuildOnly sets the route to never match: it is only used to build URLs.
func (r *RouteItem) BuildOnly() *RouteItem {
	r.buildOnly = true
	return r
}

// Handler --------------------------------------------------------------------
/*
func InterfaceOf(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Interface {
		panic("Called inject.InterfaceOf with a value that is not a pointer to an interface. (*MyInterface)(nil)")
	}

	return t
}
*/

func (r *RouteItem) Controller(c ControllerInterface) *RouteItem {
	if r.err == nil {
		//默认 controller 都是 http, 可以通过 OnlyScheme() 来改变
		r.onlyscheme = "http"
		r.controller = c
		//r.cType = InterfaceOf(c)
		reflectVal := reflect.ValueOf(c)
		r.cType = reflect.Indirect(reflectVal).Type()
	}
	return r
}

// Handler sets a handler for the route.
func (r *RouteItem) Handler(handler http.Handler) *RouteItem {
	if r.err == nil {
		r.handler = handler
	}
	return r
}

// HandlerFunc sets a handler function for the route.
func (r *RouteItem) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *RouteItem {
	return r.Handler(http.HandlerFunc(f))
}

/*
// GetHandler returns the handler for the route, if any.
func (r *RouteItem) GetHandler() http.Handler {
	return r.handler
}
*/
// ----------------------------------------------------------------------------
// Matchers
// ----------------------------------------------------------------------------

// matcher types try to match a request.
type matcher interface {
	Match(*http.Request, *RouteMatch) bool
}

// addMatcher adds a matcher to the route.
func (r *RouteItem) addMatcher(m matcher) *RouteItem {
	if r.err == nil {
		r.matchers = append(r.matchers, m)
	}
	return r
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (r *RouteItem) addRegexpMatcher(tpl string, matchHost, matchPrefix bool) error {
	if r.err != nil {
		return r.err
	}
	r.regexp = r.getRegexpGroup()
	if !matchHost {
		if len(tpl) == 0 || tpl[0] != '/' {
			return fmt.Errorf("mux: path must start with a slash, got %q", tpl)
		}
		if r.regexp.path != nil {
			tpl = strings.TrimRight(r.regexp.path.template, "/") + tpl
		}
	}
	rr, err := newRouteRegexp(tpl, matchHost, matchPrefix, r.strictSlash)
	if err != nil {
		return err
	}
	if matchHost {
		if r.regexp.path != nil {
			if err = uniqueVars(rr.varsN, r.regexp.path.varsN); err != nil {
				return err
			}
		}
		r.regexp.host = rr
	} else {
		if r.regexp.host != nil {
			if err = uniqueVars(rr.varsN, r.regexp.host.varsN); err != nil {
				return err
			}
		}
		r.regexp.path = rr
	}
	r.addMatcher(rr)
	return nil
}

// Headers --------------------------------------------------------------------

// headerMatcher matches the request against header values.
type headerMatcher map[string]string

func (m headerMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchMap(m, r.Header, true)
}

// Headers adds a matcher for request header values.
// It accepts a sequence of key/value pairs to be matched. For example:
//
//     r := mux.NewRouter()
//     r.Headers("Content-Type", "application/json",
//               "X-Requested-With", "XMLHttpRequest")
//
// The above route will only match if both request header values match.
//
// It the value is an empty string, it will match any value if the key is set.
func (r *RouteItem) Headers(pairs ...string) *RouteItem {
	if r.err == nil {
		var headers map[string]string
		headers, r.err = mapFromPairs(pairs...)
		return r.addMatcher(headerMatcher(headers))
	}
	return r
}

// Host -----------------------------------------------------------------------

// Host adds a matcher for the URL host.
// It accepts a template with zero or more URL variables enclosed by {}.
// Variables can define an optional regexp pattern to me matched:
//
// - {name} matches anything until the next dot.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     r := mux.NewRouter()
//     r.Host("www.domain.com")
//     r.Host("{subdomain}.domain.com")
//     r.Host("{subdomain:[a-z]+}.domain.com")
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (r *RouteItem) Host(tpl string) *RouteItem {
	r.err = r.addRegexpMatcher(tpl, true, false)
	return r
}

// MatcherFunc ----------------------------------------------------------------

// MatcherFunc is the function signature used by custom matchers.
type MatcherFunc func(*http.Request, *RouteMatch) bool

func (m MatcherFunc) Match(r *http.Request, match *RouteMatch) bool {
	return m(r, match)
}

// MatcherFunc adds a custom function to be used as request matcher.
func (r *RouteItem) MatcherFunc(f MatcherFunc) *RouteItem {
	return r.addMatcher(f)
}

// Methods --------------------------------------------------------------------

// methodMatcher matches the request against HTTP methods.
type methodMatcher []string

func (m methodMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchInArray(m, r.Method)
}

// Methods adds a matcher for HTTP methods.
// It accepts a sequence of one or more methods to be matched, e.g.:
// "GET", "POST", "PUT".
func (r *RouteItem) Methods(methods ...string) *RouteItem {
	for k, v := range methods {
		methods[k] = strings.ToUpper(v)
	}
	return r.addMatcher(methodMatcher(methods))
}

// Path -----------------------------------------------------------------------

// Path adds a matcher for the URL path.
// It accepts a template with zero or more URL variables enclosed by {}.
// Variables can define an optional regexp pattern to me matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     r := mux.NewRouter()
//     r.Path("/products/").Handler(ProductsHandler)
//     r.Path("/products/{key}").Handler(ProductsHandler)
//     r.Path("/articles/{category}/{id:[0-9]+}").
//       Handler(ArticleHandler)
//
// Variable names must be unique in a given route. They can be retrieved
// calling mux.Vars(request).
func (r *RouteItem) Path(tpl string) *RouteItem {
	r.err = r.addRegexpMatcher(tpl, false, false)
	return r
}

// PathPrefix -----------------------------------------------------------------

// PathPrefix adds a matcher for the URL path prefix.
func (r *RouteItem) PathPrefix(tpl string) *RouteItem {
	r.strictSlash = false
	r.err = r.addRegexpMatcher(tpl, false, true)
	return r
}

// Query ----------------------------------------------------------------------

// queryMatcher matches the request against URL queries.
type queryMatcher map[string]string

func (m queryMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchMap(m, r.URL.Query(), false)
}

// Queries adds a matcher for URL query values.
// It accepts a sequence of key/value pairs. For example:
//
//     r := mux.NewRouter()
//     r.Queries("foo", "bar", "baz", "ding")
//
// The above route will only match if the URL contains the defined queries
// values, e.g.: ?foo=bar&baz=ding.
//
// It the value is an empty string, it will match any value if the key is set.
func (r *RouteItem) Queries(pairs ...string) *RouteItem {
	if r.err == nil {
		var queries map[string]string
		queries, r.err = mapFromPairs(pairs...)
		return r.addMatcher(queryMatcher(queries))
	}
	return r
}

// Schemes --------------------------------------------------------------------
func (r *RouteItem) OnlyScheme(scheme string) *RouteItem {
	r.onlyscheme = strings.ToLower(scheme)
	return r
}

// ----------------------------------------------------------------------------
// URL building
// ----------------------------------------------------------------------------

// URL builds a URL for the route.
//
// It accepts a sequence of key/value pairs for the route variables. For
// example, given this route:
//
//     r := mux.NewRouter()
//     r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
//       Name("article")
//
// ...a URL for it can be built using:
//
//     url, err := r.Get("article").URL("category", "technology", "id", "42")
//
// ...which will return an url.URL with the following path:
//
//     "/articles/technology/42"
//
// This also works for host variables:
//
//     r := mux.NewRouter()
//     r.Host("{subdomain}.domain.com").
//       HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
//       Name("article")
//
//     // url.String() will be "http://news.domain.com/articles/technology/42"
//     url, err := r.Get("article").URL("subdomain", "news",
//                                      "category", "technology",
//                                      "id", "42")
//
// All variables defined in the route are required, and their values must
// conform to the corresponding patterns.
func (r *RouteItem) URL(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp == nil {
		return nil, errors.New("mux: route doesn't have a host or path")
	}
	var scheme, host, path string
	var err error
	if r.regexp.host != nil {
		// Set a default scheme.
		scheme = "http"
		if host, err = r.regexp.host.url(pairs...); err != nil {
			return nil, err
		}
	}
	if r.regexp.path != nil {
		if path, err = r.regexp.path.url(pairs...); err != nil {
			return nil, err
		}
	}
	return &url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}, nil
}

// URLHost builds the host part of the URL for a route. See RouteItem.URL().
//
// The route must have a host defined.
func (r *RouteItem) URLHost(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp == nil || r.regexp.host == nil {
		return nil, errors.New("mux: route doesn't have a host")
	}
	host, err := r.regexp.host.url(pairs...)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Scheme: "http",
		Host:   host,
	}, nil
}

// URLPath builds the path part of the URL for a route. See RouteItem.URL().
//
// The route must have a path defined.
func (r *RouteItem) URLPath(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.regexp == nil || r.regexp.path == nil {
		return nil, errors.New("mux: route doesn't have a path")
	}
	path, err := r.regexp.path.url(pairs...)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Path: path,
	}, nil
}

// getRegexpGroup returns regexp definitions from this route.
func (r *RouteItem) getRegexpGroup() *routeRegexpGroup {
	if r.regexp == nil {
		r.regexp = new(routeRegexpGroup)
	}
	return r.regexp
}
