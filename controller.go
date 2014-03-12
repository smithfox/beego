package beego

import (
	//"bytes"
	// "crypto/hmac"
	// "crypto/sha1"
	// "encoding/base64"
	//"errors"
	"fmt"
	"github.com/smithfox/beego/context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	// "strconv"
	// "strings"
	// "time"
)

type Controller struct {
	Ctx         *context.Context
	Data        map[interface{}]interface{}
	TplNames    string
	TplExt      string
	_xsrf_token string
	XSRFExpire  int
}

func (c *Controller) Init(ctx *context.Context) {
	c.Data = make(map[interface{}]interface{})
	c.TplNames = ""
	c.Ctx = ctx
	c.TplExt = "tpl"
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

func (c *Controller) Render() error {
	rb, err := c.RenderBytes()

	if err != nil {
		return err
	} else {
		//DENY ： 不允许被任何页面嵌入；
		//SAMEORIGIN ： 不允许被本域以外的页面嵌入；
		//ALLOW-FROM uri： 不允许被指定的域名以外的页面嵌入（Chrome现阶段不支持）
		c.Ctx.SetHeader("X-Frame-Options", "DENY")
		c.Ctx.SetHeader("X-XSS-Protection", "1; mode=block")
		c.Ctx.SetHeader("X-Content-Type-Options", "nosniff")
		c.Ctx.SetHeader("Content-Type-Options", "nosniff")
		//c.Ctx.SetHeader("Content-Security-Policy", "default-src 'self'")

		c.Ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
		c.Ctx.Body(rb)
	}
	return nil
}

/*
func (c *Controller) RenderString() (string, error) {
	b, e := c.RenderBytes()
	return string(b), e
}
*/

func (c *Controller) RenderBytes() ([]byte, error) {
	// if c.TplNames == "" {
	// 	c.TplNames = c.ChildName + "/" + strings.ToLower(c.Ctx.R.Method) + "." + c.TplExt
	// }

	ibytes, err := RenderTemplate(c.TplNames, c.Data)
	if err != nil {
		fmt.Printf("Beego ExecuteTemplate %s err=%v\n", c.TplNames, err)
	}
	icontent, _ := ioutil.ReadAll(ibytes)
	return icontent, nil
}

func (c *Controller) ServeJson(encoding ...bool) {
	var hasIndent bool
	var hasencoding bool
	if RunMode == "prod" {
		hasIndent = false
	} else {
		hasIndent = true
	}
	if len(encoding) > 0 && encoding[0] == true {
		hasencoding = true
	}
	c.Ctx.Json(c.Data["json"], hasIndent, hasencoding)
}

func (c *Controller) ServeJsonp() {
	var hasIndent bool
	if RunMode == "prod" {
		hasIndent = false
	} else {
		hasIndent = true
	}
	c.Ctx.Jsonp(c.Data["jsonp"], hasIndent)
}

func (c *Controller) ServeXml() {
	var hasIndent bool
	if RunMode == "prod" {
		hasIndent = false
	} else {
		hasIndent = true
	}
	c.Ctx.Xml(c.Data["xml"], hasIndent)
}

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

/*
func (c *Controller) StartSession() session.SessionStore {
	// if c.Session == nil {
	// 	c.Session = c.Ctx.Input.CruSession
	// }
	return c.Session
}

func (c *Controller) SetSession(name interface{}, value interface{}) {
	// if c.Session == nil {
	// 	c.StartSession()
	// }
	c.Session.Set(name, value)
}

func (c *Controller) GetSession(name interface{}) interface{} {
	// if c.Session == nil {
	// 	c.StartSession()
	// }
	return c.Session.Get(name)
}

func (c *Controller) DelSession(name interface{}) {
	// if c.Session == nil {
	// 	c.StartSession()
	// }
	c.Session.Delete(name)
}
*/

/*
func (c *Controller) IsAjax() bool {
	return c.Ctx.Input.IsAjax()
}
*/
/*
func (c *Controller) GetSecureCookie(Secret, key string) (string, bool) {
	val := c.Ctx.GetCookie(key)
	if val == "" {
		return "", false
	}

	parts := strings.SplitN(val, "|", 3)

	if len(parts) != 3 {
		return "", false
	}

	vs := parts[0]
	timestamp := parts[1]
	sig := parts[2]

	h := hmac.New(sha1.New, []byte(Secret))
	fmt.Fprintf(h, "%s%s", vs, timestamp)

	if fmt.Sprintf("%02x", h.Sum(nil)) != sig {
		return "", false
	}
	res, _ := base64.URLEncoding.DecodeString(vs)
	return string(res), true
}

func (c *Controller) SetSecureCookie(Secret, name, val string, age int64) {
	vs := base64.URLEncoding.EncodeToString([]byte(val))
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	h := hmac.New(sha1.New, []byte(Secret))
	fmt.Fprintf(h, "%s%s", vs, timestamp)
	sig := fmt.Sprintf("%02x", h.Sum(nil))
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	c.Ctx.SetCookie(name, cookie, age, "/")
}
*/

/*
func (c *Controller) XsrfToken() string {
	if c._xsrf_token == "" {
		token, ok := c.GetSecureCookie(XSRFKEY, "_xsrf")
		if !ok {
			var expire int64
			if c.XSRFExpire > 0 {
				expire = int64(c.XSRFExpire)
			} else {
				expire = int64(XSRFExpire)
			}
			token = GetRandomString(15)
			c.SetSecureCookie(XSRFKEY, "_xsrf", token, expire)
		}
		c._xsrf_token = token
	}
	return c._xsrf_token
}

func (c *Controller) CheckXsrfCookie() bool {
	token := c.Ctx.GetString("_xsrf")
	if token == "" {
		token = c.Ctx.GetHeader("X-Xsrftoken")
	}
	if token == "" {
		token = c.Ctx.GetHeader("X-Csrftoken")
	}
	if token == "" {
		c.Ctx.SetStatus(403)
		c.Ctx.Body([]byte("'_xsrf' argument missing from POST"))
	} else if c._xsrf_token != token {
		c.Ctx.SetStatus(403)
		c.Ctx.Body([]byte("XSRF cookie does not match POST argument"))
	}
	return true
}

func (c *Controller) XsrfFormHtml() string {
	return "<input type=\"hidden\" name=\"_xsrf\" value=\"" +
		c._xsrf_token + "\"/>"
}
*/
