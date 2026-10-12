<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">КЕ без владельца</h1>
  <a class="btn btn-outline-secondary" href="/reports/no-owner/export.csv">Выгрузить CSV</a>
</div>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Имя</th><th>Класс</th><th>Локация</th><th>Компания</th><th>Критичность</th><th>Стадия ЖЦ</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.class_name}}</td><td>{{.location_code}}</td>
          <td>{{.company_code}}</td><td>{{.criticality}}</td><td>{{.lifecycle_stage}}</td>
        </tr>
        {{else}}<tr><td colspan="6" class="text-muted p-3">У всех действующих КЕ указан владелец.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
