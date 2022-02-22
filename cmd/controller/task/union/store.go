package union

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"errors"
	"gorm.io/gorm"
)

const queryWithId = "task_id=? and project_id=?"

func Store(info TaskInfo, enable bool) error {
	var old model.UnionTask
	if err := db.MysqlClient.DB.Where(queryWithId, info.GetTaskId(), info.GetProjectId()).First(&old).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// create
		record := model.UnionTask{
			TaskId:    info.GetTaskId(),
			ProjectId: info.GetProjectId(),
			Content:   string(info.MarshToJson()),
			Enable:    enable,
		}
		if err := db.MysqlClient.DB.Create(&record).Error; err != nil {
			return err
		}
	} else { // update
		old.Content = string(info.MarshToJson())
		old.Enable = enable
		if err := db.MysqlClient.DB.Save(&old).Error; err != nil {
			return err
		}
	}
	return nil
}

func GetAll() ([]model.UnionTask, error) {
	records := make([]model.UnionTask, 0)
	err := db.MysqlClient.DB.Find(&records).Error
	return records, err
}

func Del(taskId, projectId string) error {
	return db.MysqlClient.DB.Where(queryWithId, taskId, projectId).Delete(model.UnionTask{}).Error
}
