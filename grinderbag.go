package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/scryner/logg"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	expirationSec int64
	listenPort    int

	s      *storage
	logger *logg.Logger
)

type SetQuery struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type GetQuery struct {
	Key string `json:"key"`
}

type item struct {
	t             time.Time
	expirationSec int64

	key, val string
}

type storage struct {
	m    map[string]item
	lock *sync.RWMutex
}

func (s *storage) set(key, val string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s._get(key)
	if ok {
		return fmt.Errorf("'%s' is already existed", key)
	}

	it := item{
		t:             time.Now(),
		expirationSec: expirationSec,
		key:           key,
		val:           val,
	}

	s.m[key] = it

	return nil
}

func (s *storage) get(key string) (string, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s._get(key)
}

func (s *storage) _get(key string) (string, bool) {
	it, ok := s.m[key]
	if !ok {
		return ``, false
	}

	now := time.Now().Unix()
	exp := it.t.Unix() + it.expirationSec

	if exp-now > 0 {
		return it.val, true
	} else {
		delete(s.m, key)
		return ``, false
	}

	panic(`never reached`)
}

func init() {
	var expSec int

	flag.IntVar(&expSec, "exp", 60, "specify item expiration")
	expirationSec = int64(expSec)

	flag.IntVar(&listenPort, "l", 8091, "listen port")

	s = &storage{
		m:    make(map[string]item),
		lock: new(sync.RWMutex),
	}

	logger = logg.GetDefaultLogger("grindbag")
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("set error while reading from %v", r.RemoteAddr)
		http.Error(w, "can't reading", http.StatusInternalServerError)
		return
	}

	var query SetQuery
	err = json.Unmarshal(b, &query)
	if err != nil {
		logger.Errorf("set error while unmarshaling / %v", r.RemoteAddr)
		http.Error(w, "can't unmarshaling", http.StatusInternalServerError)
		return
	}

	err = s.set(query.Key, query.Val)
	if err != nil {
		logger.Errorf("set error: %v / %v", query, r.RemoteAddr)
		http.Error(w, "can't set", http.StatusInternalServerError)
		return
	}

	logger.Debugf("set success: %v / %v", query, r.RemoteAddr)
	fmt.Fprintf(w, "ok")
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("get error while reading from %v", r.RemoteAddr)
		http.Error(w, "can't reading", http.StatusInternalServerError)
		return
	}

	var query GetQuery
	err = json.Unmarshal(b, &query)
	if err != nil {
		logger.Errorf("get error while unmarshaling / %v", r.RemoteAddr)
		http.Error(w, "can't unmarshaling", http.StatusInternalServerError)
		return
	}

	val, ok := s.get(query.Key)
	if !ok {
		logger.Debugf("get success: %s NOT_FOUND / %v", query.Key, r.RemoteAddr)
		fmt.Fprintf(w, "")
	}

	logger.Debugf("get success: %s(%v) / %v", query.Key, val, r.RemoteAddr)
	fmt.Fprintf(w, "%s", val)
}

func main() {
	flag.Parse()

	defer func() {
		logg.Flush()
		os.Exit(1)
	}()

	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)

	logger.Infof("starting server at port %d", listenPort)
	logger.Infof("item will expired after %v seconds", expirationSec)

	err := http.ListenAndServe(fmt.Sprintf(":%d", listenPort), nil)
	if err != nil {
		logger.Errorf("does not starting server: %v", err)
	}
}
