package service

import (
	"anomaly-detect/cmd/controller/db"
	"anomaly-detect/cmd/controller/model"
	"container/list"
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ModelMeta struct {
	Url  string
	Type ModelType
}

type orderedModelElement struct {
	Key   string    // name
	Value ModelMeta // url
}

type orderedModelMap struct {
	kv map[string]*list.Element
	ll *list.List
}

func newOrderedModelMap() *orderedModelMap {
	return &orderedModelMap{
		kv: make(map[string]*list.Element),
		ll: list.New(),
	}
}

func (m *orderedModelMap) Get(key string) (ModelMeta, bool) {
	v, ok := m.kv[key]
	if !ok {
		return ModelMeta{}, false
	}
	return v.Value.(*orderedModelElement).Value, true
}

func (m *orderedModelMap) Set(name, url string, t ModelType) {
	_, exist := m.kv[name]
	if exist {
		m.kv[name].Value = ModelMeta{
			Url:  url,
			Type: t,
		}
	} else {
		element := m.ll.PushBack(&orderedModelElement{
			Key: name,
			Value: ModelMeta{
				Url:  url,
				Type: t,
			},
		})
		m.kv[name] = element
	}
}

func (m *orderedModelMap) Del(key string) {
	element, ok := m.kv[key]
	if ok {
		m.ll.Remove(element)
		delete(m.kv, key)
	}
}

func (m *orderedModelMap) Keys() []string {
	keys := make([]string, m.ll.Len())
	element := m.ll.Front()
	for i := 0; element != nil; i++ {
		logrus.Info(element.Value)
		keys[i] = element.Value.(*orderedModelElement).Key
		element = element.Next()
	}
	return keys
}

var Model *orderedModelMap

func init() {
	Model = newOrderedModelMap()
}

// Load 从数据库中载入模型
func Load() {
	var models []model.InvokeService
	if err := db.MysqlClient.DB.Model(model.InvokeService{}).Find(&models).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logrus.Errorf("load model failed: %s", err.Error())
		}
		return
	}
	for _, m := range models {
		Model.Set(m.Name, m.Url, ModelType(m.Type))
		logrus.Infof("load detect model: %s url:%s", m.Name, m.Url)
	}
}

const queryString = "name=? and type=?"

func Save(name, url string, t int) error {
	var oldModel model.InvokeService
	if err := db.MysqlClient.DB.Where(queryString, name, t).First(&oldModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record := model.InvokeService{
				Name: name,
				Type: t,
				Url:  url,
			}
			if err := db.MysqlClient.DB.Create(&record).Error; err != nil {
				return err
			}
		}
	} else {
		oldModel.Url = url
		if err := db.MysqlClient.DB.Save(&oldModel).Error; err != nil {
			return err
		}
	}
	return nil
}

func Delete(name string) error {
	return db.MysqlClient.DB.Where("name=?", name).Delete(model.InvokeService{}).Error
}
