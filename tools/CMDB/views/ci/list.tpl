<link href="/static/vendor/tabulator.min.css" rel="stylesheet">
<script src="/static/vendor/tabulator.min.js"></script>

<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Конфигурационные единицы</h1>
  <div class="d-flex gap-2">
    <div class="dropdown">
      <button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown">Представления</button>
      <div class="dropdown-menu p-2" style="min-width: 16rem" id="views-menu">
        <div class="input-group input-group-sm mb-2">
          <input type="text" class="form-control" id="view-name" placeholder="Название представления">
          <button class="btn btn-outline-primary" type="button" id="view-save">Сохранить текущее</button>
        </div>
        <div id="views-list" class="small"></div>
      </div>
    </div>
    <div class="dropdown">
      <button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown">Столбцы</button>
      <ul class="dropdown-menu p-2" id="columns-menu"></ul>
    </div>
    <div class="dropdown">
      <button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown">Экспорт</button>
      <ul class="dropdown-menu">
        <li><a class="dropdown-item" id="export-csv" href="/cis/export.csv">CSV (по текущему фильтру)</a></li>
        <li><a class="dropdown-item" id="export-xlsx" href="/cis/export.xlsx">XLSX (по текущему фильтру)</a></li>
      </ul>
    </div>
    {{if index .Perms "ci.write"}}<a class="btn btn-primary" href="/cis/new">Создать КЕ</a>{{end}}
  </div>
</div>

