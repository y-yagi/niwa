package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/router"
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
		return 0
	}

	conf, err := config.ParseConfigfile(configFilename)
	if err != nil {
		log.Fatal(err)
	}

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

	if len(conf.Certfile) != 0 && len(conf.Keyfile) != 0 {
		go func() {
			if err = s.ListenAndServeTLS(conf.Certfile, conf.Keyfile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
	} else {
		go func() {
			if err = s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	return
}
