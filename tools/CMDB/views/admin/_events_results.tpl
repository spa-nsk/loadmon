<table class="table table-sm table-hover align-middle">
  <thead>
    <tr>
      <th>№</th><th>Когда (UTC)</th><th>Тип события</th><th>Пользователь</th><th>IP</th><th>Детали</th>
    </tr>
  </thead>
  <tbody>
    {{range .Result.Rows}}
    <tr>
      <td>{{.ID}}</td>
      <td>{{.At}}</td>
      <td><span class="badge text-bg-secondary">{{.EventType}}</span></td>
      <td>{{if .Username}}{{.Username}}{{else if .UserID}}#{{.UserID}}{{else}}<span class="text-muted">—</span>{{end}}</td>
      <td>{{.ClientIP}}</td>
      <td><code class="small">{{.Details}}</code></td>
    </tr>
    {{else}}
    <tr><td colspan="6" class="text-center text-body-secondary py-4">Ничего не найдено</td></tr>
    {{end}}
  </tbody>
</table>

<div class="d-flex justify-content-between align-items-center">
  <small class="text-body-secondary">Найдено: {{.Result.Total}}</small>
  <nav aria-label="Страницы">
    <ul class="pagination pagination-sm mb-0">
      {{if gt .Result.Page 1}}<li class="page-item"><a class="page-link" href="{{call .PageURL (sub .Result.Page 1)}}" hx-get="{{call .PageURL (sub .Result.Page 1)}}" hx-target="#events-results" hx-push-url="true">‹</a></li>{{end}}
      {{range .PageNums}}
      <li class="page-item {{if eq . $.Result.Page}}active{{end}}"><a class="page-link" href="{{call $.PageURL .}}" hx-get="{{call $.PageURL .}}" hx-target="#events-results" hx-push-url="true">{{.}}</a></li>
      {{end}}
      {{if lt .Result.Page .Result.Pages}}<li class="page-item"><a class="page-link" href="{{call .PageURL (add .Result.Page 1)}}" hx-get="{{call .PageURL (add .Result.Page 1)}}" hx-target="#events-results" hx-push-url="true">›</a></li>{{end}}
    </ul>
  </nav>
</div>
