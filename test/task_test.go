package task_test

import (
	"anomaly-detect/cmd/controller/model"
	"anomaly-detect/pkg/mysql"
	"bytes"
	"encoding/json"
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
	var tasks []model.Task
	if err := connector.DB.Model(&model.Task{}).Where("project_id=3").Find(&tasks).Error; err != nil {
		t.Fatalf("falied to get tasks: %s", err.Error())
		return
	}

	type temp struct {
		Lower float64 `json:"lower"`
		Upper float64 `json:"upper"`
	}

	for _, task := range tasks {
		data := temp{
			Lower: 10.0,
			Upper: 29.5,
		}
		content, _ := json.Marshal(data)
		resp, err := http.Post(fmt.Sprintf("http://10.203.96.205:3030/api/task/set?taskId=%s&projectId=3", task.TaskId), "application/json", bytes.NewBuffer(content))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			t.Fatalf("failed with code %d", resp.StatusCode)
		}
		params := fmt.Sprintf("taskId=%s&projectId=3&isUpdate=false&enable=true", task.TaskId)
		resp, err = http.Post("http://10.203.96.205:3030/api/task/control?"+params, "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			t.Fatalf("failed with code %d", resp.StatusCode)
		}
	}
}

func TestDeleteTask(t *testing.T) {
	connector, err := mysql.NewConnector("10.203.96.205:3306", "my_database", "rwuser", "nmlab301")
	if err != nil {
		t.Fatalf("failed connect to mysql: %s", err.Error())
		return
	}
	var tasks []model.Task
	if err := connector.DB.Model(&model.Task{}).Where("project_id=3").Find(&tasks).Error; err != nil {
		t.Fatalf("falied to get tasks: %s", err.Error())
		return
	}
	for _, task := range tasks {
		params := fmt.Sprintf("taskId=%s&projectId=3", task.TaskId)
		url := "http://10.203.96.205:3030/api/task?" + params
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			t.Fatalf("failed with code %d", resp.StatusCode)
		}
	}
}

func TestStartUnionTask(t *testing.T) {
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

func TestDeleteUnionTask(t *testing.T) {
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
		params := fmt.Sprintf("taskId=%s&projectId=3", task.TaskId)
		url := "http://10.203.96.205:3030/api/task?" + params
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			t.Fatalf("failed with code %d", resp.StatusCode)
		}
	}
}
