package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/y-yagi/niwa/internal/config"
)

type Router struct {
	conf *config.Config
}

func New(conf *config.Config) http.Handler {
	var handler http.Handler
	handler = &Router{conf: conf}

	if conf.Timelimit != 0 {
		handler = http.TimeoutHandler(handler, conf.Timelimit, "")
	}

	if conf.RequestBodyMaxSize > 0 {
		handler = http.MaxBytesHandler(handler, int64(conf.RequestBodyMaxSize))
	}

	return handler
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router.conf.RequestBodyMaxSize > 0 {
		l := r.ContentLength
		body := make([]byte, l)
		_, err := r.Body.Read(body)
		if err != nil && err.Error() == "http: request body too large" {
			_ = router.conf.Logging.WriteHTTPLog(w, r, http.StatusRequestEntityTooLarge, 0)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
	}

	if router.conf.ReverseProxy != nil {
		router.conf.ReverseProxy.ServeHTTP(w, r)
		return
	}

	if to, found := router.conf.RuleMap[r.URL.Path]; found {
		http.Redirect(w, r, to, http.StatusPermanentRedirect)
		return
	}

	if routing, found := router.conf.RoutingMap[r.URL.Path]; found {
		for _, h := range routing.Headers {
			w.Header().Set(h.Key, h.Value)
		}

		if routing.ReverseProxy != nil {
			routing.ReverseProxy.ServeHTTP(w, r)
		}
		return
	}

	for _, h := range router.conf.Headers {
		w.Header().Set(h.Key, h.Value)
	}

	if strings.HasPrefix(r.URL.Path, "/public/") {
		fh := http.StripPrefix("/public", http.FileServer(http.Dir(router.conf.Root)))
		scw := &captureWriter{ResponseWriter: w}
		fh.ServeHTTP(scw, r)
		_ = router.conf.Logging.WriteHTTPLog(w, r, scw.status, scw.size)
		return
	}

	msg := "Hello, world"
	_ = router.conf.Logging.WriteHTTPLog(w, r, 200, len(msg))
	fmt.Fprint(w, msg)
}
