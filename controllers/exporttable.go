package controllers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"time"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// exportTableFile отдаёт таблицу (заголовок + строки) файлом XLSX (xlsx=true)
// или CSV (utf8, разделитель "|") на скачивание. Файл сохраняется в ./reports/
// под именем <filePrefix>_<дата>.<csv|xlsx>. Пустая таблица -> сообщение
// "Нет записей для экспорта".
func (this *MainController) exportTableFile(header []string, data [][]string, xlsx bool, filePrefix, sheet string, flash *beego.FlashData) {
	if len(data) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}
	crt := time.Now().Local()
	ext := "csv"
	if xlsx {
		ext = "xlsx"
	}
	shortName := fmt.Sprintf("%s_%02d_%02d_%04d_%02d_%02d_%02d_%d.%s", filePrefix, crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano(), ext)
	destination := "./reports/" + shortName

	if xlsx {
		file := excelize.NewFile()
		index, err := file.NewSheet(sheet)
		if err != nil {
			flash.Error("Ошибка создания листа XLSX файла: %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
		file.SetActiveSheet(index)
		file.DeleteSheet("Sheet1")
		all := append([][]string{header}, data...)
		for i, line := range all {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			vals := make([]interface{}, len(line))
			for k, v := range line {
				vals[k] = v
			}
			if err := file.SetSheetRow(sheet, cell, &vals); err != nil {
				flash.Error("Ошибка записи строки XLSX файла: %v", err)
				flash.Store(&this.Controller)
				file.Close()
				return
			}
		}
		if err := file.SaveAs(destination); err != nil {
			flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
		file.Close()
		this.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	} else {
		csvFile, err := os.Create(destination)
		if err != nil {
			flash.Error("Ошибка создания CSV файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			return
		}
		w := csv.NewWriter(csvFile)
		w.Comma = '|'
		_ = w.Write(header)
		for _, line := range data {
			_ = w.Write(line)
		}
		w.Flush()
		csvFile.Close()
		this.Ctx.Output.Header("Content-Type", "application/csv")
	}

	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+shortName+"\"")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")
	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}
