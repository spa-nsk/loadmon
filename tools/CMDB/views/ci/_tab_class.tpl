<form class="card card-body" hx-post="/cis/{{.CIID}}/ext" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
  <input type="hidden" name="row_version" value="{{index .D.Ext "row_version"}}">
  <h2 class="h6">{{.Spec.Title}}</h2>
  <div class="row g-3">{{template "ci/_class_fields.tpl" .}}</div>
  <div class="row g-3 mt-2">
    <div class="col-md-8">
      <label class="form-label" for="c_reason">Причина изменения <span class="text-danger">*</span></label>
      <input id="c_reason" name="reason" class="form-control">
    </div>
    <div class="col-md-4 d-flex align-items-end"><button class="btn btn-primary" type="submit">Сохранить</button></div>
  </div>
</form>
