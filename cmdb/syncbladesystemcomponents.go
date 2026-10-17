package cmdb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// SyncBladeSystemsFromOneView опрашивает /rest/enclosures на каждой
// зарегистрированной точке подключения HP OneView (ov.Endpoints) и
// актуализирует справочник блейд-корзин (models.BladeSystems) вместе с
// компонентами корзины (interconnect/management/питание/охлаждение),
// которые хранятся в той же таблице server_hardware_components, что и
// компоненты серверов (models.ServerHardwareComponents.BladeSystemId,
// Type=6/7/12/13 - см. cmdb/syncserverhardwarecomponents.go).
//
// Это РЕАЛИЗАЦИЯ ранее отключённой функции UpdateHwSprHPBladeSystem (была
// no-op стабом после миграции на oneview2 - см. историю этого файла и
// kontext/05-conventions-and-gotchas.md): в oneview2.Enclosure (в отличие
// от старого пакета loadmon/oneview) типизированы только сами отсеки
// устройств (DeviceBays []EnclosureDeviceBay) - остальные списки
// (InterconnectBays/ManagerBays/PowerSupplyBays/FanBays) SDK отдаёт как
// []map[string]any (сырой JSON), без готовых полей. Эта функция разбирает
// их вручную по общеупотребимым ключам HPE OneView REST API
// (bayNumber/model/partNumber/serialNumber/status/state/...) - см.
// bayString/bayInt ниже.
//
// ⚠️ Поля сырых bay-словарей (interconnect/manager/power/fan) взяты по
// документации HPE OneView REST API (устойчивый паттерн именования по всем
// bay-ресурсам этого API), а НЕ проверены против реального ответа
// appliance в этой песочнице (нет доступа к живому HP OneView) - если после
// включения окажется, что часть колонок остаётся пустой, нужно свериться с
// фактическим JSON (например, залогировав один "сырой" bay через
// fmt.Printf("%#v", bay) и сверив реальные ключи).
func SyncBladeSystemsFromOneView(ov *oneview.Infrastructure) {
	if ov == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Фаза 1: собрать корзины со ВСЕХ точек подключения вместе с мощностью,
	// доступной на каждой (локальный /utilization с refresh или поля объекта GD).
	// Фаза 2: объединить по корзине и записать один раз - приоритет источников
	// не зависит от порядка опроса endpoint (раньше побеждал последний).
	var cands []encCandidate
	for _, cfg := range ov.EndpointList() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log_BladeSystemsPanic(cfg.Endpoint, r)
				}
			}()
			cands = append(cands, collectEnclosuresFromEndpoint(ctx, cfg)...)
		}()
	}
	syncEnclosureCandidates(cands)
}

func log_BladeSystemsPanic(endpoint string, r any) {
	fmt.Println("SyncBladeSystemsFromOneView: ПАНИКА при опросе", endpoint, ":", r)
}

// encCandidate - корзина, полученная с одной точки подключения, и мощность,
// которую удалось получить на ней.
type encCandidate struct {
	Endpoint string
	GD       bool // Global Dashboard (агрегатор): мощность - только поля объекта
	Enc      oneview.Enclosure
	Power    oneview.PowerWatts
}

func collectEnclosuresFromEndpoint(ctx context.Context, cfg oneview.Config) []encCandidate {
	c, err := oneview.New(cfg)
	if err != nil {
		fmt.Println("SyncBladeSystemsFromOneView: ошибка подключения к", cfg.Endpoint, err)
		return nil
	}
	if err := c.Login(ctx); err != nil {
		fmt.Println("SyncBladeSystemsFromOneView: ошибка авторизации на", cfg.Endpoint, err)
		return nil
	}
	defer c.Logout(ctx)

	col, err := c.ListEnclosures(ctx, oneview.ListOptions{Count: -1})
	if err != nil || col == nil {
		fmt.Println("SyncBladeSystemsFromOneView: ошибка чтения /rest/enclosures на", cfg.Endpoint, err)
		return nil
	}

	gd := c.IsGlobalDashboard()
	out := make([]encCandidate, 0, len(col.Members))
	for i := range col.Members {
		enc := col.Members[i]
		// Локальный appliance: /utilization (+refresh, calibratedMaxPower);
		// GD: только свежие поля объекта. Пустой результат = данных нет.
		p := c.EnclosurePower(ctx, &enc)
		out = append(out, encCandidate{Endpoint: cfg.Endpoint, GD: gd, Enc: enc, Power: p})
	}
	return out
}

