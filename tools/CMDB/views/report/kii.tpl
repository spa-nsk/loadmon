<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Объекты КИИ и их состав</h1>
  <a class="btn btn-outline-secondary" href="/reports/kii/export.csv">Выгрузить CSV</a>
</div>
<p class="text-muted small">Действующие КЕ с признаком объекта критической информационной инфраструктуры
  (<code>kii_object</code>) и связанные с ними КЕ — состав. Признак и категория задаются на вкладке «Общие»
  карточки КЕ.</p>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Объект КИИ</th><th>Категория</th><th>Класс объекта</th><th>Состав: КЕ</th><th>Класс</th><th>Тип связи</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.kii_id}}">{{.kii_name}}</a></td>
          <td>{{if .kii_category}}{{.kii_category}}{{else}}<span class="text-muted">не указана</span>{{end}}</td>
          <td>{{.kii_class_name}}</td>
          <td>{{if .member_id}}<a href="/cis/{{.member_id}}">{{.member_name}}</a>{{else}}<span class="text-muted">нет связей</span>{{end}}</td>
          <td>{{.member_class_name}}</td>
          <td>{{.rel_type}}</td>
        </tr>
        {{else}}<tr><td colspan="6" class="text-muted p-3">Объектов КИИ не найдено.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
