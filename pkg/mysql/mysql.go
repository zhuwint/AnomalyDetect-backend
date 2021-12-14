package mysql

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Connector struct {
	Address  string
	Database string
	Username string
	Password string
	DB       *gorm.DB
}

func NewConnector(address, database, username, password string) (*Connector, error) {
	link := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", username, password, address, database)
	db, err := gorm.Open(mysql.Open(link), &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		return nil, err
	}

	return &Connector{
		Address:  address,
		Database: database,
		Username: username,
		Password: password,
		DB:       db,
	}, nil
}