// encKey - ключ объединения корзины между точками подключения: серийный номер,
// при его отсутствии нормализованное имя.
func encKey(enc *oneview.Enclosure) string {
	if sn := strings.ToUpper(strings.TrimSpace(enc.SerialNumber)); sn != "" {
		return "SN:" + sn
	}
	return "NAME:" + strings.ToUpper(bladeSystemNameFromEnclosure(strings.TrimSpace(enc.Name)))
}

// syncEnclosureCandidates объединяет корзины по ключу и записывает каждую один раз.
// Описание корзины (модель, отсеки, компоненты) берётся с локального appliance,
// если корзина видна и на нём, иначе из Global Dashboard. Мощность - по
// приоритету oneview.MergeEnclosurePower (измерение > свежие поля GD),
// а при отсутствии измерения - оценка по блейдам (ApplyBladeSystemPowerVariantC).
func syncEnclosureCandidates(cands []encCandidate) {
	if len(cands) == 0 {
		return
	}
	order := make([]string, 0, len(cands))
	groups := make(map[string][]encCandidate, len(cands))
	for _, cd := range cands {
		if strings.TrimSpace(cd.Enc.Name) == "" {
			continue
		}
		k := encKey(&cd.Enc)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], cd)
	}

	o := orm.NewOrm()
	for _, k := range order {
		inv, power := pickEnclosureSources(groups[k])
		enc := inv.Enc
		fmt.Println("SyncBladeSystemsFromOneView:", enc.Name, "("+enc.SerialNumber+")", "описание:", inv.Endpoint,
			"| мощность: источник="+fmt.Sprintf("%q", power.Source),
			"avg="+FormatPowerW(power.Avg24hW), "peak="+FormatPowerW(power.Peak24hW),
			"номинал="+FormatPowerW(power.RatedMaxW)+" ("+power.RatedSource+")")
		syncOneEnclosure(o, &enc, power)
	}
}

// pickEnclosureSources выбирает для одной корзины (кандидаты со всех endpoint):
// источник описания (предпочтительно локальный appliance, не Global Dashboard) и
// итоговую мощность по приоритету oneview.MergeEnclosurePower: измерение
// /utilization > свежие поля объекта GD; номинал calibratedMaxPower >
// powerCapacity24Hrs. Результат без измерения (Source == "") означает, что
// в БД нужно использовать оценку по блейдам (ApplyBladeSystemPowerVariantC).
func pickEnclosureSources(g []encCandidate) (inv encCandidate, power oneview.PowerWatts) {
	if len(g) == 0 {
		return encCandidate{}, oneview.PowerWatts{}
	}
	inv = g[0]
	powers := make([]oneview.PowerWatts, 0, len(g))
	for _, cd := range g {
		if inv.GD && !cd.GD {
			inv = cd
		}
		powers = append(powers, cd.Power)
	}
	return inv, oneview.MergeEnclosurePower(powers...)
}

// bladeSystemNameFromEnclosure нормализует имя корзины ("c7000-1" ->
// "c7000-01") - тот же приём выравнивания номера, что уже применяется в
// cmdb/initrec.go:ImportHWSprRecord при определении корзины по одиночному
// серверу; вынесен сюда как отдельная функция, чтобы оба места гарантированно
// использовали ОДИНАКОВУЮ нормализацию (иначе один и тот же физический
// шкаф мог бы завестись в справочнике дважды под разными именами).
func bladeSystemNameFromEnclosure(name string) string {
	s := strings.Split(name, "-")
	if len(s) == 2 {
		if num, err := strconv.Atoi(s[1]); err == nil {
			return fmt.Sprintf("%s-%02d", s[0], num)
		}
	}
	return name
}

