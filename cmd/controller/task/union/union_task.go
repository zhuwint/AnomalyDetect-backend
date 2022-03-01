package union

import (
	"anomaly-detect/cmd/controller/task/api"
	"anomaly-detect/cmd/controller/task/record"
	"anomaly-detect/pkg/validator"
	"fmt"
	"time"
)

type Task struct {
	info      TaskInfo
	state     map[string]RunTimeState
	enabled   bool
	isAnomaly bool          // 当前告警状态
	duration  time.Duration // 持续多少时间告警
	timer     time.Time
	created   time.Time
	updated   time.Time
}

func NewUnionTask(info api.Info) (*Task, error) {
	taskInfo, ok := info.(TaskInfo)
	if !ok {
		return nil, fmt.Errorf("invalid task info")
	}
	if err := taskInfo.Validate(); err != nil {
		return nil, err
	}
	d, _ := validator.CheckDurationPositive(taskInfo.Duration)
	t := &Task{
		info:      taskInfo,
		state:     make(map[string]RunTimeState, len(taskInfo.Series)),
		enabled:   false,
		isAnomaly: false,
		duration:  d,
		timer:     time.Now(),
		created:   time.Now(),
		updated:   time.Now(),
	}
	return t, nil
}

func (t Task) ProjectId() string {
	return t.info.GetProjectId()
}

func (t Task) TaskId() string {
	return t.info.TaskId
}

func (t Task) IsStream() bool {
	return true
}

func (t Task) IsUnion() bool {
	return true
}

func (t *Task) Start() error {
	return nil
}

func (t *Task) Stop() error {
	return nil
}

func (t *Task) Restart() error {
	return nil
}

func (t *Task) Update(i interface{}) error {
	newTaskInfo, ok := i.(TaskInfo)
	if !ok {
		return fmt.Errorf("invalid task info")
	}
	if err := newTaskInfo.Validate(); err != nil {
		return err
	}

	d, _ := validator.CheckDurationPositive(newTaskInfo.Duration)
	t.duration = d

	_taskId, _projectId := t.info.TaskId, t.info.ProjectId
	for _, s := range t.info.Series { // 删除所有当前测点状态
		delete(t.state, s.Key())
	}
	t.info = newTaskInfo
	t.info.TaskId, t.info.ProjectId = _taskId, _projectId
	t.updated = time.Now()
	return t.Save()
}

func (t *Task) SubKey() []string {
	series := make([]string, len(t.info.Series))
	for i, s := range t.info.Series {
		series[i] = fmt.Sprintf("%s#%s#%s#%s", t.info.GetProjectId(), s.SensorMac, s.SensorType, s.ReceiveNo)
	}
	return series
}

func (t *Task) Run(projectId, sensorMac, sensorType, receiveNo string, value float64, pt time.Time) {
	if !t.enabled {
		return
	}

	// 判断键是否存在
	flag := false
	for _, s := range t.info.Series {
		if s.SensorMac == sensorMac && s.SensorType == sensorType && s.ReceiveNo == receiveNo {
			flag = true
			break
		}
	}
	if !flag {
		return
	}

	// 更新值
	key := fmt.Sprintf("%s#%s#%s", sensorMac, sensorType, receiveNo)
	if st, ok := t.state[key]; !ok {
		t.state[key] = RunTimeState{
			Last:      pt,
			Triggered: 1,
			Value:     value,
		}
	} else if pt.After(st.Last) {
		t.state[key] = RunTimeState{
			Last:      pt,
			Triggered: st.Triggered + 1,
			Value:     value,
		}
	} else {
		return // 舍弃乱序的点
	}
	// 告警判断
	isAnomaly := true
	for i, s := range t.info.Series {
		if st, ok := t.state[s.Key()]; ok {
			if pt.Sub(st.Last) > 30*time.Minute || pt.Sub(st.Last) < -30*time.Minute {
				isAnomaly = t.isAnomaly // 若两点间隔大于30分钟, 则维持原有状态不变
				break
			}
			if i == 0 {
				isAnomaly = !(st.Value <= s.ThresholdUpper && st.Value >= s.ThresholdLower)
			} else {
				// TODO:
				if t.info.Operate[i-1] == 0 { // 且
					isAnomaly = isAnomaly || !(st.Value <= s.ThresholdUpper && st.Value >= s.ThresholdLower)
				} else { // 或
					isAnomaly = isAnomaly && !(st.Value <= s.ThresholdUpper && st.Value >= s.ThresholdLower)
				}
			}
		} else {
			isAnomaly = false // 数据不全不告警
		}
	}
	// TODO:考虑持续时间
	if isAnomaly { // 如果当前是异常
		if !t.isAnomaly { // 如果之前是正常的,改为异常
			t.timer = pt
			t.isAnomaly = isAnomaly
			t.publish(t.info.Level, pt)
		}
	} else {
		if t.isAnomaly {
			t.publish(int(api.InfoLevel), pt)
		}
		t.isAnomaly = isAnomaly
	}
}

func (t *Task) publish(level int, pt time.Time) {
	for _, s := range t.info.Series {
		key := fmt.Sprintf("%s#%s#%s", s.SensorMac, s.SensorType, s.ReceiveNo)
		r := record.Record{
			SensorMac:      s.SensorMac,
			SensorType:     s.SensorType,
			ReceiveNo:      s.ReceiveNo,
			ThresholdUpper: s.ThresholdUpper,
			ThresholdLower: s.ThresholdLower,
			Value:          t.state[key].Value,
			Time:           pt,
			Start:          pt,
			Stop:           pt,
			Level:          level,
		}
		_ = record.SaveAlertRecord(t.info.TaskId, t.info.ProjectId, r)
	}
}

func (t Task) Save() error {
	return Store(t.info, t.enabled)
}

func (t Task) Status() api.Status {
	st := Status{
		Info:      t.info,
		Created:   t.created,
		Updated:   t.updated,
		State:     t.state,
		Enable:    t.enabled,
		IsAnomaly: t.isAnomaly,
	}
	return st
}

func (t Task) SimpleStatus() api.Status {
	st := SimpleStatus{
		ProjectId: t.info.ProjectId,
		TaskId:    t.info.TaskId,
		TaskName:  t.info.TaskName,
		Enable:    t.enabled,
		IsAnomaly: t.isAnomaly,
	}
	return st
}

func (t *Task) EnableModelUpdate(b bool) error {
	return nil // 没有此项
}

func (t *Task) EnableAnomalyDetect(b bool) error {
	t.enabled = b
	return t.Save()
}

func (t *Task) SetThreshold(sensorMac, sensorType, receiveNo string, lower *float64, upper *float64) error {
	for i, s := range t.info.Series {
		if s.SensorMac == sensorMac && s.SensorType == sensorType && s.ReceiveNo == receiveNo {
			if upper != nil {
				t.info.Series[i].ThresholdUpper = *upper
			}
			if lower != nil {
				t.info.Series[i].ThresholdLower = *lower
			}
		}
	}
	return t.Save()
}
