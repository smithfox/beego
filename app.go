package beego

import (
	"fmt"
	"github.com/smithfox/beego/context"
	"net/http"
	"path/filepath"
	"time"
)

//return false means started=true, need not continue
type RawHttpFilterFunc func(http.ResponseWriter, *http.Request) bool

type FilterFunc func(*context.Context)

type App struct {
	Handlers *ControllerRegistor
}

// New returns a new PatternServeMux.
func NewApp() *App {
	cr := NewControllerRegistor()
	app := &App{Handlers: cr}
	return app
}

func (app *App) RunHttps() {
	addr := HttpAddr

	if HttpsPort != 0 {
		addr = fmt.Sprintf("%s:%d", HttpAddr, HttpsPort)
	}

	s := &http.Server{
		Addr:         addr,
		Handler:      app.Handlers,
		ReadTimeout:  time.Duration(HttpServerTimeOut) * time.Second,
		WriteTimeout: time.Duration(HttpServerTimeOut) * time.Second,
	}

	certfile := filepath.Join(AppPath, filepath.Clean(HttpCertFile))
	keyfile := filepath.Join(AppPath, filepath.Clean(HttpKeyFile))

	certfile = filepath.FromSlash(certfile)
	keyfile = filepath.FromSlash(keyfile)

	fmt.Printf("certfile=%s,keyfile=%s\n", certfile, keyfile)
	err := s.ListenAndServeTLS(certfile, keyfile)
	if err != nil {
		panic(err)
	}
}

func (app *App) RunHttp() {
	addr := HttpAddr

	if HttpPort != 0 {
		addr = fmt.Sprintf("%s:%d", HttpAddr, HttpPort)
	}

	s := &http.Server{
		Addr:         addr,
		Handler:      app.Handlers,
		ReadTimeout:  time.Duration(HttpServerTimeOut) * time.Second,
		WriteTimeout: time.Duration(HttpServerTimeOut) * time.Second,
	}

	err := s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (app *App) Router(path string, c ControllerInterface, mappingMethods ...string) *App {
	app.Handlers.Add(path, c, mappingMethods...)
	return app
}

func (app *App) SetRawFilter(rawfilter RawHttpFilterFunc) *App {
	app.Handlers.SetRawFilter(rawfilter)
	return app
}

func (app *App) SetViewsPath(path string) *App {
	ViewsPath = path
	return app
}

func (app *App) SetStaticPath(url string, path string) *App {
	StaticDir[url] = path
	return app
}

func (app *App) DelStaticPath(url string) *App {
	delete(StaticDir, url)
	return app
}
