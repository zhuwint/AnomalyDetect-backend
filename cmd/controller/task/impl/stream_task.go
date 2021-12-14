package impl

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/task/api"
	"anomaly-detect/cmd/controller/task/record"
	"anomaly-detect/cmd/controller/task/service"
	"anomaly-detect/cmd/controller/task/store"
	"anomaly-detect/pkg/concurrency"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type StreamTask struct {
	info    StreamTaskInfo
	created time.Time
	updated time.Time

	modelUpdateState RuntimeState // 模型更新运行时状态

	detectEnabled  concurrency.Bool // 是否开启异常检测
	thresholdUpper concurrency.Float64
	thresholdLower concurrency.Float64
	currentValue   concurrency.Float64
	triggered      concurrency.Int64

	duration  time.Duration // 持续多少时间告警
	isAnomaly bool          // 当前状态
	timer     time.Time     // 计时器

	exit context.CancelFunc

	rw sync.RWMutex
}

func NewStreamTask(info api.Info) (*StreamTask, error) {
	streamTaskInfo, ok := info.(StreamTaskInfo)
	if !ok {
		return nil, fmt.Errorf("invalid stream task info")
	}
	if err := streamTaskInfo.Validate(); err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(streamTaskInfo.AnomalyDetect.Duration)
	if err != nil {
		return nil, err
	}
	t := &StreamTask{
		info:             streamTaskInfo,
		created:          time.Now(),
		updated:          time.Now(),
		modelUpdateState: RuntimeState{},
		detectEnabled:    concurrency.Bool{},
		thresholdUpper:   concurrency.Float64{},
		thresholdLower:   concurrency.Float64{},
		duration:         d,
		isAnomaly:        false,
		exit:             nil,
		rw:               sync.RWMutex{},
	}
	return t, nil
}

func (s *StreamTask) ProjectId() string {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return s.info.GetProjectId()
}

func (s *StreamTask) TaskId() string {
	s.rw.RLock()
	defer s.rw.RUnlock()
	return s.info.GetTaskId()
}

func (s *StreamTask) IsStream() bool {
	return true
}

func (s *StreamTask) Start() error {
	if s.exit != nil {
		s.exit()
	}
	// 仅当定义了模型更新时才启动模型更新
	if s.info.DetectModel != nil && s.info.ModelUpdate != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.exit = cancel
		go s.do(ctx)
	}
	return nil
}

func (s *StreamTask) Stop() error {
	if s.exit != nil {
		s.exit()
		s.exit = nil
	}
	return nil
}

func (s *StreamTask) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}
	return nil
}

func (s *StreamTask) Update(info interface{}) error {
	newTaskInfo, ok := info.(StreamTaskInfo)
	if !ok {
		return fmt.Errorf("invalid task info")
	}
	if err := newTaskInfo.Validate(); err != nil {
		return err
	}
	d, _ := time.ParseDuration(newTaskInfo.AnomalyDetect.Duration)
	s.rw.Lock()
	defer s.rw.Unlock()
	_taskId, _projectId := s.info.TaskId, s.info.ProjectId
	s.info = newTaskInfo
	s.info.TaskId, s.info.ProjectId = _taskId, _projectId
	s.duration = d
	s.updated = time.Now()
	return s.Restart()
}

func (s *StreamTask) Save() error {
	upper, lower := s.thresholdUpper.Get(), s.thresholdLower.Get()
	return store.Store(s.info, upper, lower, s.modelUpdateState.IsEnabled(), s.detectEnabled.Get())
}

func (s *StreamTask) Status() api.Status {
	st := StreamStatus{
		Info:    s.info,
		Created: s.created,
		Updated: s.updated,
		ModelUpdate: BatchState{
			Enable:    s.modelUpdateState.IsEnabled(),
			Last:      s.modelUpdateState.Last(),
			Next:      s.modelUpdateState.Next(),
			Triggered: int(s.modelUpdateState.Triggered()),
		},
		AnomalyDetect: StreamState{
			Enable:    s.detectEnabled.Get(),
			Triggered: int(s.triggered.Get()),
		},
		ThresholdUpper: s.thresholdUpper.Get(),
		ThresholdLower: s.thresholdLower.Get(),
		CurrentValue:   s.currentValue.Get(),
		IsAnomaly:      s.isAnomaly,
	}
	return st
}

func (s *StreamTask) SimpleStatus() api.Status {
	series := s.info.Target

	st := SimpleStatus{
		ProjectId:      s.info.ProjectId,
		TaskId:         s.info.TaskId,
		IsStream:       true,
		Model:          s.info.DetectModel.Name,
		SensorMac:      series.SensorMac,
		SensorType:     series.SensorType,
		ReceiveNo:      series.ReceiveNo,
		UpdateEnable:   s.modelUpdateState.IsEnabled(),
		DetectEnable:   s.detectEnabled.Get(),
		ThresholdUpper: s.thresholdUpper.Get(),
		ThresholdLower: s.thresholdLower.Get(),
		CurrentValue:   s.currentValue.Get(),
		IsAnomaly:      s.isAnomaly,
	}
	return st
}

func (s *StreamTask) EnableModelUpdate(enable bool) error {
	switch enable {
	case true:
		s.modelUpdateState.Enable()
		s.logInfo("model update enabled")
	case false:
		s.modelUpdateState.Disable()
		s.logInfo("model update disabled")
	}
	return s.Save()
}

func (s *StreamTask) EnableAnomalyDetect(enable bool) error {
	switch enable {
	case true:
		s.detectEnabled.Set(true)
		s.logInfo("anomaly detect enabled")
	case false:
		s.detectEnabled.Set(false)
		s.logInfo("anomaly detect disabled")
	}
	return s.Save()
}

