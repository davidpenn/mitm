package daemon

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
	"github.com/julienschmidt/httprouter"
	"github.com/prologic/bitcask"
)

// Server handler
type Server struct {
	*httprouter.Router
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
	server := &Server{
		Router: httprouter.New(),
		db:     db,
		replay: replay,
	}
	server.Router.GET("/requests/:id", server.getRequestHandler)
	return server
}

// GetRequest with id from the cache database
func (m *Server) GetRequest(id string) (*RequestLogger, error) {
	var req *RequestLogger
	data, err := m.db.Get([]byte(id))
	switch {
	case err == bitcask.ErrKeyNotFound:
		return nil, nil
	case err != nil:
		return nil, err
	}
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

		// compare headers
		if !cmp.Equal(r.Headers, req.Headers) {
			continue
		}

		// compare query
		if !cmp.Equal(r.Query, req.Query) {
			continue
		}

		// compare body
		if strings.Contains(r.Headers.Get("Content-Type"), "application/json") {
			var a, b interface{}
			if err = json.Unmarshal(r.Body, &a); err != nil {
				continue
			}
			if err = json.Unmarshal(req.Body, &b); err != nil {
				continue
			}
			if !cmp.Equal(a, b) {
				continue
			}
		} else if !cmp.Equal(r.Body, req.Body) {
			continue
		}

		// everything matches
		return req, true
	}
	return r, false
}

func (m *Server) getRequestHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data, err := m.GetRequest(ps.ByName("id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if data == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
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
