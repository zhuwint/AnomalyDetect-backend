package config

import (
	"anomaly-detect/pkg/influxdb"
	"anomaly-detect/pkg/mysql"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Influxdb influxdb.Account `yaml:"influxdb"`
	Mysql    mysql.Account    `yaml:"mysql"`
}

func (c Config) Validate() error {
	if err := c.Influxdb.Validate(); err != nil {
		return err
	}
	if err := c.Mysql.Validate(); err != nil {
		return err
	}
	return nil
}

func ParseYaml(path string) (*Config, error) {
	conf := &Config{}
	if f, err := os.Open(path); err != nil {
		return nil, err
	} else {
		if err = yaml.NewDecoder(f).Decode(conf); err != nil {
			return nil, err
		}
		return conf, nil
	}
}
