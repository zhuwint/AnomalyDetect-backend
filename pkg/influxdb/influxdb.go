package influxdb

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"golang.org/x/sync/semaphore"
	"time"
)

type Connector struct {
	Address string
	Bucket  string
	Token   string
	Org     string
	client  influxdb2.Client
	sema    *semaphore.Weighted // use for concurrency
}

func NewConnector(address, bucket, token, org string) (*Connector, error) {
	return &Connector{
		Address: address,
		Bucket:  bucket,
		Token:   token,
		Org:     org,
		client:  influxdb2.NewClient(address, token),
		sema:    semaphore.NewWeighted(10),
	}, nil
}

func (c *Connector) Close() {
	c.client.Close()
}

func (c *Connector) acquire() {
	_ = c.sema.Acquire(context.Background(), 1)
}

func (c *Connector) release() {
	c.sema.Release(1)
}

func (c *Connector) QueryRaw(script string, ctx context.Context) (*api.QueryTableResult, error) {
	c.acquire()
	defer c.release()
	queryAPI := c.client.QueryAPI(c.Org)
	if result, err := queryAPI.Query(ctx, script); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func (c *Connector) Query(script string, ctx context.Context) ([]*Point, error) {
	result, err := c.QueryRaw(script, ctx)
	if err != nil {
		return nil, err
	}

	// 这里必须创建初始化大小，否则对空数组gin会返回null
	res := make([]*Point, 0)

	for result.Next() {
		var p = &Point{
			Time:     result.Record().Time(),
			Value:    nil,
			FieldTag: result.Record().ValueByKey("sensor_type").(string),
		}
		// .Format("2006-01-02 15:04:05")

		// value type only support float32 and float64
		value := result.Record().ValueByKey("_value")
		switch value.(type) {
		case float32:
			_v := float64(value.(float32))
			p.Value = &_v
		case float64:
			_v := value.(float64)
			p.Value = &_v
		default:
			if value != nil {
				return nil, fmt.Errorf("invalid value type")
			}
		}
		res = append(res, p)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query parsing error: %s", result.Err().Error())
	}

	return res, nil
}

func (c *Connector) QueryMultiple(script string, ctx context.Context) (*TimeSeries, error) {
	result, err := c.QueryRaw(script, ctx)
	if err != nil {
		return nil, err
	}

	series := &TimeSeries{
		Time:    make([]time.Time, 0),
		Columns: make([]string, 0),
		Value:   make(map[string][]*float64),
	}
	var timestamps []time.Time
	values := make([]*float64, 0)

	tableCount := -1

	for result.Next() {
		if result.TableChanged() {
			if tableCount >= 0 {
				col := fmt.Sprintf("value%d", tableCount)
				series.Columns = append(series.Columns, col)
				series.Value[col] = values
				values = make([]*float64, 0)
			}
			tableCount++
		}

		if result.TablePosition() == 0 {
			timestamps = append(timestamps, result.Record().Time())
		}
		v := result.Record().ValueByKey("_value")
		switch v.(type) {
		case float64:
			_v := v.(float64)
			values = append(values, &_v)
		case float32:
			_v := float64(v.(float32))
			values = append(values, &_v)
		default:
			if v == nil {
				values = append(values, nil)
			} else {
				return nil, fmt.Errorf("invalid value type")
			}
		}
	}

	if tableCount >= 0 {
		col := fmt.Sprintf("value%d", tableCount)
		series.Columns = append(series.Columns, col)
		series.Value[col] = values
		series.Time = timestamps
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query parsing error: %s", result.Err().Error())
	}

	return series, nil
}

func (c *Connector) WritePoint(p *write.Point) {
	writeApi := c.client.WriteAPI(c.Org, c.Bucket)
	writeApi.WritePoint(p)
	writeApi.Flush()
}
