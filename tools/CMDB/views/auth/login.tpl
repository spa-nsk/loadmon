<!doctype html>
<html lang="ru" data-bs-theme="light">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Вход — CMDB</title>
  <script>(function(){try{var t=localStorage.getItem('theme');if(t){document.documentElement.setAttribute('data-bs-theme',t);}}catch(e){}})();</script>
  <link href="/static/vendor/bootstrap.min.css" rel="stylesheet">
</head>
<body class="bg-body-tertiary">
<main class="container" style="max-width: 26rem;">
  <div class="card shadow-sm mt-5">
    <div class="card-body p-4">
      <h1 class="h4 mb-3">CMDB — вход</h1>
      {{if .Error}}<div class="alert alert-danger py-2" role="alert">{{.Error}}</div>{{end}}
      <form method="post" action="/login">
        <input type="hidden" name="_xsrf" value="{{.Xsrf}}">
        <input type="hidden" name="next" value="{{.Next}}">
        <div class="mb-3">
          <label class="form-label" for="username">Имя пользователя</label>
          <input id="username" name="username" class="form-control" autocomplete="username" autofocus required>
        </div>
        <div class="mb-3">
          <label class="form-label" for="password">Пароль</label>
          <input id="password" type="password" name="password" class="form-control" autocomplete="current-password" required>
        </div>
        <button type="submit" class="btn btn-primary w-100">Войти</button>
        {{if .LDAP}}<div class="form-text mt-2">Учётные записи домена и локальные записи CMDB.</div>{{end}}
      </form>
    </div>
  </div>
</main>
</body>
</html>
