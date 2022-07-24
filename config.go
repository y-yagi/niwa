package main

import (
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

type Config struct {
	Root            string    `toml:"root"`
	Port            string    `toml:"port"`
	Certfile        string    `toml:"certfile"`
	Keyfile         string    `toml:"keyfile"`
	Rules           []Rule    `toml:"rules"`
	ReverseProxyURL string    `toml:"reverse_proxy"`
	Headers         []Header  `toml:"headers"`
	Routings        []Routing `toml:"routings"`
	Log             Log       `toml:"log"`
	RuleMap         map[string]string
	RoutingMap      map[string]Routing
	ReverseProxy    *httputil.ReverseProxy
	Logging         *Logging
}

type Rule struct {
	From string `toml:"from"`
	To   string `toml:"to"`
}

type Header struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

type Log struct {
	Output string `toml:"output"`
	Format string `toml:"format"`
	File   File   `toml:"file"`
}

type Routing struct {
	Path            string `toml:"path"`
	ReverseProxyURL string `toml:"reverse_proxy"`
	ReverseProxy    *httputil.ReverseProxy
	Headers         []Header `toml:"headers"`
}

type File struct {
	Path string `toml:"path"`
}

func ParseConfigfile(filename string) (*Config, error) {
	cfg := &Config{RuleMap: map[string]string{}, RoutingMap: map[string]Routing{}}

	if len(filename) == 0 {
		return cfg, nil
	}

	f, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	err = toml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	for _, rule := range cfg.Rules {
		cfg.RuleMap[rule.From] = rule.To
	}

	if cfg.ReverseProxyURL != "" {
		url, err := url.Parse(cfg.ReverseProxyURL)
		if err != nil {
			return nil, err
		}
		cfg.ReverseProxy = httputil.NewSingleHostReverseProxy(url)
	}

	if cfg.Logging, err = NewLogging(&cfg.Log); err != nil {
		return nil, err
	}

	for _, routing := range cfg.Routings {
		if len(routing.ReverseProxyURL) != 0 {
			url, err := url.Parse(routing.ReverseProxyURL)
			if err != nil {
				return nil, err
			}
			routing.ReverseProxy = httputil.NewSingleHostReverseProxy(url)
		}
		cfg.RoutingMap[routing.Path] = routing
	}

	return cfg, nil
}
