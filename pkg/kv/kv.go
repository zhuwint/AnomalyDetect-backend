package kv

type KV struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
