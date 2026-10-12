<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Legacy-имена</h1>
  <a class="btn btn-outline-secondary" href="/reports/legacy-names/export.csv">Выгрузить CSV</a>
</div>
<p class="text-muted small">Действующие КЕ, названные по прежнему профилю (Guide 2019, статус <code>legacy</code>), с
  предложенным именем по действующему профилю CMDB-NC-2026. Предложение — только шаблон (<code>NNNNN</code> —
  разряды номера): настоящий номер выделяет seqalloc в момент переименования, не отчёт.</p>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Текущее имя</th><th>Класс</th><th>Локация</th><th>Предложенное имя</th><th>Исходное имя источника</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.class_name}}</td><td>{{.location_code}}</td>
          <td>{{if .suggested_name}}<code>{{.suggested_name}}</code>{{else}}<span class="text-muted">н/д</span>{{end}}</td>
          <td>{{.source_name}}</td>
        </tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Legacy-имён среди действующих КЕ не найдено.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
