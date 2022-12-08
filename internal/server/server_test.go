package server_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/madflojo/testcerts"
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
	dir, err := os.MkdirTemp("", "niwa_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cerfile := path.Join(dir, "niwatest.pem")
	keyfile := path.Join(dir, "niwatest-key.pem")
	if err = testcerts.GenerateCertsToFile(cerfile, keyfile); err != nil {
		t.Fatal(err)
	}

	conf := &config.Config{ConfigFile: config.ConfigFile{UseHttp3: true, Certfile: cerfile, Keyfile: keyfile}, Port: "18080"}

	ctx, done := context.WithCancel(context.Background())
	defer done()
	g, gctx := errgroup.WithContext(ctx)
	server := server.New(conf)
	server.Start(g, gctx, done)
	time.Sleep(100 * time.Millisecond)

	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		time.Sleep(200 * time.Millisecond)
	}()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get("https://localhost:18080")
	if err != nil {
		t.Fatal(err)
	}

	expected := "h3=\":18080\""
	if !strings.Contains(res.Header.Get("Alt-Svc"), expected) {
		t.Errorf("expect '%s' included in '%s'", expected, res.Header.Get("Alt-Svc"))
	}
}
