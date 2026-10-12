<table class="table table-sm align-middle">
  <thead><tr><th>Компонент</th><th>Слот</th><th>Производитель</th><th>P/N</th><th>Серийный №</th><th>Прошивка</th><th>Ёмкость</th><th>Статус</th><th></th></tr></thead>
  <tbody>
    {{range .Components}}
    <tr class="{{if ne (index .Row "status") "installed"}}row-muted{{end}}">
      <td class="tree-cell" style="padding-left: {{indent .Depth}}rem">{{index .Row "type_name"}}</td>
      <td>{{index .Row "slot"}}</td>
      <td>{{index .Row "manufacturer"}}</td>
      <td>{{index .Row "part_number"}}</td>
      <td>{{index .Row "serial_number"}}</td>
      <td>{{index .Row "firmware"}}</td>
      <td>{{index .Row "capacity"}}</td>
      <td>{{index .Row "status"}}</td>
      <td>
        {{if eq (index .Row "status") "installed"}}
        <details>
          <summary class="small">заменить</summary>
          <form class="mt-2" style="min-width: 22rem" hx-post="/components/{{index .Row "id"}}/replace" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
            <input type="hidden" name="ci" value="{{$.CIID}}">
            <input type="hidden" name="row_version" value="{{index .Row "row_version"}}">
            <input class="form-control form-control-sm mb-1" name="manufacturer" placeholder="Производитель">
            <input class="form-control form-control-sm mb-1" name="part_number" placeholder="P/N">
            <input class="form-control form-control-sm mb-1" name="serial_number" placeholder="Серийный №">
            <input class="form-control form-control-sm mb-1" name="firmware" placeholder="Прошивка">
            <input class="form-control form-control-sm mb-1" name="reason" placeholder="Причина замены *">
            <button class="btn btn-outline-warning btn-sm" type="submit">Заменить</button>
          </form>
        </details>
        {{end}}
      </td>
    </tr>
    {{else}}
    <tr><td colspan="9" class="text-center text-body-secondary py-3">Компонентов нет</td></tr>
    {{end}}
  </tbody>
</table>

<button type="button" class="btn btn-primary btn-sm mt-3" data-bs-toggle="modal" data-bs-target="#modal-add-component">Добавить компонент</button>

<div class="modal fade" id="modal-add-component" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <form hx-post="/cis/{{.CIID}}/components" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
        <div class="modal-header">
          <h3 class="modal-title h6">Добавить компонент</h3>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
        </div>
        <div class="modal-body row g-2">
          <div class="col-md-6">
            <select name="component_type" class="form-select" aria-label="Тип компонента">
              <option value="">— тип —</option>
              {{range .ComponentTypes}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
            </select>
          </div>
          <div class="col-md-6">
            <select name="parent_id" class="form-select" aria-label="Родитель">
              <option value="">без родителя</option>
              {{range .Components}}{{if eq (index .Row "status") "installed"}}<option value="{{index .Row "id"}}">{{index .Row "type_name"}} {{index .Row "slot"}}</option>{{end}}{{end}}
            </select>
          </div>
          <div class="col-md-4"><input class="form-control" name="slot" placeholder="Слот"></div>
          <div class="col-md-8"><input class="form-control" name="manufacturer" placeholder="Производитель"></div>
          <div class="col-md-6"><input class="form-control" name="part_number" placeholder="P/N"></div>
          <div class="col-md-6"><input class="form-control" name="serial_number" placeholder="Серийный №"></div>
          <div class="col-md-6"><input class="form-control" name="firmware" placeholder="Прошивка"></div>
          <div class="col-md-3"><input class="form-control" name="capacity_value" placeholder="Ёмк." inputmode="decimal"></div>
          <div class="col-md-3"><input class="form-control" name="capacity_unit" placeholder="ед."></div>
          <div class="col-12"><input class="form-control" name="reason" placeholder="Причина *"></div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Отмена</button>
          <button class="btn btn-primary" type="submit">Добавить</button>
        </div>
      </form>
    </div>
  </div>
</div>
