package alert

import (
	"fmt"
	"sync"
	"time"
)

// Subscribe : subscribe associate to task (taskName), store in redis, set expire time 1h and keep heartbeat
type Subscribe struct {
	Topic         string     `json:"topic"`
	Receiver      []Receiver `json:"receiver"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Triggered     int        `json:"triggered"`
	LastTriggered time.Time  `json:"last_triggered"`
	rw            sync.RWMutex
}

func (s *Subscribe) Send(message Message) error {
	s.rw.Lock()
	defer s.rw.Unlock()
	var failed []string
	for _, r := range s.Receiver {
		err := r.Message(message)
		if err != nil {
			failed = append(failed, err.Error())
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("subscribe %s message push failed %d", s.Topic, len(failed))
	}
	return nil
}

// AddReceiver : add receiver
func (s *Subscribe) AddReceiver(receiver Receiver) error {
	s.rw.RLock()
	for _, r := range s.Receiver {
		if r.Id() == receiver.Id() {
			return fmt.Errorf("receiver %s already existed", r.Id())
		}
	}
	s.rw.RUnlock()
	s.rw.Lock()
	defer s.rw.Unlock()
	s.Receiver = append(s.Receiver, receiver)
	return nil
}

func (s *Subscribe) DeleteReceiver(id string) error {
	s.rw.Lock()
	defer s.rw.Unlock()
	for i, r := range s.Receiver {
		if r.Id() == id {
			if i == 0 {
				s.Receiver = s.Receiver[1:]
			} else if i == len(s.Receiver)-1 {
				s.Receiver = s.Receiver[:len(s.Receiver)-1]
			} else {
				s.Receiver = append(s.Receiver[:i], s.Receiver[i+1:]...)
			}
		}
	}
	return fmt.Errorf("receiver %s does not exist", id)
}

func (s *Subscribe) UpdateReceiver(receiver []Receiver) {
	s.Receiver = receiver
}
