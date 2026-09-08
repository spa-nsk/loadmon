package itilliumimport

import "sync"

// ImportProgress - состояние одного запущенного в фоне импорта блейд-корзин
// (см. controllers/adminimportblades.go: StartImportItiliumBlade запускает
// разбор и сохранение файла в отдельной горутине, а не синхронно в рамках
// одного HTTP-запроса, чтобы страница могла опрашивать прогресс через AJAX и
// показывать градуированный индикатор с процентами - 2026-09-08, по запросу
// пользователя "выводить статистику по итогу" + "отображать процесс импорта
// в виде градуированного линейного индикатора с процентами").
//
// Все поля читаются/пишутся из разных горутин (сама горутина импорта и
// HTTP-хендлер опроса статуса AdminImportBladeSystemsStatus) - доступ только
// через мьютекс-защищённые методы ниже, наружу отдаётся только неизменяемый
// Snapshot.
type ImportProgress struct {
	mu sync.Mutex

	fileRows  int // всего строк в файле, дошедших до обработки (знаменатель для процента)
	processed int // сколько строк файла уже прошли через цикл обработки (успешно или с ошибкой)

	imported int // сколько строк успешно сохранено в БД (новых + обновлённых)
	newCount int // из них - создано новых записей BladeSystems
	updCount int // из них - обновлено уже существующих записей
	inactive int // сколько из успешно сохранённых записей в итоге неактивны (Active == 2)

	done bool
	err  string
}

// importJobs - реестр запущенных импортов по идентификатору задания. В
// текущей реализации идентификатор - это имя файла, сохранённого на сервере
// (см. AdminImportBladeSystems: savedName уже уникален благодаря временной
// метке в имени). Хранится в памяти процесса - при перезапуске сервера
// прогресс "теряется" вместе с самим импортом, что ожидаемо.
var (
	importJobsMu sync.Mutex
	importJobs   = map[string]*ImportProgress{}
)

// newImportJob регистрирует новое задание импорта под ключом jobId.
func newImportJob(jobId string) *ImportProgress {
	p := &ImportProgress{}
	importJobsMu.Lock()
	importJobs[jobId] = p
	importJobsMu.Unlock()
	return p
}

// GetImportJob возвращает прогресс запущенного (или уже завершённого)
// импорта по jobId и признак, что такое задание известно - используется
// controllers/adminimportblades.go:AdminImportBladeSystemsStatus для ответа
// на AJAX-опрос страницы.
func GetImportJob(jobId string) (*ImportProgress, bool) {
	importJobsMu.Lock()
	p, ok := importJobs[jobId]
	importJobsMu.Unlock()
	return p, ok
}

// setFileRows задаёт знаменатель для процента (общее число строк в файле,
// полученных из ConvertFileBlade) - вызывается один раз сразу после
// успешного разбора файла.
func (p *ImportProgress) setFileRows(n int) {
	p.mu.Lock()
	p.fileRows = n
	p.mu.Unlock()
}

// tickProcessed отмечает переход к обработке очередной строки файла -
// вызывается в начале каждой итерации цикла по строкам, независимо от
// итогового результата (пропуск/ошибка/успех), чтобы индикатор прогресса
// двигался равномерно даже на "проблемных" строках.
func (p *ImportProgress) tickProcessed() {
	p.mu.Lock()
	p.processed++
	p.mu.Unlock()
}

// addResult фиксирует итог успешно сохранённой строки (isNew - создана новая
// запись BladeSystems или обновлена существующая; active2 - итоговое
// значение BladeSystems.Active после (если отработал) уточнения по HPE
// OneView равно 2 "выведена из эксплуатации/не размещена").
func (p *ImportProgress) addResult(isNew bool, active2 bool) {
	p.mu.Lock()
	p.imported++
	if isNew {
		p.newCount++
	} else {
		p.updCount++
	}
	if active2 {
		p.inactive++
	}
	p.mu.Unlock()
}

// finish помечает задание завершённым, опционально с текстом ошибки (общая
// ошибка загрузки конфигурации соответствий/чтения файла - т.е. импорт не
// смог даже начаться, см. ImportItiliumBlade).
func (p *ImportProgress) finish(errText string) {
	p.mu.Lock()
	p.done = true
	p.err = errText
	p.mu.Unlock()
}

// ImportProgressSnapshot - неизменяемый снимок ImportProgress для отдачи в
// JSON (см. AdminImportBladeSystemsStatus) - поля названы под прямое
// использование на странице (see admin/importblades.tpl).
type ImportProgressSnapshot struct {
	FileRows  int    `json:"file_rows"`
	Processed int    `json:"processed"`
	Imported  int    `json:"imported"`
	New       int    `json:"new"`
	Updated   int    `json:"updated"`
	Inactive  int    `json:"inactive"`
	Done      bool   `json:"done"`
	Err       string `json:"err,omitempty"`
	Percent   int    `json:"percent"`
}

// Snapshot возвращает текущее состояние задания - см. ImportProgressSnapshot.
// Percent считается по processed/fileRows (а не по imported), чтобы строки,
// пропущенные из-за ошибок, тоже продвигали индикатор - иначе он мог бы
// "зависнуть" на файле, где часть строк ошибочна. Если fileRows==0 (ошибка
// разбора файла ещё до цикла), процент принудительно 100 при done==true,
// чтобы индикатор не висел на 0% для явно завершившегося (с ошибкой)
// задания.
func (p *ImportProgress) Snapshot() ImportProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	percent := 0
	switch {
	case p.fileRows > 0:
		percent = p.processed * 100 / p.fileRows
		if percent > 100 {
			percent = 100
		}
	case p.done:
		percent = 100
	}

	return ImportProgressSnapshot{
		FileRows:  p.fileRows,
		Processed: p.processed,
		Imported:  p.imported,
		New:       p.newCount,
		Updated:   p.updCount,
		Inactive:  p.inactive,
		Done:      p.done,
		Err:       p.err,
		Percent:   percent,
	}
}
