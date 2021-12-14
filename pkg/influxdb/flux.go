package influxdb

import (
	"anomaly-detect/pkg/kv"
	"anomaly-detect/pkg/validator"
	"fmt"
	"strings"
	"time"
)

// InfluxQuery : influxdb query interface
type InfluxQuery interface {
	TransToFlux() string // transform struct to flux script
}

type Filter interface {
}

// BaseQuery : use for unvaried time series query
// use []BaseQuery to query multivariable time series
// filters: string, like
//          ["host=ubuntu", "cpu=cpu0,cpu=cpu1"]
// must provide at least one field
type BaseQuery struct {
	Alias       string   `json:"alias"` // name of the query, use for table join. must be unique
	Bucket      string   `json:"bucket"`
	Measurement string   `json:"measurement"`
	Fields      []string `json:"fields"`
	Filters     []kv.KV  `json:"filters"`
}

func (q BaseQuery) Validate() error {
	if q.Alias == "" {
		return fmt.Errorf("alias could not be empty")
	}
	if q.Bucket == "" {
		return fmt.Errorf("bucket could not be empty")
	}
	if q.Measurement == "" {
		return fmt.Errorf("measurement could not be empty")
	}
	if len(q.Fields) == 0 {
		return fmt.Errorf("must provide at least one field")
	}
	for _, f := range q.Filters {
		if f.Key == "" || f.Value == "" {
			return fmt.Errorf("filter key(value) could not be empty")
		}
	}
	return nil
}

// Aggregate : aggregate with interval and aggregate function
type Aggregate struct {
	Enable      bool   `json:"enable"`
	Every       string `json:"every"`
	Fn          string `json:"fn"`
	CreateEmpty bool   `json:"create_empty"`
}

func (a Aggregate) Validate() error {
	if !a.Enable {
		return nil
	}
	if a.Fn != "mean" && a.Fn != "median" {
		return fmt.Errorf("fn must in [mean, median]")
	}
	if d, err := time.ParseDuration(a.Every); err != nil {
		return err
	} else {
		now := time.Now()
		if !now.Add(d).After(now) {
			return fmt.Errorf("every must be positive time duration")
		}
	}
	return nil
}

// Range : query with time range
type Range struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

func (r Range) Validate() error {
	if r.Start == "" || r.Stop == "" {
		return fmt.Errorf("range parse failed: could not be empty string")
	}

	// start can be all supported unix duration unit relative to stop or absolute time.
	// for example -20h5m3s, 2019-08-28T22:00:00Z.
	// stop can be all supported unix duration unit relative to now or absolute time or now().
	// for example -20h5m3s, 2019-08-28T22:00:00Z, now().
	// supported duration-types are "ns", "us" (or "µs"), "ms", "s", "m", "h".

	start, err := validator.CheckTimeBeforeNow(r.Start)
	if err != nil {
		return fmt.Errorf("range (start) parse failed: %s", err.Error())
	}

	stop, err := validator.CheckTimeBeforeNow(r.Stop)
	if err != nil {
		return fmt.Errorf("range (stop) parse failed: %s", err.Error())
	}

	if !start.Before(stop) {
		return fmt.Errorf("range parse failed: start should before stop")
	}
	return nil
}

// GeneralQuery : the general query option. User can also design their own query option
type GeneralQuery struct {
	Bucket      string              `json:"bucket"`
	Measurement string              `json:"measurement"`
	Fields      []string            `json:"fields"`
	Filters     []kv.KV             `json:"filters"`
	Aggregate   Aggregate           `json:"aggregate"`
	Range       Range               `json:"range"`
	GroupBy     map[string][]string `json:"group_by"`
}

func (g GeneralQuery) Validate() error {
	if g.Bucket == "" {
		return fmt.Errorf("bucket could not be empty")
	}
	if g.Measurement == "" {
		return fmt.Errorf("measurement could not be empty")
	}
	if len(g.Fields) == 0 {
		return fmt.Errorf("must provide at least one field")
	}
	for _, f := range g.Filters {
		if f.Key == "" || f.Value == "" {
			return fmt.Errorf("filter key(value) could not be empty")
		}
	}
	if err := g.Aggregate.Validate(); err != nil {
		return err
	}
	if err := g.Range.Validate(); err != nil {
		return err
	}
	return nil
}

const (
	BucketSnippet      string = "from(bucket: \"%s\")"
	TimeRangeSnippet   string = " |> range(start: %s, stop: %s)"
	FilterSnippet      string = " |> filter(fn: (r) => %s)"
	AggregateSnippet   string = " |> aggregateWindow(every: %s, fn: %s, createEmpty: %v)"
	YieldSnippet       string = " |> yield(name: \"%s\")"
	GroupSnippet       string = " |> group(columns: [%s])"
	JoinSnippet        string = "join(tables: {%s}, on: [\"_time\"])"
	MeasurementSnippet string = "r._measurement == \"%s\""
	FieldSnippet       string = "r._field == \"%s\""
	TagSnippet         string = "r.%s == \"%v\""
)

func (g GeneralQuery) TransToFlux() string {
	var scripts []string
	// bucket
	scripts = append(scripts, fmt.Sprintf(BucketSnippet, g.Bucket))

	// range
	scripts = append(scripts, fmt.Sprintf(TimeRangeSnippet, g.Range.Start, g.Range.Stop))

	// measurement
	measurement := fmt.Sprintf("r._measurement == \"%s\"", g.Measurement)
	scripts = append(scripts, fmt.Sprintf(FilterSnippet, measurement))

	// fields
	var fields []string
	for _, f := range g.Fields {
		fields = append(fields, fmt.Sprintf("r._field == \"%s\"", f))
	}
	scripts = append(scripts, fmt.Sprintf(FilterSnippet, strings.Join(fields, " or ")))

	// filters
	if len(g.Filters) > 0 {
		var filters []string
		for _, f := range g.Filters {
			filters = append(filters, fmt.Sprintf("r.%s == \"%v\"", f.Key, f.Value))
		}
		scripts = append(scripts, fmt.Sprintf(FilterSnippet, strings.Join(filters, " and ")))
	}

	// group filters
	var groupKey []string
	if g.GroupBy != nil {
		for k, v := range g.GroupBy {
			groupKey = append(groupKey, fmt.Sprintf("\"%s\"", k))
			if len(v) > 0 {
				_vs := []string{}
				for _, _v := range v {
					_vs = append(_vs, fmt.Sprintf("r.%s == \"%v\"", k, _v))
				}
				scripts = append(scripts, fmt.Sprintf(FilterSnippet, strings.Join(_vs, " or ")))
			}
		}
	}

	// aggregate
	if g.Aggregate.Enable {
		scripts = append(scripts, fmt.Sprintf(AggregateSnippet, g.Aggregate.Every, g.Aggregate.Fn, g.Aggregate.CreateEmpty))
	}

	// group
	if len(groupKey) > 0 {
		scripts = append(scripts, fmt.Sprintf(GroupSnippet, strings.Join(groupKey, ",")))
	}

	return strings.Join(scripts, "\n")
}
