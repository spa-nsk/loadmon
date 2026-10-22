<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Типы сетевого оборудования</h2>
        <div class="text-muted">Записей: {{.TypeCount}}</div>
    </div>
    <div class="d-flex gap-2">
        <a href="/sprsrv/networkequipmentlist/0" class="btn btn-outline-secondary rounded-pill px-3">К справочнику оборудования</a>
        <a href="/sprsrv/networkequipmenttypeedit/0" class="btn btn-success rounded-pill px-3">Добавить</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th class="text-center">Id</th>
            <th>Наименование</th>
            <th>Описание</th>
            <th class="text-end">Устройств</th>
        </tr>
    </thead>
    <tbody>
        {{range $t := .TypeList}}
        <tr>
            <td class="text-end">{{$t.Id}}</td>
            <td><a href="/sprsrv/networkequipmenttypeedit/{{$t.Id}}">{{$t.Name}}</a></td>
            <td>{{$t.Annotation}}</td>
            <td class="text-end">{{if gt $t.Count 0}}<a href="/sprsrv/networkequipmentlist/0">{{$t.Count}}</a>{{else}}0{{end}}</td>
        </tr>
        {{else}}
        <tr>
            <td colspan="4" class="text-center text-muted py-4">Типы пока не заведены - они появятся после импорта сетевого оборудования из Итиллиум или их можно добавить вручную</td>
        </tr>
        {{end}}
    </tbody>
</table>
