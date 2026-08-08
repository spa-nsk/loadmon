<div class="d-flex justify-content-center">
<form role="form" id="Update" method="POST" class="p-1" style="max-width:520px; width:100%;">
    <div class="row">
        <div class="hero-text mb-3">
            <h2>Пожалуйста зарегистрируйтесь</h2>
            {{if .flash.error}}
            <blockquote class="blockquote">{{.flash.error}}</blockquote>
            {{end}}

            {{if .flash.notice}}
            <blockquote class="blockquote">{{.flash.notice}}</blockquote>
            {{end}}
        </div>
    </div>
    <div class="row g-3">
        <div class="col-sm-6 {{if .Errors.first}}has-error has-feedback{{end}}">
            <label for="first" class="form-label">Имя {{if .Errors.first}}{{.Errors.first}}{{end}}</label>
            <input name="first" class="form-control" type="text" tabindex="1" autofocus value="{{.User.First}}" />
        </div>
        <div class="col-sm-6 {{if .Errors.last}}has-error has-feedback{{end}}">
            <label for="last" class="form-label">Фамилия {{if .Errors.last}}{{.Errors.last}}{{end}}</label>
            <input name="last" class="form-control" type="text" tabindex="2" value="{{.User.Last}}" />
        </div>
    </div>
    <div class="mb-3 mt-3 {{if .Errors.email}}has-error has-feedback{{end}}">
        <label for="email" class="form-label">Email адрес {{if .Errors.email}}{{.Errors.email}}{{end}}</label>
        <input name="email" class="form-control" type="text" tabindex="3" value="{{.User.Email}}"/>
    </div>
    <div class="mb-3 {{if .Errors.password}}has-error has-feedback{{end}}">
        <label for="password" class="form-label">Пароль (не менее 6 символов) {{if .Errors.password}}{{.Errors.password}}{{end}}</label>
        <input name="password" class="form-control" type="password" tabindex="4" />
    </div>
    <div class="mb-3 {{if .Errors.password2}}has-error has-feedback{{end}}">
        <label for="password2" class="form-label">Пароль повторно {{if .Errors.password2}}{{.Errors.password2}}{{end}}</label>
        <input name="password2" class="form-control" type="password" tabindex="5" />
    </div>
    <div class="d-grid">
        <input type="submit" title="Регистрация нового пользователя" value="Регистрация" class="btn btn-primary rounded-pill" tabindex="6" />
    </div>
</form>
</div>
