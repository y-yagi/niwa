package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"

	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/server"
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

	if conf.PidFile != "" {
		pid := []byte(strconv.Itoa(os.Getpid()) + "\n")
		/* #nosec G306 */
		err := os.WriteFile(conf.PidFile, pid, 0644)
		if err != nil {
			fmt.Printf("pid file creating was error%+v\n", err)
			exitCode = 1
			return
		}
		defer os.Remove(conf.PidFile)
	}

	ctx, done := context.WithCancel(context.Background())
	defer done()
	g, gctx := errgroup.WithContext(ctx)
	server := server.New(conf)
	server.Start(g, gctx, done)

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Println(err)
		exitCode = 1
	}

	return
}
