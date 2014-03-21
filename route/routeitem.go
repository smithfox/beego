package route

import (
	"net/http"
)

type RouteParams map[string]string

const (
	FuzzyEndSlash  = 0
	RedictEndSlash = 1
	ExactEndSlash  = 2
)

//ExactEndSlash
//是否精确匹配URL的最后'/'
//  /url  和  /url/ 被看着两个不同的URL

//RedictEndSlash
//若模糊匹配URL的最后'/', 是否强制服从 route 的定义
// route:  /url      request:  /url/   ==> redirect到  /url
// route:  /url/     request:  /url    ==> redirect到  /url/
//Redirect要用 307, 不能用301,302,303, 否则POST请求被改为了 GET

//FuzzyEndSlash
//模糊匹配, 但不redirect

type RouteItem interface {
	GetRedirectURL(req *http.Request) string
	SetSlashOption(int)
	ExactMatchSlash() bool
	RedirectSlash() bool
	CreateHandler(w http.ResponseWriter, req *http.Request) http.Handler
	GetRouteParams(req *http.Request) RouteParams
	//0-no,1-yes, otherwise no path
	EndWithSlash() int
	Match(req *http.Request) bool
}
