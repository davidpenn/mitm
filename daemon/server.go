package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
	"github.com/prologic/bitcask"
)

// Server handler
type Server struct {
	db     *bitcask.Bitcask
	replay bool
}

// NewServer constructs a new Server
func NewServer(pathToCache, maxValueSize string, replay bool) *Server {
	size, err := humanize.ParseBytes(maxValueSize)
	if err != nil {
		log.Fatal(err)
	}
	db, _ := bitcask.Open(pathToCache, bitcask.WithMaxValueSize(size))
	return &Server{db, replay}
}

// GetRequest with id from the cache database
func (m *Server) GetRequest(id string) (*RequestLogger, error) {
	var req *RequestLogger
	data, _ := m.db.Get([]byte(id))
	return req, json.Unmarshal(data, &req)
}

// Handler returns a http handler
func (m *Server) Handler(upstream http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var servedFromCache bool
		mw := NewRequestLogger(w, r)

		if m.replay {
			mw, servedFromCache = m.getFromCache(mw)
		}

		if servedFromCache {
			mw.CreatedAt = time.Now().UTC()
			for k, v := range mw.Response.Headers {
				w.Header()[k] = v
			}
			w.WriteHeader(mw.Response.Code)
			w.Write(mw.Response.Body)
			mw.FinishedAt = time.Now().UTC()
		} else {
			upstream.ServeHTTP(mw, r)
			mw.FinishedAt = time.Now().UTC()
			m.saveToCache(mw)
		}

		log.Infof("%s %s\n", mw, m.colorFromCache(servedFromCache))
	})
}

func (m *Server) colorFromCache(cached bool) string {
	if cached {
		return color.BlueString("✔")
	}
	return color.MagentaString("<--")
}

func (m *Server) getCacheMap(r *RequestLogger) map[string][]string {
	cacheMap := make(map[string][]string)
	data, _ := m.db.Get([]byte(r.Host + r.Path))
	if data != nil {
		if err := json.Unmarshal(data, &cacheMap); err != nil {
			log.Error(err)
			return cacheMap
		}
	}
	return cacheMap
}

func (m *Server) getFromCache(r *RequestLogger) (*RequestLogger, bool) {
	cacheMap := m.getCacheMap(r)
	for _, id := range cacheMap[r.Method] {
		req, err := m.GetRequest(id)
		if err != nil {
			log.Error(err)
			continue
		}
		if !cmp.Equal(r.Headers, req.Headers) {
			continue
		}
		if !cmp.Equal(r.Query, req.Query) {
			continue
		}
		if strings.Contains(r.Headers.Get("Content-Type"), "application/json") {
			var a, b interface{}
			json.Unmarshal(r.Body, &a)
			json.Unmarshal(req.Body, &b)
			if a != nil && b != nil {
				if cmp.Equal(a, b) {
					return req, true
				}
			}
		}
		if !cmp.Equal(r.Body, req.Body) {
			continue
		}
		return req, true
	}
	return r, false
}

func (m *Server) saveToCache(r *RequestLogger) {
	data, err := json.Marshal(r)
	if err != nil {
		log.Error(err)
		return
	}
	if err := m.db.Put([]byte(r.ID), data); err != nil {
		log.Error(err)
	}

	cacheMap := m.getCacheMap(r)
	cacheMap[r.Method] = append(cacheMap[r.Method], r.ID)
	data, err = json.Marshal(cacheMap)
	if err != nil {
		log.Error(err)
		return
	}
	if err := m.db.Put([]byte(r.Host+r.Path), data); err != nil {
		log.Error(err)
	}
}
