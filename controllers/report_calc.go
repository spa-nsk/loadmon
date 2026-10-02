// Package controllers реализует веб-контроллеры интерфейса пользователя (beego)
// системы мониторинга loadmon: отображение и фильтрацию списка серверов и
// виртуальных машин, карточки серверов с показателями нагрузки (CPU, память,
// диски) и графиками производительности, работу со справочником серверов
// (добавление, обновление данных из систем-источников), аутентификацию и
// профиль пользователя, журналирование действий, а также вспомогательный API
// для внешних систем.
package controllers

import (
	"fmt"
	"sort"
	"strings"

	//"github.com/astaxie/beego"
	//"github.com/astaxie/beego/orm"
	"github.com/beego/beego/v2/client/orm"

	//	"github.com/astaxie/beego/validation"

	//	exportres "loadmon/export"
	models "loadmon/models"
)

func PatchCoreCPURecords(records []*models.ReportServerRecord) {
	o := orm.NewOrm()

	for ind, rec := range records {
		serverName := strings.Split(strings.ToLower(rec.ServerName), "(")
		name := strings.Split(strings.ToLower(strings.TrimSpace(serverName[0])), ".")
		srv := models.ServerSprRecords{}
		//o.Using("default")
		qs := models.ServerSprQS(o) // без записей Stat=666
		qs = qs.OrderBy("Name").Filter("Name__icontains", name[0])
		if !rec.Virtual {
			if rec.CPUCount <= 4 && rec.CoreCount > 36 {
				records[ind].CoreCount = rec.CoreCount / 2
			}
		}
		err := qs.One(&srv)
		if err == orm.ErrMultiRows {
			fmt.Println("Найдено несколько записей, ожидалась одна: ", rec.ServerName)
			continue
		}
		if err == orm.ErrNoRows {
			fmt.Println("Обнаружен новый хост:", rec.ServerName)
			continue
		}
		if err != nil {
			fmt.Println("Ошибка чтения данных справочника:", rec.ServerName, err)
			continue
		}
		if !rec.Virtual {
			srvHW := models.ServerHardware{ServerId: srv.Id}
			err = o.Read(&srvHW, "ServerId")
			if err == nil {
				if srvHW.CPUCount != 0 {
					records[ind].CPUCount = float64(srvHW.CPUCount)
					if srvHW.CorePerCPU != 0 {
						records[ind].CoreCount = float64(srvHW.CPUCount * srvHW.CorePerCPU)
					}
				}
			}
		} else {
			VM := models.VMSprRecord{Id: srv.VMSprId}

			//					o.Using("vm")
			//o.Using("default")
			err = o.Read(&VM, "Id")
			if err == nil {
				records[ind].ServerName = VM.Vm_HostName
				if int(rec.CPUCount) != int(VM.Vm_NumCPU) || int(rec.CoreCount) != int(VM.Vm_NumCores) {
					fmt.Println(VM.Vm_HostName, int(rec.CPUCount), "=", int(VM.Vm_NumCPU), "|", int(rec.CoreCount), "=", int(VM.Vm_NumCores))
				}
				records[ind].CPUCount = float64(VM.Vm_NumCPU)
				records[ind].CoreCount = float64(VM.Vm_NumCores)
			}
		}
	}
}

// CalcInventarisation формирует текстовую рекомендацию по инвентаризации
// (rec.Inventory) на основе загрузки CPU/памяти и свободного места на дисках.
func CalcInventarisation(rec *models.ReportServerRecord) {

	if rec.UseWorkDisk {

		if rec.CPULoadMax >= 80 || rec.MemUseMax >= 80 {
			rec.Inventory = "недостаточно ресурсов по ряду метрик"
		} else if rec.CPULoadMax < 50 && rec.MemUseMax < 50 && rec.SysDiskFreePercentMin > 50 && rec.WorkDiskFreePercentMin > 50 {
			rec.Inventory = "возможно, требуется перераспределить ресурсы"
		} else {
			rec.Inventory = "нет рекомендаций по оптимизации"
		}
	} else {
		if rec.CPULoadMax >= 80 || rec.MemUseMax >= 80 {
			rec.Inventory = "недостаточно ресурсов по ряду метрик"
		} else if rec.CPULoadMax < 50 && rec.MemUseMax < 50 && rec.SysDiskFreePercentMin > 50 {
			rec.Inventory = "возможно, требуется перераспределить ресурсы"
		} else {
			rec.Inventory = "нет рекомендаций по оптимизации"
		}
	}

}

// lessFunc — функция сравнения двух записей отчёта, используемая при сортировке.
type lessFunc func(p1, p2 *models.ReportServerRecord) bool

// MaxResource реализует сортировку записей отчёта по нескольким критериям
// (реализует sort.Interface с цепочкой функций сравнения less).
type MaxResource struct {
	resource []*models.ReportServerRecord
	less     []lessFunc
}

