package beego

import (
	"github.com/smithfox/beego/config"
	"html/template"
	"os"
	"path"
	"strconv"
)

var (
	AppName       string
	AppPath       string
	AppConfigPath string
	TemplateCache map[string]*template.Template
	HttpAddr      string
	HttpPort      int
	HttpsPort     int
	HttpTLS       bool
	HttpCertFile  string
	HttpKeyFile   string
	RecoverPanic  bool
	AutoRender    bool
	PprofOn       bool
	ViewsPath     string
	RunMode       string //"dev" or "prod"
	AppConfig     config.ConfigContainer
	//related to session
	SessionOn            bool   // whether auto start session,default is false
	SessionProvider      string // default session provider  memory mysql redis
	SessionName          string // sessionName cookie's name
	SessionGCMaxLifetime int64  // session's gc maxlifetime
	SessionSavePath      string // session savepath if use mysql/redis/file this set to the connectinfo

	MaxMemory         int64
	EnableGzip        bool   // enable gzip
	DirectoryIndex    bool   //enable DirectoryIndex default is false
	HttpServerTimeOut int64  //set httpserver timeout
	ErrorsShow        bool   //set weather show errors
	XSRFKEY           string //set XSRF
	EnableXSRF        bool
	XSRFExpire        int
	TemplateLeft      string
	TemplateRight     string
)

func init() {
	AppPath, _ = os.Getwd()
	TemplateCache = make(map[string]*template.Template)
	HttpAddr = ""
	HttpPort = 8080
	HttpsPort = 4043
	AppName = "beego"
	RunMode = "dev" //default runmod
	AutoRender = true
	RecoverPanic = true
	PprofOn = false
	ViewsPath = "template"
	SessionOn = false
	SessionProvider = "memory"
	SessionName = "beegosessionID"
	SessionGCMaxLifetime = 3600
	SessionSavePath = ""
	MaxMemory = 1 << 26 //64MB
	EnableGzip = false
	AppConfigPath = path.Join(AppPath, "conf", "app.conf")
	HttpServerTimeOut = 0
	ErrorsShow = true
	XSRFKEY = "beegoxsrf"
	XSRFExpire = 0
	TemplateLeft = "{{"
	TemplateRight = "}}"
	//ParseConfig()
	//runtime.GOMAXPROCS(runtime.NumCPU())
}

func ParseConfig() (err error) {
	AppConfig, err = config.NewConfig("ini", AppConfigPath)
	if err != nil {
		return err
	} else {
		HttpAddr = AppConfig.String("http.addr")
		if v, err := AppConfig.Int("http.port"); err == nil {
			HttpPort = v
		}
		if timeout, err := AppConfig.Int64("http.servertimeout"); err == nil {
			HttpServerTimeOut = timeout
		}
		if httptls, err := AppConfig.Bool("https"); err == nil {
			HttpTLS = httptls
		}
		if v, err := AppConfig.Int("https.port"); err == nil {
			HttpsPort = v
		}
		if certfile := AppConfig.String("https.certfile"); certfile != "" {
			HttpCertFile = certfile
		}
		if keyfile := AppConfig.String("https.keyfile"); keyfile != "" {
			HttpKeyFile = keyfile
		}
		if sessionon, err := AppConfig.Bool("session"); err == nil {
			SessionOn = sessionon
		}
		if sessProvider := AppConfig.String("session.provider"); sessProvider != "" {
			SessionProvider = sessProvider
		}
		if sessName := AppConfig.String("session.name"); sessName != "" {
			SessionName = sessName
		}
		if sesssavepath := AppConfig.String("session.savepath"); sesssavepath != "" {
			SessionSavePath = sesssavepath
		}
		if sessMaxLifeTime, err := AppConfig.Int("session.gcmaxlifetime"); err == nil && sessMaxLifeTime != 0 {
			int64val, _ := strconv.ParseInt(strconv.Itoa(sessMaxLifeTime), 10, 64)
			SessionGCMaxLifetime = int64val
		}
		if maxmemory, err := AppConfig.Int64("maxmemory"); err == nil {
			MaxMemory = maxmemory
		}
		AppName = AppConfig.String("appname")
		if runmode := AppConfig.String("runmode"); runmode != "" {
			RunMode = runmode
		}
		if autorender, err := AppConfig.Bool("autorender"); err == nil {
			AutoRender = autorender
		}
		if autorecover, err := AppConfig.Bool("autorecover"); err == nil {
			RecoverPanic = autorecover
		}
		if pprofon, err := AppConfig.Bool("pprofon"); err == nil {
			PprofOn = pprofon
		}
		if views := AppConfig.String("viewspath"); views != "" {
			ViewsPath = views
		}

		if enablegzip, err := AppConfig.Bool("gzip"); err == nil {
			EnableGzip = enablegzip
		}
		if directoryindex, err := AppConfig.Bool("directoryindex"); err == nil {
			DirectoryIndex = directoryindex
		}

		if errorsshow, err := AppConfig.Bool("errorsshow"); err == nil {
			ErrorsShow = errorsshow
		}

		if enablexsrf, err := AppConfig.Bool("xsrf"); err == nil {
			EnableXSRF = enablexsrf
		}
		if xsrfkey := AppConfig.String("xsrf.key"); xsrfkey != "" {
			XSRFKEY = xsrfkey
		}
		if expire, err := AppConfig.Int("xsrf.expire"); err == nil {
			XSRFExpire = expire
		}

		if tplleft := AppConfig.String("templateleft"); tplleft != "" {
			TemplateLeft = tplleft
		}
		if tplright := AppConfig.String("templateright"); tplright != "" {
			TemplateRight = tplright
		}

	}
	return nil
}
