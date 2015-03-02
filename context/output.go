package context

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func (ctx *Context) SetHeader(key, val string) {
	ctx.W.Header().Set(key, val)
}

func (ctx *Context) write_gzip(content []byte) {
	output_writer := ctx.W.(io.Writer)
	splitted := strings.SplitN(ctx.GetHeader("Accept-Encoding"), ",", -1)
	encodings := make([]string, len(splitted))

	for i, val := range splitted {
		encodings[i] = strings.TrimSpace(val)
	}
	for _, val := range encodings {
		if val == "gzip" {
			ctx.SetHeader("Content-Encoding", "gzip")
			output_writer, _ = gzip.NewWriterLevel(ctx.W, gzip.BestSpeed)
			break
		} else if val == "deflate" {
			ctx.SetHeader("Content-Encoding", "deflate")
			output_writer, _ = flate.NewWriter(ctx.W, flate.BestSpeed)
			break
		}
	}
	output_writer.Write(content)
	switch output_writer.(type) {
	case *gzip.Writer:
		output_writer.(*gzip.Writer).Close()
	case *flate.Writer:
		output_writer.(*flate.Writer).Close()
		/*	case io.WriteCloser:
			output_writer.(io.WriteCloser).Close()*/
	}
}

func (ctx *Context) write(content []byte) {
	ctx.SetHeader("Content-Length", strconv.Itoa(len(content)))
	ctx.W.Write(content)
}

func (ctx *Context) Body(content []byte) {
	if ctx.EnableGzip == true && ctx.GetHeader("Accept-Encoding") != "" {
		ctx.write_gzip(content)
	} else {
		ctx.write(content)
	}
	ctx.SetWritten()
}

/*
func (ctx *Context) SetCookie(name string, value string, others ...interface{}) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s=%s", sanitizeName(name), sanitizeValue(value))
	if len(others) > 0 {
		switch others[0].(type) {
		case int:
			if others[0].(int) > 0 {
				fmt.Fprintf(&b, "; Max-Age=%d", others[0].(int))
			} else if others[0].(int) < 0 {
				fmt.Fprintf(&b, "; Max-Age=0")
			}
		case int64:
			if others[0].(int64) > 0 {
				fmt.Fprintf(&b, "; Max-Age=%d", others[0].(int64))
			} else if others[0].(int64) < 0 {
				fmt.Fprintf(&b, "; Max-Age=0")
			}
		case int32:
			if others[0].(int32) > 0 {
				fmt.Fprintf(&b, "; Max-Age=%d", others[0].(int32))
			} else if others[0].(int32) < 0 {
				fmt.Fprintf(&b, "; Max-Age=0")
			}
		}
	}
	if len(others) > 1 {
		fmt.Fprintf(&b, "; Path=%s", sanitizeValue(others[1].(string)))
	}
	if len(others) > 2 {
		fmt.Fprintf(&b, "; Domain=%s", sanitizeValue(others[2].(string)))
	}
	if len(others) > 3 {
		fmt.Fprintf(&b, "; Secure")
	}
	if len(others) > 4 {
		fmt.Fprintf(&b, "; HttpOnly")
	}
	ctx.W.Header().Add("Set-Cookie", b.String())
}
*/

var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")

func sanitizeName(n string) string {
	return cookieNameSanitizer.Replace(n)
}

var cookieValueSanitizer = strings.NewReplacer("\n", " ", "\r", " ", ";", " ")

func sanitizeValue(v string) string {
	return cookieValueSanitizer.Replace(v)
}

func (ctx *Context) Json(data interface{}, hasIndent bool, coding bool) error {
	ctx.SetHeader("Content-Type", "application/json;charset=UTF-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = json.MarshalIndent(data, "", "  ")
	} else {
		content, err = json.Marshal(data)
	}
	if err != nil {
		http.Error(ctx.W, err.Error(), http.StatusInternalServerError)
		return err
	}
	if coding {
		content = []byte(stringsToJson(string(content)))
	}
	ctx.Body(content)
	return nil
}

func (ctx *Context) Jsonp(data interface{}, hasIndent bool) error {
	ctx.SetHeader("Content-Type", "application/javascript;charset=UTF-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = json.MarshalIndent(data, "", "  ")
	} else {
		content, err = json.Marshal(data)
	}
	if err != nil {
		http.Error(ctx.W, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := ctx.GetString("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback_content := bytes.NewBufferString(callback)
	callback_content.WriteString("(")
	callback_content.Write(content)
	callback_content.WriteString(");\r\n")
	ctx.Body(callback_content.Bytes())
	return nil
}

func (ctx *Context) Xml(data interface{}, hasIndent bool) error {
	ctx.SetHeader("Content-Type", "application/xml;charset=UTF-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = xml.MarshalIndent(data, "", "  ")
	} else {
		content, err = xml.Marshal(data)
	}
	if err != nil {
		http.Error(ctx.W, err.Error(), http.StatusInternalServerError)
		return err
	}
	ctx.Body(content)
	return nil
}

func (ctx *Context) Download(file string) {
	ctx.SetHeader("Content-Description", "File Transfer")
	ctx.SetHeader("Content-Type", "application/octet-stream")
	ctx.SetHeader("Content-Disposition", "attachment; filename="+filepath.Base(file))
	ctx.SetHeader("Content-Transfer-Encoding", "binary")
	ctx.SetHeader("Expires", "0")
	ctx.SetHeader("Cache-Control", "must-revalidate")
	ctx.SetHeader("Pragma", "public")
	http.ServeFile(ctx.W, ctx.R, file)
	ctx.SetWritten()
}

func (ctx *Context) ContentType(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		ctx.SetHeader("Content-Type", ctype)
	}
}

func (ctx *Context) SetStatus(status int) {
	ctx.W.WriteHeader(status)
	ctx.SetWritten()
}

func (ctx *Context) SetWritten() {
	ctx.written = true
}

func (ctx *Context) Written() bool {
	return ctx.written
}

/*
func (ctx *Context) IsCachable(status int) bool {
	return ctx.Status >= 200 && ctx.Status < 300 || ctx.Status == 304
}

func (ctx *Context) IsEmpty(status int) bool {
	return ctx.Status == 201 || ctx.Status == 204 || ctx.Status == 304
}

func (ctx *Context) IsOk(status int) bool {
	return ctx.Status == 200
}

func (ctx *Context) IsSuccessful(status int) bool {
	return ctx.Status >= 200 && ctx.Status < 300
}

func (ctx *Context) IsRedirect(status int) bool {
	return ctx.Status == 301 || ctx.Status == 302 || ctx.Status == 303 || ctx.Status == 307
}

func (ctx *Context) IsForbidden(status int) bool {
	return ctx.Status == 403
}

func (ctx *Context) IsNotFound(status int) bool {
	return ctx.Status == 404
}

func (ctx *Context) IsClientError(status int) bool {
	return ctx.Status >= 400 && ctx.Status < 500
}

func (ctx *Context) IsServerError(status int) bool {
	return ctx.Status >= 500 && ctx.Status < 600
}
*/

func stringsToJson(str string) string {
	rs := []rune(str)
	jsons := ""
	for _, r := range rs {
		rint := int(r)
		if rint < 128 {
			jsons += string(r)
		} else {
			jsons += "\\u" + strconv.FormatInt(int64(rint), 16) // json
		}
	}
	return jsons
}
