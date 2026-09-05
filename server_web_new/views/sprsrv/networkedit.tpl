<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая сеть{{else}}Сеть{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/networklist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .NW}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Общие сведения</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.NW.Name}}">
            </div>
            <div class="col-6">
                <label for="Vlan" class="form-label">VLAN</label>
                <input type="number" id="Vlan" name="Vlan" class="form-control" value="{{.NW.Vlan}}">
            </div>
            <div class="col-6">
                <label for="Subnet" class="form-label">Подсеть (CIDR)</label>
                <input type="text" id="Subnet" name="Subnet" class="form-control" value="{{.NW.Subnet}}" placeholder="10.10.10.0/24">
            </div>
            <div class="col-6">
                <label for="Gateway" class="form-label">Шлюз</label>
                <input type="text" id="Gateway" name="Gateway" class="form-control" value="{{.NW.Gateway}}">
            </div>
            <div class="col-12">
                <label for="Annotation" class="form-label">Описание</label>
                <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.NW.Annotation}}</textarea>
            </div>
        </div>
    </div>

    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
    </div>
</form>
{{end}}
