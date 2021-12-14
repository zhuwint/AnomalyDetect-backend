package alert

import (
	"fmt"
	gomail "gopkg.in/mail.v2"
	"sync"
)

var client *gomail.Dialer
var mailFrom string
var once sync.Once

func InitMail(host string, port int, account string, password string) {
	once.Do(func() {
		client = gomail.NewDialer(host, port, account, password)
		mailFrom = account
	})
}

type Mail struct {
	BaseApi
}

func (m Mail) Id() string {
	return m.Address
}

func (m Mail) Message(message Message) error {
	if client == nil {
		return fmt.Errorf("mail client not init")
	}
	ms := gomail.NewMessage()
	ms.SetHeader("From", mailFrom)
	ms.SetHeader("To", m.Address)
	ms.SetHeader("Subject", message.Title())
	ms.SetBody("text/html", message.Content())
	if err := client.DialAndSend(ms); err != nil {
		return err
	}
	return nil
}
