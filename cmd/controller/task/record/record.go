package record

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/pkg/influxdb"
	"context"
	"errors"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

const queryString = "task_id=? and project_id=?"

const (
	alertLogMeasurement  = "alert_logs"
	systemLogMeasurement = "system_logs"
)

const (
	InfoLevel  = "info"
	ErrorLevel = "error"
)

type Record struct {
	Time           time.Time `json:"time"`
	Start          time.Time `json:"start"`
	Stop           time.Time `json:"stop"`
	SensorMac      string    `json:"sensor_mac"`
	SensorType     string    `json:"sensor_type"`
	ReceiveNo      string    `json:"receive_no"`
	ThresholdUpper float64   `json:"threshold_upper"`
	ThresholdLower float64   `json:"threshold_lower"`
	Value          float64   `json:"value"`
	Level          int       `json:"level"`
	Description    string    `json:"description"`
}

// SaveAlertRecord 保存告警日志
func SaveAlertRecord(taskId string, projectId int, data Record) error {
	point := influxdb2.NewPoint(
		alertLogMeasurement,
		map[string]string{
			"task_id":     taskId,
			"project_id":  strconv.Itoa(projectId),
			"sensor_mac":  data.SensorMac,
			"sensor_type": data.SensorType,
			"receive_no":  data.ReceiveNo,
		},
		map[string]interface{}{
			"threshold_upper": data.ThresholdUpper,
			"threshold_lower": data.ThresholdLower,
			"alert":           data.Level > 0,
			"value":           data.Value,
			"start":           data.Start.Format(timeFormatTz),
			"stop":            data.Stop.Format(timeFormatTz),
		},
		data.Time)

	db.InfluxdbClient.WritePoint(point)
	return nil
}

// SaveSystemRecord 保存系统日志
func SaveSystemRecord(taskId string, projectId int, data Record, level string) error {
	point := influxdb2.NewPoint(
		systemLogMeasurement,
		map[string]string{
			"task_id":     taskId,
			"project_id":  strconv.Itoa(projectId),
			"sensor_mac":  data.SensorMac,
			"sensor_type": data.SensorType,
			"receive_no":  data.ReceiveNo,
			"level":       level,
		},
		map[string]interface{}{
			"threshold_upper": data.ThresholdUpper,
			"threshold_lower": data.ThresholdLower,
			"value":           data.Value,
			"description":     data.Description,
		},
		time.Now())

	db.InfluxdbClient.WritePoint(point)
	return nil
}

const timeFormat = "2006-01-02 15:04:05"
const timeFormatTz = "2006-01-02T15:04:05Z"

const pivot = "|> pivot(\nrowKey:[\"_time\"],\ncolumnKey: [\"_field\"],\nvalueColumn: \"_value\"\n)"

type SysResponse struct {
	Time           time.Time `json:"time"`
	TaskId         string    `json:"task_id"`
	ProjectId      string    `json:"project_id"`
	SensorMac      string    `json:"sensor_mac"`
	SensorType     string    `json:"sensor_type"`
	ReceiveNo      string    `json:"receive_no"`
	ThresholdUpper float64   `json:"threshold_upper"`
	ThresholdLower float64   `json:"threshold_lower"`
	Level          string    `json:"level"`
	Description    string    `json:"description"`
}

type AlertResponse struct {
	Time           time.Time `json:"time"`
	Start          string    `json:"start"`
	Stop           string    `json:"stop"`
	TaskId         string    `json:"task_id"`
	ProjectId      string    `json:"project_id"`
	SensorMac      string    `json:"sensor_mac"`
	SensorType     string    `json:"sensor_type"`
	ReceiveNo      string    `json:"receive_no"`
	ThresholdUpper float64   `json:"threshold_upper"`
	ThresholdLower float64   `json:"threshold_lower"`
	Value          float64   `json:"value"`
	Alert          bool      `json:"alert"`
}

