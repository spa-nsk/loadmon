<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Инвентарь по локации/стойке</h1>
  <a id="export-csv" class="btn btn-outline-secondary" href="/reports/inventory/export.csv">Выгрузить CSV (построчно)</a>
</div>

<form id="filter" class="card card-body mb-3" method="get" action="/reports/inventory">
  <div class="row g-2">
    <div class="col-md-3">
      <label class="form-label" for="location">Площадка</label>
      <select id="location" name="location" class="form-select" onchange="this.form.submit()">
        <option value="">Все площадки</option>
        {{range .Locations}}<option value="{{.Value}}" {{if eq .Value $.Location}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
  </div>
</form>

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>Площадка</th><th>Стойка</th><th>Класс</th><th class="text-end">КЕ</th></tr></thead>
      <tbody>
        {{range .Summary}}
        <tr><td>{{.location_code}}</td><td>{{.rack_position}}</td><td>{{.class_name}}</td><td class="text-end">{{.n}}</td></tr>
        {{else}}<tr><td colspan="4" class="text-muted p-3">Нет данных по выбранному фильтру.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>

<script>
  document.getElementById('export-csv').addEventListener('click', function (e) {
    var q = new URLSearchParams(new FormData(document.getElementById('filter')));
    e.currentTarget.href = '/reports/inventory/export.csv?' + q.toString();
  });
</script>
