<h1 class="h4 mb-3">Проверка имени</h1>
<p class="text-body-secondary">Разбор по активному профилю CMDB-NC-2026, а если не подходит — по legacy-правилам Guide 2019 (intent §5.6/§5.7). Ничего не сохраняется.</p>

<form class="card card-body mb-3" autocomplete="off" hx-get="/admin/naming/check" hx-target="#naming-result" hx-trigger="keyup changed delay:300ms from:#name, submit">
  <label class="form-label" for="name">Имя</label>
  <input id="name" name="name" class="form-control" placeholder="например OVBDC1-S-01001 или AMSDC1-S-01000" value="{{.Name}}">
</form>

<div id="naming-result">{{template "admin/_naming_result.tpl" .}}</div>
