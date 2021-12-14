package server

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ginResponse struct {
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"` // 用户名
	Token    string `json:"token"`    // token
	Org      string `json:"org"`      // 用户所属组织
}

func (c *Controller) login(ctx *gin.Context) {
	var req loginRequest
	if ctx.BindJSON(&req) == nil && req.Username != "" && req.Password != "" {
		// TODO: 登录校验
		resp := loginResponse{
			Username: req.Username,
			Token:    "1232131231232",
			Org:      "admin",
		}
		ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "登录成功", Data: resp})
	} else {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "用户数据解析失败", Data: nil})
	}
}

type projectResponse struct {
	ProjectId   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
}

func (c *Controller) getProjects(ctx *gin.Context) {
	var resp []projectResponse
	if err := db.MysqlClient.DB.Model(&model.ProjectIdName{}).Select("PROJECT_ID as project_id, " +
		"PROJECT_NAME as project_name").Find(&resp).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, ginResponse{Status: -1, Msg: "获取失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: resp})
}

// 这里使用了自定义 tag：filter ，用于在使用反射生成查询语句
// filter 值即为 mysql table 中对应的字段
type sensorRequest struct {
	ProjectId *string `json:"project_id" filter:"project_id"`
	Location1 *string `json:"location1" filter:"location_1_id"`
	Location2 *string `json:"location2" filter:"location_2_id"`
	Location3 *string `json:"location3" filter:"location_3_id"`
	Location4 *string `json:"location4" filter:"location_4_id"`
}

type sensorResponse struct {
	SensorMac string `json:"sensor_mac"`
	TypeId    int    `json:"type_id"`
	TypeName  string `json:"type_name"`
	Location1 string `json:"location1"`
	Location2 string `json:"location2"`
	Location3 string `json:"location3"`
	Location4 string `json:"location4"`
}

const sensorQuery = "select s.SENSOR_MAC as sensor_mac, s.TYPE_ID as type_id, st.TYPE_NAME as type_name, " +
	"sl_1.location_name as location1, sl_2.location_name as location2, sl_3.location_name as location3, sl_4.location_name as location4 " +
	"from sensor_location s  " +
	"left join sensor_type st on s.TYPE_ID = st.ID  " +
	"left join site_location_name sl_1 on s.project_id=sl_1.project_id and s.location_1_id=sl_1.LOCATION_1_ID and sl_1.location_2_id=0  " +
	"left join site_location_name sl_2 on s.project_id=sl_2.project_id and s.location_1_id=sl_2.LOCATION_1_ID and s.location_2_id=sl_2.LOCATION_2_ID and sl_2.location_3_id=0  " +
	"left join site_location_name sl_3 on s.project_id=sl_3.project_id and s.location_1_id=sl_3.LOCATION_1_ID and s.location_2_id=sl_3.LOCATION_2_ID and s.location_3_id=sl_3.LOCATION_3_ID and sl_3.location_4_id=0 " +
	"left join site_location_name sl_4 on s.project_id=sl_4.project_id and s.location_1_id=sl_4.LOCATION_1_ID and s.location_2_id=sl_4.LOCATION_2_ID and s.location_3_id=sl_4.LOCATION_3_ID and s.location_4_id=sl_4.LOCATION_4_ID  " +
	"where "

const sensorFilter = "s.%s=?"

// 查询传感器列表
// body:
// project_id 	项目id，不能为空
// location1 	对应table location_1_id字段，可为空
// location2 	对应table location_2_id字段，可为空
// location3 	对应table location_3_id字段，可为空
// location4 	对应table location_4_id字段，可为空
func (c *Controller) getSensors(ctx *gin.Context) {
	var req sensorRequest
	if ctx.BindJSON(&req) != nil || req.ProjectId == nil {
		// 必须要有 project_id 字段
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "请求参数错误", Data: nil})
		return
	}
	var filters []string     // 查询过滤器
	var values []interface{} // 过滤器对应值
	t := reflect.TypeOf(req)
	v := reflect.ValueOf(req)
	// 反射遍历查询字段
	for i := 0; i < t.NumField(); i++ {
		value, ok := v.Field(i).Interface().(*string)
		if !ok {
			ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "获取失败", Data: nil})
			return
		}
		if value == nil {
			continue
		}
		filters = append(filters, fmt.Sprintf(sensorFilter, t.Field(i).Tag.Get("filter")))
		values = append(values, *value)
	}
	queryString := sensorQuery + strings.Join(filters, " and ")
	var resp []sensorResponse
	if err := db.MysqlClient.DB.Raw(queryString, values...).Find(&resp).Error; err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusInternalServerError, ginResponse{Status: -1, Msg: "获取失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: resp})
}

