package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/router"
	"golang.org/x/sync/errgroup"
)

const cmd = "niwa"

var (
	flags          *flag.FlagSet
	configFilename string
	showVersion    bool

	version = "devel"
)

func main() {
	setFlags()
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func setFlags() {
	flags = flag.NewFlagSet(cmd, flag.ExitOnError)
	flags.BoolVar(&showVersion, "v", false, "print version number")
	flags.StringVar(&configFilename, "c", "", "config file name")
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	_ = flags.Parse(args[1:])

	var err error
	exitCode = 0

	if showVersion {
		fmt.Fprintf(outStream, "%s %s (runtime: %s)\n", cmd, version, runtime.Version())
		return
	}

	conf, err := config.ParseConfigfile(configFilename)
	if err != nil {
		fmt.Printf("parse config file error %+v\n", err)
		exitCode = 1
		return
	}

	ctx, done := context.WithCancel(context.Background())
	defer done()
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return startServer(gctx, conf)
	})

	g.Go(func() error {
		sighup := make(chan os.Signal, 1)
		signal.Notify(sighup, syscall.SIGHUP)

		for {
			select {
			case <-sighup:
				if err := conf.Logging.Reopen(); err != nil {
					return err
				}
			case <-gctx.Done():
				return gctx.Err()
			}

		}
	})

	g.Go(func() error {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		select {
		case <-stop:
			done()
		case <-gctx.Done():
			return gctx.Err()
		}

		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Println(err)
		exitCode = 1
	}

	return
}

func startServer(ctx context.Context, conf *config.Config) error {
	port := "8080"
	if conf.Port != "" {
		port = conf.Port
	}

	mux := http.NewServeMux()
	mux.Handle("/", router.New(conf))

	s := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if len(conf.Certfile) > 0 && len(conf.Keyfile) > 0 {
			if err := s.ListenAndServeTLS(conf.Certfile, conf.Keyfile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		} else {
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		return s.Shutdown(tctx)
	}
}
