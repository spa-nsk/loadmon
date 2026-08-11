package controllers

import (
	"fmt"
	models "loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

/*
type UserLog struct {
	Id      int64     `form:"id:"`                // id записи
	UserId  int       `orm:"null" form:"userid"`  // id пользователя
	Login   time.Time `orm:"null" form:"login"`   // время регистрации
	Time    time.Time `orm:"null" form:"last"`    // время операции
	Type    int       `form:"type"`               // тип операции (1 авторизация, 2 выход, 3 сброс пароля, 4 обновление профиля, 5 чтение справочника, 6 изменение данных справочника, 7 чтение таблицы...
	Request string    `form:"request"`            // текст запроса
}
*/

// type тип операции (1 авторизация, 2 выход, 3 сброс пароля, 4 обновление профиля, 5 чтение справочника, 6 изменение данных справочника, 7 чтение таблицы загрузки, 8 получение деталей по серверу, 9 выгрузка в...

// RegisterRequest записывает в журнал (таблица user_log) факт обращения
// пользователя к приложению с указанным типом операции type_request
// (авторизация, выход, чтение/изменение справочника и т.д.).
func RegisterRequest(this *MainController, type_request int) error {

	sess := this.GetSession("loadmon")
	if sess == nil {
		return fmt.Errorf("Нет активного сеанса. Требуется авторизация.")
	}

	rec := models.UserLog{}
	filter := sess.(models.LoadFilter)
	m := filter.Map

	rec.UserId = m["user_id"].(int64)      //Id пользователя
	rec.Login = m["timestamp"].(time.Time) //время регистрации
	rec.Time = time.Now()                  //время запроса
	rec.Type = type_request                //тип запроса
	rec.Request = this.Ctx.Input.Method() + "-" + this.Ctx.Input.URL()

	o := orm.NewOrm()
	//o.Using("default")

	_, err := o.Insert(&rec)
	if err != nil {
		return err
	}
	this.Data["First"] = m["first"]       //отобразить логин
	this.Data["Last"] = m["last"]         //отобразить логин
	this.Data["Username"] = m["username"] //отобразить логин

	//log.Println(this.Data["First"].(string), " ", this.Data["Last"].(string), " ", this.Data["Username"].(string), " | ", rec.Request)
	return nil
}

// ViewUserLog отображает журнал действий пользователей.
func (this *MainController) ViewUserLog() {

}
