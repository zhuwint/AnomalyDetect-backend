package server

import (
	"anomaly-detect/cmd/alertengine/alert"
	"anomaly-detect/pkg/dapr"
	"anomaly-detect/pkg/env"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	*dapr.Dapr
	subscribes map[string]*alert.Subscribe
	rw         sync.RWMutex
	exit       chan error
}

func NewServer(exit chan error) *Server {
	serviceName := env.GetEnvString(dapr.AppIdEnv, defaultServiceName)
	servicePort := env.GetEnvInt(dapr.AppPortEnv, defaultHttpPort)
	daprInstance := dapr.NewDapr(serviceName, servicePort)
	return &Server{
		Dapr:       daprInstance,
		subscribes: make(map[string]*alert.Subscribe),
		exit:       exit,
	}
}

func (s *Server) Start() {
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	//// routes
	api := r.PathPrefix(BasePath).Subrouter()
	api.HandleFunc(AlertPath, s.push).Methods(http.MethodPost)
	api.HandleFunc(SubscribePath, s.createSubscribe).Methods(http.MethodPost)
	api.HandleFunc(SubscribePath, s.getSubscribe).Methods(http.MethodGet)
	api.HandleFunc(SubscribePath, s.updateSubscribe).Methods(http.MethodPatch)
	api.HandleFunc(SubscribePath, s.deleteSubscribe).Methods(http.MethodDelete)

	addr := fmt.Sprintf(":%d", s.AppPort)
	logrus.Infof("start listening on port %d", s.AppPort)
	if err := http.ListenAndServe(addr, r); err != nil {
		s.exit <- err
	}
}

func (s *Server) Stop() {
	//TODO: do something clean
}

func (s *Server) Save() {

}

func writeJson(w http.ResponseWriter, data interface{}) error {
	content, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err = w.Write(content); err != nil {
		return err
	}
	return nil
}

func write(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	_ = writeJson(w, data)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logrus.Infof("from=%s req=%s method=%s", req.RemoteAddr, req.RequestURI, req.Method)
		next.ServeHTTP(w, req)
	})
}

type StringMessage struct {
	Subject string `json:"subject"`
	Msg     string `json:"msg"`
}

func (s StringMessage) Title() string {
	return s.Subject
}

func (s StringMessage) Content() string {
	return s.Msg
}

func (s *Server) push(resp http.ResponseWriter, req *http.Request) {
	type requestBody struct {
		Topic string        `json:"topic"`
		Msg   StringMessage `json:"msg"`
	}
	data := requestBody{}
	body, _ := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	} else {
		err = s.sendMessage(data.Topic, data.Msg)
		if err != nil {
			write(resp, http.StatusInternalServerError, map[string]interface{}{"error": "push message failed: " + err.Error()})
		} else {
			write(resp, http.StatusOK, map[string]interface{}{"msg": "write success"})
		}
	}
}

// create subscribe
func (s *Server) createSubscribe(resp http.ResponseWriter, req *http.Request) {
	type requestBody struct {
		Topic string          `json:"topic"`
		To    []alert.BaseApi `json:"to"`
	}
	data := requestBody{}
	body, _ := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	s.rw.Lock()
	defer s.rw.Unlock()
	if _, ok := s.subscribes[data.Topic]; ok {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": "topic already existed"})
		return
	} else {
		var rs []alert.Receiver
		for _, r := range data.To {
			if temp := createReceiver(r); temp != nil {
				rs = append(rs, temp)
			} else {
				write(resp, http.StatusBadRequest, map[string]interface{}{"error": "type not match"})
			}
		}
		s.subscribes[data.Topic] = &alert.Subscribe{
			Topic:         data.Topic,
			Receiver:      rs,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Triggered:     0,
			LastTriggered: time.Time{},
		}
	}
	write(resp, http.StatusOK, map[string]interface{}{"msg": "create success"})
}

func createReceiver(r alert.BaseApi) alert.Receiver {
	switch r.Type {
	case "email":
		return alert.Mail{BaseApi: r}
	case "dingTalk":
		return alert.DingTalk{BaseApi: r}
	default:
		return nil
	}
}

// update subscribe
func (s *Server) updateSubscribe(resp http.ResponseWriter, req *http.Request) {
	topic := req.URL.Query().Get("topic")
	if topic == "" {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": "topic could not be empty"})
		return
	}

	type requestBody struct {
		To []alert.Receiver `json:"to"`
	}
	data := requestBody{}
	body, _ := ioutil.ReadAll(req.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	s.rw.Lock()
	defer s.rw.Unlock()
	if sub, ok := s.subscribes[topic]; !ok {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": "topic not exists"})
	} else {
		sub.UpdateReceiver(data.To)
		write(resp, http.StatusOK, map[string]interface{}{"msg": "update success"})
	}
}

// get subscribe
func (s *Server) getSubscribe(resp http.ResponseWriter, req *http.Request) {
	type response struct {
		Msg  string             `json:"msg"`
		Data []*alert.Subscribe `json:"data"`
	}
	data := response{}
	s.rw.RLock()
	defer s.rw.RUnlock()
	for _, sub := range s.subscribes {
		data.Data = append(data.Data, sub)
	}
	write(resp, http.StatusOK, data)
}

// delete subscribe
func (s *Server) deleteSubscribe(resp http.ResponseWriter, req *http.Request) {
	topic := req.URL.Query().Get("topic")
	s.rw.Lock()
	defer s.rw.Unlock()
	if _, ok := s.subscribes[topic]; !ok {
		write(resp, http.StatusBadRequest, map[string]interface{}{"error": "topic not exists"})
	} else {
		delete(s.subscribes, topic)
		write(resp, http.StatusOK, map[string]interface{}{"msg": "delete success"})
	}
}

func (s *Server) sendMessage(taskName string, message alert.Message) error {
	s.rw.Lock()
	defer s.rw.Unlock()
	if sub, ok := s.subscribes[taskName]; !ok {
		return nil
	} else {
		sub.Triggered++
		sub.LastTriggered = time.Now()
		return sub.Send(message)
	}
}
