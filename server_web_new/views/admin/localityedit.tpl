<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новый населённый пункт{{else}}Населённый пункт{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/admin/localities" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .Loc}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Сведения</h6>
        <div class="row g-3">
            <div class="col-12 col-md-4">
                <label for="Oktmo" class="form-label">Код ОКТМО (11 цифр)</label>
                <input type="text" id="Oktmo" name="Oktmo" class="form-control" maxlength="11" required value="{{.Loc.Oktmo}}">
                <div class="form-text">На этот код ссылается поле «Код населённого пункта» площадок/ЦОД.</div>
            </div>
            <div class="col-6 col-md-2">
                <label for="TypeShort" class="form-label">Тип</label>
                <input type="text" id="TypeShort" name="TypeShort" class="form-control" value="{{.Loc.TypeShort}}">
            </div>
            <div class="col-12 col-md-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" required value="{{.Loc.Name}}">
            </div>
            <div class="col-12 col-md-6">
                <label for="FullName" class="form-label">Полное наименование</label>
                <input type="text" id="FullName" name="FullName" class="form-control" value="{{.Loc.FullName}}">
                <div class="form-text">Если пусто - формируется из типа и наименования.</div>
            </div>
            <div class="col-12 col-md-6">
                <label for="Municipality" class="form-label">Муниципальное образование</label>
                <input type="text" id="Municipality" name="Municipality" class="form-control" value="{{.Loc.Municipality}}">
            </div>
            <div class="col-12 col-md-8">
                <label for="RegionName" class="form-label">Субъект РФ</label>
                <input type="text" id="RegionName" name="RegionName" class="form-control" required value="{{.Loc.RegionName}}">
            </div>
            <div class="col-6 col-md-4">
                <label for="RegionCode" class="form-label">Код региона</label>
                <input type="text" id="RegionCode" name="RegionCode" class="form-control" maxlength="2" value="{{.Loc.RegionCode}}">
                <div class="form-text">Если пусто - первые 2 цифры кода ОКТМО.</div>
            </div>
        </div>
    </div>

    <div class="mt-3 d-flex gap-2">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
        {{if not .IsNew}}
        <button type="submit" name="Delete" value="1" class="btn btn-outline-danger rounded-pill px-4" onclick="return confirm('Удалить населённый пункт?');">Удалить</button>
        {{end}}
    </div>
</form>
{{end}}
