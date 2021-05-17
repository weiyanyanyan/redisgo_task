package config

import (
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	config *Config
	once   sync.Once
)

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type Config struct {
	RedisInfo `toml:"redis"`
}

type RedisInfo struct {
	Host          string   `toml:"host"`
	Key           string   `toml:"key"`
	Value         string   `toml:"value"`
	Expire        duration `toml:"expire"`
	RetriesCount  float64 	`toml:"retries_count"`
	MonitorTryAll bool     `toml:"monitor_try_all"`
	Cron          duration `toml:"cron"`
}

func LoadConfig(fname string) *Config {
	once.Do(func() {
		if _, err := toml.DecodeFile(fname, &config); err != nil {
			panic(err)
		}
	})
	return config
}

func GetConfig() *Config {
	return config
}
