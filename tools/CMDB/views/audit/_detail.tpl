<table class="table table-sm mb-3 align-middle">
  <thead><tr><th>Объект</th><th>Операция</th><th>Атрибут</th><th>Было → стало</th></tr></thead>
  <tbody>
    {{range .CS.Entries}}
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
    {{else}}
    <tr><td colspan="4" class="text-center text-body-secondary py-3">Нет записей</td></tr>
    {{end}}
  </tbody>
</table>
