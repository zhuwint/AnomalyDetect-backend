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
	"strconv"
	"sync"
	"time"
)

type BatchTask struct {
	info BatchTaskInfo

	created time.Time // 创建时间
	updated time.Time // 更新时间

	modelUpdateState   RuntimeState // 模型更新运行时状态
	anomalyDetectState RuntimeState // 异常检测运行时状态

	thresholdUpper concurrency.Float64 // 阈值上限
	thresholdLower concurrency.Float64 // 阈值下限
	currentValue   concurrency.Float64 // 当前特征值

	isAnomaly bool

	exit context.CancelFunc
	rw   sync.RWMutex
}

func NewBatchTask(info api.Info) (*BatchTask, error) {
	batchTaskInfo, ok := info.(BatchTaskInfo)
	if !ok {
		return nil, fmt.Errorf("invalid batch task info")
	}
	if err := batchTaskInfo.Validate(); err != nil {
		return nil, err
	}
	t := &BatchTask{
		info:               batchTaskInfo,
		created:            time.Now(),
		updated:            time.Now(),
		modelUpdateState:   RuntimeState{},
		anomalyDetectState: RuntimeState{},
		thresholdUpper:     concurrency.Float64{},
		thresholdLower:     concurrency.Float64{},
		currentValue:       concurrency.Float64{},
		rw:                 sync.RWMutex{},
		isAnomaly:          false,
	}
	return t, nil
}

func (t *BatchTask) ProjectId() string {
	t.rw.RLock()
	defer t.rw.RUnlock()
	return strconv.Itoa(t.info.ProjectId)
}

func (t *BatchTask) TaskId() string {
	t.rw.RLock()
	defer t.rw.RUnlock()
	return t.info.TaskId
}

func (t *BatchTask) IsStream() bool {
	return false
}

func (t *BatchTask) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	t.exit = cancel
	go t.do(ctx)
	return nil
}

func (t *BatchTask) Stop() error {
	if t.exit != nil {
		t.exit()
	}
	t.exit = nil
	return t.Save() // 退出时保存
}

func (t *BatchTask) Restart() error {
	if err := t.Stop(); err != nil {
		return err
	}
	if err := t.Start(); err != nil {
		return err
	}
	return nil
}

func (t *BatchTask) Save() error {
	return store.Store(t.info, t.thresholdUpper.Get(), t.thresholdLower.Get(), t.modelUpdateState.IsEnabled(), t.anomalyDetectState.IsEnabled())
}

func (t *BatchTask) Update(info interface{}) error {
	newTaskInfo, ok := info.(BatchTaskInfo)
	if !ok {
		return fmt.Errorf("invalid task info")
	}
	if err := newTaskInfo.Validate(); err != nil {
		return err
	}
	t.rw.Lock()
	defer t.rw.Unlock()
	// 更新时无法更改 taskId 与 projectId
	_taskId, _projectId := t.info.TaskId, t.info.ProjectId
	t.info = newTaskInfo
	t.info.TaskId, t.info.ProjectId = _taskId, _projectId
	t.updated = time.Now()
	return t.Restart()
}

func (t *BatchTask) Status() api.Status {
	// 这里不需要加锁
	st := BatchStatus{
		Info:    t.info,
		Created: t.created,
		Updated: t.updated,
		ModelUpdate: BatchState{
			Enable:    t.modelUpdateState.IsEnabled(),
			Last:      t.modelUpdateState.Last(),
			Next:      t.modelUpdateState.Next(),
			Triggered: int(t.modelUpdateState.Triggered()),
		},
		AnomalyDetect: BatchState{
			Enable:    t.anomalyDetectState.IsEnabled(),
			Last:      t.anomalyDetectState.Last(),
			Next:      t.anomalyDetectState.Next(),
			Triggered: int(t.anomalyDetectState.Triggered()),
		},
		ThresholdUpper: t.thresholdUpper.Get(),
		ThresholdLower: t.thresholdLower.Get(),
		CurrentValue:   t.currentValue.Get(),
		IsAnomaly:      t.isAnomaly,
	}
	return st
}

