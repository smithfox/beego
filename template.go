//copy from https://github.com/codegangsta/martini-contrib/blob/master/render/render.go
package beego

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Included helper functions for use when rendering html
var helperFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield called with no layout defined")
	},
}

var gRenderer *renderer

// Delims represents a set of Left and Right delimiters for HTML template rendering
type Delims struct {
	// Left delimiter, defaults to {{
	Left string
	// Right delimiter, defaults to }}
	Right string
}

// Options is a struct for specifying configuration options for the render.Renderer middleware
type Options struct {
	// Directory to load templates. Default is "templates"
	Directory string
	// Layout template name. Will not render a layout if "". Defaults to "".
	Layout string
	// Extensions to parse template files from. Defaults to [".tmpl"]
	Extensions []string
	// Funcs is a slice of FuncMaps to apply to the template upon compilation. This is useful for helper functions. Defaults to [].
	Funcs []template.FuncMap
	// Delims sets the action delimiters to the specified strings in the Delims struct.
	Delims Delims
	// Appends the given charset to the Content-Type header. Default is "UTF-8".
	Charset string
	// Outputs human readable JSON
	IndentJSON bool
}

// RenderOptions is a struct for overriding some rendering Options
type RenderOptions struct {
	// Layout template name. Overrides Options.Layout.
	Layout string
}

func buildFuncs() {
	helperFuncs["dateformat"] = DateFormat
	helperFuncs["date"] = Date
	helperFuncs["compare"] = Compare
	helperFuncs["substr"] = Substr
	helperFuncs["html2str"] = Html2str
	helperFuncs["str2html"] = Str2html
	helperFuncs["htmlquote"] = Htmlquote
	helperFuncs["htmlunquote"] = Htmlunquote
	helperFuncs["renderform"] = RenderForm
}

// AddFuncMap let user to register a func in the template
func AddFuncMap(key string, funname interface{}) error {
	if _, ok := helperFuncs[key]; ok {
		return errors.New("funcmap already has the key")
	}
	helperFuncs[key] = funname
	return nil
}

func BuildTemplate(dir string) error {
	var opt Options
	opt.Directory = dir
	BuildTemplateWithOption(opt)
	return nil
}

func BuildTemplateWithOption(options ...Options) {
	buildFuncs()
	opt := prepareOptions(options)
	t := compile(opt)
	gRenderer = &renderer{t, opt}
}

func RenderTemplate(tplName string, binding interface{}, renderOpt ...RenderOptions) (*bytes.Buffer, error) {
	return gRenderer.renderhtml(tplName, binding, renderOpt...)
}

func prepareOptions(options []Options) Options {
	var opt Options
	if len(options) > 0 {
		opt = options[0]
	}

	// Defaults
	if len(opt.Directory) == 0 {
		opt.Directory = "template"
	}
	if len(opt.Extensions) == 0 {
		opt.Extensions = []string{".tpl"}
	}

	return opt
}

func compile(options Options) *template.Template {
	dir := options.Directory
	t := template.New(dir)
	t.Delims(options.Delims.Left, options.Delims.Right)
	// parse an initial template in case we don't have any
	template.Must(t.Parse("Martini"))

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		r, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := filepath.Ext(r)
		for _, extension := range options.Extensions {
			if ext == extension {

				buf, err := ioutil.ReadFile(path)
				if err != nil {
					panic(err)
				}

				name := (r[0 : len(r)-len(ext)])
				fname := filepath.ToSlash(r)
				name = filepath.ToSlash(name)
				tmpl := template.New(name)

				// add our funcmaps
				for _, funcs := range options.Funcs {
					tmpl.Funcs(funcs)
				}

				template.Must(tmpl.Funcs(helperFuncs).Parse(string(buf)))

				t.AddParseTree(name, tmpl.Tree)
				t.AddParseTree(fname, tmpl.Tree)
				/*
					tmpl := t.New()

					// add our funcmaps
					for _, funcs := range options.Funcs {
						tmpl.Funcs(funcs)
					}

					// Bomb out if parse fails. We don't want any silent server starts.
					template.Must(tmpl.Funcs(helperFuncs).Parse(string(buf)))
				*/
				break
			}
		}

		return nil
	})

	for _, funcs := range options.Funcs {
		t.Funcs(funcs)
	}

	t.Funcs(helperFuncs)

	return t
}

type renderer struct {
	t   *template.Template
	opt Options
}

func (r *renderer) renderhtml(name string, binding interface{}, renderOpt ...RenderOptions) (*bytes.Buffer, error) {
	opt := r.prepareRenderOptions(renderOpt)
	// assign a layout if there is one
	if len(opt.Layout) > 0 {
		r.addYield(name, binding)
		name = opt.Layout
	}

	return r.execute(name, binding)
}

func (r *renderer) Template() *template.Template {
	return r.t
}

func (r *renderer) execute(name string, binding interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	return buf, r.t.ExecuteTemplate(buf, name, binding)
}

func (r *renderer) addYield(name string, binding interface{}) {
	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf, err := r.execute(name, binding)
			// return safe html here since we are rendering our own template
			return template.HTML(buf.String()), err
		},
	}
	r.t.Funcs(funcs)
}

func (r *renderer) prepareRenderOptions(renderOpt []RenderOptions) RenderOptions {
	if len(renderOpt) > 0 {
		return renderOpt[0]
	}

	return RenderOptions{
		Layout: r.opt.Layout,
	}
}
