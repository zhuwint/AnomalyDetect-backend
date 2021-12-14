package config

import (
	"anomaly-detect/pkg/mysql"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

type SmtpService struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

func (s SmtpService) Validate() error {
	if len(strings.Split(s.Host, ".")) != 4 {
		return fmt.Errorf("invalid smtp host")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("invalid smtp port")
	}
	if s.Account == "" {
		return fmt.Errorf("invalid smtp account")
	}
	if s.Password == "" {
		return fmt.Errorf("invalid smtp password")
	}
	return nil
}

type Config struct {
	Mysql mysql.Account `yaml:"mysql"`
	Smtp  SmtpService   `yaml:"smtp"`
}

func (c Config) Validate() error {
	if err := c.Mysql.Validate(); err != nil {
		return err
	}
	if err := c.Smtp.Validate(); err != nil {
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
