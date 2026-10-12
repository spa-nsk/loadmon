<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Токены API</h1>
  <a class="btn btn-outline-secondary" href="/admin/users">Пользователи</a>
</div>

<form class="card card-body mb-3" hx-post="/admin/api-tokens/accounts" hx-swap="none">
  <h2 class="h6">Создать сервисную учётную запись</h2>
  <p class="text-muted small">Без пароля — вход только по токену API. Роль назначается на странице «Пользователи».</p>
  <div class="row g-2">
    <div class="col-md-3"><label class="form-label" for="sa-name">Имя</label><input id="sa-name" name="username" class="form-control" required autocomplete="off" placeholder="zabbix-sync"></div>
    <div class="col-md-4"><label class="form-label" for="sa-disp">Описание</label><input id="sa-disp" name="display_name" class="form-control" placeholder="Синхронизация с Zabbix"></div>
  </div>
  <div class="mt-2"><button type="submit" class="btn btn-primary">Создать</button></div>
</form>

<form class="card card-body mb-3" hx-post="/admin/api-tokens/issue" hx-target="#issued-token" hx-swap="innerHTML">
  <h2 class="h6">Выпустить токен</h2>
  <div class="row g-2">
    <div class="col-md-4"><label class="form-label" for="tk-user">Учётная запись</label>
      <select id="tk-user" name="user_id" class="form-select" required>
        <option value="">— выберите —</option>
        {{range .Users}}<option value="{{.ID}}">{{.Username}} ({{.Source}})</option>{{end}}
      </select>
    </div>
    <div class="col-md-4"><label class="form-label" for="tk-name">Название токена</label><input id="tk-name" name="name" class="form-control" required placeholder="prod-integration"></div>
    <div class="col-md-2"><label class="form-label" for="tk-ttl">Срок, дней</label><input id="tk-ttl" name="ttl_days" type="number" min="1" class="form-control" placeholder="без срока"></div>
  </div>
  <div class="mt-2"><button type="submit" class="btn btn-primary">Выпустить</button></div>
  <div id="issued-token" class="mt-2"></div>
</form>

<table class="table table-sm align-middle">
  <thead><tr><th>Учётная запись</th><th>Название</th><th>Префикс</th><th>Создан (UTC)</th><th>Истекает</th><th>Использован</th><th>Статус</th><th></th></tr></thead>
  <tbody>
    {{range .Tokens}}
    <tr>
      <td>{{.Username}}</td><td>{{.Name}}</td><td class="font-monospace">{{.Prefix}}…</td>
      <td>{{.CreatedAt}}</td><td>{{if .ExpiresAt}}{{.ExpiresAt}}{{else}}—{{end}}</td><td>{{if .LastUsedAt}}{{.LastUsedAt}}{{else}}—{{end}}</td>
      <td>{{if .Revoked}}<span class="badge text-bg-secondary">отозван</span>{{else}}<span class="badge text-bg-success">действует</span>{{end}}</td>
      <td class="text-end">{{if not .Revoked}}<button class="btn btn-sm btn-outline-danger" hx-post="/admin/api-tokens/{{.ID}}/revoke" hx-swap="none" hx-confirm="Отозвать токен «{{.Name}}»?">Отозвать</button>{{end}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
