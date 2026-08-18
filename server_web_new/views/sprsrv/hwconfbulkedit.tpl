<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Групповое редактирование конфигураций</h2>
        <div class="text-muted">Выбрано записей: <b>{{.IdsCount}}</b></div>
    </div>
    <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if gt .IdsCount 0}}
<form role="form" id="hwBulkEditForm" method="POST">
    <input type="hidden" name="ApplyBulk" value="1">
    <input type="hidden" name="ids" value="{{.IdsCSV}}">

    <div class="text-muted small mb-3">
        Отметьте чекбокс рядом с полем, чтобы применить его значение ко всем выбранным записям.
        Неотмеченные поля выбранных серверов останутся без изменений.
    </div>

    <div class="hw-card mb-3">
        <div class="row g-3">
            <div class="col-6 col-lg-3">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_Active" name="apply_Active">
                    </div>
                    <div class="flex-grow-1">
                        <label for="Active" class="form-label text-nowrap">Статус</label>
                        <select id="Active" name="Active" class="form-select">
                            <option value="0" {{if eq .HW.Active 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .HW.Active 1}}selected{{end}}>Активен</option>
                            <option value="2" {{if eq .HW.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                            <option value="3" {{if eq .HW.Active 3}}selected{{end}}>Списан</option>
                        </select>
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-3">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_DatacenterRoom" name="apply_DatacenterRoom">
                    </div>
                    <div class="flex-grow-1">
                        <label for="DatacenterRoom" class="form-label text-nowrap">ЦОД</label>
                        <input type="text" id="DatacenterRoom" name="DatacenterRoom" class="form-control" value="{{.HW.DatacenterRoom}}">
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-3">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_RackNum" name="apply_RackNum">
                    </div>
                    <div class="flex-grow-1">
                        <label for="RackNum" class="form-label text-nowrap">Номер стойки</label>
                        <input type="text" id="RackNum" name="RackNum" class="form-control" value="{{.HW.RackNum}}">
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-3">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_DatacenterName" name="apply_DatacenterName">
                    </div>
                    <div class="flex-grow-1">
                        <label for="DatacenterName" class="form-label text-nowrap">Серверное помещение (OneView)</label>
                        <input type="text" id="DatacenterName" name="DatacenterName" class="form-control" value="{{.HW.DatacenterName}}">
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-4">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_RackName" name="apply_RackName">
                    </div>
                    <div class="flex-grow-1">
                        <label for="RackName" class="form-label text-nowrap">Шкаф (OneView)</label>
                        <input type="text" id="RackName" name="RackName" class="form-control" value="{{.HW.RackName}}">
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-4">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_StartUnitPos" name="apply_StartUnitPos">
                    </div>
                    <div class="flex-grow-1">
                        <label for="StartUnitPos" class="form-label text-nowrap">Юнит начала</label>
                        <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{.HW.StartUnitPos}}">
                    </div>
                </div>
            </div>
            <div class="col-6 col-lg-4">
                <div class="d-flex align-items-end gap-2">
                    <div class="form-check mt-4">
                        <input class="form-check-input" type="checkbox" id="apply_SizeUnit" name="apply_SizeUnit">
                    </div>
                    <div class="flex-grow-1">
                        <label for="SizeUnit" class="form-label text-nowrap">Занимаемых юнитов</label>
                        <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{.HW.SizeUnit}}">
                    </div>
                </div>
            </div>
        </div>
    </div>

    <button type="submit" class="btn btn-primary rounded-pill px-4">Применить к выбранным записям</button>
    <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
</form>
{{end}}

<!-- Стили .hw-card вынесены в общий server_web_new/static/css/loadmon.css
     (подключён в header.tpl). -->
