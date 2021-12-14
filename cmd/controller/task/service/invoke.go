package service

import (
	"anomaly-detect/pkg/influxdb"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	ModelUpdateMethod    string = "update"
	AnomalyDetectMethod  string = "detect"
	DataPreprocessMethod string = "process"
	GetParamsMethod      string = "params"
	ApplicationJson      string = "application/json"
)

type ModelType int

const (
	DetectType ModelType = iota
	ProcessType
)

type InvokeRequest struct {
	Params map[string]interface{} `json:"params"` // 模型参数
	Data   *influxdb.TimeSeries   `json:"data"`   // 时间序列数据
}

type InvokeResponse struct {
	ThresholdUpper *float64 `json:"threshold_upper"`
	ThresholdLower *float64 `json:"threshold_lower"`
	EigenValue     *float64 `json:"eigen_value"`
	Success        bool     `json:"success"`
	IsAnomaly      bool     `json:"is_anomaly"`
	Error          string   `json:"error"`
}

type PreprocessResponse struct {
	Success bool                 `json:"success"`
	Error   string               `json:"error"`
	Data    *influxdb.TimeSeries `json:"data"`
	Anomaly *influxdb.TimeSeries `json:"anomaly"`
}

type GetParamsResponse struct {
	Params      map[string]interface{} `json:"params"`
	Description string                 `json:"description"`
	SupportMlt  bool                   `json:"support_mlt"` // 是否支持多变量序列
}

//func InvokePost(app, method string, body interface{}) ([]byte, error) {
//	data, err := json.Marshal(body)
//	if err != nil {
//		return []byte{}, err
//	}
//
//	return daprHelper.Client.InvokeMethodWithContent(context.Background(), app, method, http.MethodPost, &dapr.DataContent{
//		Data:        data,
//		ContentType: "application/json",
//	})
//}

func InvokePost(app, method string, body interface{}) ([]byte, error) {
	var url string
	m, _ := Model.Get(app)
	url = m.Url
	if url == "" {
		return []byte{}, fmt.Errorf("model %s not register", app)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return []byte{}, err
	}
	resp, err := http.Post(fmt.Sprintf("%s/%s", url, method), ApplicationJson, bytes.NewReader(data))
	if err != nil {
		return []byte{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return []byte{}, fmt.Errorf("status code: %v", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

type ValidateParamsResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

func ParamsValidate(app string, params map[string]interface{}) error {
	var url string
	m, _ := Model.Get(app)
	url = m.Url
	if url == "" {
		return fmt.Errorf("model %s not register", app)
	}

	data, _ := json.Marshal(params)
	resp, err := http.Post(fmt.Sprintf("%s/%s", url, GetParamsMethod), ApplicationJson, bytes.NewBuffer(data))
	if err != nil {
		return err
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("status code: %v", resp.StatusCode)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var r ValidateParamsResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("unknow model response")
	}
	if r.Status == 0 {
		return nil
	}
	return fmt.Errorf(r.Msg)
}

func GetModelParams(app string) (GetParamsResponse, error) {
	m, _ := Model.Get(app)
	url := m.Url
	if url == "" {
		return GetParamsResponse{}, fmt.Errorf("model %s not register", app)
	}

	resp, err := http.Get(fmt.Sprintf("%s/%s", url, GetParamsMethod))
	if err != nil {
		return GetParamsResponse{}, err
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return GetParamsResponse{}, fmt.Errorf("status code: %v", resp.StatusCode)
	}
	data, _ := ioutil.ReadAll(resp.Body)
	var r GetParamsResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return GetParamsResponse{}, fmt.Errorf("unknow model response")
	}
	return r, nil
}

// ParamsValidate 模型参数验证
//func ParamsValidate(app string, params map[string]interface{}) error {
//	data, _ := json.Marshal(params)
//
//	fmt.Println(app, GetParamsMethod, http.MethodPost)
//
//	body, err := daprHelper.Client.InvokeMethodWithContent(context.Background(), app, GetParamsMethod, http.MethodPost,
//		&dapr.DataContent{
//			Data:        data,
//			ContentType: "application/json",
//		})
//	if err != nil {
//		return fmt.Errorf("cannot connect to model %s", app)
//	}
//
//	var resp ValidateParamsResponse
//	if err := json.Unmarshal(body, &resp); err != nil {
//		return fmt.Errorf("unknow model response")
//	}
//	if resp.Status == 0 {
//		return nil
//	}
//	return fmt.Errorf(resp.Msg)
//}

//func GetModelParams(app string) (GetParamsResponse, error) {
//	body, err := daprHelper.Client.InvokeMethod(context.Background(), app, GetParamsMethod, http.MethodGet)
//	if err != nil {
//		return GetParamsResponse{}, fmt.Errorf("cannot connect to model %s", app)
//	}
//	var resp GetParamsResponse
//	if err := json.Unmarshal(body, &resp); err != nil {
//		return GetParamsResponse{}, fmt.Errorf("unknow model response")
//	}
//	return resp, nil
//}
