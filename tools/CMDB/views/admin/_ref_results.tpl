<table class="table table-sm table-hover align-middle">
  <thead>
    <tr>
      {{range .Spec.Fields}}{{if .ListShow}}<th>{{.Label}}</th>{{end}}{{end}}
      <th></th>
    </tr>
  </thead>
  <tbody>
    {{$spec := .Spec}}
    {{range .Rows}}
    {{$row := .}}
    <tr>
      {{range $spec.Fields}}{{if .ListShow}}<td>{{index $row .Name}}</td>{{end}}{{end}}
      <td class="text-end text-nowrap">
        {{if eq $spec.Slug "location"}}<a class="btn btn-sm btn-outline-secondary" href="/admin/ref/location/{{index $row "location_code"}}/tree">Дерево</a>{{end}}
        <a class="btn btn-sm btn-outline-secondary" href="/admin/ref/{{$spec.Slug}}/{{index $row $spec.PKField.Name}}/edit">Править</a>
        <button type="button" class="btn btn-sm btn-outline-danger"
          hx-post="/admin/ref/{{$spec.Slug}}/{{index $row $spec.PKField.Name}}/delete"
          hx-confirm="Удалить запись безвозвратно?">Удалить</button>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="99" class="text-center text-body-secondary py-4">Ничего не найдено</td></tr>
    {{end}}
  </tbody>
</table>
<small class="text-body-secondary">Найдено: {{len .Rows}}</small>
