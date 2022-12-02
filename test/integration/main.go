package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		return msg(errors.New("please specify a command path"))
	}

	/* #nosec */
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	var slurp string
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return msg(err)
	}
	if err := cmd.Start(); err != nil {
		return msg(err)
	}

	go func(slurp *string) {
		t, _ := io.ReadAll(stderr)
		*slurp = string(t)
	}(&slurp)

	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			panic(fmt.Errorf("process kill failed %+v", err))
		}
	}()

	var res *http.Response
	retry_count := 3

	for {
		res, err = http.Get("http://127.0.0.1:50000/")
		if err != nil {
			if len(slurp) != 0 {
				return msg(fmt.Errorf("maybe command failed: %s", slurp))
			}

			if retry_count > 0 {
				time.Sleep(time.Second)
				retry_count--
			} else {
				return msg(err)
			}
		} else {
			break
		}
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return msg(err)
	}

	expected := "Hello, world"
	if expected != string(body) {
		return msg(fmt.Errorf("expected '%s', but got '%s'", expected, body))
	}

	pidfile := "/tmp/niwa.pid"
	if _, err := os.Stat(pidfile); err != nil {
		return msg(err)
	}

	return 0
}
