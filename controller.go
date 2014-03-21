package beego

import (
	"github.com/smithfox/beego/context"
	"net/http"
)

type Controller struct {
	Ctx     *context.Context
	_inited bool
}

func (c *Controller) Init(ctx *context.Context) {
	if c._inited {
		return
	}
	c.Ctx = ctx
	c._inited = true
}

func (c *Controller) Context() *context.Context {
	return c.Ctx
}

func (c *Controller) Prepare() {

}
func (c *Controller) CheckAuth() bool {
	return true
}
func (c *Controller) CheckCsrf() bool {
	return true
}
func (c *Controller) Finish() {

}

func (c *Controller) Get() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Post() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Delete() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Put() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Head() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Patch() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

func (c *Controller) Options() {
	http.Error(c.Ctx.W, "Method Not Allowed", 405)
}

/*
func (c *Controller) SaveToFile(fromfile, tofile string) error {
	file, _, err := c.Ctx.R.FormFile(fromfile)
	if err != nil {
		return err
	}
	defer file.Close()
	f, err := os.OpenFile(tofile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	io.Copy(f, file)
	return nil
}
*/
