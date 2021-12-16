package server

import (
	"anomaly-detect/cmd/controller/task/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type modelResp struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Health bool   `json:"health"`
}

func (c *Controller) getStreamModel(ctx *gin.Context) {
	c.getModel(ctx, service.StreamType)
}

func (c *Controller) getBatchModel(ctx *gin.Context) {
	c.getModel(ctx, service.BatchType)
}

func (c *Controller) getDataProcessModel(ctx *gin.Context) {
	c.getModel(ctx, service.ProcessType)
}

func (c *Controller) getModel(ctx *gin.Context, t service.ModelType) {
	models := service.Model.Keys()
	res := make([]modelResp, 0)
	for _, k := range models {
		if m, ok := service.Model.Get(k); ok {
			if m.Type == t {
				if _, err := service.GetModelParams(k); err != nil {
					res = append(res, modelResp{Name: k, Url: m.Url, Health: false})
				} else {
					res = append(res, modelResp{Name: k, Url: m.Url, Health: true})
				}
			}
		}
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: res})
}

type ModelRequest struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Type int    `json:"type"`
}

func (c *Controller) registerModel(ctx *gin.Context) {
	var req ModelRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "invalid request"})
		return
	}
	service.Model.Set(req.Name, req.Url, service.ModelType(req.Type))
	_ = service.Save(req.Name, req.Url, req.Type)
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
}

func (c *Controller) unregisterModel(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "invalid request"})
		return
	}
	service.Model.Del(name)
	_ = service.Delete(name)
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success"})
}

func (c *Controller) getModelParams(ctx *gin.Context) {
	modelName := ctx.Query("model")
	if modelName == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide model name"})
		return
	}
	resp, err := service.GetModelParams(modelName)
	if err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "success", Data: resp})
	}
}

func (c *Controller) paramsValidate(ctx *gin.Context) {
	modelName := ctx.Query("model")
	if modelName == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "must provide model name"})
		return
	}
	var req map[string]interface{}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "params parse failed"})
		return
	}
	if err := service.ParamsValidate(modelName, req); err != nil {
		ctx.JSON(http.StatusOK, ginResponse{Status: -1, Msg: err.Error()})
	} else {
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "ok"})
	}
}