type locationNode struct {
	Level        int             `json:"level"`
	Location1Id  int             `json:"location_1_id"`
	Location2Id  int             `json:"location_2_id"`
	Location3Id  int             `json:"location_3_id"`
	Location4Id  int             `json:"location_4_id"`
	LocationName string          `json:"location_name"`
	Children     []*locationNode `json:"children"`
}

// 获取传感器位置信息，4级查找树
func (c *Controller) getLocations(ctx *gin.Context) {
	projectId, err := strconv.Atoi(ctx.Query("project_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "请求参数错误", Data: nil})
		return
	}
	var _locations []model.SiteLocationName
	var response []*locationNode
	queryString := "select * from site_location_name where project_id=?"
	if err := db.MysqlClient.DB.Raw(queryString, projectId).Find(&_locations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "请求成功", Data: response})
		} else {
			ctx.JSON(http.StatusInternalServerError, ginResponse{Status: -1, Msg: err.Error(), Data: nil})
		}
		return
	}

	// data transform

	var nodes []*locationNode
	for _, l := range _locations {
		_node := &locationNode{
			Location1Id:  l.Location1Id,
			Location2Id:  l.Location2Id,
			Location3Id:  l.Location3Id,
			Location4Id:  l.Location4Id,
			LocationName: l.LocationName,
		}
		if l.Location2Id == 0 {
			_node.Level = 1
			nodes = append(nodes, _node)
		} else if l.Location3Id == 0 {
			_node.Level = 2
			nodes = append(nodes, _node)
		} else if l.Location4Id == 0 {
			_node.Level = 3
			nodes = append(nodes, _node)
		} else {
			_node.Level = 4
			nodes = append(nodes, _node)
		}
	}

	response = buildIndexTree(nodes)
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: response})
}

// 根据 location 列表 生成4级 location 树
func buildIndexTree(array []*locationNode) []*locationNode {
	maxLen := len(array)
	var isVisit = make([]bool, maxLen)

	var root []*locationNode
	for i := 0; i < maxLen; i++ {
		if array[i].Level == 1 {
			root = append(root, array[i])
			continue
		} else if array[i].Level == 2 {
			for j := 0; j < maxLen; j++ {
				if array[j].Location1Id == array[i].Location1Id && array[j].Level == 1 {
					array[j].Children = append(array[j].Children, array[i])
					isVisit[i] = true
				}
			}
		} else if array[i].Level == 3 {
			for j := 0; j < maxLen; j++ {
				if array[j].Location1Id == array[i].Location1Id && array[j].Location2Id == array[i].Location2Id && array[j].Level == 2 {
					array[j].Children = append(array[j].Children, array[i])
					isVisit[i] = true
				}
			}
		} else {
			for j := 0; j < maxLen; j++ {
				if array[j].Location1Id == array[i].Location1Id && array[j].Location2Id == array[i].Location2Id && array[j].Location3Id == array[i].Location3Id && array[j].Level == 3 {
					array[j].Children = append(array[j].Children, array[i])
					isVisit[i] = true
				}
			}
		}
	}
	return root
}

type measurementResponse struct {
	ReceiveNo      int    `json:"receive_no"`
	GatherType     string `json:"gather_type"`
	GatherTypeName string `json:"gather_type_name"`
	Unit           string `json:"unit"`
}

// 根据type_id获取传感器的采集类型信息
func (c *Controller) getMeasurements(ctx *gin.Context) {
	typeId := ctx.Query("type_id")
	if typeId == "" {
		ctx.JSON(http.StatusBadRequest, ginResponse{Status: -1, Msg: "请求参数错误", Data: nil})
		return
	}
	var _measurements []measurementResponse

	queryString := "select receive_no, gather_type, gather_type_name, unit from sensor_gather_type " +
		"where sensor_type_id=?"
	if err := db.MysqlClient.DB.Raw(queryString, typeId).Find(&_measurements).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusInternalServerError, ginResponse{Status: -1, Msg: err.Error(), Data: nil})
			return
		}
	}
	ctx.JSON(http.StatusOK, ginResponse{Status: 0, Msg: "获取成功", Data: _measurements})
}
