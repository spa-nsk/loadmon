// Package models содержит ORM-модели (записи БД) веб-фронтенда beego для
// проекта loadmon: сведения о серверах, виртуальных машинах, кластерах,
// хостах, СХД, оборудовании (CMDB), пользователях и метриках производительности.
// Структуры пакета регистрируются в config.DBInit и используются контроллерами
// для чтения/записи данных через beego ORM.
package models

import (
	"errors"
	"strconv"
	"time"
)

var (
	// Objects — демонстрационное in-memory хранилище объектов (пример из шаблона beego).
	Objects map[string]*Object
)

// Object — демонстрационная структура объекта (пример из шаблона beego), не связана с БД.
type Object struct {
	ObjectId   string
	Score      int64
	PlayerName string
}

func init() {
	Objects = make(map[string]*Object)
	Objects["hjkhsbnmn123"] = &Object{"hjkhsbnmn123", 100, "astaxie"}
	Objects["mjjkxsxsaa23"] = &Object{"mjjkxsxsaa23", 101, "someone"}
}

// AddOne добавляет объект в хранилище Objects и возвращает сгенерированный ObjectId.
func AddOne(object Object) (ObjectId string) {
	object.ObjectId = "astaxie" + strconv.FormatInt(time.Now().UnixNano(), 10)
	Objects[object.ObjectId] = &object
	return object.ObjectId
}

// GetOne возвращает объект из хранилища Objects по идентификатору.
func GetOne(ObjectId string) (object *Object, err error) {
	if v, ok := Objects[ObjectId]; ok {
		return v, nil
	}
	return nil, errors.New("ObjectId Not Exist")
}

// GetAll возвращает все объекты хранилища Objects.
func GetAll() map[string]*Object {
	return Objects
}

// Update изменяет счёт (Score) объекта по идентификатору.
func Update(ObjectId string, Score int64) (err error) {
	if v, ok := Objects[ObjectId]; ok {
		v.Score = Score
		return nil
	}
	return errors.New("ObjectId Not Exist")
}

// Delete удаляет объект из хранилища Objects по идентификатору.
func Delete(ObjectId string) {
	delete(Objects, ObjectId)
}
