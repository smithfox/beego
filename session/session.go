package session

import (
	//"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	//"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func init() {
	mrand.Seed(time.Now().UnixNano())
}

type SessionStore interface {
	Set(key, value interface{}) error //set session value
	Get(key interface{}) interface{}  //get session value
	Delete(key interface{}) error     //delete session value
	SessionID() string                //back current sessionID
	SessionRelease()                  // release the resource
	Flush() error                     //delete all data
}

type CreateSidFunc func() string

type Provider interface {
	SessionInit(maxlifetime int64, savePath string) error

	//not change the last-access-time
	HasSession(sid string) (bool, error)

	//will update the last-access-time
	GetSession(sid string) (SessionStore, error)

	//create new session by the sid, if exist will return nil and an error
	NewSession(sid string) (SessionStore, error)

	//SessionRead(sid string) (SessionStore, error)
	//SessionNewIfNo(sid string, createSidFunc CreateSidFunc) (SessionStore, error)
	//SessionRegenerate(oldsid, sid string) (SessionStore, error)

	SessionDestroy(sid string) error
	SessionGC()
}

var provides = make(map[string]Provider)

// Register makes a session provide available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, provide Provider) {
	if provide == nil {
		panic("session: Register provide is nil")
	}
	if _, dup := provides[name]; dup {
		panic("session: Register called twice for provider " + name)
	}
	provides[name] = provide
}

type Manager struct {
	cookieName    string //private cookiename
	extCookieName string //cookie session token extend info: IP, UID
	provider      Provider
	maxlifetime   int64
	domain        string
	//options     []interface{}

	//cache options setting
	secure   bool   //options[0], 为true时,只有https才传递到服务器端。http是不会传递的
	hashfunc string //options[1], support md5 & sha1
	hashkey  string //options[2]
	maxage   int    //options[3]

	lock sync.RWMutex
}

//options
//1. is https  default false
//2. hashfunc  default sha1
//3. hashkey default beegosessionkey
//4. maxage default is none
func NewManager(provideName, cookieName string, maxlifetime int64, savePath string, domain string, options ...interface{}) (*Manager, error) {
	provider, ok := provides[provideName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provide %q (forgotten import?)", provideName)
	}
	provider.SessionInit(maxlifetime, savePath)

	secure := false
	if len(options) > 0 {
		secure = options[0].(bool)
	}

	hashfunc := "sha1"
	if len(options) > 1 {
		hashfunc = options[1].(string)
	}
	hashkey := "changethedefaultkey"
	if len(options) > 2 {
		hashkey = options[2].(string)
	}

	maxage := -1
	if len(options) > 3 {
		switch options[3].(type) {
		case int:
			if options[3].(int) > 0 {
				maxage = options[3].(int)
			} else if options[3].(int) < 0 {
				maxage = 0
			}
		case int64:
			if options[3].(int64) > 0 {
				maxage = int(options[3].(int64))
			} else if options[3].(int64) < 0 {
				maxage = 0
			}
		case int32:
			if options[3].(int32) > 0 {
				maxage = int(options[3].(int32))
			} else if options[3].(int32) < 0 {
				maxage = 0
			}
		}
	}

	return &Manager{
		provider:      provider,
		cookieName:    cookieName,
		extCookieName: "eds1",
		maxlifetime:   maxlifetime,
		domain:        domain,
		hashfunc:      hashfunc,
		hashkey:       hashkey,
		maxage:        maxage,
		secure:        secure,
	}, nil
}

