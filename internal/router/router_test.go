package router_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/y-yagi/niwa/internal/config"
	"github.com/y-yagi/niwa/internal/router"
)

func TestRoot(t *testing.T) {
	conf := &config.Config{}
	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()

	body, err := getBodyFromURL(client, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello, world"
	if string(body) != expected {
		t.Errorf("got: %s, wont: %s", body, expected)
	}
}

func TestServeFile(t *testing.T) {
	conf := &config.Config{Root: "../../testdata"}
	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	body, err := getBodyFromURL(client, ts.URL+"/public/user.json")
	if err != nil {
		t.Fatal(err)
	}

	expected := `{name: "dummy","email":"dummy@example.com"}`
	if string(body) != expected {
		t.Errorf("got: %s, wont: %s", body, expected)
	}
}

func TestProxy(t *testing.T) {
	asbody := "Hello from application server"

	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, asbody)
	}))
	defer as.Close()

	url, err := url.Parse(as.URL)
	if err != nil {
		t.Fatal(err)
	}

	conf := &config.Config{ReverseProxy: httputil.NewSingleHostReverseProxy(url)}

	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	body, err := getBodyFromURL(client, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != asbody+"\n" {
		t.Errorf("got: %s, wont: %s", body, asbody)
	}
}

func TestRule(t *testing.T) {
	conf := &config.Config{Root: "../../testdata", RuleMap: map[string]string{}}
	conf.RuleMap["/public/from.html"] = "/public/user.json"

	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	body, err := getBodyFromURL(client, ts.URL+"/public/from.html")
	if err != nil {
		t.Fatal(err)
	}

	expected := `{name: "dummy","email":"dummy@example.com"}`
	if string(body) != expected {
		t.Errorf("got: %s, wont: %s", body, expected)
	}
}

func TestHeaders(t *testing.T) {
	conf := &config.Config{Root: "testdata"}
	conf.Headers = append(conf.Headers, config.Header{Key: "Key", Value: "Value"})

	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	res, err := client.Get(ts.URL + "/public/user.json")
	if err != nil {
		t.Fatal(err)
	}

	if res.Header.Get("Key") != "Value" {
		t.Errorf("got: %+v, wont: %s", res.Header, "Value")
	}
}

func TestRoutings(t *testing.T) {
	asbody := "Hello from application server"

	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, asbody)
	}))
	defer as.Close()

	url, err := url.Parse(as.URL)
	if err != nil {
		t.Fatal(err)
	}

	conf := &config.Config{}
	conf.RoutingMap = map[string]config.Routing{}
	routing := config.Routing{ReverseProxy: httputil.NewSingleHostReverseProxy(url)}
	conf.RoutingMap["/app"] = routing

	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	body, err := getBodyFromURL(client, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello, world"
	if string(body) != expected {
		t.Errorf("got: %s, wont: %s", body, expected)
	}

	body, err = getBodyFromURL(client, ts.URL+"/app")
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != asbody+"\n" {
		t.Errorf("got: %s, wont: %s", body, asbody)
	}
}

func TestRequestBodyMaxSize(t *testing.T) {
	conf := &config.Config{RequestBodyMaxSize: 20}
	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	body := strings.NewReader("This is a short text")
	client := ts.Client()
	res, err := client.Post(ts.URL, "text/plain", body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("got: %v, wont: %v", res.StatusCode, http.StatusOK)
	}

	body = strings.NewReader("This is a long long long data")
	res, err = client.Post(ts.URL, "text/plain", body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("got: %v, wont: %v", res.StatusCode, http.StatusRequestEntityTooLarge)
	}
}

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

func TestTimelimit(t *testing.T) {
	asbody := "Hello from application server"

	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		fmt.Fprintln(w, asbody)
	}))
	defer as.Close()

	url, err := url.Parse(as.URL)
	if err != nil {
		t.Fatal(err)
	}

	conf := &config.Config{ReverseProxy: httputil.NewSingleHostReverseProxy(url), Timelimit: 10 * time.Millisecond, RequestBodyMaxSize: 1000}

	ts := httptest.NewServer(router.New(conf))
	defer ts.Close()

	client := ts.Client()
	body, err := getBodyFromURL(client, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	wont := "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"
	if string(body) != wont {
		t.Errorf("got: %s, wont: %s", body, wont)
	}
}
