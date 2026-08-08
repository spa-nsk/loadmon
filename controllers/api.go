package controllers

import (
	"encoding/json"
	"io"
	"loadmon/grpcclient"
	models "loadmon/models"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type Response_1 struct {
	Id       int64  `json:"Id"`
	Name     string `json:"Name"`
	NameFQDN string `json:"NameFQDN"`
	Owner    string `json:"Owner"`
}

type Request_1 struct {
	NameFQDN string `json:"NameFQDN"`
}

func (this *MainController) LoginAPI() {
	m := make(map[string]interface{})
	m["first"] = ""
	m["last"] = ""
	m["username"] = ""
	m["user_id"] = int64(0)
	m["privilage"] = 0        // 0 - обычный пользователь 1 - разрешено редактировать справочник
	m["podrazdelenie"] = 0    // 0 - не указано 1 - ДАТИ 2 - СТО
	m["timestamp"] = time.Now()
	m["SprSearchString"] = ""  //строка фильтра справочника
	m["SprSearchField"] = "1" //тип строки для фильтра справочника

	to := time.Now()
	to = CorrectTimeFilter(to)
	from := to.AddDate(0, 0, -7)
	from = CorrectTimeFilter(from)

	f := models.LoadFilter{FromTime: from, ToTime: to, Map: m}
	this.SetSession("loadmon", f)
}

func (this *MainController) RequestAPI() {
	var err error
	/*
		sess := this.GetSession("loadmon")
		if sess == nil {
			this.LoginAPI()
			sess = this.GetSession("loadmon")
		}
	*/
	//this.activeContent("api") // api.tpl отсутствует
	rec := models.LoadmonRequest{Data: Request_1{}}

	json_rec := this.Ctx.Input.Param(":json")
	if this.Ctx.Input.IsGet() {
		err = json.Unmarshal([]byte(json_rec), &rec)
	} else if this.Ctx.Input.IsPost() && json_rec == "post" {
		body, _ := io.ReadAll(this.Ctx.Request.Body)
		err = json.Unmarshal(body, &rec)
	}
	if err != nil {
		this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Count: 0, Status: 3, Data: json_rec + " - " + err.Error()}, false, false) //недопустимый код запроса
		return
	}
	if rec.Code == 1 {

		pRec, ok := rec.Data.(map[string]interface{})
		if ok {
			o := orm.NewOrm()
			//o.Using("default")
			cond := orm.NewCondition()
			cond = cond.And("stat__ne", 666)
			nameFQDN := strings.TrimSpace(pRec["NameFQDN"].(string))
			if nameFQDN != "*" {
				if nameFQDN != "" {
					condName := orm.NewCondition()
					names := strings.Split(nameFQDN, ";")
					for m := range names {
						s := strings.TrimSpace(names[m])
						if s != "" {
							condName = condName.Or("name_f_q_d_n__icontains", s)
						}
					}
					cond = cond.AndCond(condName)
				}
			}
			var records []*models.ServerSprRecords
			qs := o.QueryTable("server_spr_records")
			qs = qs.OrderBy("active", "name_f_q_d_n")

			qs = qs.SetCond(cond)

			count, err := qs.All(&records)
			if err == nil {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 1, Count: (int)(count), Data: records}, false, false)
			} else {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 0, Count: (int)(count), Data: records}, false, false)
			}
		} else {
			this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Count: 0, Status: 2, Data: nil}, false, false)
		}
		return
	}
	if rec.Code == 2 { //запрос из базы по серийному номеру
		pRec, ok := rec.Data.(map[string]interface{})
		if ok {
			o := orm.NewOrm()
			//o.Using("default")
			cond := orm.NewCondition()
			SerialNum := strings.TrimSpace(pRec["SerialNum"].(string))

			if SerialNum != "" {
				names := strings.Split(SerialNum, ";")
				for n := range names {
					s := strings.TrimSpace(names[n])
					if s != "" {
						cond = cond.Or("serial_num", s)
					}
				}
			}

			var records []*models.ServerHardware
			qs := o.QueryTable("server_hardware")
			qs = qs.OrderBy("serial_num")

			if SerialNum != "*" {
				qs = qs.SetCond(cond)
			}

			count, err := qs.All(&records)
			if err == nil {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 1, Count: (int)(count), Data: records}, false, false)
			} else {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 0, Count: (int)(count), Data: records}, false, false)
			}
		} else {
			this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Count: 0, Status: 2, Data: nil}, false, false)
		}
		return
	}
	if rec.Code == 3 { //запрос оперативный к HP OneView
		pRec, ok := rec.Data.(map[string]interface{})
		if ok {
			pRecord, ok := grpcclient.GetServerHardwareBySN(pRec["SerialNum"].(string))
			/*
				o := orm.NewOrm()
				//o.Using("default")
				cond := orm.NewCondition()
				SerialNum := strings.TrimSpace(pRec["SerialNum"].(string))

				if SerialNum != "" {
					names := strings.Split(SerialNum, ";")
					for m := range names {
						s := strings.TrimSpace(names[m])
						if s != "" {
							cond = cond.Or("serial_num", s)
						}
					}
				}

				var records []*models.ServerHardware
				qs := o.QueryTable("server_hardware")
				qs = qs.OrderBy("serial_num")

				if SerialNum != "" {
					qs = qs.SetCond(cond)
				}

				count, err := qs.All(&records)
			*/
			if ok {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 1, Count: 1, Data: pRecord}, false, false)
			} else {
				this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Status: 0, Count: 1, Data: pRecord}, false, false)
			}
		} else {
			this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Count: 0, Status: 2, Data: nil}, false, false)
		}
		return
	}
	this.Ctx.Output.JSON(models.LoadmonResponse{Code: rec.Code, Count: 0, Status: 3, Data: json_rec}, false, false) //недопустимый код запроса
	//this.ServeJSON()
}
