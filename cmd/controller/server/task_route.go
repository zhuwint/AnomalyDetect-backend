package server

import (
	"anomaly-detect/cmd/controller/task/impl"
	"anomaly-detect/pkg/models"
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
