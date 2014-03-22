package route

import (
	"fmt"
	"github.com/smithfox/beego/context"
	"net/http"
	"reflect"
)

type ContextRouteItem struct {
	HandlerRouteItem
	createCtxHandler createCtxHandlerFunc
}

func (r *ContextRouteItem) Path(tpl string) *ContextRouteItem {
	r.err = r.addRegexpMatcher(tpl, false, false)
	return r
}

func (r *ContextRouteItem) CreateHandler(w http.ResponseWriter, req *http.Request) http.Handler {
	routeParams := r.GetRouteParams(req)
	context := &context.Context{W: w, R: req, Param: routeParams, EnableGzip: true}
	//fmt.Printf("ContextRouteItem\n")
	return r.createCtxHandler(r.Router.services, context)
}

// Context --------------------------------------------------------------------

func (r *ContextRouteItem) action(f ContextHandler) {
	rt := reflect.TypeOf(f)
	if rt.Kind() != reflect.Ptr {
		panic("RoutItem.Get parameter must pointer!")
	}

	rt = rt.Elem()

	r.createCtxHandler = func(services map[string]DatabusService, ctx *context.Context) http.Handler {
		//fmt.Printf("RouteItem.createCtxHandler\n")
		vc := reflect.New(rt)
		ci := vc.Interface()
		bus := WrapperBusValue(vc)
		wch := &WrapperContextHandler{}
		wch.context = ctx
		wch.bus = bus
		wch.services = services
		wch.handler = ci.(ContextHandler)

		return wch
	}
}

func (r *ContextRouteItem) Get(f ContextHandler) *ContextRouteItem {
	if r.err == nil {
		r._methods("GET")
		//默认 controller 都是 http, 可以通过 OnlyScheme() 来改变
		r.onlyscheme = "http"
		r.action(f)
	}
	return r
}

func (r *ContextRouteItem) Post(f ContextHandler) *ContextRouteItem {
	if r.err == nil {
		r._methods("POST")
		//默认 controller 都是 http, 可以通过 OnlyScheme() 来改变
		r.onlyscheme = "http"
		r.action(f)
	}
	return r
}

////===================== ContextHandler ====================
type DatabusService func(*context.Context, Databus)

type ServeContextFunc func(*context.Context)

func (scf ServeContextFunc) ServeContext(context *context.Context) {
	scf(context)
}

type ContextHandler interface {
	ServeContext(context *context.Context)
}

type createCtxHandlerFunc func(map[string]DatabusService, *context.Context) http.Handler

type WrapperContextHandler struct {
	context  *context.Context
	bus      Databus
	handler  ContextHandler
	services map[string]DatabusService
}

func (c *WrapperContextHandler) RunBus() {
	for _, name := range c.bus.Fields() {
		//fmt.Printf("RunBus, name=%s\n", name)
		service, _ := c.services[name]
		if service != nil {
			service(c.context, c.bus)
		} else {
			fmt.Printf("RunBus, NOT find Service with name=%s\n", name)
		}
	}
}

func (c *WrapperContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("WrapperContextHandler.ServeHTTP\n")
	c.RunBus()
	c.handler.ServeContext(c.context)
}
