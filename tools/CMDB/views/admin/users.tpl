<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Пользователи</h1>
  <a class="btn btn-outline-secondary" href="/admin/groups">Группы каталога → роли</a>
</div>

<form class="card card-body mb-4" hx-post="/admin/users" hx-swap="none">
  <h2 class="h6">Создать локального пользователя</h2>
  <div class="row g-2">
    <div class="col-md-2"><label class="form-label" for="u-name">Имя</label><input id="u-name" name="username" class="form-control" required autocomplete="off"></div>
    <div class="col-md-3"><label class="form-label" for="u-disp">Отображаемое имя</label><input id="u-disp" name="display_name" class="form-control"></div>
    <div class="col-md-3"><label class="form-label" for="u-mail">Почта</label><input id="u-mail" name="email" class="form-control"></div>
    <div class="col-md-2"><label class="form-label" for="u-pass">Пароль (от {{.MinPass}} симв.)</label><input id="u-pass" type="password" name="password" class="form-control" required autocomplete="new-password"></div>
    <div class="col-md-2"><label class="form-label" for="u-role">Роль</label>
      <select id="u-role" name="role" class="form-select"><option value="">— без роли —</option>{{range .Roles}}<option value="{{.}}">{{.}}</option>{{end}}</select></div>
    <div class="col-md-2"><label class="form-label" for="u-class">Класс (class_owner)</label><input id="u-class" name="ci_class" class="form-control" placeholder="server"></div>
  </div>
  <div class="mt-2"><button type="submit" class="btn btn-primary">Создать</button></div>
</form>

<table class="table table-sm align-middle">
  <thead><tr><th>Имя</th><th>ФИО</th><th>Источник</th><th>Роли</th><th>Последний вход (UTC)</th><th>Статус</th><th></th></tr></thead>
  <tbody>
    {{range .Users}}
    <tr>
      <td>{{.Username}}</td><td>{{.DisplayName}}</td><td>{{.Source}}</td><td>{{.Roles}}</td><td>{{.LastLogin}}</td>
      <td>{{if .Enabled}}<span class="badge text-bg-success">активен</span>{{else}}<span class="badge text-bg-secondary">отключён</span>{{end}}</td>
      <td class="text-end">
        <form class="d-inline-flex gap-1" hx-post="/admin/users/{{.ID}}/role" hx-swap="none">
          <select name="role" class="form-select form-select-sm" aria-label="Роль">{{range $.Roles}}<option value="{{.}}">{{.}}</option>{{end}}</select>
          <input name="ci_class" class="form-control form-control-sm" placeholder="класс" style="width: 6rem;" aria-label="Класс для class_owner">
          <button class="btn btn-sm btn-outline-secondary" type="submit">Назначить</button>
        </form>
        <button class="btn btn-sm btn-outline-secondary" hx-post="/admin/users/{{.ID}}/enabled" hx-vals='{"enabled": "{{if .Enabled}}0{{else}}1{{end}}"}' hx-swap="none">{{if .Enabled}}Отключить{{else}}Включить{{end}}</button>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
