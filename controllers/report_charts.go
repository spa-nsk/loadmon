// Package controllers реализует веб-контроллеры интерфейса пользователя (beego)
// системы мониторинга loadmon: отображение и фильтрацию списка серверов и
// виртуальных машин, карточки серверов с показателями нагрузки (CPU, память,
// диски) и графиками производительности, работу со справочником серверов
// (добавление, обновление данных из систем-источников), аутентификацию и
// профиль пользователя, журналирование действий, а также вспомогательный API
// для внешних систем.
package controllers

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	//"github.com/astaxie/beego"
	//"github.com/astaxie/beego/orm"

	chart "github.com/wcharczuk/go-chart"

	//	"github.com/astaxie/beego/validation"

	//	exportres "loadmon/export"
	models "loadmon/models"
)

func releases() []chart.GridLine {
	return []chart.GridLine{
		{Value: float64(time.Date(2019, 12, 2, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 3, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 4, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 5, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 6, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 7, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 8, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 9, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 10, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 11, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 12, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 13, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 14, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 15, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 16, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 17, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 18, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 19, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 20, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 21, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 22, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 23, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 24, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 25, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 26, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 27, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 28, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 29, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 30, 0, 0, 0, 0, time.Local).UnixNano())},
	}
}

// MakeImageReportCPU1 строит и сохраняет в destination PNG-график загрузки CPU
// по серверам на основе переданных записей отчёта records.
func MakeImageReportCPU1(records []*models.ReportServerRecord, destination string) error {

	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64
	num := len(records)
	count_series := 3
	series = make([]chart.Series, count_series) // +1 если добавить закомментированный кусок

	xValues := make([]time.Time, num)
	yCPUMax := make([]float64, num)
	yCPUAvg := make([]float64, num)
	yCPUMin := make([]float64, num)
	ServerType := "Сервер: "
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yCPUMax[j] = record.CPULoadMax
		yCPUAvg[j] = record.CPULoadAvg
		yCPUMin[j] = record.CPULoadMin
	}

	series[0] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Процессор макс.",
		XValues: xValues,
		YValues: yCPUMax,
	}

	series[1] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Процессор ср.",
		XValues: xValues,
		YValues: yCPUAvg,
	}

	series[2] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(15),
		},
		Name:    "Процессор мин.",
		XValues: xValues,
		YValues: yCPUMin,
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

// MakeImageReportMEM1 строит и сохраняет в destination PNG-график использования
// памяти по серверам на основе переданных записей отчёта records.
func MakeImageReportMEM1(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	series = make([]chart.Series, count_series) // +1 если добавить закомментированный кусок

	xValues := make([]time.Time, num)
	yMEMMax := make([]float64, num)
	yMEMAvg := make([]float64, num)
	yMEMMin := make([]float64, num)
	ServerType := "Сервер: "
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yMEMMax[j] = record.MemUseMax
		yMEMAvg[j] = record.MemUseAvg
		yMEMMin[j] = record.MemUseMin
	}

	series[0] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Память макс.",
		XValues: xValues,
		YValues: yMEMMax,
	}

	series[1] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Память ср.",
		XValues: xValues,
		YValues: yMEMAvg,
	}

	series[2] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(15),
		},
		Name:    "Память мин.",
		XValues: xValues,
		YValues: yMEMMin,
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

