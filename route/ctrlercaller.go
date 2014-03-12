package route

import (
	"github.com/smithfox/beego/context"
	//"net/http"
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

var noparams []reflect.Value = []reflect.Value{}

func callVoidMethod(vc reflect.Value, name string) {
	method := vc.MethodByName(name)
	method.Call(noparams)
}

func callBoolMethod(vc reflect.Value, name string) bool {
	method := vc.MethodByName(name)
	out := method.Call(noparams)
	ff := out[0].Interface().(bool)
	return ff
}

func controllerHTTP(match *RouteMatch, context *context.Context) {
	r := context.R
	//Invoke the request handler
	vc := reflect.New(match.CType)

	//call the controller init function
	method := vc.MethodByName("Init")
	in := make([]reflect.Value, 1)
	in[0] = reflect.ValueOf(context)
	method.Call(in)

	//call prepare function
	in = make([]reflect.Value, 0)
	method = vc.MethodByName("Prepare")
	method.Call(in)

	if match.CheckAuth {
		passed := callBoolMethod(vc, "CheckAuth")
		if !passed {
			return
		}
	}

	if r.Method == "GET" {
		if match.CheckCsrf {
			passed := callBoolMethod(vc, "CheckCsrf")
			if !passed {
				return
			}
		}
		callVoidMethod(vc, "Get")
	} else if r.Method == "HEAD" {
		callVoidMethod(vc, "Head")
	} else if r.Method == "DELETE" || (r.Method == "POST" && r.Form.Get("_method") == "delete") {
		if !callBoolMethod(vc, "CheckCsrf") {
			return
		}
		callVoidMethod(vc, "Delete")
	} else if r.Method == "PUT" || (r.Method == "POST" && r.Form.Get("_method") == "put") {
		if !callBoolMethod(vc, "CheckCsrf") {
			return
		}
		callVoidMethod(vc, "Put")
	} else if r.Method == "POST" {
		if !callBoolMethod(vc, "CheckCsrf") {
			return
		}
		callVoidMethod(vc, "Post")
	} else if r.Method == "PATCH" {
		callVoidMethod(vc, "Patch")
	} else if r.Method == "OPTIONS" {
		callVoidMethod(vc, "Options")
	}

	callVoidMethod(vc, "Finish")
}
