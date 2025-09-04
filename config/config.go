package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	cfgProvider ConfigProvider
}

const (
	Port           = "PORT"
	BindAddress    = "BIND_ADDRESS"
	DatabaseURL    = "DATABASE_URL"
	MaxConnections = "MAX_CONNECTIONS"
	MaxThreads     = "MAX_THREADS"
)

func (c *Config) GetValue(key string) string {
	return c.cfgProvider.GetValue(key)
}

type ConfigProvider interface {
	GetValue(key string) string
}

type DefaultProvider struct {
}

func (p *DefaultProvider) GetValue(key string) string {
	return os.Getenv(key)
}

func NewFileProvider(filePath string) ConfigProvider {
	godotenv.Load(filePath)
	return &DefaultProvider{}
}

func NewConfig(provider ConfigProvider) *Config {
	return &Config{
		cfgProvider: provider,
	}
}