//get SessionCookie, is sid
func (manager *Manager) GetSessionCookie(r *http.Request) (string, error) {
	//	fmt.Printf("GetSessionCookie, manager.cookieName=%s\n", manager.cookieName)
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

//set session cookie
func (manager *Manager) SetSessionCookie(w http.ResponseWriter, sid string) {
	manager._deleteOldSessionCookie(w)

	cookie := &http.Cookie{
		Name:     manager.cookieName,
		Value:    sid,
		Path:     "/",
		Domain:   manager.domain,
		HttpOnly: true,
		Secure:   manager.secure}
	if manager.maxage >= 0 {
		cookie.MaxAge = manager.maxage
	}

	//cookie.Expires = time.Now().Add(time.Duration(manager.maxlifetime) * time.Second)
	http.SetCookie(w, cookie)
}

func (manager *Manager) _deleteOldSessionCookie(w http.ResponseWriter) {
	expiration := time.Now().AddDate(-1, 0, 0)

	cookie := http.Cookie{
		Name:     "nsxnid",
		Path:     "/",
		Domain:   "www.dota2sp.com",
		HttpOnly: true,
		Expires:  expiration,
		MaxAge:   -1}
	http.SetCookie(w, &cookie)
}

//delete session cookie
func (manager *Manager) DeleteSessionCookie(w http.ResponseWriter) {
	manager._deleteOldSessionCookie(w)
	expiration := time.Now().AddDate(-1, 0, 0)
	cookie := http.Cookie{
		Name:     manager.cookieName,
		Path:     "/",
		Domain:   manager.domain,
		HttpOnly: true,
		Expires:  expiration,
		MaxAge:   -1}
	http.SetCookie(w, &cookie)
}

//get extra session cookie: IP, UID
func (manager *Manager) GetSessionExtCookie(r *http.Request) (string, string, error) {
	cookie, err := r.Cookie(manager.extCookieName)
	if err != nil {
		return "", "", err
	}
	value := cookie.Value
	bb := []byte(value)
	for i := 0; i < len(bb); i++ {
		bb[i] = bb[i] - 64
	}
	value = string(bb)

	ss := strings.Split(value, ",")
	if len(ss) == 2 {
		IP := ss[0]
		UID := ss[1]
		if IP == "" || UID == "" {
			return "", "", errors.New("Invalid extra session cookie value")
		} else {
			return IP, UID, nil
		}
	} else {
		return "", "", errors.New("Invalid extra session cookie format")
	}
}

//set extra session cookie: IP, UID
func (manager *Manager) SetSessionExtCookie(w http.ResponseWriter, IP string, UID string) {
	manager._deleteOldSessionExtCookie(w)

	value := fmt.Sprintf("%s,%s", IP, UID)
	bb := []byte(value)
	for i := 0; i < len(bb); i++ {
		bb[i] = bb[i] + 64
	}

	value = string(bb)

	cookie := &http.Cookie{
		Name:     manager.extCookieName,
		Value:    value,
		Path:     "/",
		Domain:   manager.domain,
		HttpOnly: true,
		Secure:   manager.secure}
	if manager.maxage >= 0 {
		cookie.MaxAge = manager.maxage
	}
	//cookie.Expires = time.Now().Add(time.Duration(manager.maxlifetime) * time.Second)
	http.SetCookie(w, cookie)
}

func (manager *Manager) _deleteOldSessionExtCookie(w http.ResponseWriter) {
	expiration := time.Now().AddDate(-1, 0, 0)
	cookie := http.Cookie{
		Name:     "eds",
		Path:     "/",
		Domain:   "www.dota2sp.com",
		HttpOnly: true,
		Expires:  expiration,
		MaxAge:   -1}
	http.SetCookie(w, &cookie)
}

//delete extra session cookie: IP, UID
func (manager *Manager) DeleteSessionExtCookie(w http.ResponseWriter) {
	manager._deleteOldSessionExtCookie(w)

	expiration := time.Now().AddDate(-1, 0, 0)
	cookie := http.Cookie{
		Name:     manager.extCookieName,
		Path:     "/",
		Domain:   manager.domain,
		HttpOnly: true,
		Expires:  expiration,
		MaxAge:   -1}
	http.SetCookie(w, &cookie)
}

//get Session By sessionId
func (manager *Manager) GetSession(sid string) (SessionStore, error) {
	return manager.provider.GetSession(sid)
}

//get Session By sessionId
func (manager *Manager) NewSession(sid string) (SessionStore, error) {
	return manager.provider.NewSession(sid)
}

func (manager *Manager) DeleteSession(sid string) {
	manager.provider.SessionDestroy(sid)
}

func (manager *Manager) GC() {
	manager.provider.SessionGC()
	time.AfterFunc(time.Duration(manager.maxlifetime)*time.Second, func() { manager.GC() })
}

//remote_addr cruunixnano randdata
func (manager *Manager) NewSessionId(r *http.Request) string {
	const randlen int = 12
	randbb := make([]byte, randlen)
	if _, err := io.ReadFull(rand.Reader, randbb); err != nil {
		return ""
	}

	sig := fmt.Sprintf("%s%d", r.RemoteAddr, time.Now().UnixNano())
	signbytes := []byte(sig)
	signlen := len(signbytes)
	h := md5.New()
	n, _ := h.Write(signbytes)
	for n < signlen {
		fmt.Printf("!!!why1")
		m, _ := h.Write(signbytes[n:])
		n += m
	}

	n, _ = h.Write(randbb)
	for n < randlen {
		fmt.Printf("!!!why2")
		m, _ := h.Write(randbb[n:])
		n += m
	}

	return url.QueryEscape(hex.EncodeToString(h.Sum(nil)))

	/*
		if manager.hashfunc == "md5" {
			h := md5.New()
			h.Write([]byte(sig))
			h.Write(randbb)
			sid = hex.EncodeToString(h.Sum(nil))
		} else if manager.hashfunc == "sha1" {
			h := hmac.New(sha1.New, []byte(manager.hashkey))
			h.Write([]byte(sig))
			h.Write(randbb)
			sid = hex.EncodeToString(h.Sum(nil))
		} else {
			h := hmac.New(sha1.New, []byte(manager.hashkey))
			h.Write([]byte(sig))
			h.Write(randbb)
			sid = hex.EncodeToString(h.Sum(nil))
		}
	*/
}