// syncOneEnclosure записывает одну корзину; power - итог объединения источников
// мощности (может быть пустым: тогда используется оценка по блейдам).
func syncOneEnclosure(o orm.QueryExecutor, enc *oneview.Enclosure, power oneview.PowerWatts) {
	if enc == nil || strings.TrimSpace(enc.Name) == "" {
		return
	}
	name := bladeSystemNameFromEnclosure(enc.Name)

	bls := models.BladeSystems{Name: name}
	found := o.Read(&bls, "Name") == nil
	// prev - снимок ДО перезаписи полей ниже, нужен, чтобы после сохранения
	// сравнить старое/новое значение каждого значимого поля и залогировать
	// изменение в hardware_event_log (см. cmdb/hardwareeventlog.go,
	// 2026-09-04, "фиксировать изменение конфигурации... в отдельном логе").
	// Для НОВОЙ корзины (found==false) prev - нулевые значения, поэтому все
	// поля будут выглядеть как "изменившиеся с пустого" - это ожидаемо и
	// сознательно НЕ подавляется отдельным HardwareEventComponentAdded,
	// чтобы не дублировать разные типы событий ради одного факта создания
	// записи справочника.
	prev := bls

	bls.Name = name
	bls.Vendor = "HPE"
	bls.Model = enc.EnclosureModel
	bls.Type = enc.EnclosureType
	bls.SerialNum = enc.SerialNumber
	bls.Size = enc.DeviceBayCount.Int()
	occupied := 0
	for _, bay := range enc.DeviceBays {
		if strings.EqualFold(bay.DevicePresence, "Present") {
			occupied++
		}
	}
	bls.Used = occupied
	bls.Time_Mfy = time.Now().Local()
	if bls.Anotation == "" {
		bls.Anotation = enc.SerialNumber + " Данные добавлены из HP OneView"
	}

	ApplyBladeSystemPowerVariantC(o, &bls, power)

	if found {
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "PowerAvg24hW", FormatPowerW(prev.PowerAvg24hW), FormatPowerW(bls.PowerAvg24hW))
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "PowerPeak24hW", FormatPowerW(prev.PowerPeak24hW), FormatPowerW(bls.PowerPeak24hW))
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "PowerRatedMaxW", FormatPowerW(prev.PowerRatedMaxW), FormatPowerW(bls.PowerRatedMaxW))
	}

	if found {
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "Model", prev.Model, bls.Model)
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "Type", prev.Type, bls.Type)
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "SerialNum", prev.SerialNum, bls.SerialNum)
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "Size", strconv.Itoa(prev.Size), strconv.Itoa(bls.Size))
		logHardwareFieldChange(o, "BladeSystem", 0, prev.Id, bls.SerialNum, "Used", strconv.Itoa(prev.Used), strconv.Itoa(bls.Used))
	}

	var err error
	if found {
		_, err = o.Update(&bls)
	} else {
		_, err = o.Insert(&bls)
	}
	if err != nil {
		fmt.Println("SyncBladeSystemsFromOneView: ошибка сохранения корзины", name, err)
		return
	}
	if bls.Id <= 0 {
		return
	}

	syncBladeSystemComponents(o, bls.Id, enc)
}