func (s *StreamTask) SetThreshold(lower *float64, upper *float64) error {
	if lower != nil {
		s.thresholdLower.Set(*lower)
		s.logInfo("set threshold lower: %v", *lower)
	}
	if upper != nil {
		s.thresholdUpper.Set(*upper)
		s.logInfo("set threshold upper: %v", *upper)
	}
	return s.Save()
}

func (s *StreamTask) SubKey() string {
	series := s.info.Target
	return fmt.Sprintf("%s#%s#%s#%s", s.info.GetProjectId(), series.SensorMac, series.SensorType, series.ReceiveNo)
}

func (s *StreamTask) do(ctx context.Context) {
	d, _ := time.ParseDuration(s.info.ModelUpdate.Interval)
	timeTicker := time.NewTicker(d)

	s.doModelUpdate()
	s.modelUpdateState.SetNext(time.Now().Add(d))

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeTicker.C:
			s.modelUpdateState.SetNext(time.Now().Add(d))
			if s.modelUpdateState.IsEnabled() {
				s.doModelUpdate()
			}
			_ = s.Save()
		}
	}
}

func (s *StreamTask) doModelUpdate() {
	s.modelUpdateState.SetLast(time.Now())
	s.modelUpdateState.SetTriggered(s.modelUpdateState.Triggered() + 1)

	series := append([]UnvariedSeries{s.info.Target}, s.info.Independent...)
	fluxScript := s.info.ModelUpdate.Query.TransToFlux(s.info.ProjectId, series)

	result, err := s.modelInvoke(fluxScript)
	if err != nil {
		s.logError("model update failed: %s", err.Error())
		return
	}

	r := record.Record{
		SensorMac:      s.info.Target.SensorMac,
		SensorType:     s.info.Target.SensorType,
		ReceiveNo:      s.info.Target.ReceiveNo,
		ThresholdUpper: s.thresholdUpper.Get(),
		ThresholdLower: s.thresholdLower.Get(),
		Value:          s.currentValue.Get(),
		Level:          int(api.InfoLevel),
	}

	if result.Success {
		_ = s.SetThreshold(result.ThresholdLower, result.ThresholdUpper)
		r.Description = "阈值更新成功"
	} else {
		s.logError("model update failed: %s", result.Error)
		r.Description = fmt.Sprintf("阈值更新失败: %s", result.Error)
	}
	if err := record.SaveAlertRecord(s.TaskId(), s.info.ProjectId, r); err != nil {
		s.logError("save record failed: %s", err.Error())
	}
}

func (s *StreamTask) modelInvoke(flux string) (service.InvokeResponse, error) {
	if flux == "" {
		return service.InvokeResponse{}, fmt.Errorf("trans to flux failed")
	}

	queryRes, err := db.InfluxdbClient.QueryMultiple(flux, context.Background())
	if err != nil {
		return service.InvokeResponse{}, err
	}

	data := queryRes

	if s.info.Preprocess != nil && s.info.Preprocess.Name != "" {
		req := service.InvokeRequest{
			Params: s.info.Preprocess.Params,
			Data:   queryRes,
		}
		out, err := service.InvokePost(s.info.Preprocess.Name, service.DataPreprocessMethod, req)
		if err != nil {
			s.logError("data preprocess failed %s", err.Error())
		} else {
			var resp service.PreprocessResponse
			if err := json.Unmarshal(out, &resp); err != nil {
				s.logError("data preprocess failed %s", err.Error())
			} else {
				data = resp.Data
				s.logInfo("data preprocess success")
			}
		}
	}

	if s.info.DetectModel != nil && s.info.DetectModel.Name != "" {
		req := service.InvokeRequest{
			Params: s.info.DetectModel.Params,
			Data:   data,
		}
		out, err := service.InvokePost(s.info.DetectModel.Name, service.ModelUpdateMethod, req)
		if err != nil {
			return service.InvokeResponse{}, err
		}
		var resp service.InvokeResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return service.InvokeResponse{}, err
		} else {
			return resp, nil
		}
	} else {
		return service.InvokeResponse{}, fmt.Errorf("undefined model")
	}
}

func (s *StreamTask) Run(value float64, pt time.Time) {
	// 执行异常检测，复用goroutine
	if !s.detectEnabled.Get() {
		return
	}
	s.triggered.Set(s.triggered.Get() + 1)
	s.currentValue.Set(value)
	upper, lower := s.thresholdUpper.Get(), s.thresholdLower.Get()
	if value < lower || value > upper {
		if !s.isAnomaly && pt.After(s.timer) { // 舍弃乱序的点
			s.isAnomaly = true
			s.timer = pt
		}
		if s.timer.Add(s.duration).Before(time.Now()) {
			// TODO: message pub
			s.logInfo("alert message pub")
		}
		s.logInfo("anomaly detect: upper %v lower %v current %v, is anomaly", upper, lower, value)
	} else {
		s.isAnomaly = false
		s.logInfo("anomaly detect: upper %v lower %v current %v, is normal", upper, lower, value)
	}
}

func (s *StreamTask) logInfo(format string, opts ...interface{}) {
	var msg string
	if len(opts) > 0 {
		msg = fmt.Sprintf(format, opts...)
	} else {
		msg = format
	}
	logrus.Infof("task %s: %s", s.info.TaskId, msg)
}

func (s *StreamTask) logError(format string, opts ...interface{}) {
	var msg string
	if len(opts) > 0 {
		msg = fmt.Sprintf(format, opts...)
	} else {
		msg = format
	}
	logrus.Errorf("task %s: %s", s.info.TaskId, msg)
}
