{{if .Changes}}
<form class="card card-body mb-3" hx-get="/cis/{{.CIID}}/compare" hx-target="#compare-result" hx-swap="innerHTML">
  <div class="row g-2 align-items-end">
    <div class="col-auto">
      <label class="form-label mb-0 small">Версия A</label>
      <select name="a" class="form-select form-select-sm">
        {{range .Changes}}<option value="{{.ID}}">№{{.ID}} — {{.At}}</option>{{end}}
      </select>
    </div>
    <div class="col-auto">
      <label class="form-label mb-0 small">Версия B</label>
      <select name="b" class="form-select form-select-sm">
        {{range .Changes}}<option value="{{.ID}}">№{{.ID}} — {{.At}}</option>{{end}}
      </select>
    </div>
    <div class="col-auto"><button type="submit" class="btn btn-outline-primary btn-sm">Сравнить версии</button></div>
  </div>
</form>
<div id="compare-result"></div>
{{end}}
{{range .Changes}}
<div class="card mb-3">
  <div class="card-header d-flex flex-wrap gap-3 small">
    <strong>Набор изменений №{{.ID}}</strong>
    <span>{{.At}} UTC</span>
    <span class="badge text-bg-secondary">{{.Source}}</span>
    <span>пользователь: {{.UserID}}</span>
    {{if .RFC}}<span>RFC: {{.RFC}}</span>{{end}}
    <span class="w-100">Причина: {{.Reason}}</span>
  </div>
  <table class="table table-sm mb-0 align-middle">
    <thead><tr><th>Объект</th><th>Операция</th><th>Атрибут</th><th>Было → стало</th></tr></thead>
    <tbody>
      {{range .Entries}}
      <tr>
        <td>{{.Entity}}</td>
        <td>{{if eq .Op "I"}}<span class="badge text-bg-success">создание</span>{{else if eq .Op "D"}}<span class="badge text-bg-danger">удаление</span>{{else}}<span class="badge text-bg-warning">изменение</span>{{end}}</td>
        <td>{{.Attr}}</td>
        <td class="json-cell">
          {{if eq .Op "U"}}<span class="diff-old">{{.Old}}</span> → <span class="diff-new">{{.New}}</span>
          {{else if eq .Op "I"}}<details><summary class="small">снимок строки</summary><code>{{.New}}</code></details>
          {{else}}<details><summary class="small">снимок строки</summary><code>{{.Old}}</code></details>{{end}}
        </td>
      </tr>
      {{end}}
    </tbody>
  </table>
</div>
{{else}}
<p class="text-body-secondary">История пуста.</p>
{{end}}
