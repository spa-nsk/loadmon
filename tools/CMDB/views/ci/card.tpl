{{if .Created}}<div class="alert alert-success">КЕ создана: <strong>{{.Created}}</strong></div>{{end}}
<div class="d-flex flex-wrap align-items-center gap-2 mb-3">
  <h1 class="h4 mb-0">{{index .D.CI "name"}}</h1>
  <span class="badge text-bg-primary">{{.D.ClassName}}</span>
  <span class="badge {{stageClass (index .D.CI "lifecycle_stage")}}">{{index .D.CI "lifecycle_stage"}}</span>
  {{if index .D.CI "environment_code"}}<span class="badge text-bg-secondary">{{index .D.CI "environment_code"}}</span>{{end}}
  <span class="badge {{namingClass (index .D.CI "naming_status")}}" title="Соответствие имени профилю именования">{{index .D.CI "naming_status"}}</span>
  <a class="ms-auto btn btn-outline-secondary btn-sm" href="/cis">← К списку</a>
</div>
{{template "ci/_tabs.tpl" .}}
