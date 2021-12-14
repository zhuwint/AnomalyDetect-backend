package store

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"anomaly-detect/cmd/controller/task/api"
	"errors"
	"gorm.io/gorm"
)

const queryWithId = "task_id=? and project_id=?"

func Store(task api.Info, upper, lower float64, update, detect bool) error {
	var old model.Task
	if err := db.MysqlClient.DB.Where(queryWithId, task.GetTaskId(), task.GetProjectId()).First(&old).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// create
		record := model.Task{
			TaskId:         task.GetTaskId(),
			ProjectId:      task.GetProjectId(),
			IsStream:       task.IsStreamTask(),
			Content:        string(task.MarshToJson()),
			UpdateEnable:   update,
			DetectEnable:   detect,
			ThresholdUpper: upper,
			ThresholdLower: lower,
		}
		if err := db.MysqlClient.DB.Create(&record).Error; err != nil {
			return err
		}
	} else {
		// update
		old.Content = string(task.MarshToJson())
		old.ThresholdUpper = upper
		old.ThresholdLower = lower
		old.UpdateEnable = update
		old.DetectEnable = detect
		if err := db.MysqlClient.DB.Save(&old).Error; err != nil {
			return err
		}
	}
	return nil
}

func Get(taskId, projectId string) (model.Task, error) {
	var record model.Task
	err := db.MysqlClient.DB.Where(queryWithId, taskId, projectId).First(&record).Error
	return record, err
}

func GetAll() ([]model.Task, error) {
	var records []model.Task
	err := db.MysqlClient.DB.Find(&records).Error
	return records, err
}

func Del(taskId, projectId string) error {
	return db.MysqlClient.DB.Where(queryWithId, taskId, projectId).Delete(model.Task{}).Error
}
