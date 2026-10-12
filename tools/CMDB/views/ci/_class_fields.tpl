{{range $f := .Spec.Fields}}
<div class="col-md-4">
  {{if eq $f.Type "bool"}}
  <div class="form-check mt-4">
    <input class="form-check-input" type="checkbox" id="f_{{$f.Name}}" name="{{$f.Name}}" value="1" {{if eq (index $.Values $f.Name) "true"}}checked{{end}}>
    <label class="form-check-label" for="f_{{$f.Name}}">{{$f.Label}}</label>
  </div>
  {{else}}
  <label class="form-label" for="f_{{$f.Name}}">{{$f.Label}}</label>
  {{if eq $f.Type "select"}}
  <select class="form-select" id="f_{{$f.Name}}" name="{{$f.Name}}">
    <option value="">—</option>
    {{range $o := $f.Options}}<option value="{{$o}}" {{if eq $o (index $.Values $f.Name)}}selected{{end}}>{{$o}}</option>{{end}}
  </select>
  {{else}}
  <input class="form-control" id="f_{{$f.Name}}" name="{{$f.Name}}" value="{{index $.Values $f.Name}}"
         {{if or (eq $f.Type "int") (eq $f.Type "number")}}inputmode="numeric"{{end}}>
  {{end}}
  {{if $f.Help}}<div class="form-text">{{$f.Help}}</div>{{end}}
  {{end}}
</div>
{{end}}
