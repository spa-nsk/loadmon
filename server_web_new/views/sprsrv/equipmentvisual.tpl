<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Визуализация оборудования</h2>
        <div class="text-muted">По техническому номеру ИТ, инвентарному, бухгалтерскому инвентарному или серийному номеру — открывается 3D-вид выбранного устройства</div>
    </div>
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/equipmentplacement">Поиск размещения</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form method="get" action="/sprsrv/equipmentvisual" class="hw-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-12 col-md-8">
            <label for="eqVisualQ" class="form-label">Номер оборудования</label>
            <input type="text" id="eqVisualQ" name="q" class="form-control" autocomplete="off"
                   placeholder="технический / инвентарный / бухгалтерский / серийный номер"
                   value="{{.Query}}" autofocus>
        </div>
        <div class="col-12 col-md-4 d-flex gap-2">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Найти</button>
            <a href="/sprsrv/equipmentvisual" class="btn btn-outline-secondary rounded-pill px-3">Сброс</a>
        </div>
    </div>
</form>

{{if .NotImplemented}}
<div class="alert alert-warning" role="alert">
    <div class="fw-semibold mb-1">Визуализация пока не реализована</div>
    <div>
        Тип: <b>{{.NotImplementedHit.KindLabel}}</b>
        {{if .NotImplementedHit.Name}} — {{.NotImplementedHit.Name}}{{end}}.
        Сейчас доступны блейд-корзины, blade-серверы и rack-серверы.
    </div>
    {{if .NotImplementedHit.CardURL}}
    <div class="mt-2">
        <a class="btn btn-sm btn-outline-secondary rounded-pill px-3" href="{{.NotImplementedHit.CardURL}}">Открыть карточку</a>
    </div>
    {{end}}
</div>
{{end}}

{{if gt .ResultCount 0}}
<div class="mb-2 text-muted small">Найдено: <b>{{.ResultCount}}</b></div>
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th>Тип</th>
            <th>Наименование</th>
            <th>Статус</th>
            <th>Совпадение</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {{range $h := .Results}}
        <tr>
            <td>{{$h.KindLabel}}</td>
            <td>{{$h.Name}}</td>
            <td>
                {{if $h.StatusName}}
                <span class="text-muted">{{$h.StatusName}}</span>
                {{else}}
                <span class="text-muted">—</span>
                {{end}}
            </td>
            <td><span class="text-muted">{{$h.MatchField}}:</span> {{$h.MatchValue}}</td>
            <td class="text-end">
                {{if $h.Implemented}}
                <a class="btn btn-sm btn-outline-primary rounded-pill px-3" href="{{$h.VisualURL}}" target="_blank">3D-вид</a>
                {{else}}
                <span class="badge text-bg-secondary">пока не реализовано</span>
                {{if $h.CardURL}}
                <a class="btn btn-sm btn-outline-secondary rounded-pill px-3 ms-1" href="{{$h.CardURL}}">Карточка</a>
                {{end}}
                {{end}}
            </td>
        </tr>
        {{end}}
    </tbody>
</table>
{{else if .Query}}
{{/* сообщение уже в flash.notice */}}
{{else}}
<div class="text-muted">
    Введите номер и нажмите «Найти». При единственном результате откроется 3D-вид
    (блейд-корзина, blade- или rack-сервер). Для СХД, сетевого оборудования и ленточных
    библиотек визуализация пока не реализована.
</div>
{{end}}
