<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/admin/users">Пользователи</a></li><li class="breadcrumb-item active">Группы каталога</li></ol></nav>
<h1 class="h4 mb-3">Группы каталога → роли</h1>
<p class="text-body-secondary">Группа задаётся полным DN либо именем (CN), без учёта регистра. Роли из групп пересчитываются при каждом входе пользователя каталога; назначенные вручную роли не затрагиваются.</p>

<form class="card card-body mb-4" hx-post="/admin/groups" hx-swap="none">
  <div class="row g-2 align-items-end">
    <div class="col-md-5"><label class="form-label" for="g-name">Группа (DN или CN)</label><input id="g-name" name="group" class="form-control" required placeholder="cmdb-editors"></div>
    <div class="col-md-3"><label class="form-label" for="g-role">Роль</label><select id="g-role" name="role" class="form-select">{{range .Roles}}<option value="{{.}}">{{.}}</option>{{end}}</select></div>
    <div class="col-md-2"><label class="form-label" for="g-class">Класс (class_owner)</label><input id="g-class" name="ci_class" class="form-control"></div>
    <div class="col-md-2"><button type="submit" class="btn btn-primary">Сохранить</button></div>
  </div>
</form>

<table class="table table-sm align-middle">
  <thead><tr><th>Группа</th><th>Роль</th><th>Класс</th><th></th></tr></thead>
  <tbody>
    {{range .Mappings}}
    <tr><td>{{.Group}}</td><td>{{.Role}}</td><td>{{.Class}}</td>
      <td class="text-end"><button class="btn btn-sm btn-outline-danger" hx-post="/admin/groups/delete" hx-vals='{"group": "{{.Group}}", "role": "{{.Role}}", "ci_class": "{{.Class}}"}' hx-swap="none">Удалить</button></td></tr>
    {{else}}<tr><td colspan="4" class="text-center text-body-secondary py-3">Сопоставлений нет — пользователи каталога без ролей войти не смогут</td></tr>{{end}}
  </tbody>
</table>
