package cmdb

import (
	"fmt"
	"strconv"
	"strings"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// Типы записей models.ServerHardwareComponents (поле Type) - см. также
// комментарий к самому полю в models/models_add.go. Значения 1/2/5 были
// заведены раньше (транскрипция со скриншотов, см. kontext) и уже
// заполняются (RAM/диски/RAID-контроллер); 3/4/6/7 были ЗАДОКУМЕНТИРОВАНЫ
// изначально, но никогда не заполнялись кодом - сетевые/FC-контроллеры
// сервера (3/4) заполняются теперь этой функцией, 6/7 (interconnect/
// management card блейд-корзины) - см. cmdb/syncbladesystemcomponents.go.
// 9/10/11 - новые типы, добавленные вместе с этой функцией (2026-09-04) по
// запросу "полноценная обработка сбора конфигураций серверов из oneview2 с
// сохранением данных по всем компонентам" - раньше срv.Devices/
// srv.NetworkPorts/srv.Firmware/сокеты процессоров кроме первого от
// oneview2.Server читались (или не читались вовсе), но НИКУДА не
// сохранялись, то есть эти данные были доступны в API и просто терялись.
const (
	ServerHWComponentRAM         = 1  // модуль оперативной памяти
	ServerHWComponentDisk        = 2  // физический диск
	ServerHWComponentNetwork     = 3  // сетевой порт/контроллер (Ethernet)
	ServerHWComponentFC          = 4  // порт/контроллер Fibre Channel
	ServerHWComponentRAID        = 5  // RAID/storage-контроллер
	ServerHWComponentInterconnect = 6 // interconnect-модуль блейд-корзины
	ServerHWComponentManagement  = 7  // management/OA-модуль блейд-корзины
	ServerHWComponentPCIDevice   = 9  // прочее PCI-устройство (не диск/сеть/RAID)
	ServerHWComponentFirmware    = 10 // компонент прошивки (BIOS/iLO/адаптер/...)
	ServerHWComponentProcessor   = 11 // процессор (сокет)
	ServerHWComponentPowerSupply = 12 // блок питания блейд-корзины
	ServerHWComponentFan         = 13 // вентиляторный модуль блейд-корзины
)

// SyncServerHardwareComponents полностью пересобирает записи
// models.ServerHardwareComponents (таблица server_hardware_components) для
// ОДНОГО сервера (hw.Id) по данным oneview2.Server: память, диски, RAID-
// контроллеры (логика идентична исторически существовавшей, ранее была
// продублирована по коду трижды - в cmdb/insertHWSpr.go:InsertHWSpr,
// cmdb/updatehwsprhp.go:UpdateHwSprHPServers и
// cmdb/updatehwsprhp.go:UpdateHwSprHPServersInfra - вынесена сюда один раз,
// чтобы дальнейшие правки/новые типы компонентов не пришлось синхронизировать
// в трёх местах), а также НОВЫЕ типы компонентов, которые oneview2.Server
// уже предоставляет, но раньше нигде не сохранялись: сетевые/FC-порты
// (srv.NetworkPorts), прочие PCI-устройства (srv.Devices), компоненты
// прошивки (srv.Firmware), все процессоры по всем сокетам (srv.Processors -
// раньше в ServerHardware попадали только агрегированные CPUModel/CorePerCPU/
// CoreMHz по сокету 0, полный список нигде не хранился).
//
// Приём "удалить всё по (ServerId, Type), записать заново" - простой и
// устойчивый к дублям/переименованиям способ актуализации набора
// компонентов на каждом цикле синхронизации (сервер физически не может сам
// сообщить diff, поэтому полная пересборка дешевле и надёжнее точечного
// UPDATE по совпадению серийника слота).
func SyncServerHardwareComponents(o orm.Ormer, hw *models.ServerHardware, srv *oneview.Server) {
	if hw == nil || hw.Id <= 0 || srv == nil {
		return
	}

	fill := func(c models.ServerHardwareComponents) models.ServerHardwareComponents {
		c.ServerId = hw.Id
		c.ContractDoc = hw.ContractDoc
		c.DateContractDoc = hw.DateContractDoc
		c.Invoice = hw.Invoice
		c.DateInvoice = hw.DateInvoice
		c.ServiceContract = hw.ServiceContract
		c.DateService = hw.DateService
		c.Time_Mfy = hw.Time_Mfy
		c.Id_User = hw.Id_User
		return c
	}

	// syncType сравнивает уже сохранённый (в предыдущем цикле опроса) набор
	// строк данного типа с только что построенным newRows, пишет разницу в
	// hardware_event_log (см. cmdb/hardwareeventlog.go:diffAndLogComponents
	// - 2026-09-04, по запросу "фиксировать изменение конфигурации... в
	// отдельном логе"), и только затем выполняет прежний приём "удалить
	// всё по (ServerId, Type), записать заново".
	syncType := func(t int, rows []models.ServerHardwareComponents) {
		var old []models.ServerHardwareComponents
		if _, err := o.QueryTable("server_hardware_components").Filter("ServerId", hw.Id).Filter("Type", t).All(&old); err != nil {
			fmt.Println("SyncServerHardwareComponents: ошибка чтения предыдущих записей типа", t, "сервер", hw.SerialNum, err)
		}
		diffAndLogComponents(o, "Server", hw.Id, 0, hw.SerialNum, t, old, rows)

		m := models.ServerHardwareComponents{ServerId: hw.Id, Type: t}
		if _, err := o.Delete(&m, "ServerId", "Type"); err != nil {
			fmt.Println("SyncServerHardwareComponents: ошибка удаления старых записей типа", t, "сервер", hw.SerialNum, err)
		}
		for _, c := range rows {
			c = fill(c)
			if _, err := o.Insert(&c); err != nil {
				fmt.Println("SyncServerHardwareComponents: ошибка записи компонента типа", c.Type, "сервер", hw.SerialNum, err)
			}
		}
	}

	// --- Память (Type=1) ---
	var ram []models.ServerHardwareComponents
	for idx, mem := range srv.Memory {
		ram = append(ram, models.ServerHardwareComponents{
			Type:       ServerHWComponentRAM,
			Vendor:     mem.Manufacturer,
			Model:      fmt.Sprintf("%.0fGb %s %s %d", float64(mem.CapacityMiB)/1024.0, mem.BaseModuleType, mem.MemoryDeviceType, mem.OperatingSpeedMHz),
			CapacityGb: float64(mem.CapacityMiB) / 1024.0,
			SerialNum:  firstNonEmpty(strings.TrimSpace(mem.SerialNumber), hw.SerialNum+"-ram-"+strconv.Itoa(idx)),
			Slot:       mem.Name,
		})
	}
	syncType(ServerHWComponentRAM, ram)

	// --- Диски и RAID/storage-контроллеры (Type=2/5) ---
	var raid, disks []models.ServerHardwareComponents
	for _, storage := range srv.Controllers {
		raid = append(raid, models.ServerHardwareComponents{
			Type:       ServerHWComponentRAID,
			Vendor:     "HPE",
			Model:      strings.TrimSpace(fmt.Sprintf("%s %s FW:%s Mode:%s", storage.Model, storage.AdapterType, storage.FirmwareVersion, storage.CurrentOperatingMode)),
			CapacityGb: float64(storage.CacheMemorySizeMiB) / 1024.0,
			SerialNum:  strings.TrimSpace(storage.SerialNumber),
			Slot:       storage.Location, // формат строки расположения "ControllerPort:Box:Bay"
		})
		for _, disk := range storage.Drives {
			dsk := models.ServerHardwareComponents{
				Type:       ServerHWComponentDisk,
				Vendor:     "HPE",
				Model:      fmt.Sprintf("%.0fGb %s %s (%s) FW:%s", float64(disk.CapacityMiB)/1024.0, disk.MediaType, strings.TrimSpace(disk.Model), strings.TrimSpace(disk.SerialNumber), disk.FirmwareVersion),
				CapacityGb: float64(disk.CapacityMiB) / 1000.0,
				SerialNum:  strings.TrimSpace(disk.SerialNumber),
				Slot:       disk.Location,
			}
			switch disk.MediaType {
			case "HDD":
				dsk.SubType = 2
			case "SSD":
				dsk.SubType = 1
			}
			disks = append(disks, dsk)
		}
	}
	syncType(ServerHWComponentRAID, raid)
	syncType(ServerHWComponentDisk, disks)

	// --- Сетевые/FC-порты (Type=3/4) - НОВОЕ (2026-09-04): srv.NetworkPorts
	// раньше вообще не читался в cmdb, хотя SDK его отдаёт полностью (MAC/
	// WWN/слот/interconnect). Разделяем Ethernet и Fibre Channel по
	// подстроке в NetworkPortExport.Type ("FC"/"Fibre") - это единственный
	// признак типа порта, который есть в структуре.
	var netPorts, fcPorts []models.ServerHardwareComponents
	for idx, port := range srv.NetworkPorts {
		t := ServerHWComponentNetwork
		portType := strings.ToUpper(port.Type)
		if strings.Contains(portType, "FC") || strings.Contains(portType, "FIBRE") {
			t = ServerHWComponentFC
		}
		ident := strings.TrimSpace(port.MAC)
		if ident == "" {
			ident = strings.TrimSpace(port.WWN)
		}
		row := models.ServerHardwareComponents{
			Type:      t,
			Vendor:    port.DeviceName,
			Model:     fmt.Sprintf("%s порт %d, тип %s, MAC=%s WWN=%s", port.DeviceName, port.PortNumber, port.Type, port.MAC, port.WWN),
			SerialNum: firstNonEmpty(ident, hw.SerialNum+"-net-"+strconv.Itoa(idx)),
			Slot:      fmt.Sprintf("Слот %d, порт %d", port.DeviceSlot, port.PortNumber),
		}
		if t == ServerHWComponentFC {
			fcPorts = append(fcPorts, row)
		} else {
			netPorts = append(netPorts, row)
		}
	}
	syncType(ServerHWComponentNetwork, netPorts)
	syncType(ServerHWComponentFC, fcPorts)

	// --- Прочие PCI-устройства (Type=9) - НОВОЕ: srv.Devices (GPU/HBA/
	// доп. контроллеры, которые не попали в Controllers/NetworkPorts).
	var pci []models.ServerHardwareComponents
	for idx, dev := range srv.Devices {
		pci = append(pci, models.ServerHardwareComponents{
			Type:      ServerHWComponentPCIDevice,
			Vendor:    dev.Manufacturer,
			Model:     fmt.Sprintf("%s (%s) FW:%s", dev.Name, dev.DeviceType, dev.FirmwareVersion),
			SerialNum: firstNonEmpty(strings.TrimSpace(dev.SerialNumber), hw.SerialNum+"-pci-"+strconv.Itoa(idx)),
			Slot:      dev.Location,
		})
	}
	syncType(ServerHWComponentPCIDevice, pci)

	// --- Компоненты прошивки (Type=10) - НОВОЕ: srv.Firmware (версии BIOS/
	// iLO/адаптеров и т.п. - раньше вообще нигде не хранились, поэтому не
	// было истории версий прошивок по серверу).
	var fw []models.ServerHardwareComponents
	for idx, f := range srv.Firmware {
		fw = append(fw, models.ServerHardwareComponents{
			Type:      ServerHWComponentFirmware,
			Vendor:    "HPE",
			Model:     fmt.Sprintf("%s v%s (%s)", f.Name, f.Version, f.State),
			SerialNum: hw.SerialNum + "-fw-" + strconv.Itoa(idx),
			Slot:      f.Location,
		})
	}
	syncType(ServerHWComponentFirmware, fw)

	// --- Процессоры по ВСЕМ сокетам (Type=11) - НОВОЕ: раньше в
	// ServerHardware.CPUModel/CorePerCPU/CoreMHz агрегировался только сокет
	// 0 (см. cmdb/initrec.go:ImportHWSprRecord) - для многопроцессорных
	// серверов данные по остальным сокетам терялись полностью.
	var cpus []models.ServerHardwareComponents
	for _, p := range srv.Processors {
		speed := p.OperatingSpeedMHz
		if speed == 0 {
			speed = p.MaxSpeedMHz
		}
		cpus = append(cpus, models.ServerHardwareComponents{
			Type:      ServerHWComponentProcessor,
			Vendor:    p.Manufacturer,
			Model:     fmt.Sprintf("%s (%d ядер/%d потоков, %d МГц)", p.Model, p.TotalCores, p.TotalThreads, speed),
			SerialNum: hw.SerialNum + "-cpu-" + strconv.Itoa(p.Socket),
			Slot:      fmt.Sprintf("Сокет %d", p.Socket),
		})
	}
	syncType(ServerHWComponentProcessor, cpus)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
