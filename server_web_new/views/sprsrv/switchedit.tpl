<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новый коммутатор{{else}}Коммутатор{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/switchlist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .SW}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Общие сведения</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.SW.Name}}">
            </div>
            <div class="col-6">
                <label for="Vendor" class="form-label">Производитель</label>
                <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.SW.Vendor}}">
            </div>
            <div class="col-6">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.SW.Model}}">
            </div>
            <div class="col-6">
                <label for="SerialNum" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.SW.SerialNum}}">
            </div>
            <div class="col-6">
                <label for="ManagementIp" class="form-label">IP управления</label>
                <input type="text" id="ManagementIp" name="ManagementIp" class="form-control" value="{{.SW.ManagementIp}}">
            </div>
        </div>
    </div>

    <div class="hw-card mt-3">
        <h6>Размещение</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="DatacenterSiteId" class="form-label">ЦОД</label>
                <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                    <option value="0">-</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" {{if eq $.SW.DatacenterSiteId $dc.Id}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6">
                <label for="RackCabinetId" class="form-label">Шкаф</label>
                <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                    <option value="0">-</option>
                    {{range $rc := .RackCabinets}}
                    <option value="{{$rc.Id}}" {{if eq $.SW.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
    </div>

    <div class="hw-card mt-3">
        <h6>Связанные сети</h6>
        <div class="row g-2">
            {{range $nw := .Networks}}
            <div class="col-6 col-md-4 col-lg-3">
                <div class="form-check">
                    <input class="form-check-input" type="checkbox" name="NetworkIds" value="{{$nw.Id}}" id="net{{$nw.Id}}" {{if index $.LinkedNetworkIds $nw.Id}}checked{{end}}>
                    <label class="form-check-label" for="net{{$nw.Id}}">{{$nw.Name}}</label>
                </div>
            </div>
            {{else}}
            <div class="col-12 text-muted small">В справочнике сетей пока нет записей.</div>
            {{end}}
        </div>
    </div>

    <div class="hw-card mt-3">
        <h6>Описание</h6>
        <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.SW.Annotation}}</textarea>
    </div>

    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
    </div>
</form>
{{end}}
