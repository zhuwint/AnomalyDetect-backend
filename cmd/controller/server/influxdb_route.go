package server

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/pkg/influxdb"
	"anomaly-detect/pkg/kv"
	"context"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type dataQuery struct {
	SensorMac  *string `json:"sensor_mac"`
	SensorType *string `json:"sensor_type"`
	ReceiveNo  *string `json:"receive_no"`
}

type queryRequest struct {
	ProjectID int                 `json:"project_id"`
	Start     string              `json:"start"`
	Stop      string              `json:"stop"`
	Interval  string              `json:"interval"`
	Filter    dataQuery           `json:"filter"`
	GroupBy   map[string][]string `json:"group_by"`
}

const TIME_LAYOUT = "2006-01-02 15:04:05"
const TIME_FORMAT = "2006-01-02T15:04:05Z"

func (c *Controller) query(ctx *gin.Context) {
	var req queryRequest
	if ctx.BindJSON(&req) != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误", Data: nil})
		return
	}

	var filters []kv.KV
	_t := reflect.TypeOf(req.Filter)
	_v := reflect.ValueOf(req.Filter)
	for i := 0; i < _t.NumField(); i++ {
		value, ok := _v.Field(i).Interface().(*string)
		if !ok {
			ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "请求参数错误", Data: nil})
			return
		}
		if value == nil {
			continue
		}
		filters = append(filters, kv.KV{Key: _t.Field(i).Tag.Get("json"), Value: *value})
	}

	_start, err := time.ParseInLocation(TIME_LAYOUT, req.Start, time.Local)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "时间戳格式错误", Data: nil})
		return
	}
	_stop, err := time.ParseInLocation(TIME_LAYOUT, req.Stop, time.Local)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: "时间戳格式错误", Data: nil})
		return
	}

	query := influxdb.GeneralQuery{
		Bucket:      db.InfluxdbClient.Bucket,
		Measurement: MEASUREMENT,
		Fields:      []string{FIELD},
		Filters:     filters,
		Aggregate: influxdb.Aggregate{
			Enable: true,
			Every:  req.Interval,
			Fn:     "mean",
		},
		Range: influxdb.Range{
			Start: _start.Format(TIME_FORMAT),
			Stop:  _stop.Format(TIME_FORMAT),
		},
		GroupBy: req.GroupBy,
	}

	// if err := query.Validate(); err != nil {
	// 	ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: err.Error(), Data: nil})
	// 	return
	// }

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	res, err := db.InfluxdbClient.Query(query.TransToFlux(), timeoutCtx)

	if err != nil {
		logrus.Error(err.Error())
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error(), Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: res})
}

func (c *Controller) streamApi(ctx *gin.Context) {
	body, _ := ioutil.ReadAll(ctx.Request.Body)
	logrus.Info(string(body))
	// TODO: implement
}
