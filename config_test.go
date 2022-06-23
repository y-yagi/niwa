package main

import (
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	config, err := ParseConfigfile("testdata/config.toml")
	if err != nil {
		t.Errorf("Parse file error: %v", err)
	}

	if config.ReverseProxy == nil {
		t.Errorf("Reverse Proxy build error")
	}

	if len(config.RuleMap) != 1 {
		t.Errorf("Rule map build error: %+v", config.RuleMap)
	}

	if len(config.RoutingMap) != 1 {
		t.Errorf("Routing map build error: %+v", config.RoutingMap)
	}
}
