<div class="d-flex align-items-center justify-content-center" style="min-height: calc(100vh - 3rem);">
<form role="form" id="Update" method="POST" class="p-1" style="max-width:480px; width:100%;">
    <div class="row">
        <div class="hero-text mb-3">
            <h2>Пожалуйста авторизуйтесь</h2>
            {{if .flash.error}}
            <blockquote class="blockquote">{{.flash.error}}</blockquote>
            {{end}}

            {{if .flash.notice}}
            <blockquote class="blockquote">{{.flash.notice}}</blockquote>
            {{end}}
        </div>
    </div>
    <div class="mb-3">
        <label for="email" class="form-label">Email адрес</label>
        <input name="email" class="form-control" type="text" tabindex="1" autofocus />
    </div>
    <div class="mb-3">
        <label for="password" class="form-label">Пароль</label>
        <input name="password" class="form-control" type="password" tabindex="2" />
    </div>
    <div class="form-check mb-3">
        <input name="domain" type="checkbox" class="form-check-input" id="checkDomain" tabindex="3">
        <label class="form-check-label" for="checkDomain">Режим просмотра (без авторизации)</label>
    </div>
    <div class="d-grid gap-2 mb-2">
        <input type="submit" title="Авторизоваться" value="Войти" class="btn btn-primary rounded-pill" tabindex="4" />
    </div>
    <div class="d-flex gap-2">
        <a href="/user/forgot" title="Сбросить пароль" class="btn btn-outline-secondary rounded-pill flex-fill">Сброс пароля</a>
        <a href="/user/register" title="Регистрация нового пользователя" class="btn btn-outline-secondary rounded-pill flex-fill">Регистрация</a>
    </div>
</form>
</div>
