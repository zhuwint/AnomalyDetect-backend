package influxdb

import "time"

// Point 用于 ant-v 显示
type Point struct {
	Time     time.Time `json:"time"`
	Value    *float64  `json:"value"`
	FieldTag string    `json:"field_tag"`
}

type TimeSeries struct {
	Time    []time.Time           `json:"time"`
	Columns []string              `json:"columns"`
	Value   map[string][]*float64 `json:"value"`
}
