package mysql

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Account struct {
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (m Account) Validate() error {
	if m.Username == "" {
		return fmt.Errorf("mysql: username could not be empty")
	}
	if m.Password == "" {
		return fmt.Errorf("mysql: password could not be empty")
	}
	if p := strings.Split(m.Address, ":"); len(p) != 2 {
		return fmt.Errorf("mysql: address invalid, except xxxx.xxx.xxx.xx:xx")
	} else {
		if len(net.ParseIP(p[0])) == 0 {
			return fmt.Errorf("mysql: ip address invalid")
		}
		if port, err := strconv.ParseInt(p[1], 10, 32); err != nil || port < 0 || port > 65535 {
			return fmt.Errorf("mysql: port invalid")
		}
	}
	if m.Database == "" {
		return fmt.Errorf("mysql: database could not be empty")
	}
	return nil
}
