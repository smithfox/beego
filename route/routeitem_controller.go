package route

import (
	//"fmt"
	"github.com/smithfox/beego/context"
	"net/http"
	"reflect"
)

type ControllerRouteItem struct {
	HandlerRouteItem
	createCtrlHandler NewControllerHandlerFunc
}

func (r *ControllerRouteItem) Path(tpl string) *ControllerRouteItem {
	r.err = r.addRegexpMatcher(tpl, false, false)
	return r
}

func (r *ControllerRouteItem) CreateHandler(w http.ResponseWriter, req *http.Request) http.Handler {
	routeParams := r.GetRouteParams(req)
	context := &context.Context{W: w, R: req, Param: routeParams, EnableGzip: true}
	//fmt.Printf("ControllerRouteItem\n")
	return r.createCtrlHandler(context)
}

func (r *ControllerRouteItem) ControllerFunc(f func() Controller) *ControllerRouteItem {
	if r.err == nil {
		//默认 controller 都是 http, 可以通过 OnlyScheme() 来改变
		r.onlyscheme = "http"
		r.createCtrlHandler = func(c *context.Context) ControllerHandler {
			//fmt.Printf("RouteItem.createCtrlHandler\n")
			ci := f()
			ci.Init(c)
			return &WrapperController{Controller: ci}
		}
	}
	return r
}

func (r *ControllerRouteItem) Controller(c Controller) *ControllerRouteItem {
	if r.err == nil {
		//默认 controller 都是 http, 可以通过 OnlyScheme() 来改变
		r.onlyscheme = "http"
		rt := reflect.TypeOf(c)
		if rt.Kind() != reflect.Ptr {
			panic("RoutItem.Controller parameter must pointer!")
		}

		rt = rt.Elem()

		r.createCtrlHandler = func(ctx *context.Context) ControllerHandler {
			vc := reflect.New(rt)
			ci := vc.Interface().(Controller)
			ci.Init(ctx)
			return &WrapperController{Controller: ci}
		}
	}
	return r
}

///===================== controller ========================

type ControllerOption struct {
	AuthAny bool
	CrsfAny bool
}

type Controller interface {
	Context() *context.Context
	Init(*context.Context)
	CheckCsrf() bool
	CheckAuth() bool
	Prepare()
	Get()
	Post()
	Delete()
	Put()
	Head()
	Patch()
	Options()
	Finish()
}

//即实现了 Controller, 也实现了 ServerHTTP
type ControllerHandler interface {
	Controller
	http.Handler
}

//var noparams []reflect.Value = []reflect.Value{}

type NewControllerHandlerFunc func(*context.Context) ControllerHandler

type WrapperController struct {
	Controller
}

func (c *WrapperController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("WrapperController.ServeHTTP, method=%s\n", r.Method)
	CallMatchedMethod(c.Controller)
}

func CallMatchedMethod(c Controller) {
	//fmt.Printf("CallMatchedMethod\n")
	r := c.Context().R

	c.Init(c.Context())
	c.Prepare()

	if r.Method == "GET" {
		c.Get()
	} else if r.Method == "HEAD" {
		c.Head()
	} else if r.Method == "DELETE" || (r.Method == "POST" && r.Form.Get("_method") == "delete") {
		c.Delete()
	} else if r.Method == "PUT" || (r.Method == "POST" && r.Form.Get("_method") == "put") {
		c.Put()
	} else if r.Method == "POST" {
		c.Post()
	} else if r.Method == "PATCH" {
		c.Patch()
	} else if r.Method == "OPTIONS" {
		c.Options()
	}

	c.Finish()
}