func (t *BatchTask) SimpleStatus() api.Status {
	// 参数验证时已经保证了series存在
	series := t.info.Target

	sst := SimpleStatus{
		ProjectId:      t.info.ProjectId,
		TaskId:         t.info.TaskId,
		IsStream:       false,
		Model:          t.info.DetectModel.Name,
		SensorMac:      series.SensorMac,
		SensorType:     series.SensorType,
		ReceiveNo:      series.ReceiveNo,
		UpdateEnable:   t.modelUpdateState.IsEnabled(),
		DetectEnable:   t.anomalyDetectState.IsEnabled(),
		ThresholdUpper: t.thresholdUpper.Get(),
		ThresholdLower: t.thresholdLower.Get(),
		CurrentValue:   t.currentValue.Get(),
		IsAnomaly:      t.isAnomaly,
	}
	return sst
}

func (t *BatchTask) EnableModelUpdate(enable bool) error {
	switch enable {
	case true:
		t.modelUpdateState.Enable()
		t.logInfo("model update enabled")
	case false:
		t.modelUpdateState.Disable()
		t.logInfo("model update disabled")
	}
	return t.Save()
}

func (t *BatchTask) EnableAnomalyDetect(enable bool) error {
	switch enable {
	case true:
		t.anomalyDetectState.Enable()
		t.logInfo("anomaly detect enabled")
	case false:
		t.anomalyDetectState.Disable()
		t.logInfo("anomaly detect disabled")
	}
	return t.Save()
}

func (t *BatchTask) SetThreshold(lower *float64, upper *float64) error {
	if lower != nil {
		t.thresholdLower.Set(*lower)
		t.logInfo("set threshold lower: %v", *lower)
	}
	if upper != nil {
		t.thresholdUpper.Set(*upper)
		t.logInfo("set threshold upper: %v", *upper)
	}
	return t.Save()
}

func (t *BatchTask) do(ctx context.Context) {
	// because duration has been validated before, ignore error in the below duration parse
	updateDuration, _ := time.ParseDuration(t.info.ModelUpdate.Interval)
	detectDuration, _ := time.ParseDuration(t.info.AnomalyDetect.Interval)
	// create time ticker
	updateTicker := time.NewTicker(updateDuration)
	detectTicker := time.NewTicker(detectDuration)

	// 延迟执行
	go time.AfterFunc(5*time.Second, func() {
		if t.modelUpdateState.IsEnabled() {
			t.doModelUpdate()
			t.modelUpdateState.SetNext(time.Now().Add(updateDuration))
		}
		if t.anomalyDetectState.IsEnabled() {
			t.doAnomalyDetect()
			t.anomalyDetectState.SetNext(time.Now().Add(detectDuration))
		}
	})

	for {
		select {
		case <-updateTicker.C:
			t.modelUpdateState.SetNext(time.Now().Add(updateDuration))
			if t.modelUpdateState.IsEnabled() {
				t.doModelUpdate()
			}
			_ = t.Save()
		case <-detectTicker.C:
			t.anomalyDetectState.SetNext(time.Now().Add(detectDuration))
			if t.anomalyDetectState.IsEnabled() {
				t.doAnomalyDetect()
			}
			_ = t.Save()
		case <-ctx.Done():
			return
		}
	}
}

func (t *BatchTask) doAnomalyDetect() {
	t.anomalyDetectState.SetLast(time.Now())
	t.anomalyDetectState.SetTriggered(t.anomalyDetectState.Triggered() + 1)
	startAt := time.Now()

	series := append([]UnvariedSeries{t.info.Target}, t.info.Independent...)
	fluxScript := t.info.AnomalyDetect.Query.TransToFlux(t.info.ProjectId, series)

	stop := time.Now()
	offset, _ := time.ParseDuration(t.info.AnomalyDetect.Query.Range.Start)
	start := stop.Add(offset)

	result, err := t.modelInvoke(fluxScript, service.AnomalyDetectMethod)
	if err != nil {
		t.logError("anomaly detect failed: %s", err.Error())
		return
	}
	if !result.Success {
		t.logError("anomaly detect failed: %s", result.Error)
		return
	}
	t.currentValue.Set(*result.EigenValue)
	t.logInfo("anomaly detect success cost: %v", time.Now().Sub(startAt).String())
	// 判断是否异常
	r := record.Record{
		SensorMac:      t.info.Target.SensorMac,
		SensorType:     t.info.Target.SensorType,
		ReceiveNo:      t.info.Target.ReceiveNo,
		ThresholdUpper: t.thresholdUpper.Get(),
		ThresholdLower: t.thresholdLower.Get(),
		Value:          t.currentValue.Get(),
		Time:           time.Now(),
		Start:          start,
		Stop:           stop,
	}
	if result.IsAnomaly || t.currentValue.Get() > t.thresholdUpper.Get() || t.currentValue.Get() < t.thresholdLower.Get() {
		r.Level = t.info.Level
		r.Description = "检测异常"
		t.isAnomaly = true
	} else {
		r.Level = int(api.InfoLevel)
		t.isAnomaly = false
	}
	if err := record.SaveAlertRecord(t.TaskId(), t.info.ProjectId, r); err != nil {
		t.logError("save record failed: %s", err.Error())
	}
}

