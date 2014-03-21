// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package route

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HandlerRouteItem stores information to match a request and build URLs.
type HandlerRouteItem struct {
	*Router
	// Request handler for the route.
	handler http.Handler

	// List of matchers.
	matchers []matcher
	// Manager for the variables from host and path.
	regexp *routeRegexpGroup

	// How to deal with the traviling slash
	// FuzzyEndSlash  = 0
	// RedictEndSlash = 1
	// ExactEndSlash  = 2
	endSlashOption int

	// If true, this route never matches: it is only used to build URLs.
	buildOnly bool
	// The name used to build URLs.
	name string
	// Error resulted from building a route.
	err error

	// OnlyScheme=http,  https will force redirect to http
	// OnlyScheme=https, http will force redirect to https
	// otherwise, ignore
	onlyscheme string

	//POST/DEL/PUT auto forcely check csrf, the option means GET check
	checkCsrf bool

	checkAuth bool
}

// Match matches the route against the request.
func (r *HandlerRouteItem) Match(req *http.Request) bool {
	if r.buildOnly || r.err != nil {
		return false
	}
	// Match everything.
	for _, m := range r.matchers {
		if matched := m.Match(req); !matched {
			return false
		}
	}

	return true
}

// ----------------------------------------------------------------------------
// HandlerRouteItem attributes
// ----------------------------------------------------------------------------

// GetError returns an error resulted from building the route, if any.
func (r *HandlerRouteItem) GetError() error {
	return r.err
}

// BuildOnly sets the route to never match: it is only used to build URLs.
func (r *HandlerRouteItem) BuildOnly() *HandlerRouteItem {
	r.buildOnly = true
	return r
}

func (r *HandlerRouteItem) GetRedirectURL(req *http.Request) string {
	//{{处理 https和 http 的redirect
	var redirectURL string = ""
	if r.onlyscheme == "http" && req.TLS != nil {
		redirectURL = r.Router.httphost + req.URL.Path
	} else if r.onlyscheme == "https" && req.TLS == nil {
		redirectURL = r.Router.httpshost + req.URL.Path
	} else {
		redirectURL = ""
	}
	return redirectURL
}

func (r *HandlerRouteItem) SetSlashOption(option int) {
	r.endSlashOption = option
}

//是否精确匹配URL的最后'/'
func (r *HandlerRouteItem) ExactMatchSlash() bool {
	return r.endSlashOption == ExactEndSlash
}

//若模糊匹配URL的最后'/', 是否强制服从 route 的定义
// route:  /url      request:  /url/   ==> redirect到  /url
// route:  /url/     request:  /url    ==> redirect到  /url/
func (r *HandlerRouteItem) RedirectSlash() bool {
	return r.endSlashOption == RedictEndSlash
}

// Handler --------------------------------------------------------------------

// Handler sets a handler for the route.
func (r *HandlerRouteItem) Handler(handler http.Handler) *HandlerRouteItem {
	if r.err == nil {
		r.handler = handler
	}
	return r
}

// HandlerFunc sets a handler function for the route.
func (r *HandlerRouteItem) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *HandlerRouteItem {
	return r.Handler(http.HandlerFunc(f))
}

// GetHandler returns the handler for the route, if any.
func (r *HandlerRouteItem) CreateHandler(w http.ResponseWriter, req *http.Request) http.Handler {
	return r.handler
}

// ----------------------------------------------------------------------------
// Matchers
// ----------------------------------------------------------------------------

// matcher types try to match a request.
type matcher interface {
	Match(*http.Request) bool
}

// addMatcher adds a matcher to the route.
func (r *HandlerRouteItem) addMatcher(m matcher) *HandlerRouteItem {
	if r.err == nil {
		r.matchers = append(r.matchers, m)
	}
	return r
}

func (r *HandlerRouteItem) GetRouteParams(req *http.Request) RouteParams {
	routeParams := make(RouteParams)
	r.regexp.setMatch(req, routeParams)
	return routeParams
}

