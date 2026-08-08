<div class="container">
    <div class="row">
        <div class="hero-text">
            <h2>Удаление учетной записи пользователя</h2>
            {{if .flash.error}}
            <blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
            {{end}}

            {{if .flash.notice}}
            <blockquote class="blockquote text-warning-emphasis">{{.flash.notice}}</blockquote>
            {{end}}
        </div>
    </div>
</div>
<div class="d-flex justify-content-center">
    <form role="form" id="Remove" method="POST" class="p-1" style="max-width:480px; width:100%;">
        <p class="mb-3">Осторожно: данные по учетной записи ({{.First}} {{.Last}} [{{.Username}}]) будут удалены. Вы уверены?</p>
        <div class="mb-3 {{if .Errors.current}}has-error has-feedback{{end}}">
            <label for="current" class="form-label">Текущий пароль {{if .Errors.current}}{{.Errors.current}}{{end}}</label>
            <input name="current" class="form-control" type="password" tabindex="2" />
        </div>
        <div class="d-flex gap-2">
            <input type="submit" title="Удалить учетную запись пользователя {{.First}} {{.Last}} [{{.Username}}]" value="Удалить" class="btn btn-danger rounded-pill px-4" tabindex="3" />
            <a href="/user/profile" title="Ничего не делать" class="btn btn-outline-secondary rounded-pill px-4">Отменить</a>
        </div>
    </form>
</div>
