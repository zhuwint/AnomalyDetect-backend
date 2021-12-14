package main

import (
	"anomaly-detect/cmd/alertengine/alert"
	"anomaly-detect/cmd/alertengine/config"
	"anomaly-detect/cmd/alertengine/server"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var configFilePath = flag.String("config", "", "input config file path")

const info = "usage \n" + "		-config string		config file path"

func main() {
	flag.Parse()
	if *configFilePath == "" {
		fmt.Printf("please input config file path\n\n")
		fmt.Print(info)
		return
	}

	conf, err := config.ParseYaml(*configFilePath)
	if err != nil {
		logrus.Errorf("server start failed: %s", err.Error())
		return
	}

	alert.InitMail(conf.Smtp.Host, conf.Smtp.Port, conf.Smtp.Account, conf.Smtp.Password)

	exit := make(chan error)      // internal exit signal, cause by program error
	sg := make(chan os.Signal, 1) // external interrupt signal, send by user
	signal.Notify(sg, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	serv := server.NewServer(exit)
	go serv.Start()

	select {
	case info := <-sg:
		serv.Stop()
		logrus.Infof("service stop: %s", info.String())
	case err := <-exit:
		serv.Stop()
		logrus.Errorf("service stop: %s", err.Error())
	}
}
