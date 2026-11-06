<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новый IP-адрес{{else}}IP-адрес{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/ipaddresslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .IP}}
<form role="form" method="POST" class="pb-4">
    <div class="alert alert-warning small" role="alert">
        Справочник IP-адресов пересобирается автосинхронизацией из карточек серверов
        (раз в сутки и при запуске). Ручные правки могут быть перезаписаны следующим циклом синхронизации.
    </div>

    <div class="hw-card">
        <h6>Данные адреса</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="IPAddress" class="form-label">IP-адрес</label>
                <input type="text" id="IPAddress" name="IPAddress" class="form-control" value="{{.IP.IPAddress}}" required>
            </div>
            <div class="col-6">
                <label for="MacAddress" class="form-label">MAC-адрес</label>
                <input type="text" id="MacAddress" name="MacAddress" class="form-control" value="{{.IP.MacAddress}}">
            </div>
            <div class="col-6">
                <label for="NetworkId" class="form-label">Сеть</label>
                <select id="NetworkId" name="NetworkId" class="form-select">
                    <option value="0">-</option>
                    {{range $nw := .NetworkList}}
                    <option value="{{$nw.Id}}" {{if eq $.IP.NetworkId $nw.Id}}selected{{end}}>{{$nw.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6">
                <label for="ServerName" class="form-label">Имя сервера</label>
                <input type="text" id="ServerName" name="ServerName" class="form-control" value="{{.IP.ServerName}}">
            </div>
            <div class="col-6">
                <label for="ServerSprId" class="form-label">Id сервера в справочнике</label>
                <input type="number" id="ServerSprId" name="ServerSprId" class="form-control" value="{{if gt .IP.ServerSprId 0}}{{.IP.ServerSprId}}{{end}}">
            </div>
            <div class="col-6">
                <label class="form-label">Карточка сервера</label>
                <div class="form-control-plaintext">
                    {{if gt .IP.ServerSprId 0}}
                    <a href="/sprsrv/updatebase-tab/{{.IP.ServerSprId}}" target="_blank">{{if .IP.ServerName}}{{.IP.ServerName}}{{else}}сервер {{.IP.ServerSprId}}{{end}}</a>
                    {{else}}—{{end}}
                </div>
            </div>
            {{if not .IsNew}}
            <div class="col-6">
                <label class="form-label">Обновлено</label>
                <div class="form-control-plaintext">{{.IP.Time_Mfy.Format "2006-01-02 15:04:05"}}</div>
            </div>
            {{end}}
        </div>
    </div>

    <div class="d-flex gap-2 mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
        <a href="/sprsrv/ipaddresslist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
{{end}}
