package route

import (
	"fmt"
	"github.com/smithfox/beego/context"
	"github.com/smithfox/beego/recovery"
	"log"
	"net/http"
	"path"
	//"reflect"
)

type Router struct {
	// Configurable Handler to be used when no route matches.
	NotFoundHandler http.Handler
	// Routes to be matched, in order.
	routes    []*RouteItem
	filters   []Filter
	httphost  string
	httpshost string
}

// NewRouter returns a new router instance.
func NewRouter() *Router {
	return &Router{}
}

// NewRouter returns a new router instance.
func NewRouterWithHost(httphost, httpshost string) *Router {
	return &Router{httphost: httphost, httpshost: httpshost}
}

func (r *Router) NewRouteItem() *RouteItem {
	routeitem := &RouteItem{}
	routeitem.strictSlash = true
	r.routes = append(r.routes, routeitem)
	return routeitem
}

// Match matches registered routes against the request.
func (r *Router) Match(req *http.Request, match *RouteMatch) bool {
	for _, route := range r.routes {
		if route.Match(req, match) {
			return true
		}
	}
	return false
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			stack := recovery.Stack(0)
			log.Printf("Router ServeHTTP PANIC: %s\n%s\n", err, stack)
		}
	}()

	//debug.PrintStack()
	//fmt.Printf("router ServeHTTP, url=%q\n", req.URL)
	for _, filter := range r.filters {
		if ok := filter.FilterHTTP(w, req); !ok {
			return
		}
	}

	var match RouteMatch
	var handler http.Handler
	if r.Match(req, &match) {
		fmt.Printf("router.ServHTTP, matched\n")
		//{{处理 https和 http 的redirect
		var redirectURL string = ""
		if match.OnlyScheme == "http" && req.TLS != nil {
			redirectURL = r.httphost + req.URL.Path
		} else if match.OnlyScheme == "https" && req.TLS == nil {
			redirectURL = r.httpshost + req.URL.Path
		} else {
			redirectURL = ""
		}

		if redirectURL != "" {
			fmt.Printf("Router ServeHTTP redirectURL=%s, match.OnlyScheme=%s,req.TLS=%t\n", redirectURL, match.OnlyScheme, (req.TLS != nil))
			http.Redirect(w, req, redirectURL, http.StatusMovedPermanently)
			return
		}
		//}}

		if match.NewControllerHandler != nil {
			context := &context.Context{W: w, R: req, Param: match.Vars, EnableGzip: true}
			fmt.Printf("router.ServHTTP, new NewControllerHandler\n")
			handler = match.NewControllerHandler(context)
		} else {
			handler = match.Handler
		}
	}

	if handler == nil {
		if r.NotFoundHandler == nil {
			r.NotFoundHandler = http.NotFoundHandler()
		}
		handler = r.NotFoundHandler
	}

	handler.ServeHTTP(w, req)
}

func (r *Router) ControllerFunc(path string, f func() Controller) *RouteItem {
	return r.NewRouteItem().Path(path).ControllerFunc(f)
}

func (r *Router) Controller(path string, c Controller) *RouteItem {
	return r.NewRouteItem().Path(path).Controller(c)
}

// Handle registers a new route with a matcher for the URL path.
// See RouteItem.Path() and RouteItem.Handler().
func (r *Router) Handle(path string, handler http.Handler) *RouteItem {
	return r.NewRouteItem().Path(path).Handler(handler)
}

// HandleFunc registers a new route with a matcher for the URL path.
// See Route.Path() and Route.HandlerFunc().
func (r *Router) HandleFunc(path string, f func(http.ResponseWriter,
	*http.Request)) *RouteItem {
	return r.NewRouteItem().Path(path).HandlerFunc(f)
}

func (r *Router) Filter(filter Filter) {
	r.filters = append(r.filters, filter)
}

func (r *Router) FilterFunc(f func(http.ResponseWriter,
	*http.Request) bool) {
	r.Filter(FilterFunc(f))
}

// Headers registers a new route with a matcher for request header values.
// See RouteItem.Headers().
func (r *Router) Headers(pairs ...string) *RouteItem {
	return r.NewRouteItem().Headers(pairs...)
}

// Host registers a new route with a matcher for the URL host.
// See RouteItem.Host().
func (r *Router) Host(tpl string) *RouteItem {
	return r.NewRouteItem().Host(tpl)
}

// MatcherFunc registers a new route with a custom matcher function.
// See RouteItem.MatcherFunc().
func (r *Router) MatcherFunc(f MatcherFunc) *RouteItem {
	return r.NewRouteItem().MatcherFunc(f)
}

// Methods registers a new route with a matcher for HTTP methods.
// See RouteItem.Methods().
func (r *Router) Methods(methods ...string) *RouteItem {
	return r.NewRouteItem().Methods(methods...)
}

// Path registers a new route with a matcher for the URL path.
// See RouteItem.Path().
func (r *Router) Path(tpl string) *RouteItem {
	return r.NewRouteItem().Path(tpl)
}

// PathPrefix registers a new route with a matcher for the URL path prefix.
// See RouteItem.PathPrefix().
func (r *Router) PathPrefix(tpl string) *RouteItem {
	return r.NewRouteItem().PathPrefix(tpl)
}

// Queries registers a new route with a matcher for URL query values.
// See RouteItem.Queries().
func (r *Router) Queries(pairs ...string) *RouteItem {
	return r.NewRouteItem().Queries(pairs...)
}

// RouteMatch stores information about a matched route.
type RouteMatch struct {
	RouteItem            *RouteItem
	Handler              http.Handler
	NewControllerHandler NewControllerHandlerFunc
	Vars                 map[string]string
	OnlyScheme           string
	CheckCsrf            bool
	CheckAuth            bool
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// cleanPath returns the canonical path for p, eliminating . and .. elements.
// Borrowed from the net/http package.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// uniqueVars returns an error if two slices contain duplicated strings.
func uniqueVars(s1, s2 []string) error {
	for _, v1 := range s1 {
		for _, v2 := range s2 {
			if v1 == v2 {
				return fmt.Errorf("mux: duplicated route variable %q", v2)
			}
		}
	}
	return nil
}

// mapFromPairs converts variadic string parameters to a string map.
func mapFromPairs(pairs ...string) (map[string]string, error) {
	length := len(pairs)
	if length%2 != 0 {
		return nil, fmt.Errorf(
			"mux: number of parameters must be multiple of 2, got %v", pairs)
	}
	m := make(map[string]string, length/2)
	for i := 0; i < length; i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m, nil
}

// matchInArray returns true if the given string value is in the array.
func matchInArray(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

// matchMap returns true if the given key/value pairs exist in a given map.
func matchMap(toCheck map[string]string, toMatch map[string][]string,
	canonicalKey bool) bool {
	for k, v := range toCheck {
		// Check if key exists.
		if canonicalKey {
			k = http.CanonicalHeaderKey(k)
		}
		if values := toMatch[k]; values == nil {
			return false
		} else if v != "" {
			// If value was defined as an empty string we only check that the
			// key exists. Otherwise we also check for equality.
			valueExists := false
			for _, value := range values {
				if v == value {
					valueExists = true
					break
				}
			}
			if !valueExists {
				return false
			}
		}
	}
	return true
}
