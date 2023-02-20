package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerAddr string    `yaml:"server_addr"` // e.g., 0.0.0.0:29443
	Timeout    int       `yaml:"timeout"`     // in seconds
	WebSignal  WebSignal `yaml:"web_signal"`
	ICEServer  ICEServer `yaml:"ice_server"`
}

type WebSignal struct {
	DefaultUsers map[uint64]string `yaml:"default_users"`
}

type ICEServer struct {
	URLs []string `yaml:"urls"`
}

func loadConfig(configFile string) (*Config, error) {
	var config Config

	// Load the config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal the config file
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
