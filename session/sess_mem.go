package session

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var mempder = &MemProvider{list: list.New(), sessions: make(map[string]*list.Element)}

type MemSessionStore struct {
	sid          string                      //session id唯一标示
	timeAccessed time.Time                   //最后访问时间
	value        map[interface{}]interface{} //session里面存储的值
	lock         sync.RWMutex
}

func (st *MemSessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value[key] = value
	return nil
}

func (st *MemSessionStore) Get(key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.value[key]; ok {
		return v
	} else {
		return nil
	}
	return nil
}

func (st *MemSessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.value, key)
	return nil
}

func (st *MemSessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value = make(map[interface{}]interface{})
	return nil
}

func (st *MemSessionStore) SessionID() string {
	return st.sid
}

func (st *MemSessionStore) SessionRelease() {

}

type MemProvider struct {
	lock        sync.RWMutex             //用来锁
	sessions    map[string]*list.Element //用来存储在内存
	list        *list.List               //用来做gc
	maxlifetime int64
	savePath    string
}

func (pder *MemProvider) SessionInit(maxlifetime int64, savePath string) error {
	pder.maxlifetime = maxlifetime
	pder.savePath = savePath
	return nil
}

//not change the last-access-time
func (pder *MemProvider) HasSession(sid string) (bool, error) {
	pder.lock.RLock()
	defer pder.lock.RUnlock()

	_, ok := pder.sessions[sid]
	return ok, nil
}

//will update the last-access-time
func (pder *MemProvider) GetSession(sid string) (SessionStore, error) {
	pder.lock.RLock()
	defer pder.lock.RUnlock()

	if element, ok := pder.sessions[sid]; ok {
		go pder.SessionUpdate(sid)
		return element.Value.(*MemSessionStore), nil
	} else {
		return nil, nil
	}
}

func (pder *MemProvider) NewSession(sid string) (SessionStore, error) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	if sid == "" {
		return nil, errors.New("can not create session with empty sid")
	}

	if _, ok := pder.sessions[sid]; ok {
		return nil, errors.New("session with sid=" + sid + " exist")
	}
	newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
	element := pder.list.PushBack(newsess)
	pder.sessions[sid] = element
	return newsess, nil
}

/*
func (pder *MemProvider) SessionRead(sid string) (SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[sid]; ok {
		go pder.SessionUpdate(sid)
		pder.lock.RUnlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
	return nil, nil
}

func (pder *MemProvider) SessionNewIfNo(sid string, createSidFunc CreateSidFunc) (SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[sid]; ok {
		go pder.SessionUpdate(sid)
		pder.lock.RUnlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		newsid := createSidFunc()
		newsess := &MemSessionStore{sid: newsid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[newsid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
	return nil, nil
}
*/
/*
func (pder *MemProvider) SessionRegenerate(oldsid, sid string) (SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[oldsid]; ok {
		go pder.SessionUpdate(oldsid)
		pder.lock.RUnlock()
		pder.lock.Lock()
		element.Value.(*MemSessionStore).sid = sid
		pder.sessions[sid] = element
		delete(pder.sessions, oldsid)
		pder.lock.Unlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
	return nil, nil
}
*/
func (pder *MemProvider) SessionDestroy(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}

func (pder *MemProvider) SessionGC() {
	pder.lock.RLock()
	for {
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*MemSessionStore).timeAccessed.Unix() + pder.maxlifetime) < time.Now().Unix() {
			pder.lock.RUnlock()
			pder.lock.Lock()
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*MemSessionStore).sid)
			pder.lock.Unlock()
			pder.lock.RLock()
		} else {
			break
		}
	}
	pder.lock.RUnlock()
}

func (pder *MemProvider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*MemSessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}

func init() {
	Register("memory", mempder)
}
