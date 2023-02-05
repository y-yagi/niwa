package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/router"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	conf *config.Config
}

func New(conf *config.Config) *Server {
	return &Server{conf: conf}
}

func (s *Server) Start(g *errgroup.Group, ctx context.Context, done context.CancelFunc) {
	g.Go(func() error {
		if s.conf.UseHttp3 {
			return s.startHttp3Server(ctx)
		} else {
			return s.startHttpServer(ctx)
		}

	})

	g.Go(func() error {
		sighup := make(chan os.Signal, 1)
		signal.Notify(sighup, syscall.SIGHUP)

		for {
			select {
			case <-sighup:
				if err := s.conf.Logging.Reopen(); err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			}

		}
	})

	g.Go(func() error {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		select {
		case <-stop:
			done()
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}

func (s *Server) startHttpServer(ctx context.Context) error {
	httpserver := &http.Server{
		Addr:              ":" + s.port(),
		Handler:           s.buildServeMux(),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if len(s.conf.Certfile) > 0 && len(s.conf.Keyfile) > 0 {
			if err := httpserver.ListenAndServeTLS(s.conf.Certfile, s.conf.Keyfile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		} else {
			if err := httpserver.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return httpserver.Shutdown(tctx)
	}
}

func (s *Server) startHttp3Server(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if err := http3.ListenAndServe(":"+s.port(), s.conf.Certfile, s.conf.Keyfile, s.buildServeMux()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) port() string {
	port := "8080"
	if s.conf.Port != "" {
		port = s.conf.Port
	}

	return port
}

func (s *Server) buildServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", router.New(s.conf))
	return mux
}
