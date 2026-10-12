<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Свежесть инвентаризации</h1>
  <a class="btn btn-outline-secondary" href="/reports/stale-inventory/export.csv">Выгрузить CSV</a>
</div>
<p class="text-muted small">Действующие КЕ, которые ни один источник не подтверждал более 30 дней (KPI из intent §9),
  включая никогда не подтверждённые (только ручной ввод, без источника вовсе — см. дашборд
  <a href="/quality">качества данных</a> для доли по всей базе).</p>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Имя</th><th>Класс</th><th>Локация</th><th>Последнее подтверждение (UTC)</th><th class="text-end">Дней без подтверждения</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.class_name}}</td><td>{{.location_code}}</td>
          <td>{{if .last_seen_at}}{{.last_seen_at}}{{else}}<span class="text-muted">никогда</span>{{end}}</td>
          <td class="text-end">{{.days_stale}}</td>
        </tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Все действующие КЕ подтверждены за последние 30 дней.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
