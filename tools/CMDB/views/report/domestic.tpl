<h1 class="h4 mb-3">Доля отечественного оборудования и ПО</h1>
<p class="text-muted small">Признак отечественного происхождения и ссылка на реестр задаются в
  <a href="/admin/ref/model">справочнике моделей</a> и <a href="/admin/ref/software">справочнике ПО</a>.
  КЕ без указанной модели и ПО без установок в подсчёт доли не входят — по ним признак попросту неизвестен.</p>

<div class="row g-3 mb-4">
  <div class="col-md-6">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">Оборудование (КЕ с указанной моделью)</div>
      <div class="h3 mb-0 {{if lt .HWSummary.Pct 50.0}}text-warning{{else}}text-success{{end}}">{{printf "%.1f" .HWSummary.Pct}}%</div>
      <div class="small text-muted">{{.HWSummary.Domestic}} отечественных из {{.HWSummary.Total}}</div>
    </div></div>
  </div>
  <div class="col-md-6">
    <div class="card h-100"><div class="card-body">
      <div class="text-muted small">ПО (установки на действующих КЕ)</div>
      <div class="h3 mb-0 {{if lt .SWSummary.Pct 50.0}}text-warning{{else}}text-success{{end}}">{{printf "%.1f" .SWSummary.Pct}}%</div>
      <div class="small text-muted">{{.SWSummary.Domestic}} отечественных из {{.SWSummary.Total}}</div>
    </div></div>
  </div>
</div>

<div class="card mb-4">
  <div class="card-header d-flex justify-content-between align-items-center">
    <span>Оборудование по моделям</span>
    <a class="btn btn-sm btn-outline-secondary" href="/reports/domestic/hardware.csv">Выгрузить CSV</a>
  </div>
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Производитель</th><th>Модель</th><th>Отечественная</th><th>Реестр</th><th class="text-end">КЕ</th></tr></thead>
      <tbody>
        {{range .Hardware}}
        <tr>
          <td>{{.manufacturer}}</td><td>{{.model_name}}</td>
          <td><span class="badge {{if eq .is_domestic "да"}}text-bg-success{{else}}text-bg-secondary{{end}}">{{.is_domestic}}</span></td>
          <td>{{.registry_ref}}</td><td class="text-end">{{.n}}</td>
        </tr>
        {{else}}<tr><td colspan="5" class="text-muted p-3">Нет действующих КЕ с указанной моделью.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>

<div class="card">
  <div class="card-header d-flex justify-content-between align-items-center">
    <span>ПО по продуктам</span>
    <a class="btn btn-sm btn-outline-secondary" href="/reports/domestic/software.csv">Выгрузить CSV</a>
  </div>
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Вендор</th><th>Продукт</th><th>Версия</th><th>Отечественное</th><th>Реестр</th><th class="text-end">Установок</th></tr></thead>
      <tbody>
        {{range .Software}}
        <tr>
          <td>{{.vendor}}</td><td>{{.product}}</td><td>{{.version}}</td>
          <td><span class="badge {{if eq .is_domestic "да"}}text-bg-success{{else}}text-bg-secondary{{end}}">{{.is_domestic}}</span></td>
          <td>{{.registry_ref}}</td><td class="text-end">{{.n}}</td>
        </tr>
        {{else}}<tr><td colspan="6" class="text-muted p-3">Установок ПО на действующих КЕ нет.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>
