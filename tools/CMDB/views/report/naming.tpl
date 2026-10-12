<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Отклонения именования</h1>
  <a class="btn btn-outline-secondary" href="/reports/naming/export.csv">Выгрузить CSV</a>
</div>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Имя</th><th>Класс</th><th>Статус</th><th>Локация</th><th>Исходное имя источника</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.class_name}}</td>
          <td><span class="badge text-bg-warning">{{.naming_status}}</span></td>
          <td>{{.location_code}}</td><td>{{.source_name}}</td>
        </tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Все действующие КЕ соответствуют профилю именования.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
