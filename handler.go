package main

import (
	"fmt"
	"net/http"
	"strings"
)

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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if config.ReverseProxy != nil {
		config.ReverseProxy.ServeHTTP(w, r)
		return
	}

	if to, found := config.RuleMap[r.URL.Path]; found {
		http.Redirect(w, r, to, http.StatusPermanentRedirect)
		return
	}

	if routing, found := config.RoutingMap[r.URL.Path]; found {
		for _, h := range routing.Headers {
			w.Header().Set(h.Key, h.Value)
		}

		if routing.ReverseProxy != nil {
			routing.ReverseProxy.ServeHTTP(w, r)
		}
		return
	}

	for _, h := range config.Headers {
		w.Header().Set(h.Key, h.Value)
	}

	if strings.HasPrefix(r.URL.Path, "/public/") {
		fh := http.StripPrefix("/public", http.FileServer(http.Dir(config.Root)))
		scw := &captureWriter{ResponseWriter: w}
		fh.ServeHTTP(scw, r)
		_ = config.Logging.Write(w, r, scw.status, scw.size)
		return
	}

	msg := "Hello, world"
	_ = config.Logging.Write(w, r, 200, len(msg))
	fmt.Fprint(w, msg)
}