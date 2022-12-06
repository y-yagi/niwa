package server_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/server"
	"golang.org/x/sync/errgroup"
)

func getBodyFromURL(c *http.Client, url string) ([]byte, error) {
	res, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	return body, err
}

func TestStart(t *testing.T) {
	conf := &config.Config{Port: "18080"}

	ctx, done := context.WithCancel(context.Background())
	defer done()
	g, gctx := errgroup.WithContext(ctx)
	server := server.New(conf)
	server.Start(g, gctx, done)

	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		time.Sleep(200 * time.Millisecond)
	}()

	client := &http.Client{}
	body, err := getBodyFromURL(client, "http://localhost:18080")
	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello, world"
	if string(body) != expected {
		t.Errorf("got: %s, wont: %s", body, expected)
	}
}

func TestStart_UseHTTP3(t *testing.T) {
	cerfile := "../../localhost.pem"
	keyfile := "../../localhost-key.pem"

	if _, err := os.Stat(cerfile); errors.Is(err, os.ErrNotExist) {
		t.Skip()
	}
	if _, err := os.Stat(keyfile); errors.Is(err, os.ErrNotExist) {
		t.Skip()
	}

	conf := &config.Config{ConfigFile: config.ConfigFile{UseHttp3: true, Certfile: cerfile, Keyfile: keyfile}}

	ctx, done := context.WithCancel(context.Background())
	defer done()
	g, gctx := errgroup.WithContext(ctx)
	server := server.New(conf)
	server.Start(g, gctx, done)

	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		time.Sleep(200 * time.Millisecond)
	}()

	client := &http.Client{}
	res, err := client.Get("https://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}

	expected := "h3=\":8080\""
	if !strings.Contains(res.Header.Get("Alt-Svc"), expected) {
		t.Errorf("expect '%s' included in '%s'", expected, res.Header.Get("Alt-Svc"))
	}
}
