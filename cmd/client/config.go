package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LocalAddress  string    `yaml:"local_address"` // e.g., 127.0.0.1:8080
	SignalBaseURL string    `yaml:"signal_base_url"`
	Timeout       int       `yaml:"timeout"` // in seconds
	UserID        uint64    `yaml:"user_id"`
	Password      string    `yaml:"password"`
	ICEServer     ICEServer `yaml:"ice_server"`
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
