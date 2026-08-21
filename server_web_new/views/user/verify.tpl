<div id="content">
{{if .flash.error}}
<blockquote class="blockquote">{{.flash.error}}</blockquote>
{{else if .Verified}}
<h2>Ваша учетная запись подтверждена. Вы можете авторизоваться.</h2>
{{else}}
<h2>Ваша учетная запись <b>не подтверждена</b>.</h2>
{{end}}
</div>
