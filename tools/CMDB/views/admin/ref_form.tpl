<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/admin/ref">Справочники</a></li><li class="breadcrumb-item"><a href="/admin/ref/{{.Spec.Slug}}">{{.Spec.Title}}</a></li><li class="breadcrumb-item active">{{if .IsCreate}}Новая запись{{else}}Правка{{end}}</li></ol></nav>

<h1 class="h4 mb-3">{{.Spec.Title}} — {{if .IsCreate}}новая запись{{else}}правка{{end}}</h1>

<form hx-post="{{if .IsCreate}}/admin/ref/{{.Spec.Slug}}{{else}}/admin/ref/{{.Spec.Slug}}/{{.PK}}/edit{{end}}" hx-swap="none" autocomplete="off" class="card card-body" style="max-width: 640px;">
  {{$values := .Values}}
  {{$fk := .FKOptions}}
  {{$isCreate := .IsCreate}}
  {{range .Spec.Fields}}
  {{if not .Generated}}
  {{$f := .}}
  {{$fname := .Name}}
  {{$cur := index $values .Name}}
  <div class="mb-3">
    <label class="form-label" for="f-{{$fname}}">{{.Label}}{{if .Required}} *{{end}}</label>

    {{if .PK}}
      {{if $isCreate}}
        <input id="f-{{$fname}}" name="{{$fname}}" class="form-control" value="{{$cur}}" required>
      {{else}}
        <input class="form-control" value="{{$cur}}" disabled>
      {{end}}
    {{else if eq (fieldType .) "bool"}}
      <div class="form-check">
        <input type="checkbox" class="form-check-input" id="f-{{$fname}}" name="{{$fname}}" {{if eq $cur "true"}}checked{{end}}>
      </div>
    {{else if eq (fieldType .) "select"}}
      <select id="f-{{$fname}}" name="{{$fname}}" class="form-select" {{if .Required}}required{{end}}>
        {{if not .Required}}<option value="">—</option>{{end}}
        {{if .FK}}
          {{range index $fk $fname}}<option value="{{.Value}}" {{if eq .Value $cur}}selected{{end}}>{{.Label}}</option>{{end}}
        {{else}}
          {{range $f.Options}}<option value="{{.}}" {{if eq . $cur}}selected{{end}}>{{.}}</option>{{end}}
        {{end}}
      </select>
    {{else if eq (fieldType .) "textarea"}}
      <textarea id="f-{{$fname}}" name="{{$fname}}" class="form-control" rows="3" {{if .Required}}required{{end}}>{{$cur}}</textarea>
    {{else if eq (fieldType .) "jsonb"}}
      <textarea id="f-{{$fname}}" name="{{$fname}}" class="form-control font-monospace" rows="4" placeholder="{}">{{$cur}}</textarea>
    {{else if eq (fieldType .) "date"}}
      <input type="date" id="f-{{$fname}}" name="{{$fname}}" class="form-control" value="{{$cur}}" {{if .Required}}required{{end}}>
    {{else if or (eq (fieldType .) "int") (eq (fieldType .) "bigint")}}
      <input type="number" step="1" id="f-{{$fname}}" name="{{$fname}}" class="form-control" value="{{$cur}}" {{if .Required}}required{{end}}>
    {{else if eq (fieldType .) "numeric"}}
      <input type="number" step="any" id="f-{{$fname}}" name="{{$fname}}" class="form-control" value="{{$cur}}" {{if .Required}}required{{end}}>
    {{else}}
      <input id="f-{{$fname}}" name="{{$fname}}" class="form-control" value="{{$cur}}" {{if .Required}}required{{end}}>
    {{end}}
    {{if .Help}}<div class="form-text">{{.Help}}</div>{{end}}
  </div>
  {{end}}
  {{end}}

  <div class="d-flex gap-2">
    <button type="submit" class="btn btn-primary">Сохранить</button>
    <a class="btn btn-outline-secondary" href="/admin/ref/{{.Spec.Slug}}">Отмена</a>
  </div>
</form>
