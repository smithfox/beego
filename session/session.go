package session

import (
	//"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	//"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
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
	SessionRead(sid string) (SessionStore, error)
	SessionNewIfNo(sid string, createSidFunc CreateSidFunc) (SessionStore, error)
	SessionRegenerate(oldsid, sid string) (SessionStore, error)
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
	cookieName  string //private cookiename
	provider    Provider
	maxlifetime int64
	//options     []interface{}

	//cache options setting
	secure   bool   //options[0]
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
func NewManager(provideName, cookieName string, maxlifetime int64, savePath string, options ...interface{}) (*Manager, error) {
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
		provider:    provider,
		cookieName:  cookieName,
		maxlifetime: maxlifetime,
		hashfunc:    hashfunc,
		hashkey:     hashkey,
		maxage:      maxage,
		secure:      secure,
	}, nil
}

//get Session
func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session SessionStore) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		sid := manager.sessionId(r)
		session, _ = manager.provider.SessionRead(sid)

		cookie = &http.Cookie{Name: manager.cookieName,
			Value:    url.QueryEscape(session.SessionID()),
			Path:     "/",
			HttpOnly: true,
			Secure:   manager.secure}
		if manager.maxage >= 0 {
			cookie.MaxAge = manager.maxage
		}
		//cookie.Expires = time.Now().Add(time.Duration(manager.maxlifetime) * time.Second)
		http.SetCookie(w, cookie)
		//r.AddCookie(cookie)

	} else {
		//cookie.Expires = time.Now().Add(time.Duration(manager.maxlifetime) * time.Second)
		sessionchanged := false

		oldsid, _ := url.QueryUnescape(cookie.Value)
		var newsid string = ""

		session, _ = manager.provider.SessionNewIfNo(oldsid, func() string { newsid = manager.sessionId(r); return newsid })

		cookie.HttpOnly = true
		cookie.Path = "/"

		newescapesid := url.QueryEscape(session.SessionID())
		if cookie.Value != newescapesid {
			cookie.Value = newescapesid
			sessionchanged = true
		}

		if manager.maxage >= 0 {
			cookie.MaxAge = manager.maxage
		}

		if sessionchanged {
			http.SetCookie(w, cookie)
		}
	}
	return
}

//Destroy sessionid
func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {
		manager.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()
		cookie := http.Cookie{Name: manager.cookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

func (manager *Manager) GC() {
	manager.provider.SessionGC()
	time.AfterFunc(time.Duration(manager.maxlifetime)*time.Second, func() { manager.GC() })
}

func (manager *Manager) SessionRegenerateId(w http.ResponseWriter, r *http.Request) (session SessionStore) {
	sid := manager.sessionId(r)
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil && cookie.Value == "" {
		//delete old cookie
		session, _ = manager.provider.SessionRead(sid)

		cookie = &http.Cookie{Name: manager.cookieName,
			Value:    url.QueryEscape(sid),
			Path:     "/",
			HttpOnly: true,
			Secure:   manager.secure,
		}
	} else {
		oldsid, _ := url.QueryUnescape(cookie.Value)
		session, _ = manager.provider.SessionRegenerate(oldsid, sid)
		cookie.Value = url.QueryEscape(sid)
		cookie.HttpOnly = true
		cookie.Path = "/"
	}
	if manager.maxage >= 0 {
		cookie.MaxAge = manager.maxage
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return
}

//remote_addr cruunixnano randdata

func (manager *Manager) sessionId(r *http.Request) (sid string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	randbb := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, randbb); err != nil {
		return ""
	}

	sig := fmt.Sprintf("%s%d", r.RemoteAddr, time.Now().UnixNano())
	h := md5.New()
	h.Write([]byte(sig))
	h.Write(randbb)
	sid = hex.EncodeToString(h.Sum(nil))

	//怀疑hmac.sha1容易conflict
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
	return
}
