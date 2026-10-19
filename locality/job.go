package locality

import "sync"

// Job - состояние фоновой загрузки справочника населённых пунктов (аналог
// itilliumimport.ImportProgress): страница администрирования опрашивает его
// через AJAX и показывает индикатор прогресса.
type Job struct {
	mu        sync.Mutex
	fileRows  int
	processed int
	done      bool
	err       string
}

// JobSnapshot - неизменяемый снимок Job для отдачи в JSON.
type JobSnapshot struct {
	FileRows  int    `json:"file_rows"`
	Processed int    `json:"processed"`
	Imported  int    `json:"imported"`
	Done      bool   `json:"done"`
	Err       string `json:"err,omitempty"`
	Percent   int    `json:"percent"`
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*Job{}
)

// GetJob возвращает задание загрузки по идентификатору (имени сохранённого файла).
func GetJob(id string) (*Job, bool) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	j, ok := jobs[id]
	return j, ok
}

// Start запускает загрузку файла path в фоне и регистрирует задание под id.
func Start(id, path string) *Job {
	j := &Job{}
	jobsMu.Lock()
	jobs[id] = j
	jobsMu.Unlock()
	go func() {
		_, err := load(path, j)
		if err != nil {
			j.finish(err.Error())
			return
		}
		j.finish("")
	}()
	return j
}

func (j *Job) set(rows, processed int) {
	if j == nil {
		return
	}
	j.mu.Lock()
	if rows >= 0 {
		j.fileRows = rows
	}
	j.processed = processed
	j.mu.Unlock()
}

func (j *Job) finish(errText string) {
	j.mu.Lock()
	j.done = true
	j.err = errText
	j.mu.Unlock()
}

// Snapshot возвращает текущее состояние задания.
func (j *Job) Snapshot() JobSnapshot {
	j.mu.Lock()
	defer j.mu.Unlock()
	pct := 0
	switch {
	case j.fileRows > 0:
		pct = j.processed * 100 / j.fileRows
	case j.done:
		pct = 100
	}
	return JobSnapshot{FileRows: j.fileRows, Processed: j.processed, Imported: j.processed,
		Done: j.done, Err: j.err, Percent: pct}
}
