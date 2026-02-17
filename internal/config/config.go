package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AppConfig contains application and server configuration.
type AppConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	Name         string `json:"name"`
	DefaultGreet string `json:"default_greet"`
	LogLevel     string `json:"log_level"`

	JSONOutput bool `json:"json_output"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		Name:         "go-hello-world",
		DefaultGreet: "world",
		LogLevel:     "info",
		JSONOutput:   false,
	}
}

// Load reads JSON config from path (if provided), then applies APP_* env overrides.
func Load(path string) (AppConfig, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				applyEnvOverrides(&cfg)
				return cfg, nil
			}
			return cfg, fmt.Errorf("read config %s: %w", path, err)
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", path, err)
		}
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

func MustLoad(path string) AppConfig {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

func (c AppConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func applyEnvOverrides(cfg *AppConfig) {
	if v := os.Getenv("APP_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("APP_NAME"); v != "" {
		cfg.Name = v
	}
	if v := os.Getenv("APP_DEFAULT_GREET"); v != "" {
		cfg.DefaultGreet = v
	}
	if v := os.Getenv("APP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := os.Getenv("APP_JSON_OUTPUT"); v != "" {
		cfg.JSONOutput = v == "true" || v == "1"
	}
}
