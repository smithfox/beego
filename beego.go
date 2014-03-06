package beego

import (
	"fmt"
	"github.com/smithfox/beego/middleware"
	"github.com/smithfox/beego/session"
	"net/http"
	"path"
)

const VERSION = "0.9.9"

func Router(rootpath string, c ControllerInterface, mappingMethods ...string) *App {
	BeeApp.Router(rootpath, c, mappingMethods...)
	return BeeApp
}

func Errorhandler(err string, h http.HandlerFunc) *App {
	middleware.Errorhandler(err, h)
	return BeeApp
}

func SetViewsPath(path string) *App {
	BeeApp.SetViewsPath(path)
	return BeeApp
}

func SetStaticPath(url string, path string) *App {
	StaticDir[url] = path
	return BeeApp
}

func DelStaticPath(url string) *App {
	delete(StaticDir, url)
	return BeeApp
}

func SetRawFilter(rawfilter RawHttpFilterFunc) *App {
	BeeApp.SetRawFilter(rawfilter)
	return BeeApp
}

func Prepare() error {
	//if AppConfigPath not In the conf/app.conf reParse config
	if AppConfigPath != path.Join(AppPath, "conf", "app.conf") {
		err := ParseConfig()
		if err != nil {
			fmt.Printf("Beego ParseConfig err=%v\n", err)
		}
	}

	if SessionOn {
		GlobalSessions, _ = session.NewManager(SessionProvider, SessionName, SessionGCMaxLifetime, SessionSavePath, HttpTLS)
		go GlobalSessions.GC()
	}

	err := BuildTemplate(ViewsPath)
	if err != nil {
		fmt.Printf("Beego BuildTemplate err=%v\n", err)
		return err
	}

	middleware.VERSION = VERSION
	middleware.AppName = AppName
	middleware.RegisterErrorHander()
	return nil
}

func RunHttp() {
	BeeApp.RunHttp()
}

func RunHttps() {
	BeeApp.RunHttps()
}
