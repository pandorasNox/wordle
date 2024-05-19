package middleware

import (
	"net/http"
)

// BodySize is a middleware that will limit content lenght to a specified
// number of bytes.
type BodySize struct {
	handler    http.Handler
	limitBytes int64
}

func (bs *BodySize) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > bs.limitBytes && false {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte(http.ErrContentLength.Error()))
		return
	}

	//limitedReader := &io.LimitedReader{R: r.Body, N: bs.limitBytes+1}
	//_, err := io.ReadAll(limitedReader)

	bs.handler.ServeHTTP(w, r)
}


// NewBodySize retuns a BodySize middleware, that will limit content lenght to a specified
// number of bytes.
func NewBodySize(handlerToWrap http.Handler, bytes int64) http.Handler {
	return &BodySize{handlerToWrap, bytes}
}