// MakeImageReportDiskLoad строит и сохраняет в destination PNG-график нагрузки
// на диски по серверам на основе переданных записей отчёта records.
func MakeImageReportDiskLoad(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 0
	xValues := make([]time.Time, num)
	ySYS_QUEUE := make([]float64, num)
	yWORK_QUEUE := make([]float64, num)
	ySYS_DISK_LOADMax := make([]float64, num)
	ySYS_DISK_LOADAvg := make([]float64, num)
	ySYS_DISK_LOADMin := make([]float64, num)
	yWORK_DISK_LOADMax := make([]float64, num)
	yWORK_DISK_LOADAvg := make([]float64, num)
	yWORK_DISK_LOADMin := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	flag_FromSCOM := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 4
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			if !(record.SysDiskQueueMax == 0.0 && record.SysDiskIdleTimeMin == 0.0) {
				count_series = count_series + 4
				flag_FromSCOM = true
			}
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		ySYS_QUEUE[j] = record.SysDiskQueueMax
		yWORK_QUEUE[j] = record.WorkDiskQueueMax
		ySYS_DISK_LOADMax[j] = 100.0 - record.SysDiskIdleTimeMin
		ySYS_DISK_LOADAvg[j] = 100.0 - record.SysDiskIdleTimeAvg
		ySYS_DISK_LOADMin[j] = 100.0 - record.SysDiskIdleTimeMax
		yWORK_DISK_LOADMax[j] = 100.0 - record.WorkDiskIdleTimeMin
		yWORK_DISK_LOADAvg[j] = 100.0 - record.WorkDiskIdleTimeAvg
		yWORK_DISK_LOADMin[j] = 100.0 - record.WorkDiskIdleTimeMax
	}
	k := 0
	if count_series == 0 { // нет данных по дискам
		return errors.New("Нет данных для отчета")
	}
	if flag_FromSCOM {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlack,
				FillColor:   chart.ColorBlack.WithAlpha(5),
			},
			Name:    "Очередь системного диска",
			XValues: xValues,
			YValues: ySYS_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorRed,
				FillColor:   chart.ColorRed.WithAlpha(15),
			},
			Name:    "Загрузка системного диска макс.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorGreen,
				FillColor:   chart.ColorGreen.WithAlpha(15),
			},
			Name:    "Загрузка системного диска ср.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlue,
				FillColor:   chart.ColorBlue.WithAlpha(15),
			},
			Name:    "Загрузка системного диска мин.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADMin,
		}
		k = k + 1
	}
	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorCyan,
				FillColor:   chart.ColorCyan.WithAlpha(15),
			},
			Name:    "Очередь рабочего диска",
			XValues: xValues,
			YValues: yWORK_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска макс.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска ср.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска мин.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADMin,
		}
		k = k + 1
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

// MakeImageReportDiskUse строит и сохраняет в destination PNG-график использования
// дискового пространства по серверам на основе переданных записей отчёта records.
func MakeImageReportDiskUse(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	xValues := make([]time.Time, num)
	ySYS_DISK_USEMax := make([]float64, num)
	ySYS_DISK_USEAvg := make([]float64, num)
	ySYS_DISK_USEMin := make([]float64, num)
	yWORK_DISK_USEMax := make([]float64, num)
	yWORK_DISK_USEAvg := make([]float64, num)
	yWORK_DISK_USEMin := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 3
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		ySYS_DISK_USEMax[j] = 100.0 - record.SysDiskFreePercentMin
		ySYS_DISK_USEAvg[j] = 100.0 - record.SysDiskFreePercentAvg
		ySYS_DISK_USEMin[j] = 100.0 - record.SysDiskFreePercentMax
		yWORK_DISK_USEMax[j] = 100.0 - record.WorkDiskFreePercentMin
		yWORK_DISK_USEAvg[j] = 100.0 - record.WorkDiskFreePercentAvg
		yWORK_DISK_USEMin[j] = 100.0 - record.WorkDiskFreePercentMax
	}
	k := 0
	if count_series == 0 { // нет данных по дискам
		return errors.New("Нет данных для отчета")
	}

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(5),
		},
		Name:    "Занято на системном диске макс.",
		XValues: xValues,
		YValues: ySYS_DISK_USEMax,
	}
	k = k + 1
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(5),
		},
		Name:    "Занято на системном диске ср.",
		XValues: xValues,
		YValues: ySYS_DISK_USEAvg,
	}
	k = k + 1
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(5),
		},
		Name:    "Занято на системном диске мин.",
		XValues: xValues,
		YValues: ySYS_DISK_USEMin,
	}
	k = k + 1

	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске макс.",
			XValues: xValues,
			YValues: yWORK_DISK_USEMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске ср.",
			XValues: xValues,
			YValues: yWORK_DISK_USEAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске мин.",
			XValues: xValues,
			YValues: yWORK_DISK_USEMin,
		}
		k = k + 1
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

