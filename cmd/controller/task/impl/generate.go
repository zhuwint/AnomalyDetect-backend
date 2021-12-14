package impl

import (
	"anomaly-detect/pkg/snowflake"
)

var sf *snowflake.Snowflake

func init() {
	s, err := snowflake.NewSnowflake(int64(0), int64(0))
	if err != nil {
		panic(err.Error())
	}
	sf = s
}

const dict = "abcdefghijklmnopqrstuvwxyz"

func GenerateTaskId() string {
	id := sf.NextVal()

	// 将int64转化为字母
	var res []byte
	for id > 0 {
		res = append(res, dict[id%10])
		id = id / 10
	}
	return string(res)
}
