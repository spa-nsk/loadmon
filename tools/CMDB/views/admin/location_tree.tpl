<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/admin/ref">Справочники</a></li><li class="breadcrumb-item"><a href="/admin/ref/location">Площадки</a></li><li class="breadcrumb-item active">{{.LocationCode}}</li></ol></nav>

<h1 class="h4 mb-3">Размещение: {{index .Location "location_name"}} ({{.LocationCode}})</h1>

<table class="table table-sm align-middle">
  <thead><tr><th>Уровень</th><th>Код</th><th>Название</th><th>Юнитов</th></tr></thead>
  <tbody>
    {{range .Nodes}}
    <tr>
      <td class="tree-cell" style="padding-left: {{indent .Depth}}rem">{{index .Row "kind"}}</td>
      <td>{{index .Row "code"}}</td>
      <td>{{index .Row "name"}}</td>
      <td>{{index .Row "rack_units"}}</td>
    </tr>
    {{else}}
    <tr><td colspan="4" class="text-center text-body-secondary py-3">Размещение не заведено</td></tr>
    {{end}}
  </tbody>
</table>

<a class="btn btn-outline-primary btn-sm" href="/admin/ref/site_place/new">Добавить уровень размещения</a>
