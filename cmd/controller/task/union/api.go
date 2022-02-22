package union

import (
	"anomaly-detect/pkg/validator"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Meta struct {
	SensorMac      string  `json:"sensor_mac"`
	ReceiveNo      string  `json:"receive_no"`
	SensorType     string  `json:"sensor_type"`
	ThresholdUpper float64 `json:"threshold_upper"`
	ThresholdLower float64 `json:"threshold_lower"`
}

func (u Meta) Key() string {
	return fmt.Sprintf("%s#%s#%s", u.SensorMac, u.SensorType, u.ReceiveNo)
}

func (u Meta) Validate() error {
	if u.SensorMac == "" || u.ReceiveNo == "" || u.SensorType == "" {
		return fmt.Errorf("unvalid series")
	}
	return nil
}

type TaskInfo struct {
	TaskId      string `json:"task_id"`
	TaskName    string `json:"task_name"`
	ProjectId   int    `json:"project_id"`
	Bucket      string `json:"bucket"`
	Measurement string `json:"measurement"`
	Series      []Meta `json:"series"`
	Operate     []int  `json:"operate"` // 0表示与， 其它表示或
	Duration    string `json:"duration"`
	IsStream    bool   `json:"is_stream"`
	Level       int    `json:"level"`
}

func (u TaskInfo) Validate() error {
	if u.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	if len(u.TaskId) < 4 || len(u.TaskId) > 20 {
		return fmt.Errorf("length of task id must between 4 and 20")
	}
	if len(u.TaskName) < 4 || len(u.TaskName) > 20 {
		return fmt.Errorf("length of task name must between 4 and 20")
	}
	if u.ProjectId <= 0 {
		return fmt.Errorf("projcet id must > 0")
	}
	if u.Bucket == "" {
		return fmt.Errorf("bucket cannot be empty")
	}
	if u.Measurement == "" {
		return fmt.Errorf("measurement cannot be empty")
	}
	if len(u.Series) == 0 {
		return fmt.Errorf("series cannot be empty")
	}
	if len(u.Operate) != len(u.Series)-1 {
		return fmt.Errorf("length of operate not match series")
	}
	for _, s := range u.Series {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	if _, err := validator.CheckDurationPositive(u.Duration); err != nil {
		return err
	}
	if u.Level < 0 {
		return fmt.Errorf("level must >= 0")
	}
	return nil
}

func (u TaskInfo) GetTaskId() string {
	return u.TaskId
}

func (u TaskInfo) IsStreamTask() bool {
	return true
}

func (u TaskInfo) IsUnionTask() bool {
	return true
}

func (u TaskInfo) GetProjectId() string {
	return strconv.Itoa(u.ProjectId)
}

func (u TaskInfo) MarshToJson() []byte {
	data, _ := json.Marshal(u)
	return data
}

type RunTimeState struct {
	Last      time.Time `json:"last"`
	Triggered int       `json:"triggered"`
	Value     float64   `json:"value"`
}

type Status struct {
	Info      TaskInfo                `json:"info"`
	Created   time.Time               `json:"created"`
	Updated   time.Time               `json:"updated"`
	State     map[string]RunTimeState `json:"state"`
	Enable    bool                    `json:"enable"`
	IsAnomaly bool                    `json:"is_anomaly"`
}

func (s Status) GetProjectId() string {
	return s.Info.GetProjectId()
}

func (s Status) GetTaskId() string {
	return s.Info.TaskId
}

func (s Status) MarshToJson() []byte {
	data, _ := json.Marshal(s)
	return data
}

type SimpleStatus struct {
	ProjectId int    `json:"project_id"`
	TaskId    string `json:"task_id"`
	TaskName  string `json:"task_name"`
	Enable    bool   `json:"enable"`
	IsAnomaly bool   `json:"is_anomaly"`
}

func (s SimpleStatus) GetProjectId() string {
	return strconv.Itoa(s.ProjectId)
}

func (s SimpleStatus) GetTaskId() string {
	return s.TaskId
}

func (s SimpleStatus) MarshToJson() []byte {
	data, _ := json.Marshal(s)
	return data
}