// syncBladeSystemComponents полностью пересобирает компоненты ОДНОЙ корзины
// (interconnect/management/питание/охлаждение) - тот же приём "удалить всё
// по (BladeSystemId, Type), записать заново", что и
// cmdb/syncserverhardwarecomponents.go для компонентов сервера.
func syncBladeSystemComponents(o orm.QueryExecutor, bladeSystemId int64, enc *oneview.Enclosure) {
	// syncType - тот же приём diff+delete+insert, что и
	// cmdb/syncserverhardwarecomponents.go:SyncServerHardwareComponents
	// (2026-09-04) - перед перезаписью читает предыдущий набор строк этого
	// типа и логирует разницу в hardware_event_log.
	syncType := func(t int, rows []models.ServerHardwareComponents) {
		var old []models.ServerHardwareComponents
		if _, err := o.QueryTable("server_hardware_components").Filter("BladeSystemId", bladeSystemId).Filter("Type", t).All(&old); err != nil {
			fmt.Println("syncBladeSystemComponents: ошибка чтения предыдущих записей типа", t, "корзина Id=", bladeSystemId, err)
		}
		diffAndLogComponents(o, "BladeSystem", 0, bladeSystemId, enc.SerialNumber, t, old, rows)

		m := models.ServerHardwareComponents{BladeSystemId: bladeSystemId, Type: t}
		if _, err := o.Delete(&m, "BladeSystemId", "Type"); err != nil {
			fmt.Println("syncBladeSystemComponents: ошибка удаления старых записей типа", t, "корзина Id=", bladeSystemId, err)
		}
		for _, c := range rows {
			c.BladeSystemId = bladeSystemId
			c.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&c); err != nil {
				fmt.Println("syncBladeSystemComponents: ошибка записи компонента типа", c.Type, "корзина Id=", bladeSystemId, err)
			}
		}
	}

	var interconnect []models.ServerHardwareComponents
	for idx, bay := range enc.InterconnectBays {
		interconnect = append(interconnect, models.ServerHardwareComponents{
			Type:      ServerHWComponentInterconnect,
			Vendor:    "HPE",
			Model:     bayString(bay, "model", "partNumber"),
			SerialNum: firstNonEmpty(bayString(bay, "serialNumber"), "bay-ic-"+strconv.Itoa(idx)),
			Slot:      fmt.Sprintf("Bay %d, %s", bayInt(bay, "bayNumber", idx+1), bayString(bay, "status", "state")),
		})
	}
	syncType(ServerHWComponentInterconnect, interconnect)

	var management []models.ServerHardwareComponents
	for idx, bay := range enc.ManagerBays {
		management = append(management, models.ServerHardwareComponents{
			Type:      ServerHWComponentManagement,
			Vendor:    "HPE",
			Model:     bayString(bay, "model", "role"),
			SerialNum: firstNonEmpty(bayString(bay, "serialNumber"), "bay-mgmt-"+strconv.Itoa(idx)),
			Slot:      fmt.Sprintf("Bay %d, IP %s", bayInt(bay, "bayNumber", idx+1), bayString(bay, "mgmtIpAddress")),
		})
	}
	syncType(ServerHWComponentManagement, management)

	var power []models.ServerHardwareComponents
	for idx, bay := range enc.PowerSupplyBays {
		power = append(power, models.ServerHardwareComponents{
			Type:      ServerHWComponentPowerSupply,
			Vendor:    "HPE",
			Model:     bayString(bay, "model", "partNumber"),
			SerialNum: firstNonEmpty(bayString(bay, "serialNumber"), "bay-power-"+strconv.Itoa(idx)),
			Slot:      fmt.Sprintf("Bay %d, %s", bayInt(bay, "bayNumber", idx+1), bayString(bay, "status", "state")),
		})
	}
	syncType(ServerHWComponentPowerSupply, power)

	var fans []models.ServerHardwareComponents
	for idx, bay := range enc.FanBays {
		fans = append(fans, models.ServerHardwareComponents{
			Type:      ServerHWComponentFan,
			Vendor:    "HPE",
			Model:     bayString(bay, "model"),
			SerialNum: "bay-fan-" + strconv.Itoa(idx),
			Slot:      fmt.Sprintf("Bay %d, %s", bayInt(bay, "bayNumber", idx+1), bayString(bay, "status", "state")),
		})
	}
	syncType(ServerHWComponentFan, fans)
}

// bayString читает первый непустой строковый ключ из сырого JSON-объекта
// отсека (interconnect/manager/power/fan bay) - см. предупреждение о
// неподтверждённых именах ключей в комментарии к SyncBladeSystemsFromOneView.
func bayString(bay map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := bay[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

// bayInt читает числовой ключ (JSON-числа декодируются в float64) из сырого
// bay-объекта, с запасным значением по умолчанию.
func bayInt(bay map[string]any, key string, def int) int {
	if v, ok := bay[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}
