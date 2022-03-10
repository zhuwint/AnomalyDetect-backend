package task_test

import (
	"anomaly-detect/cmd/controller/model"
	"anomaly-detect/pkg/mysql"
	"fmt"
	"net/http"
	"testing"
)

func TestStartTask(t *testing.T) {
	connector, err := mysql.NewConnector("10.203.96.205:3306", "my_database", "rwuser", "nmlab301")
	if err != nil {
		t.Fatalf("failed connect to mysql: %s", err.Error())
		return
	}
	var tasks []model.UnionTask
	if err := connector.DB.Model(&model.UnionTask{}).Where("project_id=3").Find(&tasks).Error; err != nil {
		t.Fatalf("falied to get tasks: %s", err.Error())
		return
	}
	for _, task := range tasks {
		params := fmt.Sprintf("taskId=%s&projectId=3&isUpdate=false&enable=true", task.TaskId)
		resp, err := http.Post("http://10.203.96.205:3030/api/task/control?"+params, "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			t.Fatalf("failed with code %d", resp.StatusCode)
		}
	}
}

// func TestDeleteTask(t *testing.T) {
// 	connector, err := mysql.NewConnector("10.203.96.205:3306", "my_database", "rwuser", "nmlab301")
// 	if err != nil {
// 		t.Fatalf("failed connect to mysql: %s", err.Error())
// 		return
// 	}
// 	var tasks []model.UnionTask
// 	if err := connector.DB.Model(&model.UnionTask{}).Where("project_id=3").Find(&tasks).Error; err != nil {
// 		t.Fatalf("falied to get tasks: %s", err.Error())
// 		return
// 	}
// 	for _, task := range tasks {
// 		params := fmt.Sprintf("taskId=%s&projectId=3", task.TaskId)
// 		url := "http://10.203.96.205:3030/api/task?" + params
// 		req, _ := http.NewRequest(http.MethodDelete, url, nil)
// 		resp, err := http.DefaultClient.Do(req)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		if resp.StatusCode < 200 || resp.StatusCode > 299 {
// 			t.Fatalf("failed with code %d", resp.StatusCode)
// 		}
// 	}
// }
