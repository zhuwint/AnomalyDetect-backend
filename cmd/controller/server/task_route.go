package server

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/task/api"
	"anomaly-detect/cmd/controller/task/impl"
	"anomaly-detect/cmd/controller/task/service"
	"anomaly-detect/pkg/models"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func (c *Controller) createBatchTask(ctx *gin.Context) {
	var taskInfo impl.BatchTaskInfo
	if err := ctx.BindJSON(&taskInfo); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	taskInfo.TaskId = impl.GenerateTaskId()
	if err := c.taskManager.Create(taskInfo); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) createStreamTask(ctx *gin.Context) {
	var taskInfo impl.StreamTaskInfo
	if err := ctx.BindJSON(&taskInfo); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	taskInfo.TaskId = impl.GenerateTaskId()
	if err := c.taskManager.Create(taskInfo); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) getTask(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	if projectId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "must provide project_id"})
		return
	}
	if taskId == "" {
		data := c.taskManager.SimpleStatus(projectId)
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: data})
	} else {
		data, err := c.taskManager.TaskStatus(taskId, projectId)
		if err != nil {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
		} else {
			ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: data})
		}
	}
}

func (c *Controller) getTaskByTargetSeries(ctx *gin.Context) {
	projectId := ctx.Query("projectId")
	sensorMac := ctx.Query("sensorMac")
	sensorType := ctx.Query("sensorType")
	receiveNo := ctx.Query("receiveNo")
	if projectId == "" || sensorMac == "" || sensorType == "" || receiveNo == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "must provide project_id, sensorMac, sensorType and receiveNo"})
		return
	}
	data := c.taskManager.SimpleStatus(projectId)
	resp := make([]api.Status, 0)
	for i := range data {
		d, ok := data[i].(impl.SimpleStatus)
		if ok {
			if d.SensorMac == sensorMac && d.SensorType == sensorType && d.ReceiveNo == receiveNo {
				resp = append(resp, d)
			}
		}
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: resp})
}

func (c *Controller) deleteTask(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	if taskId == "" || projectId == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide taskId and projectId"})
		return
	}
	if err := c.taskManager.Delete(taskId, projectId); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) updateBatchTask(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	if taskId == "" || projectId == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide taskId and projectId"})
		return
	}
	var taskInfo impl.BatchTaskInfo
	if err := ctx.BindJSON(&taskInfo); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	if err := c.taskManager.Update(taskId, projectId, taskInfo); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) updateStreamTask(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	if taskId == "" || projectId == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide taskId and projectId"})
		return
	}
	var taskInfo impl.StreamTaskInfo
	if err := ctx.BindJSON(&taskInfo); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	if err := c.taskManager.Update(taskId, projectId, taskInfo); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) taskControl(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	updateOrDetect, err1 := strconv.ParseBool(ctx.Query("isUpdate"))
	enable, err2 := strconv.ParseBool(ctx.Query("enable"))
	if taskId == "" || projectId == "" || err1 != nil || err2 != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "invalid params"})
		return
	}
	var _err error
	if updateOrDetect {
		// is model update
		_err = c.taskManager.EnableModelUpdate(taskId, projectId, enable)
	} else {
		_err = c.taskManager.EnableAnomalyDetect(taskId, projectId, enable)
	}
	if _err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: _err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

func (c *Controller) setThreshold(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	if taskId == "" || projectId == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide taskId and projectId"})
		return
	}
	type requestType struct {
		Upper *float64 `json:"upper"`
		Lower *float64 `json:"lower"`
	}
	var req requestType
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	if err := c.taskManager.SetThreshold(taskId, projectId, req.Lower, req.Upper); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
	}
}

// 进行流处理
func (c *Controller) writeStream(ctx *gin.Context) {
	precision := ctx.Query("precision")
	if precision == "" {
		precision = "n"
	}
	body, _ := ioutil.ReadAll(ctx.Request.Body)
	points, err := models.ParsePointsWithPrecision(body, time.Now().UTC(), precision)
	if err != nil {
		if err.Error() == "EOF" {
			ctx.JSON(http.StatusOK, nil)
		} else {
			ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, nil)
	c.taskManager.WritePoints(points)
}

type computeRequest struct {
	ProjectId   int                   `json:"project_id"`
	Preprocess  *impl.ModelService    `json:"preprocess"`
	DetectModel *impl.ModelService    `json:"detect_model"`
	Target      impl.UnvariedSeries   `json:"target"`       // 目标检测序列
	Independent []impl.UnvariedSeries `json:"independent"`  // 其它序列（自变量）
	ModelUpdate *impl.BatchMeta       `json:"model_update"` // 用于更新阈值
}

func (c *Controller) computeThreshold(ctx *gin.Context) {
	var req computeRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
		return
	}

	if req.ProjectId <= 0 {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "project id cannot be empty"})
		return
	}

	if req.DetectModel == nil || req.ModelUpdate == nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "undefined model update"})
		return
	}

	series := append([]impl.UnvariedSeries{req.Target}, req.Independent...)
	flux := req.ModelUpdate.Query.TransToFlux(req.ProjectId, series)

	queryRes, err := db.InfluxdbClient.QueryMultiple(flux, context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	}

	data := queryRes

	if req.Preprocess != nil && req.Preprocess.Name != "" {
		d := service.InvokeRequest{
			Params: req.Preprocess.Params,
			Data:   queryRes,
		}
		out, err := service.InvokePost(req.Preprocess.Name, service.DataPreprocessMethod, d)
		if err == nil {
			var resp service.PreprocessResponse
			if err := json.Unmarshal(out, &resp); err == nil {
				data = resp.Data
			}
		}
	}

	d := service.InvokeRequest{
		Params: req.DetectModel.Params,
		Data:   data,
	}
	out, err := service.InvokePost(req.DetectModel.Name, service.ModelUpdateMethod, d)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
		return
	}
	var resp service.InvokeResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: resp})
	}
}
