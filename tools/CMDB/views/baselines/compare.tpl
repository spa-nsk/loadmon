<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/baselines">Срезы</a></li><li class="breadcrumb-item active">Сравнение</li></ol></nav>
<h1 class="h4 mb-3">Сравнение</h1>

<form class="card card-body mb-3" method="get" action="/baselines/compare">
  <p class="text-body-secondary small mb-2">Сторона задаётся как <code>b:&lt;№ среза&gt;</code>, <code>t:&lt;момент UTC&gt;</code> (например <code>t:2026-10-01T00:00</code>) или <code>now</code>. Фильтр области нужен только для сравнения «дата ↔ дата»: у среза своя область.</p>
  <div class="row g-2 align-items-end">
    <div class="col-md-3"><label class="form-label" for="ca">Сторона A</label><input id="ca" name="a" class="form-control" value="{{.A}}"></div>
    <div class="col-md-3"><label class="form-label" for="cb">Сторона B</label><input id="cb" name="b" class="form-control" value="{{.B}}"></div>
    <div class="col-md-2"><label class="form-label" for="cl">Площадка</label><input id="cl" name="location_code" class="form-control" value="{{.Scope.LocationCode}}"></div>
    <div class="col-md-2"><label class="form-label" for="cc">Классы</label><input id="cc" name="classes" class="form-control" value="{{range $i, $c := .Scope.Classes}}{{if $i}},{{end}}{{$c}}{{end}}"></div>
    <div class="col-md-2 d-flex gap-2">
      <button class="btn btn-primary" type="submit">Сравнить</button>
      <button class="btn btn-outline-secondary" type="submit" name="format" value="csv">CSV</button>
    </div>
  </div>
</form>

<div class="row g-3 mb-3">
  <div class="col-md-6"><div class="card card-body"><div class="small text-body-secondary">A</div>{{.Res.LabelA}}</div></div>
  <div class="col-md-6"><div class="card card-body"><div class="small text-body-secondary">B</div>{{.Res.LabelB}}</div></div>
</div>

<div class="d-flex gap-3 mb-3">
  <span class="badge text-bg-success fs-6">добавлено: {{.Res.Added}}</span>
  <span class="badge text-bg-danger fs-6">удалено: {{.Res.Removed}}</span>
  <span class="badge text-bg-warning fs-6">изменено: {{.Res.Changed}}</span>
</div>

{{if .Res.Summary}}
<table class="table table-sm w-auto">
  <thead><tr><th>Сущность</th><th>Добавлено</th><th>Удалено</th><th>Изменено</th></tr></thead>
  <tbody>{{range .Res.Summary}}<tr><td>{{.Entity}}</td><td>{{.Added}}</td><td>{{.Removed}}</td><td>{{.Changed}}</td></tr>{{end}}</tbody>
</table>
{{else}}<div class="alert alert-secondary">Различий нет.</div>{{end}}

{{if .Res.Truncated}}<div class="alert alert-warning">Подробно показаны первые различия, счётчики полные. Полный отчёт — «CSV».</div>{{end}}

{{range .Res.Changes}}
<details class="border-bottom py-1">
  <summary>
    {{if eq .Kind "added"}}<span class="badge text-bg-success">добавлено</span>{{else if eq .Kind "removed"}}<span class="badge text-bg-danger">удалено</span>{{else}}<span class="badge text-bg-warning">изменено</span>{{end}}
    <code>{{.Entity}}</code> #{{.EntityID}}{{if .CIName}} — <a href="/cis/{{.CIID}}">{{.CIName}}</a>{{end}}
    {{if .Fields}}<small class="text-body-secondary">({{len .Fields}} полей)</small>{{end}}
  </summary>
  {{if .Fields}}
  <table class="table table-sm mb-2 mt-1">
    <thead><tr><th>Поле</th><th>A</th><th>B</th></tr></thead>
    <tbody>{{range .Fields}}<tr><td>{{.Name}}</td><td class="text-danger">{{.A}}</td><td class="text-success">{{.B}}</td></tr>{{end}}</tbody>
  </table>
  {{else}}<div class="small text-body-secondary ms-3">версии: A {{.AVersion}} → B {{.BVersion}}</div>{{end}}
</details>
{{end}}
