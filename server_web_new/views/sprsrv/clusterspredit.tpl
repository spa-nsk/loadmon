<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Кластер</h2>
        <div class="text-muted">Id записи: <b>{{.Id}}</b></div>
    </div>
    <a href="/sprsrv/clustersprlist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .CL}}
<form role="form" id="clEditForm" method="POST">
    <div class="row g-3 align-items-start">
        <div class="col-12">
            <div class="hw-card">
                <h6>Общие сведения</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="Name" class="form-label">Наименование</label>
                        <input type="text" id="Name" name="Name" class="form-control" value="{{.CL.Name}}">
                    </div>
                    <div class="col-6">
                        <label for="ClusterId" class="form-label">ClusterId</label>
                        <input type="text" id="ClusterId" name="ClusterId" class="form-control" value="{{.CL.ClusterId}}" placeholder="например, s-msk-vc-01.global.bcs:domain-c2638">
                    </div>
                    <div class="col-6">
                        <label for="Vcenter" class="form-label">vCenter</label>
                        <input type="text" id="Vcenter" name="Vcenter" class="form-control" value="{{.CL.Vcenter}}">
                    </div>
                    <div class="col-6">
                        <label for="Datacenter" class="form-label">Датацентр</label>
                        <input type="text" id="Datacenter" name="Datacenter" class="form-control" value="{{.CL.Datacenter}}">
                    </div>
                    <div class="col-6">
                        <label for="TypeWork" class="form-label">Среда</label>
                        <select id="TypeWork" name="TypeWork" class="form-select">
                            <option value="0" {{if eq .CL.TypeWork 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .CL.TypeWork 1}}selected{{end}}>Продуктовая среда</option>
                            <option value="2" {{if eq .CL.TypeWork 2}}selected{{end}}>Внутренний тест</option>
                            <option value="3" {{if eq .CL.TypeWork 3}}selected{{end}}>Внешний тест</option>
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="Stat" class="form-label">Статус</label>
                        <select id="Stat" name="Stat" class="form-select">
                            <option value="0" {{if eq .CL.Stat 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .CL.Stat 1}}selected{{end}}>Активен</option>
                            <option value="2" {{if eq .CL.Stat 2}}selected{{end}}>Выведен из работы</option>
                        </select>
                    </div>
                    <div class="col-12">
                        <label for="Annotation" class="form-label">Примечание</label>
                        <input type="text" id="Annotation" name="Annotation" class="form-control" value="{{.CL.Annotation}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
            <a href="/sprsrv/clustersprlist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
        </div>
    </div>
</form>
{{end}}

<!-- Базовые стили .hw-card/.hw-card hr вынесены в общий
     server_web_new/static/css/loadmon.css - здесь остаётся только
     дополнение для заголовков карточек (заглавными буквами), идентичное
     /sprsrv/storagesystemsedit и /sprsrv/bladesystemsedit. Добавление новой
     записи с этой формы не предусмотрено - справочник наполняется
     автоматически. -->
<style>
    .hw-card h6 {
        text-transform: uppercase;
        font-size: 0.8rem;
        letter-spacing: 0.03em;
    }
</style>
