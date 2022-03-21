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
	state     map[string]*State
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
		state:     make(map[string]*State, len(taskInfo.Series)),
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
	if _, ok := t.state[key]; !ok { // 如果之前没有状态，则创建
		t.state[key] = NewState()
	}

	if pt.Before(t.state[key].Last.T) {
		return // 如果时间在已经处理过的点的时间之前，则舍弃，保证重复发的点不会被重复处理
	}

	// if t.state[key].Len() > 0 && t.state[key].Top().T.Sub(pt) > 15*time.Minute { // 乱序点，过时15分钟，舍弃
	// 	return
	// }

	// 插入值, 并保证缓存在一定数量范围内, 该范围暂时设为10
	t.state[key].Push(PV{
		T: pt,
		V: value,
	})

	for t.state[key].Len() > 10 {
		t.state[key].Pop()
	}

	// 数据不全不告警
	for _, s := range t.info.Series {
		if _, ok := t.state[s.Key()]; !ok {
			return
		}
	}

	// 此刻到达的点作为基准点
	base_point := t.state[key].Top()

	// 触发告警前进行时间规整
	for _, s := range t.info.Series {
		st := t.state[s.Key()]
		for t.state[s.Key()].Len() > 0 && st.Top().T.Sub(base_point.T) < -1*time.Minute { // 其它数据过时
			_ = st.Pop()
		}
		if st.Len() == 0 {
			return // 数据不全不告警
		} else {
			for st.Top().T.Sub(base_point.T) > 1*time.Minute && t.state[key].Len() > 0 { // 当前数据过时
				base_point = t.state[key].Pop()
			}
		}
	}

	// 告警判断
	isAnomaly := true
	for i, s := range t.info.Series {
		st := t.state[s.Key()]
		if st == nil || st.Len() == 0 {
			return
		}
		flag := !(st.Top().V <= s.ThresholdUpper && st.Top().V >= s.ThresholdLower)
		if i == 0 {
			isAnomaly = flag
		} else {
			if t.info.Operate[i-1] == 0 { // 且
				isAnomaly = isAnomaly || flag
			} else { // 或
				isAnomaly = isAnomaly && flag
			}
		}
	}

	// 经过一轮比较之后把位于队列头部的点全部删除
	for _, s := range t.info.Series {
		t.state[s.Key()].SetLast(t.state[s.Key()].Pop())
	}

	// 推送判断 TODO:考虑持续时间
	if isAnomaly { // 如果当前为异常
		if !t.isAnomaly && t.timer != pt { // 如果之前是正常的,改为异常
			t.timer = pt
			t.isAnomaly = isAnomaly
			t.publish(t.info.Level, pt)
		}
	} else {
		if t.isAnomaly && t.timer != pt {
			t.timer = pt
			t.publish(int(api.InfoLevel), pt)
		}
		t.isAnomaly = isAnomaly
	}
}

func (t *Task) publish(level int, pt time.Time) {
	// fmt.Println("save alert record", level, pt.Local().String())
	for _, s := range t.info.Series {
		key := fmt.Sprintf("%s#%s#%s", s.SensorMac, s.SensorType, s.ReceiveNo)
		r := record.Record{
			SensorMac:      s.SensorMac,
			SensorType:     s.SensorType,
			ReceiveNo:      s.ReceiveNo,
			ThresholdUpper: s.ThresholdUpper,
			ThresholdLower: s.ThresholdLower,
			Value:          t.state[key].Last.V,
			Time:           pt,
			Start:          pt,
			Stop:           pt,
			Level:          level,
		}
		err := record.SaveAlertRecord(t.info.TaskId, t.info.ProjectId, r)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (t Task) Save() error {
	return Store(t.info, t.enabled)
}

func (t Task) Status() api.Status {
	state := make(map[string]RunTimeState)
	for k := range t.state {
		state[k] = RunTimeState{
			Last:      t.state[k].Last.T,
			Triggered: t.state[k].Triggered,
			Value:     t.state[k].Last.V,
		}
	}
	st := Status{
		Info:      t.info,
		Created:   t.created,
		Updated:   t.updated,
		State:     state,
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
