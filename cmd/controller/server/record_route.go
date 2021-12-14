package server

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const taskFilter = "task_id=?"
const projectFilter = "project_id=?"
const viewFilter = "view=?"
const timeStartFilter = "created> ?"
const timeEndFilter = "created< ?"
const timeFormat = "2006-01-02 15:04:05"

func (c *Controller) getRecord(ctx *gin.Context) {
	taskId := ctx.Query("taskId")
	projectId := ctx.Query("projectId")
	view := ctx.Query("view")

	var query []string
	var values []interface{}

	if projectId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误"})
		return
	} else {
		query = append(query, projectFilter)
		values = append(values, projectId)
	}
	if taskId != "" {
		query = append(query, taskFilter)
		values = append(values, taskId)
	}
	if view != "" {
		if v, err := strconv.ParseBool(view); err != nil {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误"})
			return
		} else {
			query = append(query, viewFilter)
			values = append(values, v)
		}
	}
	if st, err := time.ParseInLocation(timeFormat, ctx.Query("start"), time.Local); err == nil {
		query = append(query, timeStartFilter)
		values = append(values, st)
	}
	if et, err := time.ParseInLocation(timeFormat, ctx.Query("end"), time.Local); err == nil {
		query = append(query, timeEndFilter)
		values = append(values, et)
	}
	queryString := strings.Join(query, " and ")

	var res []model.AlertRecord
	if err := db.MysqlClient.DB.Where(queryString, values...).Order("created DESC").Find(&res).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "获取失败"})
			return
		}
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: res})
}

func (c *Controller) setRecordView(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误"})
		return
	}

	var r model.AlertRecord
	if err := db.MysqlClient.DB.Where("id=?", id).First(&r).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "操作失败"})
		} else {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "记录不存在"})
		}
		return
	}
	r.View = true
	if err := db.MysqlClient.DB.Save(&r).Error; err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "操作失败"})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "操作成功"})
	}
}

func (c *Controller) acceptAll(ctx *gin.Context) {
	var querys []string
	var values []interface{}

	projectId := ctx.Query("projectId")
	if projectId == "" {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误"})
		return
	} else {
		querys = append(querys, projectFilter)
		values = append(values, projectId)
	}
	if taskId := ctx.Query("taskId"); taskId != "" {
		querys = append(querys, taskFilter)
		values = append(values, taskId)
	}
	if st, err := time.ParseInLocation(timeFormat, ctx.Query("start"), time.Local); err == nil {
		querys = append(querys, timeStartFilter)
		values = append(values, st)
	}
	if et, err := time.ParseInLocation(timeFormat, ctx.Query("end"), time.Local); err == nil {
		querys = append(querys, timeEndFilter)
		values = append(values, et)
	}

	queryString := strings.Join(querys, " and ")

	if err := db.MysqlClient.DB.Model(&model.AlertRecord{}).Where(queryString, values...).Update("view", true).Error; err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "操作失败"})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "操作成功"})
	}
}
