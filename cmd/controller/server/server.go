package server

import (
	"anomaly-detect/cmd/controller/task"
	"anomaly-detect/cmd/controller/task/api"
	"anomaly-detect/cmd/controller/task/impl"
	"anomaly-detect/cmd/controller/task/store"
	"anomaly-detect/cmd/controller/task/union"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	MEASUREMENT = "sensor_data"
	FIELD       = "value"
)

type Controller struct {
	//*dapr.Dapr
	httpServer  *gin.Engine // http server
	taskManager *task.Manager
}

const (
	defaultServiceName = "controller"
	defaultHttpPort    = 3030
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewController() (*Controller, error) {
	//serviceName := env.GetEnvString(dapr.AppIdEnv, defaultServiceName)
	//servicePort := env.GetEnvInt(dapr.AppPortEnv, defaultHttpPort)
	//daprInstance := dapr.NewDapr(serviceName, servicePort)
	// e := gin.Default()
	e := gin.New()
	e.Use(gin.Recovery())

	return &Controller{
		//Dapr:        daprInstance,
		httpServer:  e,
		taskManager: task.NewManager(),
	}, nil
}

func (c *Controller) Start() error {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("server runtime failed")
		}
	}()

	c.initRouter()
	c.registerKong()
	// go time.AfterFunc(7*time.Second, c.initTask)
	go time.AfterFunc(7*time.Second, c.initUnionTask)

	// 此处阻塞
	if err := c.httpServer.Run(fmt.Sprintf(":%d", defaultHttpPort)); err != nil {
		return err
	}
	return nil
}

// 向kong网关注册路由
func (c *Controller) registerKong() {

}

// 初始化路由
func (c *Controller) initRouter() {
	api := c.httpServer.Group("/api")
	user := api.Group("/user")
	{
		user.POST("/login", c.login)
	}
	sensor := api.Group("/manage")
	{
		sensor.GET("/projects", c.getProjects)
		sensor.POST("/sensors", c.getSensors)
		sensor.GET("/locations", c.getLocations)
		sensor.GET("/measurements", c.getMeasurements)
	}

	api.GET("/task", c.getTask)
	api.DELETE("/task", c.deleteTask)
	tasks := api.Group("/task")
	{
		tasks.GET("/target", c.getTaskByTargetSeries)
		tasks.POST("/set", c.setThreshold)
		// 批处理任务的创建与更新
		tasks.POST("/batch", c.createBatchTask)
		tasks.PUT("/batch", c.updateBatchTask)
		// 流处理任务的创建与更新
		tasks.POST("/stream", c.createStreamTask)
		tasks.PUT("/stream", c.updateStreamTask)
		// 联合告警任务的创与更新
		tasks.POST("/union", c.createUnionTask)
		tasks.PUT("/union", c.updateUnionTask)
		// 控制模型更新与异常检测 开启/暂停
		tasks.POST("/control", c.taskControl)
		tasks.POST("/compute", c.computeThreshold)
	}
	model := api.Group("/model")
	{
		model.POST("/register", c.registerModel)
		model.DELETE("/register", c.unregisterModel)
		model.GET("/stream", c.getStreamModel)
		model.GET("/batch", c.getBatchModel)
		model.GET("/process", c.getDataProcessModel)
		model.GET("/params", c.getModelParams)
		model.POST("/validate", c.paramsValidate)
	}
	api.POST("/v2/write", c.writeStream) // 流处理
	data := api.Group("/data")
	{
		data.POST("/query", c.query)
	}
	record := api.Group("/record")
	{
		record.GET("/system", c.getSystemRecord)
		record.GET("/alert", c.getAlertRecord)
	}
}

func (c *Controller) initTask() {
	tasks, err := store.GetAll()
	if err != nil {
		logrus.Error("init task failed: %s", err.Error())
		return
	}
	// 注： 这列不能用 _, t := range 遍历，因为 t 用的是同一片内存
	for i := range tasks {
		var info api.Info
		switch tasks[i].IsStream {
		case false:
			var taskInfo impl.BatchTaskInfo
			if err := json.Unmarshal([]byte(tasks[i].Content), &taskInfo); err != nil {
				logrus.Errorf("init task %s failed: %s", tasks[i].TaskId, err.Error())
				_ = store.Del(tasks[i].TaskId, tasks[i].ProjectId)
				continue
			}
			info = taskInfo
		case true:
			var taskInfo impl.StreamTaskInfo
			if err := json.Unmarshal([]byte(tasks[i].Content), &taskInfo); err != nil {
				logrus.Errorf("init task %s failed: %s", tasks[i].TaskId, err.Error())
				_ = store.Del(tasks[i].TaskId, tasks[i].ProjectId)
				continue
			}
			info = taskInfo
		}
		if err := c.taskManager.Create(info); err != nil {
			logrus.Errorf("init task %s failed: %s", tasks[i].TaskId, err.Error())
		} else {
			logrus.Infof("init task %s success", tasks[i].TaskId)
			// TODO:
			_ = c.taskManager.SetThreshold(tasks[i].TaskId, tasks[i].ProjectId, "", "", "", &tasks[i].ThresholdLower, &tasks[i].ThresholdUpper)
			_ = c.taskManager.EnableAnomalyDetect(tasks[i].TaskId, tasks[i].ProjectId, tasks[i].DetectEnable)
			_ = c.taskManager.EnableModelUpdate(tasks[i].TaskId, tasks[i].ProjectId, tasks[i].UpdateEnable)
		}
	}
}

func (c *Controller) initUnionTask() {
	tasks, err := union.GetAll()
	if err != nil {
		logrus.Error("init union task failed: %s", err.Error())
		return
	}
	for i := range tasks {
		var taskInfo union.TaskInfo
		if err := json.Unmarshal([]byte(tasks[i].Content), &taskInfo); err != nil {
			logrus.Errorf("init union task %s failed: %s", tasks[i].TaskId, err.Error())
			_ = union.Del(tasks[i].TaskId, tasks[i].ProjectId)
			continue
		}
		if err := c.taskManager.Create(taskInfo); err != nil {
			logrus.Errorf("init union task %s failed:%s", tasks[i].TaskId, err.Error())
		} else {
			logrus.Infof("init union task %s success", tasks[i].TaskId)
			_ = c.taskManager.EnableAnomalyDetect(tasks[i].TaskId, tasks[i].ProjectId, tasks[i].Enable)
		}
	}
}

func (c *Controller) Stop() {
	// TODO: implement
}
