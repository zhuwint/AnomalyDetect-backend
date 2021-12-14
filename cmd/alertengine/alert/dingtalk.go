package alert

import (
	"github.com/blinkbean/dingtalk"
)

type DingTalk struct {
	BaseApi
}

func (d DingTalk) Id() string {
	return d.Address
}

func (d DingTalk) Message(message Message) error {
	robot := dingtalk.InitDingTalkWithSecret(d.Address, d.Token)
	return robot.SendMarkDownMessage(message.Title(), message.Content())
}