//0-no,1-yes, otherwise no path
func (r *HandlerRouteItem) EndWithSlash() int {
	if r.regexp == nil || r.regexp.path == nil {
		return 99
	}
	if strings.HasSuffix(r.regexp.path.template, "/") {
		return 1
	} else {
		return 0
	}
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (r *HandlerRouteItem) addRegexpMatcher(tpl string, matchHost, matchPrefix bool) error {
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
	rr, err := newRouteRegexp(tpl, matchHost, matchPrefix, !r.ExactMatchSlash())
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

func (m headerMatcher) Match(r *http.Request) bool {
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
func (r *HandlerRouteItem) Headers(pairs ...string) *HandlerRouteItem {
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
func (r *HandlerRouteItem) Host(tpl string) *HandlerRouteItem {
	r.err = r.addRegexpMatcher(tpl, true, false)
	return r
}

// MatcherFunc ----------------------------------------------------------------

// MatcherFunc is the function signature used by custom matchers.
type MatcherFunc func(*http.Request) bool

func (m MatcherFunc) Match(r *http.Request) bool {
	return m(r)
}

// MatcherFunc adds a custom function to be used as request matcher.
func (r *HandlerRouteItem) MatcherFunc(f MatcherFunc) *HandlerRouteItem {
	return r.addMatcher(f)
}

// Methods --------------------------------------------------------------------

// methodMatcher matches the request against HTTP methods.
type methodMatcher []string

func (m methodMatcher) Match(r *http.Request) bool {
	return matchInArray(m, r.Method)
}

// Methods adds a matcher for HTTP methods.
// It accepts a sequence of one or more methods to be matched, e.g.:
// "GET", "POST", "PUT".
func (r *HandlerRouteItem) _methods(methods ...string) *HandlerRouteItem {
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
func (r *HandlerRouteItem) Path(tpl string) *HandlerRouteItem {
	r.err = r.addRegexpMatcher(tpl, false, false)
	return r
}

// PathPrefix -----------------------------------------------------------------

// PathPrefix adds a matcher for the URL path prefix.
func (r *HandlerRouteItem) PathPrefix(tpl string) *HandlerRouteItem {
	r.endSlashOption = ExactEndSlash
	r.err = r.addRegexpMatcher(tpl, false, true)
	return r
}

// Query ----------------------------------------------------------------------

// queryMatcher matches the request against URL queries.
type queryMatcher map[string]string

func (m queryMatcher) Match(r *http.Request) bool {
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
func (r *HandlerRouteItem) Queries(pairs ...string) *HandlerRouteItem {
	if r.err == nil {
		var queries map[string]string
		queries, r.err = mapFromPairs(pairs...)
		return r.addMatcher(queryMatcher(queries))
	}
	return r
}

// Schemes --------------------------------------------------------------------
func (r *HandlerRouteItem) OnlyScheme(scheme string) *HandlerRouteItem {
	r.onlyscheme = strings.ToLower(scheme)
	return r
}

func (r *HandlerRouteItem) CheckCsrf() *HandlerRouteItem {
	r.checkCsrf = true
	return r
}

func (r *HandlerRouteItem) CheckAuth() *HandlerRouteItem {
	r.checkAuth = true
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
func (r *HandlerRouteItem) URL(pairs ...string) (*url.URL, error) {
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

// URLHost builds the host part of the URL for a route. See HandlerRouteItem.URL().
//
// The route must have a host defined.
func (r *HandlerRouteItem) URLHost(pairs ...string) (*url.URL, error) {
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

// URLPath builds the path part of the URL for a route. See HandlerRouteItem.URL().
//
// The route must have a path defined.
func (r *HandlerRouteItem) URLPath(pairs ...string) (*url.URL, error) {
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
func (r *HandlerRouteItem) getRegexpGroup() *routeRegexpGroup {
	if r.regexp == nil {
		r.regexp = new(routeRegexpGroup)
	}
	return r.regexp
}
