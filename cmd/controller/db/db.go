package db

import (
	"anomaly-detect/cmd/controller/model"
	"anomaly-detect/pkg/influxdb"
	"anomaly-detect/pkg/mysql"
	"fmt"
	"sync"
)

var (
	MysqlClient    *mysql.Connector
	InfluxdbClient *influxdb.Connector
	mysqlOnce      = &sync.Once{}
	influxOnce     = &sync.Once{}
)

func InitMysqlClient(c mysql.Account) {
	mysqlOnce.Do(func() {
		conn, err := mysql.NewConnector(c.Address, c.Database, c.Username, c.Password)
		if err != nil {
			panic(fmt.Errorf("init mysql conn failed %s", err.Error()))
		}
		MysqlClient = conn
	})
}

func InitInfluxdbClient(c influxdb.Account) {
	influxOnce.Do(func() {
		conn, err := influxdb.NewConnector(c.Address, c.Bucket, c.Token, c.Org)
		if err != nil {
			panic(fmt.Errorf("init influxdb conn failed %s", err.Error()))
		}
		InfluxdbClient = conn
	})
}

func InfluxdbClientClose() {
	if InfluxdbClient != nil {
		InfluxdbClient.Close()
	}
}

func Init() {
	// init table
	if MysqlClient == nil {
		panic("mysql client has not init")
	}
	if !MysqlClient.DB.Migrator().HasTable(&model.Task{}) {
		if err := MysqlClient.DB.AutoMigrate(&model.Task{}); err != nil {
			panic(fmt.Sprintf("cannot migrate database: %s", err.Error()))
		}
	}
	if !MysqlClient.DB.Migrator().HasTable(&model.InvokeService{}) {
		if err := MysqlClient.DB.AutoMigrate(&model.InvokeService{}); err != nil {
			panic(fmt.Sprintf("cannot migrate database: %s", err.Error()))
		}
	}
	//if !MysqlClient.DB.Migrator().HasTable(&model.AlertRecord{}) {
	//	if err := MysqlClient.DB.AutoMigrate(&model.AlertRecord{}); err != nil {
	//		panic(fmt.Sprintf("cannot migrate database: %s", err.Error()))
	//	}
	//}
}
