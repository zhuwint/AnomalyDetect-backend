package task

import (
	"anomaly-detect/cmd/controller/task/api"
	"anomaly-detect/cmd/controller/task/impl"
	"anomaly-detect/cmd/controller/task/store"
	imodels "anomaly-detect/pkg/models"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type Manager struct {
	tasks    map[string]api.Task
	taskList []string // 用于有序遍历 map
	//数据到task的映射, project_id#sensor_mac#sensor_type#receive_no -> project_id#taskId
	pubSub map[string][]string
	rw     sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		tasks:    make(map[string]api.Task, 0),
		taskList: make([]string, 0),
		pubSub:   make(map[string][]string),
		rw:       sync.RWMutex{},
	}
}

func (m *Manager) Create(info api.Info) error {
	m.rw.Lock()
	defer m.rw.Unlock()
	// 判断 task 在 project 中是否存在, 以 taskId#projectId 作为键
	taskKey := buildTaskKey(info.GetTaskId(), info.GetProjectId())
	if _, ok := m.tasks[taskKey]; ok {
		return fmt.Errorf("task %s in project %s already exist", info.GetTaskId(), info.GetProjectId())
	}
	// 根据 task 是否为 stream 创建对应类型的 task
	var task api.Task
	var err error
	switch info.IsStreamTask() {
	case true:
		task, err = impl.NewStreamTask(info)
		if err != nil {
			return fmt.Errorf("create task %s failed %s", info.GetTaskId(), err.Error())
		}
		m.pubSub[task.SubKey()] = append(m.pubSub[task.SubKey()], taskKey)
	case false:
		task, err = impl.NewBatchTask(info)
		if err != nil {
			return fmt.Errorf("create task %s failed %s", info.GetTaskId(), err.Error())
		}
	}
	// 保存
	if err = task.Save(); err != nil {
		return fmt.Errorf("create task %s failed %s", info.GetTaskId(), err.Error())
	}
	m.tasks[taskKey] = task
	m.taskList = append(m.taskList, taskKey)
	// 启动
	_ = m.tasks[taskKey].Start()
	return nil
}

func (m *Manager) Delete(taskId, projectId string) error {
	m.rw.Lock()
	defer m.rw.Unlock()
	taskKey := buildTaskKey(taskId, projectId)
	if _, ok := m.tasks[taskKey]; !ok {
		return fmt.Errorf("task %s in project %s not exist", taskId, projectId)
	}
	_ = m.tasks[taskKey].Stop()
	if m.tasks[taskKey].IsStream() {
		// 删除数据订阅
		ks, ok := m.pubSub[m.tasks[taskKey].SubKey()]
		if ok {
			var newKs []string
			for i := range ks {
				if ks[i] != taskKey {
					newKs = append(newKs, ks[i])
				}
			}
			m.pubSub[m.tasks[taskKey].SubKey()] = newKs
		}
	}
	delete(m.tasks, taskKey)
	i := 0
	for ; i < len(m.taskList); i++ {
		if m.taskList[i] == taskKey {
			break
		}
	}
	if i == 0 {
		m.taskList = m.taskList[1:]
	} else if i < len(m.taskList) {
		m.taskList = append(m.taskList[:i], m.taskList[i+1:]...)
	}

	// m.taskList.Remove(taskKey)
	return store.Del(taskId, projectId)
}

func (m *Manager) Update(taskId, projectId string, info api.Info) error {
	m.rw.Lock()
	defer m.rw.Unlock()
	taskKey := buildTaskKey(taskId, projectId)
	if _, ok := m.tasks[taskKey]; !ok {
		return fmt.Errorf("task %s in project %s not exist", taskId, projectId)
	}

	oldSubKey := m.tasks[taskKey].SubKey()
	if err := m.tasks[taskKey].Update(info); err != nil {
		return err
	}
	if m.tasks[taskKey].IsStream() {
		// 删除数据订阅
		ks, ok := m.pubSub[oldSubKey]
		if ok {
			var newKs []string
			for i := range ks {
				if ks[i] != taskKey { // taskKey 是不变的
					newKs = append(newKs, ks[i])
				}
			}
			m.pubSub[oldSubKey] = newKs
		}
		// 重新创建订阅
		m.pubSub[m.tasks[taskKey].SubKey()] = append(m.pubSub[m.tasks[taskKey].SubKey()], taskKey)
	}
	return nil
}

