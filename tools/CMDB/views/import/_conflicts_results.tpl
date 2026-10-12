<table class="table table-sm table-hover align-middle">
  <thead>
    <tr><th>№</th><th>Прогон</th><th>Вид</th><th>КЕ</th><th>Подробности</th><th>Статус</th><th>Когда (UTC)</th></tr>
  </thead>
  <tbody>
    {{range .Result.Rows}}
    <tr>
      <td>{{.ID}}</td>
      <td><a href="/import/runs/{{.RunID}}">№{{.RunID}}</a></td>
      <td>
        {{if eq .Kind "ambiguous_match"}}<span class="badge text-bg-warning">Неоднозначная идентификация</span>
        {{else if eq .Kind "locked_attr"}}<span class="badge text-bg-info">Заблокированный атрибут</span>
        {{else if eq .Kind "apply_error"}}<span class="badge text-bg-danger">Ошибка применения</span>
        {{else if eq .Kind "not_discovered"}}<span class="badge text-bg-secondary">Не найдена источником</span>
        {{else}}{{.Kind}}{{end}}
      </td>
      <td>{{if .CIID}}<a href="/cis/{{.CIID}}">{{if .CIName}}{{.CIName}}{{else}}#{{.CIID}}{{end}}</a>{{else}}—{{end}}</td>
      <td><code class="small">{{.Details}}</code></td>
      <td>{{.Status}}</td>
      <td>{{.CreatedAt}}</td>
    </tr>
    {{else}}
    <tr><td colspan="7" class="text-center text-body-secondary py-4">Конфликтов не найдено</td></tr>
    {{end}}
  </tbody>
</table>

<div class="d-flex justify-content-between align-items-center">
  <small class="text-body-secondary">Найдено: {{.Result.Total}}</small>
  <nav aria-label="Страницы">
    <ul class="pagination pagination-sm mb-0">
      {{if gt .Result.Page 1}}<li class="page-item"><a class="page-link" href="?page={{sub .Result.Page 1}}" hx-get="/import/conflicts?status={{.Filter.Status}}&amp;kind={{.Filter.Kind}}&amp;run_id={{.Filter.RunID}}&amp;page={{sub .Result.Page 1}}" hx-target="#conflict-results" hx-push-url="true">‹</a></li>{{end}}
      <li class="page-item disabled"><span class="page-link">{{.Result.Page}} / {{.Result.Pages}}</span></li>
      {{if lt .Result.Page .Result.Pages}}<li class="page-item"><a class="page-link" href="?page={{add .Result.Page 1}}" hx-get="/import/conflicts?status={{.Filter.Status}}&amp;kind={{.Filter.Kind}}&amp;run_id={{.Filter.RunID}}&amp;page={{add .Result.Page 1}}" hx-target="#conflict-results" hx-push-url="true">›</a></li>{{end}}
    </ul>
  </nav>
</div>
