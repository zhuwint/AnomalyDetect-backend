package api

import (
	"anomaly-detect/pkg/validator"
	"time"
)

// Status 任务状态
type Status interface {
	GetProjectId() string
	GetTaskId() string
	MarshToJson() []byte
}

// Info 任务信息，创建任务时使用
type Info interface {
	validator.Validator
	GetTaskId() string
	IsStreamTask() bool
	IsUnionTask() bool
	GetProjectId() string
	MarshToJson() []byte
}

// Task 任务
type Task interface {
	ProjectId() string                                             // 返回 task 的 project_id
	TaskId() string                                                // 返回 task_id
	IsStream() bool                                                // 是否是流任务
	IsUnion() bool                                                 // 是否是联合告警任务
	Start() error                                                  // 启动 task
	Stop() error                                                   // 停止 task
	Restart() error                                                // 重启 task
	Update(interface{}) error                                      // 更新 task
	SubKey() []string                                              // 订阅数据流， 仅 stream 类型有效
	Run(string, string, string, string, float64, time.Time)        // 仅 stream 类型的 task
	Save() error                                                   // task 持久化
	Status() Status                                                // 完整状态，用于返回单个任务时使用
	SimpleStatus() Status                                          // 简单状态，用于返回任务列表时使用
	EnableModelUpdate(bool) error                                  // 启动/停止模型更新(阈值更新)
	EnableAnomalyDetect(bool) error                                // 启动/停止异常检测
	SetThreshold(string, string, string, *float64, *float64) error // 设置阈值 lower upper
}

// 告警等级, 0 表示正常计算

type Level int

const (
	InfoLevel Level = iota
	WarnLevel
	AlertLevel
)
