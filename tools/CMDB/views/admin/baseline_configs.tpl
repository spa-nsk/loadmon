<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Эталонные конфигурации</h1>
  <div class="d-flex gap-2">
    <a class="btn btn-outline-secondary" href="/reports/baseline-config">Отчёт отклонений</a>
    <a class="btn btn-primary" href="/admin/baseline-configs/new">Новая конфигурация</a>
  </div>
</div>
<p class="text-muted small">Требуемые значения атрибутов и обязательное ПО для класса КЕ (plan.md E7.8) — источник для
  <a href="/reports/baseline-config">отчёта отклонений</a>. «Роль» — свободная подпись для человека (несколько
  конфигураций для одного класса, например «сервер БД» и «веб-сервер»); автоматически КЕ подбираются по классу и,
  если задано, окружению/компании/площадке.</p>

<table class="table table-sm align-middle">
  <thead><tr><th>Имя</th><th>Класс</th><th>Роль</th><th>Область</th><th>Атрибутов</th><th>ПО</th><th>Статус</th><th></th></tr></thead>
  <tbody>
    {{range .Configs}}
    <tr>
      <td>{{.Name}}</td>
      <td>{{.ClassName}}</td>
      <td>{{.Role}}</td>
      <td class="small text-muted">
        {{if .EnvironmentCode}}{{.EnvironmentCode}}{{end}}
        {{if .CompanyCode}} · {{.CompanyCode}}{{end}}
        {{if .LocationCode}} · {{.LocationCode}}{{end}}
        {{if and (not .EnvironmentCode) (not .CompanyCode) (not .LocationCode)}}любая{{end}}
      </td>
      <td>{{len .Attrs}}</td>
      <td>{{len .RequiredSoftware}}</td>
      <td>{{if .Enabled}}<span class="badge text-bg-success">включена</span>{{else}}<span class="badge text-bg-secondary">выключена</span>{{end}}</td>
      <td class="text-end text-nowrap">
        <a class="btn btn-sm btn-outline-secondary" href="/admin/baseline-configs/{{.ID}}/edit">Править</a>
        <button type="button" class="btn btn-sm btn-outline-danger" hx-post="/admin/baseline-configs/{{.ID}}/delete" hx-swap="none"
          hx-confirm="Удалить конфигурацию «{{.Name}}» безвозвратно?">Удалить</button>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="8" class="text-center text-body-secondary py-4">Эталонных конфигураций пока нет</td></tr>
    {{end}}
  </tbody>
</table>
