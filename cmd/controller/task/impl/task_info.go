package impl

import (
	"anomaly-detect/cmd/controller/task/service"
	"anomaly-detect/pkg/validator"
	"encoding/json"
	"fmt"
	"strconv"
)

// ModelService 模型调用：模型名称以及调用参数
type ModelService struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

func (m ModelService) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	return service.ParamsValidate(m.Name, m.Params) // 进行模型参数验证
}

type BatchMeta struct {
	Interval string        `json:"interval"`
	Query    *QueryOptions `json:"query"`
}

func (t BatchMeta) Validate() error {
	if _, err := validator.CheckDurationPositive(t.Interval); err != nil {
		return err
	}
	if t.Query == nil {
		return fmt.Errorf("query options cannot be empty")
	} else if err := t.Query.Validate(); err != nil {
		return err
	}
	return nil
}

// BatchTaskInfo 用于创建任务
type BatchTaskInfo struct {
	TaskId        string           `json:"task_id"`
	ProjectId     int              `json:"project_id"`
	Preprocess    *ModelService    `json:"preprocess"`
	DetectModel   *ModelService    `json:"detect_model"`
	Target        UnvariedSeries   `json:"target"`         // 目标检测序列
	Independent   []UnvariedSeries `json:"independent"`    // 其它序列（自变量）
	ModelUpdate   *BatchMeta       `json:"model_update"`   // 用于更新阈值
	AnomalyDetect *BatchMeta       `json:"anomaly_detect"` // 用于计算特征值
	IsStream      bool             `json:"is_stream"`
	Level         int              `json:"level"` // 告警等级
}

func (t BatchTaskInfo) GetTaskId() string {
	return t.TaskId
}

func (t BatchTaskInfo) GetProjectId() string {
	return strconv.Itoa(t.ProjectId)
}

func (t BatchTaskInfo) IsStreamTask() bool {
	return false
}

func (t BatchTaskInfo) IsUnionTask() bool {
	return false
}

func (t BatchTaskInfo) MarshToJson() []byte {
	data, _ := json.Marshal(t)
	return data
}

func (t BatchTaskInfo) Validate() error {
	if t.TaskId == "" {
		return fmt.Errorf("taskId cannot be empty")
	}
	if len(t.TaskId) < 4 || len(t.TaskId) > 20 {
		return fmt.Errorf("task id length must between 4 and 20")
	}
	if t.ProjectId <= 0 {
		return fmt.Errorf("project_id cannot be empty")
	}
	if t.Preprocess != nil {
		if err := t.Preprocess.Validate(); err != nil {
			return err
		}
	}
	if t.DetectModel != nil {
		if err := t.DetectModel.Validate(); err != nil {
			return err
		}
	}
	if err := t.Target.Validate(); err != nil {
		return fmt.Errorf("target series: %s", err.Error())
	}
	for _, s := range t.Independent {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("other independent series: %s", err.Error())
		}
	}
	if t.ModelUpdate != nil {
		if err := t.ModelUpdate.Validate(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("undefined model update")
	}
	if t.AnomalyDetect != nil {
		if err := t.AnomalyDetect.Validate(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("undefined anomaly detect")
	}
	if t.Level < 0 {
		return fmt.Errorf("level must >= 0")
	}
	return nil
}

type StreamMeta struct {
	//Bucket      string          `json:"bucket"`
	//Measurement string          `json:"measurement"`
	//Series      *UnvariedSeries `json:"series"`
	Duration string `json:"duration"` // 持续多少时间告警
}

func (s StreamMeta) Validate() error {
	//if s.Bucket == "" {
	//	return fmt.Errorf("bucket cannot be empty")
	//}
	//if s.Measurement == "" {
	//	return fmt.Errorf("measurement cannot be empty")
	//}
	//if s.Series == nil {
	//	return fmt.Errorf("series cannot be empty")
	//}
	//if s.Series.SensorMac == "" || s.Series.SensorType == "" || s.Series.ReceiveNo == "" {
	//	return fmt.Errorf("series information cannot be empty")
	//}
	if _, err := validator.CheckDurationPositive(s.Duration); err != nil {
		return err
	}
	return nil
}

// StreamTaskInfo stream 类型的任务只有模型更新时用到模型调用，异常检测为实时值判断
type StreamTaskInfo struct {
	TaskId        string           `json:"task_id"`
	ProjectId     int              `json:"project_id"`
	Preprocess    *ModelService    `json:"preprocess"`
	DetectModel   *ModelService    `json:"detect_model"`
	Target        UnvariedSeries   `json:"target"`       // 目标检测序列
	Independent   []UnvariedSeries `json:"independent"`  // 其它序列（自变量）
	ModelUpdate   *BatchMeta       `json:"model_update"` // 用于更新阈值
	AnomalyDetect *StreamMeta      `json:"anomaly_detect"`
	IsStream      bool             `json:"is_stream"`
	Level         int              `json:"level"` // 告警等级
}

func (s StreamTaskInfo) Validate() error {
	if s.TaskId == "" {
		return fmt.Errorf("taskId cannot be empty")
	}
	if len(s.TaskId) < 4 || len(s.TaskId) > 20 {
		return fmt.Errorf("task id length must between 4 and 20")
	}
	if s.ProjectId <= 0 {
		return fmt.Errorf("projectId cannot be empty")
	}
	if s.Preprocess != nil {
		if err := s.Preprocess.Validate(); err != nil {
			return err
		}
	}
	if err := s.Target.Validate(); err != nil {
		return fmt.Errorf("target series: %s", err.Error())
	}
	for _, s := range s.Independent {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("other independent series: %s", err.Error())
		}
	}
	if s.DetectModel != nil {
		if err := s.DetectModel.Validate(); err != nil {
			return err
		}
	}
	if s.ModelUpdate != nil {
		if err := s.ModelUpdate.Validate(); err != nil {
			return err
		}
	}
	if s.AnomalyDetect == nil {
		return fmt.Errorf("undefined anomaly detect")
	}
	if err := s.AnomalyDetect.Validate(); err != nil {
		return err
	}
	if s.Level < 0 {
		return fmt.Errorf("level must > 0")
	}
	return nil
}

func (s StreamTaskInfo) GetTaskId() string {
	return s.TaskId
}

func (s StreamTaskInfo) IsStreamTask() bool {
	return true
}

func (s StreamTaskInfo) IsUnionTask() bool {
	return false
}

func (s StreamTaskInfo) GetProjectId() string {
	return strconv.Itoa(s.ProjectId)
}

func (s StreamTaskInfo) MarshToJson() []byte {
	data, _ := json.Marshal(s)
	return data
}
