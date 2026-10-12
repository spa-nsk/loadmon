<div class="row g-4">
  <div class="col-lg-8">
    <form class="card card-body" hx-post="/cis/{{.CIID}}/update" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
      <input type="hidden" name="row_version" value="{{index .D.CI "row_version"}}">
      <input type="hidden" name="kii_object_present" value="1">
      <div class="row g-3">
        <div class="col-md-4"><label class="form-label">Имя</label><input class="form-control" value="{{index .D.CI "name"}}" disabled></div>
        <div class="col-md-4"><label class="form-label">Класс</label><input class="form-control" value="{{.D.ClassName}}" disabled></div>
        <div class="col-md-4"><label class="form-label">Площадка</label><input class="form-control" value="{{index .D.CI "location_code"}}" disabled></div>

        <div class="col-md-4">
          <label class="form-label" for="g_env">Окружение</label>
          <select id="g_env" name="environment_code" class="form-select">
            <option value="">—</option>
            {{range .Envs}}<option value="{{.Value}}" {{if eq .Value (index $.D.CI "environment_code")}}selected{{end}}>{{.Label}}</option>{{end}}
          </select>
        </div>
        <div class="col-md-4">
          <label class="form-label" for="g_crit">Критичность</label>
          <select id="g_crit" name="criticality" class="form-select">
            <option value="">—</option>
            {{range .Criticalities}}<option value="{{.}}" {{if eq . (index $.D.CI "criticality")}}selected{{end}}>{{.}}</option>{{end}}
          </select>
        </div>
        <div class="col-md-4">
          <label class="form-label" for="g_own">Принадлежность</label>
          <select id="g_own" name="ownership" class="form-select">
            <option value="">—</option>
            {{range .Ownerships}}<option value="{{.}}" {{if eq . (index $.D.CI "ownership")}}selected{{end}}>{{.}}</option>{{end}}
          </select>
        </div>
        <div class="col-md-4"><label class="form-label" for="g_serial">Серийный номер</label><input id="g_serial" name="serial_number" class="form-control" value="{{index .D.CI "serial_number"}}"></div>
        <div class="col-md-4"><label class="form-label" for="g_asset">Инвентарный номер</label><input id="g_asset" name="asset_tag" class="form-control" value="{{index .D.CI "asset_tag"}}"></div>
        <div class="col-md-4"><label class="form-label" for="g_rack">Позиция в стойке</label><input id="g_rack" name="rack_position" class="form-control" value="{{index .D.CI "rack_position"}}"></div>
        <div class="col-md-4"><label class="form-label" for="g_prov">Поставщик услуги</label><input id="g_prov" name="service_provider" class="form-control" value="{{index .D.CI "service_provider"}}"></div>
        <div class="col-md-4">
          <div class="form-check mt-4">
            <input class="form-check-input" type="checkbox" id="g_kii" name="kii_object" value="1" {{if eq (index .D.CI "kii_object") "true"}}checked{{end}}>
            <label class="form-check-label" for="g_kii">Объект КИИ</label>
          </div>
        </div>
        <div class="col-md-4"><label class="form-label" for="g_kiic">Категория КИИ</label><input id="g_kiic" name="kii_category" class="form-control" value="{{index .D.CI "kii_category"}}"></div>
        <div class="col-12"><label class="form-label" for="g_desc">Описание</label><textarea id="g_desc" name="description" class="form-control" rows="2">{{index .D.CI "description"}}</textarea></div>
        <div class="col-md-8">
          <label class="form-label" for="g_reason">Причина изменения <span class="text-danger">*</span></label>
          <input id="g_reason" name="reason" class="form-control">
        </div>
        <div class="col-md-4 d-flex align-items-end"><button class="btn btn-primary" type="submit">Сохранить</button></div>
      </div>
    </form>
  </div>

  <div class="col-lg-4">
    <div class="card card-body mb-3">
      <h2 class="h6">Жизненный цикл</h2>
      <p class="mb-2">Стадия: <span class="badge {{stageClass (index .D.CI "lifecycle_stage")}}">{{index .D.CI "lifecycle_stage"}}</span>
        {{if index .D.CI "lifecycle_status"}}<small class="text-body-secondary">{{index .D.CI "lifecycle_status"}}</small>{{end}}</p>
      {{if .Transitions}}
      <form hx-post="/cis/{{.CIID}}/lifecycle" hx-swap="none" autocomplete="off">
        <input type="hidden" name="row_version" value="{{index .D.CI "row_version"}}">
        <div class="mb-2">
          <label class="form-label" for="lc_stage">Перейти в стадию</label>
          <select id="lc_stage" name="to_stage" class="form-select">
            {{range .Transitions}}<option value="{{.}}">{{.}}</option>{{end}}
          </select>
        </div>
        <div class="mb-2"><label class="form-label" for="lc_status">Статус (необязательно)</label><input id="lc_status" name="status" class="form-control"></div>
        <div class="mb-2"><label class="form-label" for="lc_reason">Причина <span class="text-danger">*</span></label><input id="lc_reason" name="reason" class="form-control"></div>
        <button class="btn btn-outline-warning" type="submit">Сменить стадию</button>
      </form>
      {{else}}<p class="text-body-secondary mb-0">Из этой стадии переходов нет.</p>{{end}}
    </div>
    <div class="card card-body">
      <h2 class="h6">Служебное</h2>
      <dl class="row mb-0 small">
        <dt class="col-5">ID</dt><dd class="col-7">{{index .D.CI "id"}}</dd>
        <dt class="col-5">UID</dt><dd class="col-7 text-break">{{index .D.CI "uid"}}</dd>
        <dt class="col-5">Версия строки</dt><dd class="col-7">{{index .D.CI "row_version"}}</dd>
        <dt class="col-5">Создана</dt><dd class="col-7">{{index .D.CI "created_fmt"}} UTC</dd>
        <dt class="col-5">Изменена</dt><dd class="col-7">{{index .D.CI "updated_fmt"}} UTC</dd>
      </dl>
    </div>
  </div>
</div>
