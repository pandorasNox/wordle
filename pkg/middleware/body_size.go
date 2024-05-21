package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

// BodySize is a middleware that will limit content lenght to a specified
// number of bytes.
type BodySize struct {
	handler    http.Handler
	limitBytes int64
}

func (bs *BodySize) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > bs.limitBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte(http.ErrContentLength.Error() + " (ContentLength header)"))
		return
	}

	limitedReader := &io.LimitedReader{R: r.Body, N: bs.limitBytes + 1}
	readBytes, err := io.ReadAll(limitedReader)
	if len(readBytes) == int(bs.limitBytes+1) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte(http.ErrContentLength.Error() + " (actual body)"))
		return
	}
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		log.Printf("body_size middleware error - bad request: '%s' \n", err)
		return
	}

	readCopy := io.NopCloser(bytes.NewReader(readBytes))
	r.Body = readCopy

	bs.handler.ServeHTTP(w, r)
}

// NewBodySize retuns a BodySize middleware, that will limit content lenght to a specified
// number of bytes.
func NewBodySize(handlerToWrap http.Handler, bytes int64) http.Handler {
	return &BodySize{handlerToWrap, bytes}
}
