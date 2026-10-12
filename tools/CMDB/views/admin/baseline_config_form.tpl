<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/admin/baseline-configs">Эталонные конфигурации</a></li><li class="breadcrumb-item active">{{if .IsCreate}}Новая{{else}}Правка{{end}}</li></ol></nav>

<h1 class="h4 mb-3">Эталонная конфигурация — {{if .IsCreate}}новая{{else}}правка{{end}}</h1>

<form hx-post="{{if .IsCreate}}/admin/baseline-configs{{else}}/admin/baseline-configs/{{.Config.ID}}{{end}}" hx-swap="none" autocomplete="off" class="card card-body" style="max-width: 760px;">
  <div class="row g-3">
    <div class="col-md-6">
      <label class="form-label" for="f-name">Имя *</label>
      <input id="f-name" name="name" class="form-control" value="{{.Config.Name}}" required>
    </div>
    <div class="col-md-6">
      <label class="form-label" for="f-role">Роль (свободная подпись)</label>
      <input id="f-role" name="role" class="form-control" value="{{.Config.Role}}" placeholder="например, сервер БД">
    </div>
    <div class="col-12">
      <label class="form-label" for="f-desc">Описание</label>
      <input id="f-desc" name="description" class="form-control" value="{{.Config.Description}}">
    </div>

    <div class="col-md-3">
      <label class="form-label" for="f-class">Класс КЕ *</label>
      <select id="f-class" name="ci_class" class="form-select" required>
        <option value="">— выберите —</option>
        {{range .Classes}}<option value="{{.Value}}" {{if eq .Value $.Config.CIClass}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="f-env">Окружение</label>
      <select id="f-env" name="environment_code" class="form-select">
        <option value="">любое</option>
        {{range .Environments}}<option value="{{.Value}}" {{if eq .Value $.Config.EnvironmentCode}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="f-company">Компания</label>
      <select id="f-company" name="company_code" class="form-select">
        <option value="">любая</option>
        {{range .Companies}}<option value="{{.Value}}" {{if eq .Value $.Config.CompanyCode}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label" for="f-loc">Площадка</label>
      <select id="f-loc" name="location_code" class="form-select">
        <option value="">любая</option>
        {{range .Locations}}<option value="{{.Value}}" {{if eq .Value $.Config.LocationCode}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>

    <div class="col-12">
      <label class="form-label" for="f-attrs">Требуемые атрибуты (JSON: имя колонки таблицы-расширения → значение)</label>
      <textarea id="f-attrs" name="attrs_json" class="form-control font-monospace" rows="5">{{.AttrsJSON}}</textarea>
      <div class="form-text">Например: <code>{"os_family": "Linux", "template": false}</code>. Имена сверяются со списком реальных колонок класса при сохранении.</div>
    </div>

    <div class="col-12">
      <label class="form-label" for="f-sw">Обязательное ПО (справочник)</label>
      <select id="f-sw" name="software_ids" class="form-select" multiple size="6">
        {{$sel := .SelectedSoftware}}
        {{range .SoftwareOptions}}<option value="{{.Value}}" {{if index $sel .Value}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
      <div class="form-text">Ctrl/Cmd + клик — выбрать несколько.</div>
    </div>

    <div class="col-12">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" id="f-enabled" name="enabled" value="1" {{if .Config.Enabled}}checked{{end}}>
        <label class="form-check-label" for="f-enabled">Включена (учитывается в отчёте отклонений)</label>
      </div>
    </div>
  </div>

  <div class="d-flex gap-2 mt-3">
    <button type="submit" class="btn btn-primary">Сохранить</button>
    <a class="btn btn-outline-secondary" href="/admin/baseline-configs">Отмена</a>
  </div>
</form>
