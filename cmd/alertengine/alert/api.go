package alert

type Sender interface {
	// Push : push messages to a group of users or specific users
	Push(address string, message Message) error
	// Broadcast : push messages to all users
	Broadcast(message Message) error
}

type Receiver interface {
	Id() string
	Message(message Message) error
}

type Message interface {
	Title() string
	Content() string
}

type BaseApi struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Token   string `json:"token"`
}
