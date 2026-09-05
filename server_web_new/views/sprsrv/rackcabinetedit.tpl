<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новый шкаф{{else}}Шкаф{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/rackcabinetlist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .RC}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Общие сведения</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.RC.Name}}">
            </div>
            <div class="col-6">
                <label for="AltName" class="form-label">Альтернативное наименование</label>
                <input type="text" id="AltName" name="AltName" class="form-control" value="{{.RC.AltName}}">
            </div>
            <div class="col-6">
                <label for="DatacenterSiteId" class="form-label">ЦОД</label>
                <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                    <option value="0">-</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" {{if eq $.RC.DatacenterSiteId $dc.Id}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.RC.Model}}">
            </div>
            <div class="col-6">
                <label for="SerialNumber" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNumber" name="SerialNumber" class="form-control" value="{{.RC.SerialNumber}}">
            </div>
            <div class="col-3">
                <label for="Width" class="form-label">Ширина</label>
                <input type="number" id="Width" name="Width" class="form-control" value="{{.RC.Width}}">
            </div>
            <div class="col-3">
                <label for="Depth" class="form-label">Глубина</label>
                <input type="number" id="Depth" name="Depth" class="form-control" value="{{.RC.Depth}}">
            </div>
            <div class="col-3">
                <label for="Height" class="form-label">Высота</label>
                <input type="number" id="Height" name="Height" class="form-control" value="{{.RC.Height}}">
            </div>
            <div class="col-3">
                <label for="UHeight" class="form-label">Высота, юниты</label>
                <input type="number" id="UHeight" name="UHeight" class="form-control" value="{{.RC.UHeight}}">
            </div>
            <div class="col-12">
                <label for="Annotation" class="form-label">Описание</label>
                <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.RC.Annotation}}</textarea>
            </div>
        </div>
    </div>

    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
    </div>
</form>
{{end}}