// MakeImageReportMIX1 строит и сохраняет в destination сводный PNG-график
// (CPU, память, диски) по серверам на основе переданных записей отчёта records.
func MakeImageReportMIX1(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	xValues := make([]time.Time, num)
	yCPU := make([]float64, num)
	yRAM := make([]float64, num)
	ySYS_QUEUE := make([]float64, num)
	yWORK_QUEUE := make([]float64, num)
	ySYS_DISK_LOAD := make([]float64, num)
	yWORK_DISK_LOAD := make([]float64, num)
	ySYS_DISK_USE := make([]float64, num)
	yWORK_DISK_USE := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	flag_FromSCOM := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 3
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			if !(record.SysDiskQueueMax == 0.0 && record.SysDiskIdleTimeMin == 0.0) {
				count_series = count_series + 2
				flag_FromSCOM = true
			}
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" Количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yCPU[j] = record.CPULoadMax
		yRAM[j] = record.MemUseMax
		ySYS_QUEUE[j] = record.SysDiskQueueMax
		yWORK_QUEUE[j] = record.WorkDiskQueueMax
		ySYS_DISK_LOAD[j] = 100.0 - record.SysDiskIdleTimeMin
		yWORK_DISK_LOAD[j] = 100.0 - record.WorkDiskIdleTimeMin
		ySYS_DISK_USE[j] = 100.0 - record.SysDiskFreePercentMin
		yWORK_DISK_USE[j] = 100.0 - record.WorkDiskFreePercentMin

	}
	k := 0
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Процессор",
		XValues: xValues,
		YValues: yCPU,
	}
	k = k + 1

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Память",
		XValues: xValues,
		YValues: yRAM,
	}
	k = k + 1

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlack,
			FillColor:   chart.ColorBlack.WithAlpha(5),
		},
		Name:    "Занято на системном диске",
		XValues: xValues,
		YValues: ySYS_DISK_USE,
	}
	k = k + 1

	if flag_FromSCOM {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlue,
				FillColor:   chart.ColorBlue.WithAlpha(15),
			},
			Name:    "Очередь системного диска",
			XValues: xValues,
			YValues: ySYS_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorCyan,
				FillColor:   chart.ColorCyan.WithAlpha(15),
			},
			Name:    "Загрузка системного диска",
			XValues: xValues,
			YValues: ySYS_DISK_LOAD,
		}
		k = k + 1
	}
	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске",
			XValues: xValues,
			YValues: yWORK_DISK_USE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(15),
			},
			Name:    "Очередь рабочего диска",
			XValues: xValues,
			YValues: yWORK_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска",
			XValues: xValues,
			YValues: yWORK_DISK_LOAD,
		}
		k = k + 1
	}

	//
	//	series[3] = chart.AnnotationSeries{
	//		Annotations: []chart.Value{
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yCPU[0], Label: "Процессор", Style: chart.Style{StrokeColor: chart.ColorRed, FillColor: chart.ColorRed.WithAlpha(20)}},
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yRAM[0], Label: "Память", Style: chart.Style{StrokeColor: chart.ColorGreen, FillColor: chart.ColorGreen.WithAlpha(20)}},
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yQUEUE[0], Label: "Очередь диска", Style: chart.Style{StrokeColor: chart.ColorBlue, FillColor: chart.ColorBlue.WithAlpha(20)}},
	//		},
	//	}
	//

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

// ServerDetails отображает подробную информацию и графики нагрузки
// по конкретному серверу за выбранный период.
