package route

import (
	"fmt"
	"github.com/smithfox/beego/recovery"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type FilterFunc func(http.ResponseWriter, *http.Request) bool

func (f FilterFunc) FilterHTTP(w http.ResponseWriter, r *http.Request) bool {
	return f(w, r)
}

type Filter interface {
	FilterHTTP(http.ResponseWriter, *http.Request) bool
}

type Router struct {
	// Configurable Handler to be used when no route matches.
	NotFoundHandler http.Handler
	// Routes to be matched, in order.
	routes          []RouteItem
	filters         []Filter
	httphost        string
	httpshost       string
	enable_to_https bool //是否允许重定向到 https
	EnableGzip      bool
	services        map[string]DatabusService
}

// NewRouter returns a new router instance.
func NewRouter(enable_gzip bool) *Router {
	router := &Router{services: make(map[string]DatabusService)}
	router.EnableGzip = enable_gzip
	return router
}

// NewRouter returns a new router instance.
func NewRouterWithHost(httphost, httpshost string, enable_to_https bool, enable_gzip bool) *Router {
	router := &Router{services: make(map[string]DatabusService)}
	router.EnableGzip = enable_gzip
	router.httphost = httphost
	router.httpshost = httpshost
	if router.httphost == router.httpshost {
		enable_to_https = false
	}
	router.enable_to_https = enable_to_https
	return router
}

// func (r *Router) SetClassicMartini(mc *martini.ClassicMartini) {
// 	r.mc = mc
// }

func (r *Router) newHandlerRouteItem() *HandlerRouteItem {
	routeitem := &HandlerRouteItem{Router: r}
	r.routes = append(r.routes, routeitem)
	return routeitem
}

func (r *Router) newContextRouteItem() *ContextRouteItem {
	routeitem := &ContextRouteItem{}
	routeitem.Router = r
	r.routes = append(r.routes, routeitem)
	return routeitem
}

func (r *Router) newControllerRouteItem() *ControllerRouteItem {
	routeitem := &ControllerRouteItem{}
	routeitem.Router = r
	r.routes = append(r.routes, routeitem)
	return routeitem
}

// Match matches registered routes against the request.
func (r *Router) Match(req *http.Request) RouteItem {
	for _, route := range r.routes {
		if route.Match(req) {
			return route
		}
	}
	return nil
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

	var handler http.Handler
	routeitem := r.Match(req)
	if routeitem != nil {
		//fmt.Printf("router.ServHTTP, matched for url=%v\n", req.URL)
		//{{处理 https和 http 的redirect
		redirectURL := routeitem.GetSchemeRedirectURL(req)

		if redirectURL != "" {
			//fmt.Printf("Router ServeHTTP redirectURL=%s\n", redirectURL)
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}
		//}}

		//{{处理 RedirectSlash
		if routeitem.RedirectSlash() {
			p1 := strings.HasSuffix(req.URL.Path, "/")
			n2 := routeitem.EndWithSlash()
			if n2 <= 1 {
				p2 := (n2 == 1)
				if p1 != p2 {
					u, _ := url.Parse(req.URL.String())
					if p1 {
						u.Path = u.Path[:len(u.Path)-1]
					} else {
						u.Path += "/"
					}
					//fmt.Printf("Router ServeHTTP RedirectSlash redirectURL=%s\n", u.String())
					//此处Redirect要用 307, 不能用301,302,303, 否则POST请求被改为了 GET
					http.Redirect(w, req, u.String(), http.StatusTemporaryRedirect)
					return
				}
			}
		}
		//}}

		handler = routeitem.CreateHandler(w, req)
	}

	if handler == nil {
		if r.NotFoundHandler == nil {
			r.NotFoundHandler = http.NotFoundHandler()
		}
		handler = r.NotFoundHandler
	}

	handler.ServeHTTP(w, req)
}

/*
func (r *Router) MapDatabus(name string, f DatabusFunc) {
	r.databuses[name] = f
}
*/
func (r *Router) AddService(name string, s DatabusService) {
	r.services[name] = s
}

func (r *Router) Get(path string, v ContextHandler) *ContextRouteItem {
	return r.newContextRouteItem().Path(path).Get(v)
}

func (r *Router) Post(path string, v ContextHandler) *ContextRouteItem {
	return r.newContextRouteItem().Path(path).Post(v)
}

func (r *Router) ControllerFunc(path string, f func() Controller) *ControllerRouteItem {
	return r.newControllerRouteItem().Path(path).ControllerFunc(f)
}

func (r *Router) Controller(path string, c Controller) *ControllerRouteItem {
	return r.newControllerRouteItem().Path(path).Controller(c)
}

// Handle registers a new route with a matcher for the URL path.
func (r *Router) Handle(path string, handler http.Handler) *HandlerRouteItem {
	return r.newHandlerRouteItem().Path(path).Handler(handler)
}

// HandleFunc registers a new route with a matcher for the URL path.
// See Route.Path() and Route.HandlerFunc().
func (r *Router) HandleFunc(path string, f func(http.ResponseWriter,
	*http.Request)) *HandlerRouteItem {
	return r.newHandlerRouteItem().Path(path).HandlerFunc(f)
}

func (r *Router) Filter(filter Filter) {
	r.filters = append(r.filters, filter)
}

func (r *Router) FilterFunc(f func(http.ResponseWriter,
	*http.Request) bool) {
	r.Filter(FilterFunc(f))
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
