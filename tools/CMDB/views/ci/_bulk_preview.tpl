<div class="card card-body mb-3">
  <h2 class="h6">Предпросмотр массового изменения</h2>
  <table class="table table-sm align-middle">
    <thead><tr><th>КЕ</th><th>Было</th><th>Станет</th><th>Статус</th></tr></thead>
    <tbody>
      {{range .Items}}
      <tr class="{{if .Skip}}text-body-secondary{{end}}">
        <td>{{if .Name}}{{.Name}}{{else}}id={{.CIID}}{{end}}</td>
        <td>{{.Old}}</td>
        <td>{{.New}}</td>
        <td>{{if .Skip}}<span class="badge text-bg-secondary">пропуск: {{.SkipReason}}</span>{{else}}<span class="badge text-bg-success">будет применено</span>{{end}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <form hx-post="/cis/bulk/apply" hx-swap="none" autocomplete="off" class="row g-2 align-items-end">
    <input type="hidden" name="bulk_field" value="{{.Field}}">
    {{if eq (print .Field) "stage"}}<input type="hidden" name="bulk_stage" value="{{.Value}}">{{end}}
    {{if eq (print .Field) "company"}}<input type="hidden" name="bulk_company" value="{{.Value}}">{{end}}
    {{if eq (print .Field) "location"}}<input type="hidden" name="bulk_location" value="{{.Value}}">{{end}}
    {{range .Items}}<input type="hidden" name="ids" value="{{.CIID}}">{{end}}
    <div class="col-auto flex-grow-1">
      <input name="reason" class="form-control form-control-sm" placeholder="Причина изменения" required>
    </div>
    <div class="col-auto">
      <button type="submit" class="btn btn-primary btn-sm">Применить</button>
    </div>
  </form>
</div>
