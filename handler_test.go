package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

func TestRoot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

	config = &Config{}
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
	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

	config = &Config{Root: "testdata"}

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

	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, asbody)
	}))
	defer as.Close()

	url, err := url.Parse(as.URL)
	if err != nil {
		t.Fatal(err)
	}

	config = &Config{ReverseProxy: httputil.NewSingleHostReverseProxy(url)}
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
	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

	config = &Config{Root: "testdata", RuleMap: map[string]string{}}
	config.RuleMap["/public/from.html"] = "/public/user.json"

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
	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

	config = &Config{Root: "testdata"}
	config.Headers = append(config.Headers, Header{Key: "Key", Value: "Value"})

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

	ts := httptest.NewServer(http.HandlerFunc(rootHandler))
	defer ts.Close()

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
