<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/baselines">Срезы</a></li><li class="breadcrumb-item active">№{{.B.ID}}</li></ol></nav>

<div class="d-flex flex-wrap align-items-center gap-2 mb-3">
  <h1 class="h4 mb-0">{{.B.Name}}</h1>
  {{template "baselines/_status.tpl" .B.Status}}
  <div class="ms-auto d-flex flex-wrap gap-2">
    {{if eq .B.Status "draft"}}
      <button class="btn btn-primary" hx-post="/baselines/{{.B.ID}}/freeze" hx-swap="none" hx-confirm="Заморозить срез? Дальше он неизменяем.">Заморозить</button>
      <button class="btn btn-outline-danger" hx-post="/baselines/{{.B.ID}}/delete" hx-swap="none" hx-confirm="Удалить черновик?">Удалить</button>
    {{end}}
    {{if eq .B.Status "frozen"}}<button class="btn btn-success" hx-post="/baselines/{{.B.ID}}/approve" hx-swap="none">Утвердить</button>{{end}}
    {{if ne .B.Status "draft"}}
      <button class="btn btn-outline-secondary" hx-post="/baselines/{{.B.ID}}/verify" hx-swap="none">Проверить целостность</button>
      {{range .Formats}}<a class="btn btn-outline-secondary" href="/baselines/{{$.B.ID}}/export/{{.}}">Экспорт {{.}}</a>{{end}}
    {{end}}
  </div>
</div>

<div class="row g-3 mb-4">
  <div class="col-md-7">
    <div class="card card-body">
      <dl class="row mb-0">
        <dt class="col-sm-4">На момент (UTC)</dt><dd class="col-sm-8">{{.B.AsOf}}</dd>
        <dt class="col-sm-4">Область</dt>
        <dd class="col-sm-8">
          {{if or .Scope.Classes .Scope.LocationCode .Scope.CompanyCode .Scope.EnvironmentCode .Scope.NameLike}}
            {{if .Scope.Classes}}классы: {{range $i, $c := .Scope.Classes}}{{if $i}}, {{end}}{{$c}}{{end}}; {{end}}
            {{if .Scope.LocationCode}}площадка: {{.Scope.LocationCode}}; {{end}}{{if .Scope.CompanyCode}}владелец: {{.Scope.CompanyCode}}; {{end}}
            {{if .Scope.EnvironmentCode}}окружение: {{.Scope.EnvironmentCode}}; {{end}}{{if .Scope.NameLike}}имя содержит «{{.Scope.NameLike}}»{{end}}
          {{else}}все КЕ{{end}}
        </dd>
        <dt class="col-sm-4">Позиций</dt><dd class="col-sm-8">{{.B.ItemCount}}</dd>
        <dt class="col-sm-4">Создан</dt><dd class="col-sm-8">{{.B.CreatedAt}} (пользователь {{.B.CreatedBy}})</dd>
        {{if .B.FrozenAt}}<dt class="col-sm-4">Заморожен</dt><dd class="col-sm-8">{{.B.FrozenAt}} (пользователь {{.B.FrozenBy}})</dd>{{end}}
        {{if .B.ApprovedAt}}<dt class="col-sm-4">Утверждён</dt><dd class="col-sm-8">{{.B.ApprovedAt}} (пользователь {{.B.ApprovedBy}})</dd>{{end}}
        {{if .B.Checksum}}<dt class="col-sm-4">SHA-256</dt><dd class="col-sm-8"><code class="small text-break">{{.B.Checksum}}</code></dd>{{end}}
        {{if .B.Description}}<dt class="col-sm-4">Описание</dt><dd class="col-sm-8">{{.B.Description}}</dd>{{end}}
      </dl>
    </div>
  </div>
  <div class="col-md-5">
    <div class="card card-body">
      <h2 class="h6">Состав</h2>
      <table class="table table-sm mb-0"><tbody>
        {{range .ByEntity}}<tr><td>{{.Entity}}</td><td class="text-end">{{.N}}</td></tr>{{else}}<tr><td class="text-body-secondary">пусто</td></tr>{{end}}
      </tbody></table>
    </div>
  </div>
</div>

<form class="card card-body" method="get" action="/baselines/compare">
  <h2 class="h6">Сравнить этот срез с…</h2>
  <input type="hidden" name="a" value="b:{{.B.ID}}">
  <div class="row g-2 align-items-end">
    <div class="col-md-5">
      <label class="form-label" for="cmp-b">Другой срез</label>
      <select id="cmp-b" name="b" class="form-select">
        <option value="now">Текущее состояние</option>
        {{range .Others}}{{if ne .ID $.B.ID}}<option value="b:{{.ID}}">Срез №{{.ID}} «{{.Name}}»</option>{{end}}{{end}}
      </select>
    </div>
    <div class="col-md-2"><button class="btn btn-primary" type="submit">Сравнить</button></div>
  </div>
</form>