<form id="filter" class="filter-bar card card-body mb-3">
  <div class="row g-2">
    <div class="col-md-3">
      <input id="q" name="q" class="form-control" placeholder="Имя, серийный №, инв. №, точный IP" value="{{.Filter.Q}}">
    </div>
    <div class="col-md-2">
      <select name="class" class="form-select" aria-label="Класс">
        <option value="">Все классы</option>
        {{range .Classes}}<option value="{{.Value}}" {{if eq .Value $.Filter.Class}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-2">
      <select name="location" class="form-select" aria-label="Площадка">
        <option value="">Все площадки</option>
        {{range .Locations}}<option value="{{.Value}}" {{if eq .Value $.Filter.Location}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-1">
      <select name="env" class="form-select" aria-label="Окружение">
        <option value="">Окр.</option>
        {{range .Envs}}<option value="{{.Value}}" {{if eq .Value $.Filter.Env}}selected{{end}}>{{.Value}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-2">
      <select name="stage" class="form-select" aria-label="Стадия">
        <option value="" {{if eq $.Filter.Stage ""}}selected{{end}}>Действующие</option>
        <option value="*" {{if eq $.Filter.Stage "*"}}selected{{end}}>Все стадии</option>
        {{range .Stages}}<option value="{{.}}" {{if eq . $.Filter.Stage}}selected{{end}}>{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-1">
      <select name="naming" class="form-select" aria-label="Имя">
        <option value="">Имя</option>
        {{range .NamingStatuses}}<option value="{{.}}" {{if eq . $.Filter.NamingStatus}}selected{{end}}>{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-1">
      <select name="per" class="form-select" aria-label="На странице">
        <option value="25" {{if eq $.Filter.PerPage 25}}selected{{end}}>25</option>
        <option value="50" {{if eq $.Filter.PerPage 50}}selected{{end}}>50</option>
        <option value="100" {{if eq $.Filter.PerPage 100}}selected{{end}}>100</option>
        <option value="200" {{if eq $.Filter.PerPage 200}}selected{{end}}>200</option>
      </select>
    </div>
  </div>
</form>

{{if .BulkApplied}}<div class="alert alert-success">Массовая операция: применено {{.BulkApplied}}, пропущено {{.BulkSkipped}}.</div>{{end}}

<form id="bulk-form" class="card card-body mb-3" hx-post="/cis/bulk/preview" hx-target="#bulk-panel" hx-swap="innerHTML">
  <div class="row g-2 align-items-end">
    <div class="col-auto">
      <label class="form-label mb-0 small">Массовая операция для выбранных строк</label>
      <select name="bulk_field" class="form-select form-select-sm" data-bulk-field>
        <option value="stage">Стадия ЖЦ</option>
        <option value="company">Владелец (компания)</option>
        <option value="location">Площадка</option>
      </select>
    </div>
    <div class="col-auto" data-bulk-value="stage">
      <label class="form-label mb-0 small">Новая стадия</label>
      <select name="bulk_stage" class="form-select form-select-sm">
        {{range .Stages}}<option value="{{.}}">{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-auto" data-bulk-value="company" hidden>
      <label class="form-label mb-0 small">Новый владелец</label>
      <select name="bulk_company" class="form-select form-select-sm">
        <option value="">— сбросить —</option>
        {{range .Companies}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-auto" data-bulk-value="location" hidden>
      <label class="form-label mb-0 small">Новая площадка</label>
      <select name="bulk_location" class="form-select form-select-sm">
        <option value="">— сбросить —</option>
        {{range .Locations}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-auto">
      <button type="submit" class="btn btn-outline-primary btn-sm">Предпросмотр для выбранных</button>
    </div>
    <div class="col-auto"><small class="text-body-secondary" id="selection-count">Ничего не выбрано</small></div>
  </div>
</form>
<div id="bulk-panel"></div>

<div id="ci-grid"></div>

<script>
(function () {
  'use strict';

  var stageBadge = {Operational: 'text-bg-success', Commissioning: 'text-bg-warning', Maintenance: 'text-bg-warning',
    Decommissioning: 'text-bg-warning', Retired: 'text-bg-dark', Disposed: 'text-bg-dark'};
  var namingBadge = {compliant: 'text-bg-success', legacy: 'text-bg-info', noncompliant: 'text-bg-danger'};
  function badge(map, v) { return '<span class="badge ' + (map[v] || 'text-bg-secondary') + '">' + v + '</span>'; }

  // Соответствие поля колонки грида короткому имени сортировки, которое понимает /cis/data
  // (internal/controllers/ui/ci.go listFilter -> ci.ListFilter.Sort, internal/services/ci/query.go sortColumns).
  var sortFieldMap = {name: 'name', class_name: 'class', location_code: 'location', environment_code: 'env',
    lifecycle_stage: 'stage', naming_status: 'naming', updated_at: 'updated'};

  var filterForm = document.getElementById('filter');

  function filterParams() {
    var p = {};
    new FormData(filterForm).forEach(function (v, k) { if (v !== '') { p[k] = v; } });
    return p;
  }

  var table = new Tabulator('#ci-grid', {
    layout: 'fitDataStretch',
    height: '70vh',
    selectableRows: true,
    ajaxURL: '/cis/data',
    sortMode: 'remote',
    filterMode: 'remote',
    paginationMode: 'remote',
    pagination: true,
    paginationSize: {{.Filter.PerPage}},
    paginationSizeSelector: false,
    ajaxRequestFunc: function (url, config, params) {
      var q = new URLSearchParams(filterParams());
      q.set('page', params.page || 1);
      q.set('per', filterForm.elements['per'].value || '25');
      if (params.sorters && params.sorters.length) {
        q.set('sort', sortFieldMap[params.sorters[0].field] || 'name');
        q.set('dir', params.sorters[0].dir || 'asc');
      }
      return fetch(url + '?' + q.toString(), {headers: {'X-Requested-With': 'fetch'}}).then(function (r) {
        if (!r.ok) { throw new Error('HTTP ' + r.status); }
        return r.json();
      });
    },
    columns: [
      {formatter: 'rowSelection', titleFormatter: 'rowSelection', hozAlign: 'center', headerSort: false, width: 40, cellClick: function (e, cell) { cell.getRow().toggleSelect(); }},
      {title: 'Имя', field: 'name', formatter: function (cell) {
        var d = cell.getRow().getData();
        return '<a href="/cis/' + d.id + '">' + cell.getValue() + '</a>';
      }},
      {title: 'Класс', field: 'class_name'},
      {title: 'Площадка', field: 'location_code'},
      {title: 'Окружение', field: 'environment_code'},
      {title: 'Стадия', field: 'lifecycle_stage', formatter: function (cell) { return badge(stageBadge, cell.getValue()); }},
      {title: 'Имя по профилю', field: 'naming_status', formatter: function (cell) { return badge(namingBadge, cell.getValue()); }},
      {title: 'Серийный №', field: 'serial_number', headerSort: false},
      {title: 'Изменена (UTC)', field: 'updated_at'}
    ],
  });

  table.on('rowSelectionChanged', function (data) {
    var el = document.getElementById('selection-count');
    el.textContent = data.length ? ('Выбрано: ' + data.length) : 'Ничего не выбрано';
  });

  // Столбцы: чекбокс-меню показать/скрыть (E3.3), не трогает колонку выбора строк и "Имя".
  table.on('tableBuilt', function () {
    var menu = document.getElementById('columns-menu');
    table.getColumns().forEach(function (col) {
      var field = col.getField();
      if (!field || field === 'name') { return; }
      var li = document.createElement('li');
      var label = document.createElement('label');
      label.className = 'dropdown-item form-check py-1';
      var input = document.createElement('input');
      input.type = 'checkbox'; input.className = 'form-check-input me-1'; input.checked = true;
      input.addEventListener('change', function () { input.checked ? col.show() : col.hide(); });
      label.appendChild(input);
      label.appendChild(document.createTextNode(col.getDefinition().title));
      li.appendChild(label);
      menu.appendChild(li);
    });
  });

  filterForm.addEventListener('change', function () {
    table.setPageSize(parseInt(filterForm.elements['per'].value, 10) || 25);
    table.setPage(1);
  });
  var qDebounce;
  document.getElementById('q').addEventListener('keyup', function () {
    clearTimeout(qDebounce);
    qDebounce = setTimeout(function () { table.setPage(1); }, 400);
  });

  ['export-csv', 'export-xlsx'].forEach(function (id) {
    var a = document.getElementById(id);
    a.addEventListener('click', function () {
      var q = new URLSearchParams(filterParams());
      a.href = a.href.split('?')[0] + '?' + q.toString();
    });
  });

  // Массовые операции: перед отправкой формы — id выбранных строк как скрытые поля (E3.6).
  document.getElementById('bulk-form').addEventListener('submit', function (e) {
    var ids = table.getSelectedData().map(function (r) { return r.id; });
    this.querySelectorAll('input[name="ids"]').forEach(function (el) { el.remove(); });
    if (ids.length === 0) { e.preventDefault(); return; }
    var self = this;
    ids.forEach(function (id) {
      var input = document.createElement('input');
      input.type = 'hidden'; input.name = 'ids'; input.value = id;
      self.appendChild(input);
    });
  });

  // Сохранённые представления (E3.3): фильтр + видимость столбцов в localStorage браузера —
  // per-viewer удобство, не общие данные (пользователей/сессий пока нет, см. E7).
  var STORAGE_KEY = 'cmdb.ci.views';
  function loadViews() { try { return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}'); } catch (e) { return {}; } }
  function saveViews(v) { try { localStorage.setItem(STORAGE_KEY, JSON.stringify(v)); } catch (e) { /* недоступно */ } }
  function renderViews() {
    var views = loadViews();
    var box = document.getElementById('views-list');
    box.innerHTML = '';
    Object.keys(views).sort().forEach(function (name) {
      var row = document.createElement('div');
      row.className = 'd-flex justify-content-between align-items-center py-1';
      var applyBtn = document.createElement('button');
      applyBtn.type = 'button'; applyBtn.className = 'btn btn-link btn-sm p-0'; applyBtn.textContent = name;
      applyBtn.addEventListener('click', function () { applyView(views[name]); });
      var delBtn = document.createElement('button');
      delBtn.type = 'button'; delBtn.className = 'btn btn-sm btn-outline-danger py-0 px-1'; delBtn.textContent = '×';
      delBtn.addEventListener('click', function () { delete views[name]; saveViews(views); renderViews(); });
      row.appendChild(applyBtn); row.appendChild(delBtn);
      box.appendChild(row);
    });
    if (!Object.keys(views).length) { box.innerHTML = '<div class="text-body-secondary">Пока нет сохранённых</div>'; }
  }
  function applyView(v) {
    Object.keys(v.filter || {}).forEach(function (k) {
      var el = filterForm.elements[k];
      if (el) { el.value = v.filter[k]; }
    });
    (v.hiddenColumns || []).forEach(function (field) {
      var col = table.getColumn(field);
      if (col) { col.hide(); }
    });
    table.getColumns().forEach(function (col) {
      var field = col.getField();
      if (field && (v.hiddenColumns || []).indexOf(field) === -1) { col.show(); }
    });
    document.querySelectorAll('#columns-menu input[type=checkbox]').forEach(function (cb, i) {
      var col = table.getColumns().filter(function (c) { return c.getField() && c.getField() !== 'name'; })[i];
      if (col) { cb.checked = col.isVisible(); }
    });
    table.setPage(1);
  }
  document.getElementById('view-save').addEventListener('click', function () {
    var name = document.getElementById('view-name').value.trim();
    if (!name) { return; }
    var hidden = table.getColumns().filter(function (c) { return c.getField() && !c.isVisible(); }).map(function (c) { return c.getField(); });
    var views = loadViews();
    views[name] = {filter: filterParams(), hiddenColumns: hidden};
    saveViews(views);
    document.getElementById('view-name').value = '';
    renderViews();
  });
  renderViews();
})();
</script>
