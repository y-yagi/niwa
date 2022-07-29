package config

import (
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/pelletier/go-toml"
	"github.com/y-yagi/niwa/internal/logging"
)

type Config struct {
	Root                  string    `toml:"root"`
	Port                  string    `toml:"port"`
	Certfile              string    `toml:"certfile"`
	Keyfile               string    `toml:"keyfile"`
	Rules                 []Rule    `toml:"rules"`
	ReverseProxyURL       string    `toml:"reverse_proxy"`
	Headers               []Header  `toml:"headers"`
	Routings              []Routing `toml:"routings"`
	Log                   Log       `toml:"log"`
	RequestBodyMaxSizeStr string    `toml:"request_body_max_size"`
	RuleMap               map[string]string
	RoutingMap            map[string]Routing
	ReverseProxy          *httputil.ReverseProxy
	Logging               *logging.Logging
	RequestBodyMaxSize    uint64
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

	logconfig := logging.LogConfig{Output: cfg.Log.Output, Format: cfg.Log.Format, FilePath: cfg.Log.File.Path}
	if cfg.Logging, err = logging.New(&logconfig); err != nil {
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

	if cfg.RequestBodyMaxSizeStr != "" {
		if cfg.RequestBodyMaxSize, err = humanize.ParseBytes(cfg.RequestBodyMaxSizeStr); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
