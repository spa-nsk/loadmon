<div class="d-flex justify-content-center">
<form role="form" id="Update" method="POST" class="p-1" style="max-width:480px; width:100%;">
    <div class="row">
        <div class="hero-text mb-3">
            <h2>Запрос на сброс пароля</h2>
            {{if .flash.error}}
            <blockquote class="blockquote">{{.flash.error}}</blockquote>
            {{end}}

            {{if .flash.notice}}
            <blockquote class="blockquote">{{.flash.notice}}</blockquote>
            {{end}}
        </div>
    </div>
    <div class="mb-3 {{if .Errors.email}}has-error has-feedback{{end}}">
        <label for="email" class="form-label">Email адрес {{if .Errors.email}}{{.Errors.email}}{{end}}</label>
        <input name="email" class="form-control" type="text" tabindex="1" autofocus />
    </div>
    <div class="d-grid">
        <input type="submit" title="Запрос на сброс пароля" value="Запросить" class="btn btn-primary rounded-pill" tabindex="2" />
    </div>
</form>
</div>
