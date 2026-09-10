<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая площадка/ЦОД{{else}}Площадка/ЦОД{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/datacentersitelist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .DC}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Общие сведения</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.DC.Name}}">
            </div>
            <div class="col-6">
                <label for="AltName" class="form-label">Альтернативное наименование</label>
                <input type="text" id="AltName" name="AltName" class="form-control" value="{{.DC.AltName}}">
            </div>
            <div class="col-12">
                <div class="form-check form-switch">
                    <input class="form-check-input" type="checkbox" id="IsDatacenter" name="IsDatacenter" value="1" {{if .DC.IsDatacenter}}checked{{end}}>
                    <label class="form-check-label" for="IsDatacenter">Признак "ЦОД"</label>
                </div>
                <div class="form-text">Отмечено - площадка является ЦОД, там размещается активное оборудование (которое может быть выключено). Снято - это просто площадка размещения без активного оборудования (склад, рабочая комната, площадка для списанного и т.п.).</div>
            </div>
            <div class="col-3">
                <label for="Width" class="form-label">Ширина</label>
                <input type="number" id="Width" name="Width" class="form-control" value="{{.DC.Width}}">
            </div>
            <div class="col-3">
                <label for="Height" class="form-label">Высота</label>
                <input type="number" id="Height" name="Height" class="form-control" value="{{.DC.Height}}">
            </div>
            <div class="col-12">
                <label for="Annotation" class="form-label">Описание</label>
                <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.DC.Annotation}}</textarea>
            </div>
        </div>
    </div>

    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
    </div>
</form>
{{end}}
