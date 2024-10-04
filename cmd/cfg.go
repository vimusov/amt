package main

import (
	"github.com/BurntSushi/toml"
	"os"
	"strings"
)

type netMirror struct {
	Enabled  bool     `toml:"enabled"`
	Uri      string   `toml:"uri"`
	Arch     string   `toml:"arch"`
	Sections []string `toml:"sections"`
	Threads  uint     `toml:"threads"`
}

type netConfig struct {
	RootDir string               `toml:"rootdir"`
	Mirrors map[string]netMirror `toml:"mirror"`
}

func formatUrl(uri, arch, section string) string {
	uri = strings.Replace(uri, "%arch%", arch, -1)
	uri = strings.Replace(uri, "%section%", section, -1)
	return strings.TrimSuffix(uri, "/")
}

func readConfig(path string) (*netConfig, error) {
	var cfg netConfig
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return nil, homeErr
	}
	_, decodeErr := toml.DecodeFile(strings.Replace(path, "~", homeDir, 1), &cfg)
	if decodeErr != nil {
		return nil, decodeErr
	}
	return &cfg, nil
}
