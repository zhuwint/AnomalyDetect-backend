package impl

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/pkg/influxdb"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	DefaultMeasurement = "sensor_data"
	DefaultFieldName   = "value"
	ProjectIdTag       = "project_id"
	SensorMacTag       = "sensor_mac"
	SensorTypeTag      = "sensor_type"
	ReceiveNoTag       = "receive_no"
)

// UnvariedSeries 单变量时间序列查询
type UnvariedSeries struct {
	SensorMac  string `json:"sensor_mac"`
	ReceiveNo  string `json:"receive_no"`
	SensorType string `json:"sensor_type"`
}

func (u UnvariedSeries) Validate() error {
	if u.SensorMac == "" || u.ReceiveNo == "" || u.SensorType == "" {
		return fmt.Errorf("information cannot be empty")
	}
	return nil
}

// QueryOptions 数据查询，支持多变量时间序列查询
type QueryOptions struct {
	Range       influxdb.Range `json:"range"`
	Aggregate   string         `json:"aggregate"`
	Measurement string         `json:"measurement"`
	//ProjectId   int              `json:"project_id"`
	//Series      []UnvariedSeries `json:"series"`
}

func (d QueryOptions) Validate() error {
	//if d.ProjectId <= 0 {
	//	return fmt.Errorf("invalid project_id")
	//}
	if err := d.Range.Validate(); err != nil {
		return err
	}
	if d, err := time.ParseDuration(d.Aggregate); err != nil {
		return err
	} else {
		now := time.Now()
		if !now.Add(d).After(now) {
			return fmt.Errorf("every must be positive time duration")
		}
	}
	if d.Measurement == "" {
		return fmt.Errorf("measurement cannot be empty")
	}
	//if len(d.Series) == 0 {
	//	return fmt.Errorf("must provide at least one series")
	//}
	//for _, s := range d.Series {
	//	if s.ReceiveNo == "" || s.SensorMac == "" || s.SensorType == "" {
	//		return fmt.Errorf("series information cannot be empty")
	//	}
	//}
	return nil
}

// TransToFlux return "" if failed
func (d QueryOptions) TransToFlux(projectId int, series []UnvariedSeries) string {
	if err := d.Validate(); err != nil {
		return ""
	}
	var scripts []string

	for _, series := range series {
		var snippets []string
		// bucket
		snippets = append(snippets, fmt.Sprintf(influxdb.BucketSnippet, db.InfluxdbClient.Bucket))
		// range
		snippets = append(snippets, fmt.Sprintf(influxdb.TimeRangeSnippet, d.Range.Start, d.Range.Stop))
		// filters: measurement and field and tag filter
		var filters []string
		filters = append(filters, fmt.Sprintf(influxdb.MeasurementSnippet, d.Measurement)) // measurement
		filters = append(filters, fmt.Sprintf(influxdb.FieldSnippet, DefaultFieldName))    // field
		// project_id
		filters = append(filters, fmt.Sprintf(influxdb.TagSnippet, ProjectIdTag, projectId))
		// sensor_type, sensor_mac and receive_no
		t := reflect.TypeOf(series)
		v := reflect.ValueOf(series)
		for i := 0; i < t.NumField(); i++ {
			value, ok := v.Field(i).Interface().(string)
			if !ok {
				return ""
			}
			if value == "" {
				continue
			}
			filters = append(filters, fmt.Sprintf(influxdb.TagSnippet, t.Field(i).Tag.Get("json"), value))
		}
		snippets = append(snippets, fmt.Sprintf(influxdb.FilterSnippet, strings.Join(filters, " and ")))
		// aggregate
		snippets = append(snippets, fmt.Sprintf(influxdb.AggregateSnippet, d.Aggregate, "mean", true))
		// yield
		resultName := strings.Join([]string{series.SensorMac, series.SensorType, series.ReceiveNo}, "#")
		snippets = append(snippets, fmt.Sprintf(influxdb.YieldSnippet, resultName))

		scripts = append(scripts, strings.Join(snippets, "\n"))
	}
	return strings.Join(scripts, "\n\n")
}
