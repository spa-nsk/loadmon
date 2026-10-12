<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Запрещённое/неразрешённое ПО</h1>
  <a class="btn btn-outline-secondary" href="/reports/software/export.csv">Выгрузить CSV</a>
</div>
<p class="text-muted small">Установки ПО с политикой <span class="badge text-bg-danger">forbidden</span> или
  <span class="badge text-bg-warning">review</span> на действующих КЕ. Политику меняют в
  <a href="/admin/ref/software">справочнике ПО</a>. Можно посмотреть на дату — <code>?asof=</code> в адресе,
  как и везде (E6.1).</p>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>КЕ</th><th>Локация</th><th>Производитель</th><th>Продукт</th><th>Версия</th><th>Политика</th><th>Путь установки</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.location_code}}</td>
          <td>{{.vendor}}</td><td>{{.product}}</td><td>{{.version}}</td>
          <td><span class="badge {{if eq .policy "forbidden"}}text-bg-danger{{else}}text-bg-warning{{end}}">{{.policy}}</span></td>
          <td>{{.install_path}}</td>
        </tr>
        {{else}}<tr><td colspan="7" class="text-muted p-3">Запрещённого или неразрешённого ПО не найдено.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
