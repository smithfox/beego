package route

import (
	"fmt"
	"github.com/smithfox/beego/context"
	"net/http"
	"reflect"
)

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

var noparams []reflect.Value = []reflect.Value{}

type NewControllerHandlerFunc func(*context.Context) ControllerHandler

type reflectWrapperController struct {
	vc      reflect.Value
	context *context.Context
	option  ControllerOption
}

func (c *reflectWrapperController) callVoidMethod(name string) {
	method := c.vc.MethodByName(name)
	method.Call(noparams)
}

func (c *reflectWrapperController) callBoolMethod(name string) bool {
	method := c.vc.MethodByName(name)
	out := method.Call(noparams)
	ff := out[0].Interface().(bool)
	return ff
}

func (c *reflectWrapperController) Init(cc *context.Context) {
	ppp := make([]reflect.Value, 1)
	ppp[0] = reflect.ValueOf(c.context)
	method := c.vc.MethodByName("Init")
	method.Call(ppp)
}

func (c *reflectWrapperController) Context() *context.Context {
	return c.context
}

func (c *reflectWrapperController) Get() {
	if c.option.CrsfAny {
		if !c.CheckCsrf() {
			fmt.Printf("GET, checkcsrf return false\n")
			return
		}
	}

	c.callVoidMethod("Get")
}

func (c *reflectWrapperController) Delete() {
	if !c.CheckAuth() {
		fmt.Printf("CheckAuth return false\n")
		return
	}

	if !c.CheckCsrf() {
		return
	}
	c.callVoidMethod("Delete")
}

func (c *reflectWrapperController) Put() {
	if !c.CheckAuth() {
		fmt.Printf("CheckAuth return false\n")
		return
	}

	if !c.CheckCsrf() {
		return
	}
	c.callVoidMethod("Put")
}

func (c *reflectWrapperController) Post() {
	if !c.CheckAuth() {
		fmt.Printf("CheckAuth return false\n")
		return
	}

	if !c.CheckCsrf() {
		return
	}
	c.callVoidMethod("Post")
}

func (c *reflectWrapperController) Patch() {
	c.callVoidMethod("Patch")
}

func (c *reflectWrapperController) Options() {
	c.callVoidMethod("Options")
}

func (c *reflectWrapperController) Head() {
	c.callVoidMethod("Head")
}

func (c *reflectWrapperController) CheckAuth() bool {
	return c.callBoolMethod("CheckAuth")
}
func (c *reflectWrapperController) CheckCsrf() bool {
	return c.callBoolMethod("CheckCsrf")
}
func (c *reflectWrapperController) Prepare() {
	c.callVoidMethod("Prepare")
}
func (c *reflectWrapperController) Finish() {
	c.callVoidMethod("Finish")
}

func (c *reflectWrapperController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("reflectWrapperController.ServeHTTP\n")
	CallMatchedMethod(c)
}

func WrapperControllerHandler(vc reflect.Value, context *context.Context) *reflectWrapperController {
	fmt.Printf("WrapperControllerHandler\n")
	return &reflectWrapperController{vc: vc, context: context}
}

type WrapperController struct {
	Controller
}

func (c *WrapperController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("WrapperController\n")
	CallMatchedMethod(c.Controller)
}

func CallMatchedMethod(c Controller) {
	fmt.Printf("CallMatchedMethod\n")
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
