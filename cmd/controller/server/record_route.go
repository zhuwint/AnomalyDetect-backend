package server

import (
	"anomaly-detect/cmd/controller/task/record"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (c *Controller) getSystemRecord(ctx *gin.Context) {
	projectId := ctx.Query("projectId")
	if projectId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "project cannot be empty"})
		return
	}
	taskId := ctx.Query("taskId")
	start := ctx.Query("start")
	stop := ctx.Query("end")

	res, err := record.GetSystemRecord(projectId, taskId, start, stop)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: res})
	}
}

func (c *Controller) getAlertRecord(ctx *gin.Context) {
	projectId := ctx.Query("projectId")
	if projectId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "projectId cannot be empty"})
		return
	}
	taskId := ctx.Query("taskId")
	if taskId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "taskId cannot be empty"})
		return
	}
	start := ctx.Query("start")
	stop := ctx.Query("end")
	res, err := record.GetAlertRecord(projectId, taskId, start, stop)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: res})
	}
}
