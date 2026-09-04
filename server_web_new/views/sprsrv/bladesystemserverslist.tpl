<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Серверы в блейд-корзине</h2>
        {{if .BS}}
        <div class="text-muted">
            {{.BS.Name}} &middot; {{.BS.Vendor}} {{.BS.Model}} &middot; слотов {{.BS.Size}}, занято {{.BS.Used}}
        </div>
        {{end}}
    </div>
    <div class="d-flex gap-2">
        {{if .BS}}
        <a href="/sprsrv/bladesystemsedit/{{.BS.Id}}" class="btn btn-outline-secondary rounded-pill px-3" target="_blank">Корзина</a>
        {{end}}
        <a href="/sprsrv/bladesystemslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку корзин</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

<div class="text-muted small mb-2">Найдено серверов: {{.ServersCount}}</div>

<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th class="text-end">Слот</th>
            <th>Питание</th>
            <th>Сервер</th>
            <th>Модель</th>
            <th>Серийный №</th>
            <th class="text-end">CPU</th>
            <th class="text-end">Память, Мб</th>
            <th>Контроллер управления (iLO/OA)</th>
        </tr>
    </thead>
    <tbody>
        {{range $hw := .Servers}}
        <tr>
            <td class="text-end">{{$hw.BladeNum}}</td>
            <td>
                <svg class="bi power-led {{if eq $hw.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $hw.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg>
            </td>
            <td><a href="/sprsrv/hwconfedit/{{$hw.Id}}" target="_blank">{{if $hw.ServerName}}{{$hw.ServerName}}{{else}}&mdash;{{end}}</a></td>
            <td>
                <div style="max-width:220px; word-wrap:break-word;">
                    {{$hw.Vendor}} {{$hw.Model}}
                    <div class="small text-muted">{{$hw.ShortModel}}</div>
                </div>
            </td>
            <td>{{if $hw.SerialNum}}{{$hw.SerialNum}}{{else}}&mdash;{{end}}</td>
            <td class="text-end">{{$hw.CPUCount}}&times;{{$hw.CorePerCPU}}c @{{$hw.CoreMHz}}MHz</td>
            <td class="text-end">{{$hw.MemoryMb}}</td>
            <td>
                <div style="max-width:220px; word-wrap:break-word;">
                    {{if $hw.MpHostName}}{{$hw.MpHostName}}{{else}}&mdash;{{end}}
                    <div class="small text-muted">{{$hw.MpModel}} {{if $hw.MpIPAddress}}({{$hw.MpIPAddress}}){{end}}</div>
                </div>
            </td>
        </tr>
        {{else}}
        <tr>
            <td colspan="8" class="text-center text-muted py-4">В этой корзине не найдено ни одного сервера, привязанного через BladeSystemId (справочник конфигураций серверов заполняется автосинхронизацией из HP OneView)</td>
        </tr>
        {{end}}
    </tbody>
</table>

<!-- Стиль карточек/таблиц - общий с /sprsrv/storagesystemconfigview,
     server_web_new/static/css/loadmon.css. Список read-only - привязка
     сервера к корзине (BladeSystemId) и позиция в ней (BladeNum)
     заполняются автосинхронизацией с HP OneView
     (cmdb/initrec.go:ImportHWSprRecord), правки нужно вносить через
     /sprsrv/hwconfedit у конкретного сервера. -->
