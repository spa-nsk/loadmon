<!doctype html>
<html lang="ru" data-bs-theme="light">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CMDB</title>
  <script>(function(){try{var t=localStorage.getItem('theme');if(t){document.documentElement.setAttribute('data-bs-theme',t);}}catch(e){}})();</script>
  <link href="/static/vendor/bootstrap.min.css" rel="stylesheet">
  <link href="/static/css/app.css" rel="stylesheet">
</head>
<body hx-headers='{"X-Xsrftoken": "{{.Xsrf}}"}'>
<nav class="navbar navbar-expand-md bg-body-tertiary border-bottom">
  <div class="container-fluid">
    <a class="navbar-brand" href="/cis">CMDB</a>
    <ul class="navbar-nav me-auto">
      <li class="nav-item"><a class="nav-link {{if eq .Nav "cis"}}active{{end}}" href="/cis">Конфигурационные единицы</a></li>
      {{if index .Perms "ci.write"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "new"}}active{{end}}" href="/cis/new">Создать КЕ</a></li>{{end}}
      {{if index .Perms "import.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "import"}}active{{end}}" href="/import">Импорт</a></li>{{end}}
      {{if index .Perms "ci.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "admin"}}active{{end}}" href="/admin/ref">Справочники</a></li>{{end}}
      <li class="nav-item"><a class="nav-link {{if eq .Nav "naming"}}active{{end}}" href="/admin/naming">Проверка имени</a></li>
      {{if index .Perms "ci.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "quality"}}active{{end}}" href="/quality">Качество данных</a></li>{{end}}
      {{if index .Perms "ci.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "reports"}}active{{end}}" href="/reports">Отчёты</a></li>{{end}}
      {{if index .Perms "audit.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "changes"}}active{{end}}" href="/changes">Журнал изменений</a></li>{{end}}
      {{if index .Perms "audit.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "events"}}active{{end}}" href="/admin/events">Журнал действий</a></li>{{end}}
      {{if index .Perms "baseline.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "baselines"}}active{{end}}" href="/baselines">Срезы</a></li>{{end}}
      {{if index .Perms "baseline.read"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "periods"}}active{{end}}" href="/periods">Учётные периоды</a></li>{{end}}
      {{if index .Perms "user.admin"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "users"}}active{{end}}" href="/admin/users">Пользователи</a></li>{{end}}
      {{if index .Perms "user.admin"}}<li class="nav-item"><a class="nav-link {{if eq .Nav "users"}}active{{end}}" href="/admin/api-tokens">Токены API</a></li>{{end}}
    </ul>
    <form id="asof-form" class="d-flex align-items-center gap-1 me-3" title="Показать данные на выбранный момент (UTC), только чтение">
      <label class="visually-hidden" for="asof-input">Показать на дату (UTC)</label>
      <input id="asof-input" type="datetime-local" class="form-control form-control-sm" value="{{.AsOfInput}}" style="width: 12rem;">
      <button id="asof-apply" type="button" class="btn btn-sm btn-outline-secondary">На дату</button>
    </form>
    <span class="navbar-text me-2" title="Роли: {{.UserRoles}}">{{.User}}</span>
    {{if .LoggedIn}}<form class="me-3" method="post" action="/logout"><input type="hidden" name="_xsrf" value="{{.Xsrf}}"><button class="btn btn-outline-secondary btn-sm" type="submit">Выйти</button></form>{{end}}
    <button id="theme-toggle" type="button" class="btn btn-outline-secondary btn-sm">Тема</button>
  </div>
</nav>
{{if .AsOf}}<div id="asof-banner" class="alert alert-warning rounded-0 mb-0 py-2 d-flex align-items-center gap-3" role="status">
  <strong>Состояние на {{.AsOf}} UTC — режим только чтения.</strong>
  <span class="small">Интерфейсы, IP и источники на прошлую дату недоступны.</span>
  <button id="asof-reset" type="button" class="btn btn-sm btn-dark ms-auto">Вернуться к текущему</button>
</div>{{end}}
<main class="container-fluid py-3{{if .AsOf}} asof-readonly{{end}}">
{{.LayoutContent}}
</main>
<div id="toasts" class="toast-container position-fixed bottom-0 end-0 p-3"></div>
<script src="/static/vendor/htmx.min.js"></script>
<script src="/static/vendor/bootstrap.bundle.min.js"></script>
<script src="/static/js/app.js"></script>
</body>
</html>
