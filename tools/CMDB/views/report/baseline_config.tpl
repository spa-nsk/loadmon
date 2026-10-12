<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Отклонения от эталонных конфигураций</h1>
  <a id="export-csv" class="btn btn-outline-secondary" href="/reports/baseline-config/export.csv">Выгрузить CSV</a>
</div>
<p class="text-muted small">Требуемые атрибуты и ПО задаются в <a href="/admin/baseline-configs">эталонных конфигурациях</a>.
  По умолчанию — по действующим КЕ (можно посмотреть на дату — <code>?asof=</code>, как везде, E6.1); либо по составу
  конкретного замороженного/утверждённого среза (E6.2) — выберите его ниже.</p>

<form id="filter" class="card card-body mb-3" method="get" action="/reports/baseline-config">
  <div class="row g-2">
    <div class="col-md-4">
      <label class="form-label" for="baseline">Проверить против среза</label>
      <select id="baseline" name="baseline" class="form-select" onchange="this.form.submit()">
        <option value="">действующие КЕ (текущее/на дату)</option>
        {{range .Baselines}}<option value="{{.ID}}" {{if eq .ID (printf "%d" $.BaselineID)}}selected{{end}}>{{.Name}} ({{.Status}}, на {{.AsOf}})</option>{{end}}
      </select>
      {{if not .Baselines}}<div class="form-text">Нет замороженных или утверждённых срезов.</div>{{end}}
    </div>
  </div>
</form>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Конфигурация</th><th>КЕ</th><th>Тип</th><th>Поле</th><th>Требуется</th><th>Фактически</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td>{{.ConfigName}}</td>
          <td><a href="/cis/{{.CIID}}">{{.CIName}}</a></td>
          <td><span class="badge {{if eq .Kind "software"}}text-bg-danger{{else}}text-bg-warning{{end}}">{{.Kind}}</span></td>
          <td>{{.Field}}</td>
          <td>{{.Expected}}</td>
          <td>{{if .Actual}}{{.Actual}}{{else}}<span class="text-muted">отсутствует</span>{{end}}</td>
        </tr>
        {{else}}<tr><td colspan="6" class="text-muted p-3">Отклонений не найдено.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
