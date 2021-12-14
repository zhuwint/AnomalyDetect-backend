package influxdb

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Account struct {
	Address string `yaml:"address"`
	Bucket  string `yaml:"bucket"`
	Token   string `yaml:"token"`
	Org     string `yaml:"org"`
}

func (i Account) Validate() error {
	if i.Bucket == "" {
		return fmt.Errorf("influxdb: bucket coult not be empty")
	}
	if i.Token == "" {
		return fmt.Errorf("influxdb: token could not be empty")
	}
	if i.Org == "" {
		return fmt.Errorf("influxdb: org could not be empty")
	}
	if !strings.HasPrefix(i.Address, "http://") {
		return fmt.Errorf("influxdb: address invalid, except http://xxx.xxx.xxx.xxx:xxx")
	} else {
		host := i.Address[7:]
		if p := strings.Split(host, ":"); len(p) != 2 {
			return fmt.Errorf("influxdb: address invalid, except http://xxx.xxx.xxx.xxx:xxx")
		} else {
			if len(net.ParseIP(p[0])) == 0 {
				return fmt.Errorf("influxdb: address invalid, except http://xxx.xxx.xxx.xxx:xxx")
			}
			if port, err := strconv.ParseInt(p[1], 10, 32); err != nil || port < 0 || port > 65535 {
				return fmt.Errorf("influxdb: address invalid, except http://xxx.xxx.xxx.xxx:xxx")
			}
		}
	}
	return nil
}
