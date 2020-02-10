package daemon

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

// RequestLogger holds request/response data
type RequestLogger struct {
	ID         string      `json:"id"`
	CreatedAt  time.Time   `json:"created_at"`
	FinishedAt time.Time   `json:"finished_at"`
	Headers    http.Header `json:"headers"`
	Method     string      `json:"method"`
	Host       string      `json:"host"`
	Path       string      `json:"path"`
	Query      url.Values  `json:"query"`
	Body       []byte      `json:"body"`
	Response   struct {
		Code    int         `json:"code"`
		Headers http.Header `json:"headers"`
		Body    []byte      `json:"body"`
	} `json:"response"`

	http.ResponseWriter `json:"-"`
}

// NewRequestLogger constructs a RequestLogger
func NewRequestLogger(w http.ResponseWriter, r *http.Request) *RequestLogger {
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	return &RequestLogger{
		ID:             uuid.New().String(),
		CreatedAt:      time.Now().UTC(),
		Headers:        r.Header.Clone(),
		Method:         r.Method,
		Host:           r.Host,
		Path:           r.URL.Path,
		Query:          r.URL.Query(),
		Body:           body,
		ResponseWriter: w,
	}
}

// String implements the Stringer interface
func (m *RequestLogger) String() string {
	if m.FinishedAt.IsZero() {
		m.FinishedAt = time.Now().UTC()
	}
	return fmt.Sprintf("%s %s %s %v %s %s",
		colorFromMethod(m.Method, m.Method),
		m.Host+m.Path,
		colorFromStatus(m.Response.Code, "%d", m.Response.Code),
		m.FinishedAt.Sub(m.CreatedAt),
		humanize.Bytes(uint64(len(m.Response.Body))),
		m.ID,
	)
}

// Write writes the data to the connection as part of an HTTP reply
func (m *RequestLogger) Write(data []byte) (int, error) {
	m.Response.Body = append(m.Response.Body, data...)
	return m.ResponseWriter.Write(data)
}

// WriteHeader sends an HTTP response header with the provided status code
func (m *RequestLogger) WriteHeader(code int) {
	m.Response.Code = code
	m.Response.Headers = m.Header().Clone()
	m.ResponseWriter.WriteHeader(code)
}
