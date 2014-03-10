package route

import (
	beecontext "github.com/smithfox/beego/context"
	"net/http"
	"reflect"
)

/*
type HandlerController struct {
	ci      ControllerInterface
	context *beecontext.Context
}

func (c *HandlerController) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	if c.ci == nil {
		return
	}
	c.ci.Init(c.context)
	c.ci.Prepare()

	if r.Method == "GET" {
		c.ci.Get()
	} else if r.Method == "HEAD" {
		c.ci.Head()
	} else if r.Method == "DELETE" || (r.Method == "POST" && r.Form.Get("_method") == "delete") {
		c.ci.Delete()
	} else if r.Method == "PUT" || (r.Method == "POST" && r.Form.Get("_method") == "put") {
		c.ci.Put()
	} else if r.Method == "POST" {
		c.ci.Post()
	} else if r.Method == "PATCH" {
		c.ci.Patch()
	} else if r.Method == "OPTIONS" {
		c.ci.Options()
	}

	c.ci.Finish()
}


func (c *HandlerController) Prepare() {
	if c.ci != nil {
		c.ci.Prepare()
	}
}

func (c *HandlerController) Finish() {
	if c.ci != nil {
		c.ci.Finish()
	}
}

func (c *HandlerController) Get() {
	if c.ci != nil {
		c.ci.Get()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Post() {
	if c.ci != nil {
		c.ci.Post()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Delete() {
	if c.ci != nil {
		c.ci.Delete()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Put() {
	if c.ci != nil {
		c.ci.Put()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Head() {
	if c.ci != nil {
		c.ci.Head()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Patch() {
	if c.ci != nil {
		c.ci.Patch()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}

func (c *HandlerController) Options() {
	if c.ci != nil {
		c.ci.Options()
	} else {
		http.Error(c.Ctx.W, "Method Not Allowed", 405)
	}
}
*/

func controllerHTTP(cType reflect.Type, params map[string]string, w http.ResponseWriter, r *http.Request) {
	context := &beecontext.Context{
		W: w,
		R: r,
	}
	//context.EnableGzip = EnableGzip
	context.EnableGzip = true
	context.Param = params

	//Invoke the request handler
	vc := reflect.New(cType)

	//call the controller init function
	method := vc.MethodByName("Init")
	in := make([]reflect.Value, 1)
	in[0] = reflect.ValueOf(context)
	method.Call(in)

	//call prepare function
	in = make([]reflect.Value, 0)
	method = vc.MethodByName("Prepare")
	method.Call(in)

	if r.Method == "GET" {
		method = vc.MethodByName("Get")
		method.Call(in)
	} else if r.Method == "HEAD" {
		method = vc.MethodByName("Head")
		method.Call(in)
	} else if r.Method == "DELETE" || (r.Method == "POST" && r.Form.Get("_method") == "delete") {
		method = vc.MethodByName("Delete")
		method.Call(in)
	} else if r.Method == "PUT" || (r.Method == "POST" && r.Form.Get("_method") == "put") {
		method = vc.MethodByName("Put")
		method.Call(in)
	} else if r.Method == "POST" {
		method = vc.MethodByName("Post")
		method.Call(in)
	} else if r.Method == "PATCH" {
		method = vc.MethodByName("Patch")
		method.Call(in)
	} else if r.Method == "OPTIONS" {
		method = vc.MethodByName("Options")
		method.Call(in)
	}

	method = vc.MethodByName("Finish")
	method.Call(in)
}
