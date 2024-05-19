package middleware

import (
	"net/http"
)

// RequestSize is a middleware that will limit request sizes to a specified
// number of bytes. It uses MaxBytesReader to do so.
type RequestSize struct {
	handler http.Handler
	bytes   int64
}

func (l *RequestSize) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, l.bytes)
	l.handler.ServeHTTP(w, r)
}

// NewRequestSize retuns a RequestSize middleware, that will limit request sizes to a specified
// number of bytes. It uses MaxBytesReader to do so.
func NewRequestSize(handlerToWrap http.Handler, bytes int64) http.Handler {
	return &RequestSize{handlerToWrap, bytes}
}