// Sort сортирует переданный срез resource согласно установленным критериям сравнения.
func (mr *MaxResource) Sort(resource []*models.ReportServerRecord) {
	mr.resource = resource
	sort.Sort(mr)
}

// OrderedBy создаёt MaxResource с заданной последовательностью функций сравнения less.
func OrderedBy(less ...lessFunc) *MaxResource {
	return &MaxResource{
		less: less,
	}
}

// Len возвращает количество элементов сортируемого среза (для sort.Interface).
func (mr *MaxResource) Len() int {
	return len(mr.resource)
}

// Swap меняет местами элементы i и j сортируемого среза (для sort.Interface).
func (mr *MaxResource) Swap(i, j int) {
	mr.resource[i], mr.resource[j] = mr.resource[j], mr.resource[i]
}

// Less сравнивает элементы i и j по цепочке критериев (для sort.Interface).
func (mr *MaxResource) Less(i, j int) bool {
	p, q := mr.resource[i], mr.resource[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(mr.less)-1; k++ {
		less := mr.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return mr.less[k](p, q)
}

// CalcRecords вычисляет максимальные значения показателей нагрузки для каждого
// сервера из records и возвращает агрегированный (отсортированный) список записей.
func CalcRecords(records []*models.ReportServerRecord) []*models.ReportServerRecord {
	var out []*models.ReportServerRecord
	var counts []float64
	out = make([]*models.ReportServerRecord, 0)

	for _, rec := range records {
		if rec.CoreCount < 1.0 && rec.CPUCount > 0.0 {
			rec.CoreCount = rec.CPUCount
		}
		found := false
		for j, s := range out {
			if s.ServerName == rec.ServerName {
				//					s.CPUCount = s.CPUCount + rec.CPUCount
				//					s.CoreCount = s.CoreCount + rec.CoreCount
				//					s.MemSize = s.MemSize + rec.MemSize
				//					s.SysDiskSize = s.SysDiskSize + rec.SysDiskSize
				//					s.WorkDiskSize = s.WorkDiskSize + rec.WorkDiskSize
				s.CPULoadAvg = s.CPULoadAvg + rec.CPULoadAvg

				if s.CPULoadMin > rec.CPULoadMin {
					s.CPULoadMin = rec.CPULoadMin
				}
				if s.CPULoadMax < rec.CPULoadMax {
					s.CPULoadMax = rec.CPULoadMax
				}

				s.MemUseAvg = s.MemUseAvg + rec.MemUseAvg

				if s.MemUseMin > rec.MemUseMin {
					s.MemUseMin = rec.MemUseMin
				}
				if s.MemUseMax < rec.MemUseMax {
					s.MemUseMax = rec.MemUseMax
				}

				s.SysDiskFreePercentAvg = s.SysDiskFreePercentAvg + rec.SysDiskFreePercentAvg

				if s.SysDiskFreePercentMin > rec.SysDiskFreePercentMin {
					s.SysDiskFreePercentMin = rec.SysDiskFreePercentMin
				}
				if s.SysDiskFreePercentMax < rec.SysDiskFreePercentMax {
					s.SysDiskFreePercentMax = rec.SysDiskFreePercentMax
				}

				s.SysDiskIdleTimeAvg = s.SysDiskIdleTimeAvg + rec.SysDiskIdleTimeAvg

				if s.SysDiskIdleTimeMin > rec.SysDiskIdleTimeMin {
					s.SysDiskIdleTimeMin = rec.SysDiskIdleTimeMin
				}
				if s.SysDiskIdleTimeMax < rec.SysDiskIdleTimeMax {
					s.SysDiskIdleTimeMax = rec.SysDiskIdleTimeMax
				}

				s.SysDiskQueueAvg = s.SysDiskQueueAvg + rec.SysDiskQueueAvg

				if s.SysDiskQueueMin > rec.SysDiskQueueMin {
					s.SysDiskQueueMin = rec.SysDiskQueueMin
				}
				if s.SysDiskQueueMax < rec.SysDiskQueueMax {
					s.SysDiskQueueMax = rec.SysDiskQueueMax
				}

				s.WorkDiskFreePercentAvg = s.WorkDiskFreePercentAvg + rec.WorkDiskFreePercentAvg

				if s.WorkDiskFreePercentMin > rec.WorkDiskFreePercentMin {
					s.WorkDiskFreePercentMin = rec.WorkDiskFreePercentMin
				}
				if s.WorkDiskFreePercentMax < rec.WorkDiskFreePercentMax {
					s.WorkDiskFreePercentMax = rec.WorkDiskFreePercentMax
				}

				s.WorkDiskIdleTimeAvg = s.WorkDiskIdleTimeAvg + rec.WorkDiskIdleTimeAvg

				if s.WorkDiskIdleTimeMin > rec.WorkDiskIdleTimeMin {
					s.WorkDiskIdleTimeMin = rec.WorkDiskIdleTimeMin
				}
				if s.WorkDiskIdleTimeMax < rec.WorkDiskIdleTimeMax {
					s.WorkDiskIdleTimeMax = rec.WorkDiskIdleTimeMax
				}

				s.WorkDiskQueueAvg = s.WorkDiskQueueAvg + rec.WorkDiskQueueAvg
				if s.WorkDiskQueueMin > rec.WorkDiskQueueMin {
					s.WorkDiskQueueMin = rec.WorkDiskQueueMin
				}
				if s.WorkDiskQueueMax < rec.WorkDiskQueueMax {
					s.WorkDiskQueueMax = rec.WorkDiskQueueMax
				}

				s.SysDiskIOAvg = s.SysDiskIOAvg + rec.SysDiskIOAvg
				if s.SysDiskIOMin > rec.SysDiskIOMin {
					s.SysDiskIOMin = rec.SysDiskIOMin
				}
				if s.SysDiskIOMax < rec.SysDiskIOMax {
					s.SysDiskIOMax = rec.SysDiskIOMax
				}

				s.SysDiskFreeMbAvg = s.SysDiskFreeMbAvg + rec.SysDiskFreeMbAvg
				if s.SysDiskFreeMbMin > rec.SysDiskFreeMbMin {
					s.SysDiskFreeMbMin = rec.SysDiskFreeMbMin
				}
				if s.SysDiskFreeMbMax < rec.SysDiskFreeMbMax {
					s.SysDiskFreeMbMax = rec.SysDiskFreeMbMax
				}

				s.WorkDiskIOAvg = s.WorkDiskIOAvg + rec.WorkDiskIOAvg
				if s.WorkDiskIOMin > rec.WorkDiskIOMin {
					s.WorkDiskIOMin = rec.WorkDiskIOMin
				}
				if s.WorkDiskIOMax < rec.WorkDiskIOMax {
					s.WorkDiskIOMax = rec.WorkDiskIOMax
				}

				s.WorkDiskFreeMbAvg = s.WorkDiskFreeMbAvg + rec.WorkDiskFreeMbAvg
				if s.WorkDiskFreeMbMin > rec.WorkDiskFreeMbMin {
					s.WorkDiskFreeMbMin = rec.WorkDiskFreeMbMin
				}
				if s.WorkDiskFreeMbMax < rec.WorkDiskFreeMbMax {
					s.WorkDiskFreeMbMax = rec.WorkDiskFreeMbMax
				}

				counts[j] = counts[j] + 1.0
				found = true
			}
		}

		if !found {
			r := *rec
			out = append(out, &r)
			counts = append(counts, 1.0)
		}
	}
	for k, s := range out {
		//		out[k].CPUCount = s.CPUCount / counts[k]
		//		out[k].CoreCount = s.CoreCount / counts[k]
		//		out[k].MemSize = s.MemSize / counts[k]
		//		out[k].SysDiskSize = s.SysDiskSize / counts[k]
		//		out[k].WorkDiskSize = s.WorkDiskSize / counts[k]
		out[k].CPULoadAvg = s.CPULoadAvg / counts[k]
		out[k].MemUseAvg = s.MemUseAvg / counts[k]
		out[k].SysDiskFreePercentAvg = s.SysDiskFreePercentAvg / counts[k]
		out[k].SysDiskIdleTimeAvg = s.SysDiskIdleTimeAvg / counts[k]
		out[k].SysDiskQueueAvg = s.SysDiskQueueAvg / counts[k]
		out[k].WorkDiskFreePercentAvg = s.WorkDiskFreePercentAvg / counts[k]
		out[k].WorkDiskIdleTimeAvg = s.WorkDiskIdleTimeAvg / counts[k]
		out[k].WorkDiskQueueAvg = s.WorkDiskQueueAvg / counts[k]
		out[k].SysDiskIOAvg = s.SysDiskIOAvg / counts[k]
		out[k].SysDiskFreeMbAvg = s.SysDiskFreeMbAvg / counts[k]
		out[k].WorkDiskIOAvg = s.WorkDiskIOAvg / counts[k]
		out[k].WorkDiskFreeMbAvg = s.WorkDiskFreeMbAvg / counts[k]
		CalcInventarisation(out[k]) //пересчет инвентаризации
	}

	//	sort.Sort(MaxResource(out))

	CoreCount := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.CoreCount > c2.CoreCount
	}
	MemSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.MemSize > c2.MemSize
	}

	SysDiskSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.SysDiskSize > c2.SysDiskSize
	}

	WorkDiskSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.WorkDiskSize > c2.WorkDiskSize
	}

	OrderedBy(MemSize, CoreCount, SysDiskSize, WorkDiskSize).Sort(out)
	return out
}