func (t *BatchTask) doModelUpdate() {
	t.modelUpdateState.SetLast(time.Now())
	t.modelUpdateState.SetTriggered(t.modelUpdateState.Triggered() + 1)

	series := append([]UnvariedSeries{t.info.Target}, t.info.Independent...)
	fluxScript := t.info.ModelUpdate.Query.TransToFlux(t.info.ProjectId, series)
	result, err := t.modelInvoke(fluxScript, service.ModelUpdateMethod)
	if err != nil {
		t.logError("model update failed: %s", err.Error())
		return
	}

	r := record.Record{
		SensorMac:  t.info.Target.SensorMac,
		SensorType: t.info.Target.SensorType,
		ReceiveNo:  t.info.Target.ReceiveNo,
		Value:      t.currentValue.Get(),
		Level:      int(api.InfoLevel),
	}

	level := record.InfoLevel

	if result.Success {
		_ = t.SetThreshold(result.ThresholdLower, result.ThresholdUpper)
		r.ThresholdLower = t.thresholdLower.Get()
		r.ThresholdUpper = t.thresholdUpper.Get()
		r.Description = "阈值更新成功"
	} else {
		r.Description = fmt.Sprintf("阈值更新失败: %s", result.Error)
		level = record.ErrorLevel
	}
	if err := record.SaveSystemRecord(t.TaskId(), t.info.ProjectId, r, level); err != nil {
		t.logError("save record failed: %s", err.Error())
	}
}

func (t *BatchTask) modelInvoke(flux string, method string) (service.InvokeResponse, error) {
	if flux == "" {
		return service.InvokeResponse{}, fmt.Errorf("trans to flux failed")
	}

	queryRes, err := db.InfluxdbClient.QueryMultiple(flux, context.Background())
	if err != nil {
		return service.InvokeResponse{}, err
	}

	data := queryRes

	if t.info.Preprocess != nil && t.info.Preprocess.Name != "" {
		req := service.InvokeRequest{
			Params: t.info.Preprocess.Params,
			Data:   queryRes,
		}
		out, err := service.InvokePost(t.info.Preprocess.Name, service.DataPreprocessMethod, req)
		if err != nil {
			t.logError("data preprocess failed %s", err.Error())
		} else {
			var resp service.PreprocessResponse
			if err := json.Unmarshal(out, &resp); err != nil {
				t.logError("data preprocess failed %s", err.Error())
			} else {
				data = resp.Data
				t.logInfo("data preprocess success")
			}
		}
	}

	if t.info.DetectModel != nil && t.info.DetectModel.Name != "" {
		req := service.InvokeRequest{
			Params: t.info.DetectModel.Params,
			Data:   data,
		}
		out, err := service.InvokePost(t.info.DetectModel.Name, method, req)
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

func (t *BatchTask) SubKey() string {
	series := t.info.Target
	return fmt.Sprintf("%s#%s#%s#%s", t.info.GetProjectId(), series.SensorMac, series.SensorType, series.ReceiveNo)
}

func (t *BatchTask) Run(value float64, pt time.Time) {
	// batch task 没有此项
}

func (t *BatchTask) logInfo(format string, opts ...interface{}) {
	var msg string
	if len(opts) > 0 {
		msg = fmt.Sprintf(format, opts...)
	} else {
		msg = format
	}
	logrus.Infof("task %s: %s", t.info.TaskId, msg)
}

func (t *BatchTask) logError(format string, opts ...interface{}) {
	var msg string
	if len(opts) > 0 {
		msg = fmt.Sprintf(format, opts...)
	} else {
		msg = format
	}
	logrus.Errorf("task %s: %s", t.info.TaskId, msg)
}