// docker exec -it influxdb influx delete --bucket yinao --start '2021-12-15T00:00:00Z' --stop '2021-12-15T17:00:00Z' --predicate '_measurement="system_logs"'

func GetSystemRecord(projectId, taskId, start, stop string) ([]SysResponse, error) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	res := make([]SysResponse, 0)
	data, err := query(systemLogMeasurement, projectId, taskId, start, stop)
	if err != nil {
		return res, err
	}
	for data.Next() {
		row := SysResponse{
			Time:           data.Record().Time(),
			TaskId:         data.Record().ValueByKey("task_id").(string),
			ProjectId:      data.Record().ValueByKey("project_id").(string),
			SensorMac:      data.Record().ValueByKey("sensor_mac").(string),
			SensorType:     data.Record().ValueByKey("sensor_type").(string),
			ReceiveNo:      data.Record().ValueByKey("receive_no").(string),
			ThresholdUpper: data.Record().ValueByKey("threshold_upper").(float64),
			ThresholdLower: data.Record().ValueByKey("threshold_lower").(float64),
			Level:          data.Record().ValueByKey("level").(string),
			Description:    data.Record().ValueByKey("description").(string),
		}
		//fmt.Println(row)
		res = append(res, row)
	}
	return res, nil
}

func GetAlertRecord(projectId, taskId, start, stop string) ([]AlertResponse, error) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	res := make([]AlertResponse, 0)
	data, err := query(alertLogMeasurement, projectId, taskId, start, stop)
	if err != nil {
		return res, err
	}
	for data.Next() {
		row := AlertResponse{
			Time:           data.Record().Time(),
			TaskId:         data.Record().ValueByKey("task_id").(string),
			ProjectId:      data.Record().ValueByKey("project_id").(string),
			SensorMac:      data.Record().ValueByKey("sensor_mac").(string),
			SensorType:     data.Record().ValueByKey("sensor_type").(string),
			ReceiveNo:      data.Record().ValueByKey("receive_no").(string),
			ThresholdUpper: data.Record().ValueByKey("threshold_upper").(float64),
			ThresholdLower: data.Record().ValueByKey("threshold_lower").(float64),
			Value:          data.Record().ValueByKey("value").(float64),
			Alert:          data.Record().ValueByKey("alert").(bool),
			Start:          data.Record().ValueByKey("start").(string),
			Stop:           data.Record().ValueByKey("stop").(string),
		}
		res = append(res, row)
	}
	return res, nil
}

func query(measurement, projectId, taskId, start, stop string) (*api.QueryTableResult, error) {
	if projectId == "" {
		return nil, errors.New("projectId cannot be empty")
	}
	if start == "" {
		start = "-7d"
	} else if st, err := time.ParseInLocation(timeFormat, start, time.Local); err == nil {
		start = st.Format(timeFormatTz)
	}
	if stop == "" {
		stop = "now()"
	} else if et, err := time.ParseInLocation(timeFormat, stop, time.Local); err == nil {
		stop = et.Format(timeFormatTz)
	}

	var scripts []string
	scripts = append(scripts, fmt.Sprintf(influxdb.BucketSnippet, db.InfluxdbClient.Bucket))
	scripts = append(scripts, fmt.Sprintf(influxdb.TimeRangeSnippet, start, stop))
	var filters []string
	filters = append(filters, fmt.Sprintf(influxdb.MeasurementSnippet, measurement))
	filters = append(filters, fmt.Sprintf("r[\"project_id\"] == \"%s\"", projectId))
	if taskId != "" {
		filters = append(filters, fmt.Sprintf("r[\"task_id\"] == \"%s\"", taskId))
	}
	scripts = append(scripts, fmt.Sprintf(influxdb.FilterSnippet, strings.Join(filters, " and ")))
	scripts = append(scripts, pivot)
	flux := strings.Join(scripts, "\n")
	return db.InfluxdbClient.QueryRaw(flux, context.Background())
}
