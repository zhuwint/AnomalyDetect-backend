package record

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"time"
)

const queryString = "task_id=? and project_id=?"

type Record struct {
	SensorMac      string  `json:"sensor_mac"`
	SensorType     string  `json:"sensor_type"`
	ReceiveNo      string  `json:"receive_no"`
	ThresholdUpper float64 `json:"threshold_upper"`
	ThresholdLower float64 `json:"threshold_lower"`
	Value          float64 `json:"value"`
	Level          int     `json:"level"`
	Description    string  `json:"description"`
}

func SaveAlertRecord(taskId string, projectId int, data Record) error {
	r := model.AlertRecord{
		TaskId:         taskId,
		ProjectId:      projectId,
		SensorMac:      data.SensorMac,
		SensorType:     data.SensorType,
		ReceiveNo:      data.ReceiveNo,
		ThresholdUpper: data.ThresholdUpper,
		ThresholdLower: data.ThresholdLower,
		Value:          data.Value,
		Level:          data.Level,
		Created:        time.Now(),
		View:           false,
		Description:    data.Description,
	}
	if err := db.MysqlClient.DB.Model(&model.AlertRecord{}).Create(&r).Error; err != nil {
		return err
	}
	return nil
}
