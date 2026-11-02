<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Поиск размещения оборудования</h2>
        <div class="text-muted">По техническому номеру ИТ, инвентарному, бухгалтерскому инвентарному или серийному номеру</div>
    </div>
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacenterequipview/0">Оборудование площадки/ЦОД</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form method="get" action="/sprsrv/equipmentplacement" class="hw-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-12 col-md-8">
            <label for="eqSearchQ" class="form-label">Номер оборудования</label>
            <input type="text" id="eqSearchQ" name="q" class="form-control" autocomplete="off"
                   placeholder="технический / инвентарный / бухгалтерский / серийный номер"
                   value="{{.Query}}" autofocus>
        </div>
        <div class="col-12 col-md-4 d-flex gap-2">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Найти</button>
            <a href="/sprsrv/equipmentplacement" class="btn btn-outline-secondary rounded-pill px-3">Сброс</a>
        </div>
    </div>
</form>

{{if gt .ResultCount 0}}
<div class="mb-2 text-muted small">Найдено: <b>{{.ResultCount}}</b></div>
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th>Тип</th>
            <th>Наименование</th>
            <th>Совпадение</th>
            <th>Площадка/ЦОД</th>
            <th>Шкаф</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {{range $h := .Results}}
        <tr>
            <td>{{$h.KindLabel}}</td>
            <td>{{$h.Name}}</td>
            <td><span class="text-muted">{{$h.MatchField}}:</span> {{$h.MatchValue}}</td>
            <td>{{if $h.SiteName}}{{$h.SiteName}}{{else}}<span class="text-muted">—</span>{{end}}</td>
            <td>{{if $h.RackName}}{{$h.RackName}}{{else}}<span class="text-muted">—</span>{{end}}</td>
            <td class="text-end">
                {{if $h.ViewURL}}
                <a class="btn btn-sm btn-outline-primary rounded-pill px-3" href="{{$h.ViewURL}}">Показать на площадке</a>
                {{else}}
                <span class="text-muted small">нет размещения</span>
                {{end}}
            </td>
        </tr>
        {{end}}
    </tbody>
</table>
{{else if .Query}}
{{/* сообщение уже в flash.notice */}}
{{else}}
<div class="text-muted">Введите номер и нажмите «Найти». При единственном результате сразу откроется площадка с выделением шкафа и оборудования.</div>
{{end}}
