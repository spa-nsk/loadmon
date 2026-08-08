<div class="d-flex justify-content-center">
<form role="form" id="Update" method="POST" class="p-1" style="max-width:480px; width:100%;">
    <div class="row">
        <div class="hero-text mb-3">
            <h2>Изменить пароль по запросу</h2>
            {{if .flash.error}}
            <blockquote class="blockquote">{{.flash.error}}</blockquote>
            {{end}}

            {{if .flash.notice}}
            <blockquote class="blockquote">{{.flash.notice}}</blockquote>
            {{end}}
        </div>
    </div>
    <div class="mb-3 {{if .Errors.password}}has-error has-feedback{{end}}">
        <label for="password" class="form-label">Пароль (не менее 6 символов) {{if .Errors.password}}{{.Errors.password}}{{end}}</label>
        <input name="password" class="form-control" type="password" tabindex="1" />
    </div>
    <div class="mb-3 {{if .Errors.password2}}has-error has-feedback{{end}}">
        <label for="password2" class="form-label">Пароль повторно {{if .Errors.password2}}{{.Errors.password2}}{{end}}</label>
        <input name="password2" class="form-control" type="password" tabindex="2" />
    </div>
    <div class="d-grid">
        <input type="submit" title="Изменить пароль по запросу" value="Изменить" class="btn btn-primary rounded-pill" tabindex="3" />
    </div>
</form>
</div>
