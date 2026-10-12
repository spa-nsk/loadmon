<h1 class="h4 mb-3">Новая конфигурационная единица</h1>

<form class="card card-body" hx-post="/cis" hx-swap="none" autocomplete="off">
  <div class="row g-3">
    <div class="col-md-3">
      <label class="form-label" for="class">Класс</label>
      <select id="class" name="class" class="form-select"
              hx-get="/cis/new/fields" hx-target="#class-fields" hx-swap="innerHTML" hx-trigger="change" hx-include="this">
        {{range .Specs}}<option value="{{.Class}}" {{if eq .Class $.Spec.Class}}selected{{end}}>{{.Title}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="location_code">Площадка</label>
      <select id="location_code" name="location_code" class="form-select">
        <option value="">— выберите —</option>
        {{range .Locations}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label">Имя по профилю CMDB-NC-2026</label>
      <div class="form-control-plaintext">
        <code id="name-preview" hx-get="/cis/name-preview" hx-trigger="load, change from:closest form" hx-include="closest form"
              hx-swap="innerHTML">…</code>
        <small class="text-body-secondary d-block">Номер выдаётся при сохранении.</small>
      </div>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="environment_code">Окружение</label>
      <select id="environment_code" name="environment_code" class="form-select">
        <option value="">—</option>
        {{range .Envs}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="company_code">Компания</label>
      <select id="company_code" name="company_code" class="form-select">
        <option value="">—</option>
        {{range .Companies}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="serial_number">Серийный номер</label>
      <input id="serial_number" name="serial_number" class="form-control">
    </div>
    <div class="col-md-3">
      <label class="form-label" for="criticality">Критичность</label>
      <select id="criticality" name="criticality" class="form-select">
        <option value="">—</option>
        {{range .Criticalities}}<option value="{{.}}">{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-12">
      <label class="form-label" for="description">Описание</label>
      <textarea id="description" name="description" class="form-control" rows="2"></textarea>
    </div>
  </div>

  <h2 class="h6 mt-4">Атрибуты класса</h2>
  <div id="class-fields" class="row g-3">{{template "ci/_class_fields.tpl" .}}</div>

  <div class="row g-3 mt-2">
    <div class="col-md-8">
      <label class="form-label" for="reason">Причина создания <span class="text-danger">*</span></label>
      <input id="reason" name="reason" class="form-control" placeholder="Например: ввод сервера в эксплуатацию по заявке …">
    </div>
    <div class="col-md-4 d-flex align-items-end">
      <button class="btn btn-primary me-2" type="submit">Создать</button>
      <a class="btn btn-outline-secondary" href="/cis">Отмена</a>
    </div>
  </div>
</form>
