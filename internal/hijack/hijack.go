package hijack

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
	}
	return hj.Hijack()
}

func StatusCode(handler http.Handler, w http.ResponseWriter, r *http.Request) int {
	rec := NewResponseWriter(w)
	handler.ServeHTTP(rec, r)
	return rec.StatusCode
}
