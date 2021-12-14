package impl

import (
	"anomaly-detect/pkg/concurrency"
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"
)

// RuntimeState 运行时状态
type RuntimeState struct {
	enabled   concurrency.Bool // if the task/subtask enabled or disabled
	last      concurrency.Time // last triggered time
	next      concurrency.Time // next trigger time
	triggered int32            // triggered number
}

// IsEnabled : if the task/subtask is enabled or disabled
func (r *RuntimeState) IsEnabled() bool {
	return r.enabled.Get()
}

// Enable : enable the task/subtask
func (r *RuntimeState) Enable() {
	r.enabled.Set(true)
}

// Disable : disable the task/subtask
func (r *RuntimeState) Disable() {
	r.enabled.Set(false)
}

func (r *RuntimeState) Triggered() int32 {
	return r.triggered
}

func (r *RuntimeState) SetTriggered(t int32) {
	atomic.StoreInt32(&r.triggered, t)
}

func (r *RuntimeState) Last() time.Time {
	return r.last.Get()
}

func (r *RuntimeState) SetLast(t time.Time) {
	r.last.Set(t)
}

func (r *RuntimeState) Next() time.Time {
	return r.next.Get()
}

func (r *RuntimeState) SetNext(t time.Time) {
	r.next.Set(t)
}

// SimpleStatus 简单状态
type SimpleStatus struct {
	ProjectId      int     `json:"project_id"`
	TaskId         string  `json:"task_id"`
	Model          string  `json:"model"` // 异常检测模型
	IsStream       bool    `json:"is_stream"`
	SensorMac      string  `json:"sensor_mac"`
	SensorType     string  `json:"sensor_type"`
	ReceiveNo      string  `json:"receive_no"`
	UpdateEnable   bool    `json:"update_enable"`
	DetectEnable   bool    `json:"detect_enable"`
	ThresholdUpper float64 `json:"threshold_upper"`
	ThresholdLower float64 `json:"threshold_lower"`
	CurrentValue   float64 `json:"current_value"`
	IsAnomaly      bool    `json:"is_anomaly"`
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

type BatchState struct {
	Enable    bool      `json:"enable"`
	Last      time.Time `json:"last"`
	Next      time.Time `json:"next"`
	Triggered int       `json:"triggered"`
}

// BatchStatus 完整状态
type BatchStatus struct {
	Info           BatchTaskInfo `json:"info"`
	Created        time.Time     `json:"created"`
	Updated        time.Time     `json:"updated"`
	ModelUpdate    BatchState    `json:"model_update"`
	AnomalyDetect  BatchState    `json:"anomaly_detect"`
	ThresholdUpper float64       `json:"threshold_upper"`
	ThresholdLower float64       `json:"threshold_lower"`
	CurrentValue   float64       `json:"current_value"`
	IsAnomaly      bool          `json:"is_anomaly"`
}

func (s BatchStatus) GetProjectId() string {
	return s.Info.GetProjectId()
}

func (s BatchStatus) GetTaskId() string {
	return s.Info.GetTaskId()
}

func (s BatchStatus) MarshToJson() []byte {
	data, _ := json.Marshal(s)
	return data
}

type StreamState struct {
	Enable    bool `json:"enable"`
	Triggered int  `json:"triggered"`
}

type StreamStatus struct {
	Info           StreamTaskInfo `json:"info"`
	Created        time.Time      `json:"created"`
	Updated        time.Time      `json:"updated"`
	ModelUpdate    BatchState     `json:"model_update"`
	AnomalyDetect  StreamState    `json:"anomaly_detect"`
	ThresholdUpper float64        `json:"threshold_upper"`
	ThresholdLower float64        `json:"threshold_lower"`
	CurrentValue   float64        `json:"current_value"`
	IsAnomaly      bool           `json:"is_anomaly"`
}

func (s StreamStatus) GetProjectId() string {
	return s.Info.GetProjectId()
}

func (s StreamStatus) GetTaskId() string {
	return s.Info.GetTaskId()
}

func (s StreamStatus) MarshToJson() []byte {
	data, _ := json.Marshal(s)
	return data
}