func (m *Manager) SimpleStatus(projectId string) []api.Status {
	fmt.Println(m.taskList)
	m.rw.RLock()
	defer m.rw.RUnlock()
	matches := make([]api.Status, 0)
	for _, taskKey := range m.taskList {
		ks := parseTaskKey(taskKey)
		if len(ks) != 2 {
			_ = m.tasks[taskKey].Stop()
			delete(m.tasks, taskKey)
			continue
		}
		if projectId == ks[1] {
			matches = append(matches, m.tasks[taskKey].SimpleStatus())
		}
	}
	return matches
}

func (m *Manager) TaskStatus(taskId, projectId string) (api.Status, error) {
	m.rw.RLock()
	defer m.rw.RUnlock()
	taskKey := buildTaskKey(taskId, projectId)
	if st, ok := m.tasks[taskKey]; ok {
		return st.Status(), nil
	}
	return nil, fmt.Errorf("task %s in project %s not exist", taskId, projectId)
}

func (m *Manager) EnableModelUpdate(taskId, projectId string, enable bool) error {
	taskKey := buildTaskKey(taskId, projectId)
	m.rw.Lock()
	defer m.rw.Unlock()
	if _, ok := m.tasks[taskKey]; !ok {
		return fmt.Errorf("task %s in project %s not exist", taskId, projectId)
	}
	return m.tasks[taskKey].EnableModelUpdate(enable)
}

func (m *Manager) EnableAnomalyDetect(taskId, projectId string, enable bool) error {
	taskKey := buildTaskKey(taskId, projectId)
	m.rw.Lock()
	defer m.rw.Unlock()
	if _, ok := m.tasks[taskKey]; !ok {
		return fmt.Errorf("task %s in project %s not exist", taskId, projectId)
	}
	return m.tasks[taskKey].EnableAnomalyDetect(enable)
}

func (m *Manager) SetThreshold(taskId, projectId string, lower, upper *float64) error {
	taskKey := buildTaskKey(taskId, projectId)
	m.rw.Lock()
	defer m.rw.Unlock()
	if _, ok := m.tasks[taskKey]; !ok {
		return fmt.Errorf("task %s in project %s not exist", taskId, projectId)
	}
	return m.tasks[taskKey].SetThreshold(lower, upper)
}

func buildTaskKey(taskId, projectId string) string {
	return fmt.Sprintf("%s#%s", taskId, projectId)
}

func parseTaskKey(key string) []string {
	return strings.Split(key, "#")
}

func (m *Manager) WritePoints(points imodels.Points) {
	logrus.Infof("receive %d points", len(points))
	for _, p := range points {
		measurement := string(p.Name())
		if measurement != impl.DefaultMeasurement {
			// 丢弃不属于本 measurement 的点
			continue
		}
		projectId := string(p.Tags().Get([]byte(impl.ProjectIdTag)))
		sensorMac := string(p.Tags().Get([]byte(impl.SensorMacTag)))
		sensorType := string(p.Tags().Get([]byte(impl.SensorTypeTag)))
		receiveNo := string(p.Tags().Get([]byte(impl.ReceiveNoTag)))
		if projectId == "" || sensorMac == "" || sensorType == "" || receiveNo == "" {
			continue
		}

		field, err := p.Fields()
		if err != nil {
			logrus.Error("parse point failed: %s", err.Error())
			continue
		}

		value := field[impl.DefaultFieldName].(float64)
		key := strings.Join([]string{projectId, sensorMac, sensorType, receiveNo}, "#")

		for _, task := range m.pubSub[key] {
			if _, ok := m.tasks[task]; ok {
				m.tasks[task].Run(value, p.Time())
			}
		}
	}
}
