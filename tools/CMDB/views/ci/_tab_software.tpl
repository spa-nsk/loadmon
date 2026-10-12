<table class="table table-sm align-middle">
  <thead><tr><th>Производитель</th><th>Продукт</th><th>Версия</th><th>Путь</th><th>Политика</th><th>Отеч.</th><th>Обнаружено</th><th></th></tr></thead>
  <tbody>
    {{range .Installs}}
    <tr>
      <td>{{.Vendor}}</td><td>{{.Product}}</td>
      <td>{{if .Version}}{{.Version}}{{else}}{{.RefVersion}}{{end}}</td>
      <td>{{.InstallPath}}</td>
      <td><span class="badge {{if eq .Policy "forbidden"}}text-bg-danger{{else if eq .Policy "review"}}text-bg-warning{{else}}text-bg-success{{end}}">{{.Policy}}</span></td>
      <td>{{if .IsDomestic}}да{{end}}</td>
      <td>{{if .DetectedAt}}{{.DetectedAt}}{{else}}<span class="text-muted">вручную</span>{{end}}</td>
      <td>
        <details>
          <summary class="small">снять</summary>
          <form class="mt-2" style="min-width: 16rem" hx-post="/software-installs/{{.ID}}/remove" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
            <input type="hidden" name="ci" value="{{$.CIID}}">
            <input type="hidden" name="row_version" value="{{.RowVersion}}">
            <input class="form-control form-control-sm mb-1" name="reason" placeholder="Причина *">
            <button class="btn btn-outline-danger btn-sm" type="submit">Снять</button>
          </form>
        </details>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="8" class="text-center text-body-secondary py-3">ПО не зарегистрировано</td></tr>
    {{end}}
  </tbody>
</table>

<button type="button" class="btn btn-primary btn-sm mt-3" data-bs-toggle="modal" data-bs-target="#modal-add-software">Зарегистрировать установку</button>

<div class="modal fade" id="modal-add-software" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <form hx-post="/cis/{{.CIID}}/software" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
        <div class="modal-header">
          <h3 class="modal-title h6">Зарегистрировать установку ПО</h3>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
        </div>
        <div class="modal-body row g-2">
          <div class="col-12">
            <select name="software_id" class="form-select" aria-label="ПО">
              <option value="">— выберите из справочника —</option>
              {{range .SoftwareOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
            </select>
          </div>
          <div class="col-md-6"><input class="form-control" name="version" placeholder="Версия установки (если отличается от справочника)"></div>
          <div class="col-md-6"><input class="form-control" name="install_path" placeholder="Путь установки"></div>
          <div class="col-12"><input class="form-control" name="reason" placeholder="Причина *"></div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Отмена</button>
          <button class="btn btn-primary" type="submit">Зарегистрировать</button>
        </div>
      </form>
    </div>
  </div>
</div>
