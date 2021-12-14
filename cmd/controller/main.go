package main

import (
	"anomaly-detect/cmd/controller/config"
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/server"
	"anomaly-detect/cmd/controller/task/service"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
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

	// init influxdb connector
	db.InitInfluxdbClient(conf.Influxdb)
	logrus.Infof("init influxdb success using config bucket:%s org:%s url:%s",
		conf.Influxdb.Bucket,
		conf.Influxdb.Org,
		conf.Influxdb.Address,
	)

	// init mysql connector
	db.InitMysqlClient(conf.Mysql)
	logrus.Infof("init mysql success using config database:%s username:%s url:%s",
		conf.Mysql.Database,
		conf.Mysql.Username,
		conf.Mysql.Address,
	)

	db.Init()

	defer db.InfluxdbClientClose()

	service.Load() // 载入模型

	serv, err := server.NewController()
	if err != nil {
		logrus.Errorf("server start failed: %s", err.Error())
		return
	}

	if err := serv.Start(); err != nil {
		logrus.Errorf("server start failed: %s", err.Error())
	}
	serv.Stop()
}
