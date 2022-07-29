package router

import "net/http"

type captureWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (cw *captureWriter) WriteHeader(status int) {
	cw.status = status
	cw.ResponseWriter.WriteHeader(status)
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	size, err := cw.ResponseWriter.Write(b)
	cw.size += size
	return size, err
}
