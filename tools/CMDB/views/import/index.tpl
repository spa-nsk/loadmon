<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Импорт</h1>
  <a class="btn btn-outline-secondary" href="/import/conflicts">Очередь конфликтов{{if gt .OpenConflicts 0}} <span class="badge text-bg-danger">{{.OpenConflicts}}</span>{{end}}</a>
</div>

<div class="card card-body mb-3">
  <h2 class="h6">Загрузить файл</h2>
  <form hx-post="/import/run" hx-encoding="multipart/form-data" hx-swap="none">
    <div class="row g-2 align-items-end">
      <div class="col-md-3">
        <label class="form-label" for="connector">Источник</label>
        <select id="connector" name="connector" class="form-select" required>
          {{range .Connectors}}<option value="{{.Code}}">{{.Name}}</option>{{end}}
        </select>
      </div>
      <div class="col-md-4">
        <label class="form-label" for="file">Файл (CSV или XLSX)</label>
        <input id="file" type="file" name="file" accept=".csv,.xlsx" required class="form-control">
      </div>
      <div class="col-md-2 form-check ms-2">
        <input id="dry_run" type="checkbox" name="dry_run" value="1" class="form-check-input" checked>
        <label class="form-check-label" for="dry_run">Предпросмотр (dry-run)</label>
      </div>
      <div class="col-md-3 text-md-end">
        <button type="submit" class="btn btn-primary">Загрузить</button>
      </div>
    </div>
  </form>
  <div class="mt-2">
    {{range .Connectors}}<a class="link-secondary small me-3" href="/import/template?class={{.CIClass}}">Шаблон CSV: {{.Name}}</a>{{end}}
  </div>
</div>

<h2 class="h6">Источники по API</h2>
<table class="table table-sm align-middle mb-3">
  <thead><tr><th>Источник</th><th>Вид</th><th>Расписание</th><th>Последнее задание</th><th></th></tr></thead>
  <tbody>
    {{range .APIConnectors}}
    <tr>
      <td>{{.Name}} <small class="text-body-secondary">{{.Code}}</small></td>
      <td>{{.Kind}}</td>
      <td>{{if .ScheduleSec}}каждые {{.ScheduleSec}} с{{else}}только вручную{{end}}</td>
      <td>{{if .LastJob}}{{template "import/_job_status.tpl" .LastJob}} <small class="text-body-secondary">{{.LastJobAt}}</small>{{else}}—{{end}}</td>
      <td class="text-end">
        <form class="d-inline-flex gap-2 align-items-center" hx-post="/import/connectors/{{.Code}}/run" hx-swap="none">
          <div class="form-check mb-0"><input id="dry-{{.Code}}" type="checkbox" name="dry_run" value="1" class="form-check-input"><label class="form-check-label small" for="dry-{{.Code}}">предпросмотр</label></div>
          <button type="submit" class="btn btn-sm btn-primary">Запустить</button>
          <button type="button" class="btn btn-sm btn-outline-secondary" hx-post="/import/connectors/{{.Code}}/test" hx-swap="none">Проверить подключение</button>
        </form>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="5" class="text-center text-body-secondary py-3">API-источники не настроены (imp.connector)</td></tr>
    {{end}}
  </tbody>
</table>

<h2 class="h6">Очередь заданий</h2>
<table class="table table-sm align-middle mb-3">
  <thead><tr><th>№</th><th>Источник</th><th>Запуск</th><th>Статус</th><th>Попытка</th><th>Не раньше (UTC)</th><th>Ошибка</th><th></th></tr></thead>
  <tbody>
    {{range .Jobs}}
    <tr>
      <td>{{.ID}}</td>
      <td>{{.Connector}}</td>
      <td>{{if eq .Trigger "schedule"}}по расписанию{{else}}вручную{{end}}{{if .DryRun}} <span class="badge text-bg-secondary">предпросмотр</span>{{end}}</td>
      <td>{{template "import/_job_status.tpl" .Status}}</td>
      <td>{{.Attempts}} / {{.Max}}</td>
      <td>{{.RunAfter}}</td>
      <td class="small">{{.Error}}</td>
      <td class="text-end">
        {{if .RunID}}<a class="btn btn-sm btn-outline-secondary" href="/import/runs/{{.RunID}}">Прогон</a>{{end}}
        {{if .Cancelable}}<button type="button" class="btn btn-sm btn-outline-danger" hx-post="/import/jobs/{{.ID}}/cancel" hx-swap="none">Отменить</button>{{end}}
      </td>
    </tr>
    {{else}}
    <tr><td colspan="8" class="text-center text-body-secondary py-3">Заданий ещё не было</td></tr>
    {{end}}
  </tbody>
</table>

<h2 class="h6">Последние прогоны</h2>
<table class="table table-sm table-hover align-middle">
  <thead>
    <tr>
      <th>№</th><th>Источник</th><th>Когда (UTC)</th><th>Режим</th><th>Статус</th>
      <th>Всего</th><th>Создано</th><th>Изменено</th><th>Без изменений</th><th>Конфликты</th><th>Ошибки</th><th>Не найдено</th><th></th>
    </tr>
  </thead>
  <tbody>
    {{range .Runs}}
    <tr>
      <td>{{.ID}}</td>
      <td>{{.ConnectorName}}</td>
      <td>{{.StartedAt}}</td>
      <td>{{if .DryRun}}<span class="badge text-bg-secondary">предпросмотр</span>{{else}}<span class="badge text-bg-primary">применён</span>{{end}}</td>
      <td>
        {{if eq .Status "succeeded"}}<span class="badge text-bg-success">выполнен</span>
        {{else if eq .Status "failed"}}<span class="badge text-bg-danger">сбой</span>
        {{else}}<span class="badge text-bg-warning">выполняется</span>{{end}}
      </td>
      <td>{{.Total}}</td><td>{{.Created}}</td><td>{{.Updated}}</td><td>{{.Unchanged}}</td><td>{{.Conflicts}}</td><td>{{.Errors}}</td><td>{{.NotDiscovered}}</td>
      <td><a class="btn btn-sm btn-outline-secondary" href="/import/runs/{{.ID}}">Подробнее</a></td>
    </tr>
    {{else}}
    <tr><td colspan="13" class="text-center text-body-secondary py-4">Прогонов импорта ещё не было</td></tr>
    {{end}}
  </tbody>
</table>
