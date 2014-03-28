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
	written    bool
}

//301
//永久重定向,告诉客户端以后应从新地址访问,会影响SEO
func (ctx *Context) RedirectPermanently(redirectURL string) {
	http.Redirect(ctx.W, ctx.R, redirectURL, http.StatusMovedPermanently)
	ctx.SetWritten()
}

//302
//作为HTTP1.0的标准,以前叫做Moved Temporarily ,现在叫Found. 现在使用只是为了兼容性的处理
//HTTP 1.1 有303 和307作为详细的补充,其实是对302的细化
func (ctx *Context) RedirectFound(redirectURL string) {
	http.Redirect(ctx.W, ctx.R, redirectURL, http.StatusFound)
	ctx.SetWritten()
}

//303
//对于POST请求，它表示请求已经被服务端处理，客户端可以接着使用GET方法去请求Location里的URI
func (ctx *Context) RedirectSeeOther(redirectURL string) {
	http.Redirect(ctx.W, ctx.R, redirectURL, http.StatusSeeOther)
	ctx.SetWritten()
}

//307
//对于POST请求，表示请求还没有被处理，客户端会向Location里的URI重新发起POST请求
//意味着 POST 请求会被再次发送到服务端
func (ctx *Context) RedirectTemporary(redirectURL string) {
	http.Redirect(ctx.W, ctx.R, redirectURL, http.StatusTemporaryRedirect)
	ctx.SetWritten()
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
	return ctx.R.FormValue(key)
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
	return strconv.ParseInt(ctx.R.FormValue(key), 10, 64)
}

func (ctx *Context) GetBool(key string) (bool, error) {
	return strconv.ParseBool(ctx.R.FormValue(key))
}

func (ctx *Context) GetFloat(key string) (float64, error) {
	return strconv.ParseFloat(ctx.R.FormValue(key), 64)
}

func (ctx *Context) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.R.FormFile(key)
}

func (ctx *Context) GetHeader(key string) string {
	return ctx.R.Header.Get(key)
}

//
// func (ctx *Context) GetQuery(key string) string {
//实现有问题, 应该ctx.R.URL.Query().Get(key)
// 	return ctx.R.Form.Get(key)
// }

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

func (ctx *Context) DecodeForm(dst interface{}) error {
	return gGorillaDecoder.Decode(dst, ctx.GetForm())
}

func (ctx *Context) DecodePostForm(dst interface{}) error {
	return gGorillaDecoder.Decode(dst, ctx.GetPostForm())
}
