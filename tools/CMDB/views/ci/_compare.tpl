<div class="card card-body mb-3">
  <table class="table table-sm align-middle">
    <thead><tr><th>Поле</th><th>Версия A</th><th>Версия B</th></tr></thead>
    <tbody>
      {{range .Fields}}
      <tr class="{{if .Changed}}table-warning{{end}}">
        <td>{{.Name}}</td>
        <td class="json-cell">{{.A}}</td>
        <td class="json-cell">{{.B}}</td>
      </tr>
      {{else}}
      <tr><td colspan="3" class="text-center text-body-secondary py-3">Нет данных для сравнения</td></tr>
      {{end}}
    </tbody>
  </table>
</div>
