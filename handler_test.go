package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

func TestRoot(t *testing.T) {
	config = &Config{}
	ts := httptest.NewServer(buildRouter())
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
	config = &Config{Root: "testdata"}
	ts := httptest.NewServer(buildRouter())
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

	config = &Config{ReverseProxy: httputil.NewSingleHostReverseProxy(url)}

	ts := httptest.NewServer(buildRouter())
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
	config = &Config{Root: "testdata", RuleMap: map[string]string{}}
	config.RuleMap["/public/from.html"] = "/public/user.json"

	ts := httptest.NewServer(buildRouter())
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
	config = &Config{Root: "testdata"}
	config.Headers = append(config.Headers, Header{Key: "Key", Value: "Value"})

	ts := httptest.NewServer(buildRouter())
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

	config = &Config{}
	config.RoutingMap = map[string]Routing{}
	routing := Routing{ReverseProxy: httputil.NewSingleHostReverseProxy(url)}
	config.RoutingMap["/app"] = routing

	ts := httptest.NewServer(buildRouter())
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
	config = &Config{RequestBodyMaxSize: 20}
	ts := httptest.NewServer(buildRouter())
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
