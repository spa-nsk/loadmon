<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новый тип сетевого оборудования{{else}}Тип сетевого оборудования{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>, устройств этого типа: <b>{{.EquipmentCount}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/networkequipmenttypelist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .TP}}
<form role="form" method="POST">
    <div class="hw-card">
        <div class="row g-3">
            <div class="col-12 col-md-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.TP.Name}}" required>
            </div>
            <div class="col-12">
                <label for="Annotation" class="form-label">Описание</label>
                <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.TP.Annotation}}</textarea>
            </div>
        </div>
    </div>
    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
        <a href="/sprsrv/networkequipmenttypelist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
{{end}}
