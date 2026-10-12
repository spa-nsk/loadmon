<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Журнал действий</h1>
  <div class="d-flex gap-2">
    <a id="export-csv" class="btn btn-outline-secondary" href="/admin/events/export.csv">Выгрузить CSV</a>
    <a id="export-cef" class="btn btn-outline-secondary" href="/admin/events/export.cef">Выгрузить CEF (SIEM)</a>
  </div>
</div>
<p class="text-muted small">Действия пользователей над системой: входы, выходы, экспорт, изменения ролей и справочников
  (правила и профиль именования — тоже справочники), закрытие/переоткрытие учётных периодов. Не путать с
  <a href="/changes">журналом изменений</a> — тот про данные КЕ, этот про действия людей.</p>

<form id="filter" class="filter-bar card card-body mb-3" hx-get="/admin/events" hx-target="#events-results" hx-swap="innerHTML"
      hx-push-url="true" hx-trigger="change, submit, keyup changed delay:400ms from:#q">
  <div class="row g-2">
    <div class="col-md-3">
      <input id="q" name="q" class="form-control" placeholder="Тип события или деталь содержит…" value="{{.Filter.Q}}">
    </div>
    <div class="col-md-2">
      <select name="event_type" class="form-select" aria-label="Тип события">
        <option value="">Все типы</option>
        {{range .EventTypes}}<option value="{{.}}" {{if eq . $.Filter.EventType}}selected{{end}}>{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <select name="user_id" class="form-select" aria-label="Пользователь">
        <option value="">Все пользователи</option>
        {{range .Users}}<option value="{{.ID}}" {{if eq .ID (printf "%d" $.Filter.UserID)}}selected{{end}}>{{.Username}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-2">
      <label class="visually-hidden" for="from">С даты</label>
      <input id="from" type="date" name="from" class="form-control" value="{{.Filter.From}}">
    </div>
    <div class="col-md-2">
      <label class="visually-hidden" for="to">По дату</label>
      <input id="to" type="date" name="to" class="form-control" value="{{.Filter.To}}">
    </div>
  </div>
</form>

<div id="events-results">{{template "admin/_events_results.tpl" .}}</div>

<script>
  ['export-csv', 'export-cef'].forEach(function (id) {
    document.getElementById(id).addEventListener('click', function (e) {
      var q = new URLSearchParams(new FormData(document.getElementById('filter')));
      var base = e.currentTarget.getAttribute('href').split('?')[0];
      e.currentTarget.href = base + '?' + q.toString();
    });
  });
</script>
