<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Качество данных</h1>
  <span class="text-muted small">Снимок на {{.Snap.ComputedAt.Format "2006-01-02 15:04"}} UTC</span>
</div>

<div class="row g-3 mb-4">
  <div class="col-md-3">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Полнота обязательных атрибутов</div>
      {{if .AttrsPctOK}}<div class="h3 mb-0 {{if lt .AttrsPct 95.0}}text-danger{{else}}text-success{{end}}">{{printf "%.1f" .AttrsPct}}%</div>
      <div class="small text-muted">цель ≥ 95 %</div>{{else}}<div class="h3 mb-0 text-muted">н/д</div>{{end}}
    </div></div>
  </div>
  <div class="col-md-3">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Подтверждено источником за 7 дней</div>
      {{if .ConfirmedPctOK}}<div class="h3 mb-0 {{if lt .ConfirmedPct 90.0}}text-danger{{else}}text-success{{end}}">{{printf "%.1f" .ConfirmedPct}}%</div>
      <div class="small text-muted">цель ≥ 90 % среди КЕ с источником</div>{{else}}<div class="h3 mb-0 text-muted">н/д</div>{{end}}
    </div></div>
  </div>
  <div class="col-md-3">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Дубликаты (серийный/инв. номер)</div>
      <div class="h3 mb-0 {{if gt .DuplicatesCI 0}}text-warning{{else}}text-success{{end}}">{{.DuplicatesCI}}</div>
      <div class="small text-muted">из {{.Snap.Staleness.TotalActive}} действующих КЕ</div>
    </div></div>
  </div>
  <div class="col-md-3">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Открытые конфликты старше 14 дней</div>
      <div class="h3 mb-0 {{if gt .Snap.Conflicts.OpenOver14d 0}}text-danger{{else}}text-success{{end}}">{{.Snap.Conflicts.OpenOver14d}}</div>
      <div class="small text-muted">всего открытых: {{.Snap.Conflicts.Open}} · цель — 0</div>
    </div></div>
  </div>
</div>

<div class="row g-3 mb-4">
  <div class="col-lg-6">
    <div class="card h-100">
      <div class="card-header">Полнота атрибутов по классу и стадии ЖЦ</div>
      <div class="card-body p-0">
        {{if .Snap.Completeness}}
        <table class="table table-sm mb-0">
          <thead><tr><th>Класс</th><th>Стадия</th><th class="text-end">Заполнено</th><th class="text-end">Доля</th></tr></thead>
          <tbody>
            {{range .Snap.Completeness}}
            <tr>
              <td>{{.ClassName}}</td><td>{{.Stage}}</td>
              <td class="text-end">{{.Complete}} / {{.Total}}</td>
              <td class="text-end {{if lt .Pct 95.0}}text-danger{{else}}text-success{{end}}">{{printf "%.0f" .Pct}}%</td>
            </tr>
            {{end}}
          </tbody>
        </table>
        {{else}}<div class="p-3 text-muted small">Ни у одного класса на текущих стадиях нет обязательных атрибутов — нечего проверять.</div>{{end}}
      </div>
    </div>
  </div>
  <div class="col-lg-6">
    <div class="card h-100">
      <div class="card-header">Соответствие профилю именования</div>
      <div class="card-body p-0">
        <table class="table table-sm mb-0">
          <thead><tr><th>Статус</th><th class="text-end">КЕ</th><th class="text-end">Доля</th></tr></thead>
          <tbody>
            {{$total := .Snap.NamingTotal}}
            {{range .Snap.Naming}}
            <tr>
              <td>{{.Status}}</td><td class="text-end">{{.Count}}</td>
              <td class="text-end {{if and (ne .Status "compliant") (gt .Count 0)}}text-warning{{end}}">{{printf "%.1f" (.Pct $total)}}%</td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>

<div class="row g-3 mb-4">
  <div class="col-lg-6">
    <div class="card h-100">
      <div class="card-header">Устаревание — когда источник последний раз подтверждал КЕ</div>
      <div class="card-body p-0">
        <table class="table table-sm mb-0">
          <tbody>
            <tr><td>Подтверждена за 7 дней</td><td class="text-end">{{.Snap.Staleness.Fresh7d}}</td></tr>
            <tr><td>7–30 дней</td><td class="text-end">{{.Snap.Staleness.Stale30d}}</td></tr>
            <tr><td>Старше 30 дней</td><td class="text-end">{{.Snap.Staleness.StaleOlder}}</td></tr>
            <tr class="text-muted"><td>Без источника (введена вручную)</td><td class="text-end">{{.Snap.Staleness.NoSource}}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <div class="col-lg-6">
    <div class="card h-100">
      <div class="card-header">Открытые конфликты импорта по типу</div>
      <div class="card-body p-0">
        {{if .Snap.Conflicts.ByKind}}
        <table class="table table-sm mb-0">
          <thead><tr><th>Тип</th><th class="text-end">Открыто</th></tr></thead>
          <tbody>{{range $kind, $n := .Snap.Conflicts.ByKind}}<tr><td>{{$kind}}</td><td class="text-end">{{$n}}</td></tr>{{end}}</tbody>
        </table>
        {{else}}<div class="p-3 text-muted small">Открытых конфликтов нет.</div>{{end}}
      </div>
    </div>
  </div>
</div>

{{if .Snap.Duplicates}}
<div class="card mb-4">
  <div class="card-header">Возможные дубликаты</div>
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Признак</th><th>Значение</th><th>КЕ</th></tr></thead>
      <tbody>
        {{range .Snap.Duplicates}}
        {{$names := .Names}}
        <tr>
          <td>{{.Field}}</td><td class="font-monospace">{{.Value}}</td>
          <td>{{range $i, $id := .CIIDs}}{{if $i}}, {{end}}<a href="/cis/{{$id}}">{{index $names $i}}</a>{{end}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</div>
{{end}}

<div class="card">
  <div class="card-header">Динамика (последние 30 снимков)</div>
  <div class="card-body p-0" style="max-height: 20rem; overflow-y: auto;">
    <table class="table table-sm mb-0">
      <thead><tr><th>Дата</th><th class="text-end">Действ. КЕ</th><th class="text-end">Атрибуты</th><th class="text-end">Именование compliant</th><th class="text-end">Подтверждено 7д</th><th class="text-end">Дубликаты</th><th class="text-end">Конфликты &gt;14д</th></tr></thead>
      <tbody>
        {{range .History}}
        <tr>
          <td>{{.Date}}</td><td class="text-end">{{.TotalActiveCI}}</td>
          <td class="text-end">{{.AttrsPctDisplay}}</td>
          <td class="text-end">{{printf "%.1f" .NamingCompliantPct}}%</td>
          <td class="text-end">{{.ConfirmedPctDisplay}}</td>
          <td class="text-end">{{.DuplicatesCI}}</td>
          <td class="text-end {{if gt .OpenConflictsOver14d 0}}text-danger{{end}}">{{.OpenConflictsOver14d}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</div>
