package context

import (
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Context struct {
	R          *http.Request
	W          http.ResponseWriter
	EnableGzip bool
	Param      map[string]string
	formParsed bool
	//Status         int
}

func (ctx *Context) Redirect(status int, localurl string) {
	//ctx.Status = status
	http.Redirect(ctx.W, ctx.R, localurl, status)
}

func (ctx *Context) WriteString(content string) {
	ctx.Body([]byte(content))
}

func (ctx *Context) GetCookie(key string) string {
	ck, err := ctx.R.Cookie(key)
	if err != nil {
		return ""
	}
	return ck.Value
}

func (ctx *Context) _parseform() {
	if !ctx.formParsed {
		ct := ctx.R.Header.Get("Content-Type")
		if strings.Contains(ct, "multipart/form-data") {
			var max_memory int64 = (1 << 26) //64MB
			ctx.R.ParseMultipartForm(max_memory)
		} else {
			ctx.R.ParseForm()
		}
		ctx.formParsed = true
	}
}

func (ctx *Context) GetPostForm() url.Values {
	ctx._parseform()
	return ctx.R.PostForm
}

func (ctx *Context) GetForm() url.Values {
	ctx._parseform()
	return ctx.R.Form
}

/*func (ctx *Context) ParseForm(obj interface{}) error {
	return ParseForm(ctx.GetForm(), obj)
}*/

func (ctx *Context) GetString(key string) string {
	return ctx.GetForm().Get(key)
}

func (ctx *Context) GetStrings(key string) []string {
	r := ctx.R
	if r.Form == nil {
		return []string{}
	}
	vs := r.Form[key]
	if len(vs) > 0 {
		return vs
	}
	return []string{}
}

func (ctx *Context) GetInt(key string) (int64, error) {
	return strconv.ParseInt(ctx.GetForm().Get(key), 10, 64)
}

func (ctx *Context) GetBool(key string) (bool, error) {
	return strconv.ParseBool(ctx.GetForm().Get(key))
}

func (ctx *Context) GetFloat(key string) (float64, error) {
	return strconv.ParseFloat(ctx.GetForm().Get(key), 64)
}

func (ctx *Context) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.R.FormFile(key)
}

func (ctx *Context) GetHeader(key string) string {
	return ctx.R.Header.Get(key)
}

func (ctx *Context) GetQuery(key string) string {
	return ctx.R.Form.Get(key)
}

func (ctx *Context) IsAjax() bool {
	return ctx.GetHeader("X-Requested-With") == "XMLHttpRequest"
}

func (ctx *Context) IsWebsocket() bool {
	return ctx.GetHeader("Upgrade") == "websocket"
}

func (ctx *Context) IsUpload() bool {
	return ctx.R.MultipartForm != nil
}

func (ctx *Context) Proxy() []string {
	if ips := ctx.GetHeader("HTTP_X_FORWARDED_FOR"); ips != "" {
		return strings.Split(ips, ",")
	}
	return []string{}
}

func (ctx *Context) IP() string {
	ips := ctx.Proxy()
	if len(ips) > 0 && ips[0] != "" {
		return ips[0]
	}
	ip := strings.Split(ctx.R.RemoteAddr, ":")
	if len(ip) == 2 {
		return ip[0]
	}
	return "127.0.0.1"
}

func (ctx *Context) GetURLRouterParam(key string) string {
	if v, ok := ctx.Param[key]; ok {
		return v
	}
	return ""
}
