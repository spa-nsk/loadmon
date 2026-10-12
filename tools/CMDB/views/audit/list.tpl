<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Журнал изменений</h1>
  <a id="export-csv" class="btn btn-outline-secondary" href="/changes/export.csv">Выгрузить CSV (по текущему фильтру)</a>
</div>

<form id="filter" class="filter-bar card card-body mb-3" hx-get="/changes" hx-target="#audit-results" hx-swap="innerHTML"
      hx-push-url="true" hx-trigger="change, submit, keyup changed delay:400ms from:#q">
  <div class="row g-2">
    <div class="col-md-3">
      <input id="q" name="q" class="form-control" placeholder="Причина содержит…" value="{{.Filter.Q}}">
    </div>
    <div class="col-md-2">
      <select name="source" class="form-select" aria-label="Источник">
        <option value="">Все источники</option>
        {{range .Sources}}<option value="{{.}}" {{if eq . $.Filter.Source}}selected{{end}}>{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-2">
      <select name="entity" class="form-select" aria-label="Сущность">
        <option value="">Все сущности</option>
        {{range .Entities}}<option value="{{.}}" {{if eq . $.Filter.Entity}}selected{{end}}>{{.}}</option>{{end}}
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

<div id="audit-results">{{template "audit/_results.tpl" .}}</div>

<script>
  document.getElementById('export-csv').addEventListener('click', function (e) {
    var q = new URLSearchParams(new FormData(document.getElementById('filter')));
    e.currentTarget.href = '/changes/export.csv?' + q.toString();
  });
</script>
