package context

import (
	"net/http"
	"strings"
)

type Context struct {
	R          *http.Request
	W          http.ResponseWriter
	EnableGzip bool
	Param      map[string]string
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
