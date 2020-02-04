package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/kr/mitm"
	"github.com/prologic/bitcask"
	"github.com/spf13/cobra"
)

// CachedRequest holds request/response data
type CachedRequest struct {
	Header   http.Header
	Method   string
	Host     string
	Path     string
	Query    string
	Response struct {
		Code   int
		Header http.Header
		Body   []byte
	}
}

// NewCachedRequest constructs a CachedRequest
func NewCachedRequest(r *http.Request) *CachedRequest {
	cached := new(CachedRequest)
	cached.Header = r.Header.Clone()
	cached.Method = r.Method
	cached.Host = r.Host
	cached.Path = r.URL.Path
	cached.Query = r.URL.RawQuery
	return cached
}

// Key returns the key used to cache
func (m *CachedRequest) Key() string {
	return m.Method + m.RequestURL()
}

// RequestURL returns the requested url
func (m *CachedRequest) RequestURL() string {
	var query string
	if m.Query != "" {
		query = "?" + m.Query
	}
	return fmt.Sprintf("%s%s%s", m.Host, m.Path, query)
}

// StartCacheServer and listen for requests
func StartCacheServer(cmd *cobra.Command, args []string) {
	path, _ := cmd.Flags().GetString("database")
	maxValueSize, _ := cmd.Flags().GetString("database-max-size")
	server := NewCacheServer(path, maxValueSize)

	ca, err := genCA()
	if err != nil {
		log.Fatal(err)
	}

	proxy := &mitm.Proxy{
		CA:   &ca,
		Wrap: server.Handler,
	}

	addr, _ := cmd.Flags().GetString("listen")
	http.ListenAndServe(addr, proxy)

}

// CacheServer handler
type CacheServer struct {
	db *bitcask.Bitcask
}

// NewCacheServer constructs a new CacheServer
func NewCacheServer(pathToCache, maxValueSize string) *CacheServer {
	size, err := humanize.ParseBytes(maxValueSize)
	if err != nil {
		log.Fatal(err)
	}
	db, _ := bitcask.Open(pathToCache, bitcask.WithMaxValueSize(size))
	return &CacheServer{db}
}

// Handler returns a http handler
func (m *CacheServer) Handler(upstream http.Handler) http.Handler {
	colorFromCache := func(cached bool) string {
		if cached {
			return color.BlueString("✔")
		}
		return color.MagentaString("<--")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		cached, fromCache := m.getFromCache(r)

		if !fromCache {
			proxy := &cacher{ResponseWriter: w, dest: cached}
			upstream.ServeHTTP(proxy, r)
			m.saveToCache(cached)
		} else {
			for k, v := range cached.Response.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(cached.Response.Code)
			w.Write(cached.Response.Body)
		}

		fmt.Printf("%s %s %s %v %s %v\n",
			colorFromMethod(r.Method, r.Method),
			cached.RequestURL(),
			colorFromStatus(cached.Response.Code, "%d", cached.Response.Code),
			time.Now().Sub(start),
			humanize.Bytes(uint64(len(cached.Response.Body))),
			colorFromCache(fromCache),
		)
	})
}

func (m *CacheServer) getFromCache(r *http.Request) (*CachedRequest, bool) {
	obj := NewCachedRequest(r)
	data, _ := m.db.Get([]byte(obj.Key()))
	if data != nil {
		if err := json.Unmarshal(data, &obj); err != nil {
			log.Error(err)
			return NewCachedRequest(r), false
		}
		return obj, true
	}
	return obj, false

}

func (m *CacheServer) saveToCache(cached *CachedRequest) {
	data, err := json.Marshal(cached)
	if err != nil {
		log.Error(err)
		return
	}
	if err := m.db.Put([]byte(cached.Key()), data); err != nil {
		log.Error(err)
	}
}

type cacher struct {
	http.ResponseWriter
	dest *CachedRequest
}

// Write writes the data to the connection as part of an HTTP reply
func (m *cacher) Write(data []byte) (int, error) {
	m.dest.Response.Body = append(m.dest.Response.Body, data...)
	return m.ResponseWriter.Write(data)
}

// WriteHeader sends an HTTP response header with the provided status code
func (m *cacher) WriteHeader(code int) {
	m.dest.Response.Code = code
	m.dest.Response.Header = m.Header().Clone()
	m.ResponseWriter.WriteHeader(code)
}
