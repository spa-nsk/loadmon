<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Ёмкость СХД</h1>
  <a class="btn btn-outline-secondary" href="/reports/storage/export.csv">Выгрузить CSV (массивы)</a>
</div>

<div class="row g-3 mb-4">
  <div class="col-md-4">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Raw, ТБ (суммарно по массивам)</div>
      <div class="h3 mb-0">{{printf "%.1f" .RawTotal}}</div>
    </div></div>
  </div>
  <div class="col-md-4">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Usable, ТБ</div>
      <div class="h3 mb-0">{{printf "%.1f" .UsableTotal}}</div>
    </div></div>
  </div>
  <div class="col-md-4">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Размещено, ТБ (сумма томов)</div>
      <div class="h3 mb-0">{{printf "%.1f" .AllocatedTotal}}</div>
    </div></div>
  </div>
</div>

<div class="card mb-4">
  <div class="card-header">Массивы</div>
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Массив</th><th>Семейство</th><th class="text-end">Raw, ТБ</th><th class="text-end">Usable, ТБ</th><th class="text-end">Размещено, ТБ</th></tr></thead>
      <tbody>
        {{range .Arrays}}
        <tr><td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.array_family}}</td>
          <td class="text-end">{{.raw_tb}}</td><td class="text-end">{{.usable_tb}}</td><td class="text-end">{{.allocated_tb}}</td></tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Нет действующих массивов СХД.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>

<div class="card">
  <div class="card-header">Пулы</div>
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Пул</th><th>Массив</th><th>Уровень</th><th class="text-end">Ёмкость, ТБ</th><th class="text-end">Размещено, ТБ</th></tr></thead>
      <tbody>
        {{range .Pools}}
        <tr><td><a href="/cis/{{.id}}">{{.name}}</a></td><td>{{.array_name}}</td><td>{{.tier}}</td>
          <td class="text-end">{{.capacity_tb}}</td><td class="text-end">{{.allocated_tb}}</td></tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Нет действующих пулов СХД.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
