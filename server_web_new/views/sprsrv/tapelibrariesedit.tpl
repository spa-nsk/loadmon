<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая ленточная библиотека{{else}}Ленточная библиотека{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/tapelibrarieslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .TL}}
<form role="form" id="tlEditForm" method="POST">
    <div class="row g-3 align-items-start">
        <div class="col-12">
            <div class="hw-card">
                <h6>Общие сведения</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="Name" class="form-label">Название</label>
                        <input type="text" id="Name" name="Name" class="form-control" value="{{.TL.Name}}">
                    </div>
                    <div class="col-6">
                        <label for="Model" class="form-label">Марка / модель</label>
                        <input type="text" id="Model" name="Model" class="form-control" value="{{.TL.Model}}">
                    </div>
                    <div class="col-6">
                        <label for="SerialNum" class="form-label">Серийный номер</label>
                        <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.TL.SerialNum}}">
                    </div>
                    <div class="col-6">
                        <label for="IPAddress" class="form-label">IP адрес</label>
                        <input type="text" id="IPAddress" name="IPAddress" class="form-control" value="{{.TL.IPAddress}}">
                    </div>
                    <div class="col-6">
                        <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                        <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                            {{range $s := .ItiliumStatuses}}
                            <option value="{{$s.Value}}" {{if eq $.TL.ItiliumStatus $s.Value}}selected{{end}}>{{$s.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="StatusHistory" class="form-label">Дата статуса Итилиум</label>
                        <input type="date" id="StatusHistory" name="StatusHistory" class="form-control" value="{{.StatusHistoryStr}}">
                    </div>
                    <div class="col-12">
                        <label for="Description" class="form-label">Описание (Итилиум)</label>
                        <textarea id="Description" name="Description" class="form-control" rows="2">{{.TL.Description}}</textarea>
                    </div>
                    <div class="col-6">
                        <label for="Owner" class="form-label">Автор / ответственный</label>
                        <input type="text" id="Owner" name="Owner" class="form-control" value="{{.TL.Owner}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Подключение</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="PcName" class="form-label">Имя ПК (сервер подключения)</label>
                        <input type="text" id="PcName" name="PcName" class="form-control" value="{{.TL.PcName}}">
                        <div class="form-text">Если сервер не указан, при сохранении он ищется в справочнике серверов по этому имени.</div>
                    </div>
                    <div class="col-3">
                        <label for="ServerId" class="form-label">Id сервера в справочнике</label>
                        <input type="number" id="ServerId" name="ServerId" class="form-control" value="{{if gt .TL.ServerId 0}}{{.TL.ServerId}}{{end}}">
                    </div>
                    <div class="col-3">
                        <label class="form-label">Сервер</label>
                        <div class="form-control-plaintext">
                            {{if .ServerName}}<a href="/sprsrv/updatebase-tab/{{.TL.ServerId}}" target="_blank">{{.ServerName}}</a>{{else}}—{{end}}
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Размещение</h6>
                <div class="row g-3 align-items-end">
                    <div class="col-3">
                        <label for="tlDatacenterSite" class="form-label">ЦОД</label>
                        <select id="tlDatacenterSite" class="form-select">
                            <option value="0">-</option>
                            {{range $dc := .DatacenterSites}}
                            <option value="{{$dc.Id}}">{{$dc.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-3">
                        <label for="RackCabinetId" class="form-label">Шкаф</label>
                        <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                            <option value="0">-</option>
                            {{range $rc := .RackCabinets}}
                            <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.TL.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-3">
                        <label for="StartUnitPos" class="form-label">Начальный юнит</label>
                        <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{if gt .TL.StartUnitPos 0}}{{.TL.StartUnitPos}}{{end}}">
                    </div>
                    <div class="col-3">
                        <label for="SizeUnit" class="form-label">Размер, юнитов</label>
                        <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{if gt .TL.SizeUnit 0}}{{.TL.SizeUnit}}{{end}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Учёт и договор</h6>
                <div class="row g-3">
                    <div class="col-4">
                        <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                        <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.TL.TechnicalNumberIT}}">
                    </div>
                    <div class="col-4">
                        <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                        <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.TL.InventoryNum}}">
                    </div>
                    <div class="col-4">
                        <label for="InventoryNumBuh" class="form-label">Инв. номер (бух.)</label>
                        <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.TL.InventoryNumBuh}}">
                    </div>
                    <div class="col-6">
                        <label for="Counterparty" class="form-label">Поставщик</label>
                        <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.TL.Counterparty}}">
                    </div>
                    <div class="col-6">
                        <label for="ContractDoc" class="form-label">Документы закупки</label>
                        <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.TL.ContractDoc}}">
                    </div>
                    <div class="col-6">
                        <label for="DateContractDoc" class="form-label">Дата документа закупки</label>
                        <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
                    </div>
                    <div class="col-12">
                        <label for="Anotation" class="form-label">Описание</label>
                        <input type="text" id="Anotation" name="Anotation" class="form-control" value="{{.TL.Anotation}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
            <a href="/sprsrv/tapelibrarieslist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
        </div>
    </div>
</form>
{{end}}

<style>
    .hw-card h6 {
        text-transform: uppercase;
        font-size: 0.8rem;
        letter-spacing: 0.03em;
    }
</style>

<script>
    // Каскад "ЦОД -> Шкаф": select ЦОД сам не отправляется, он сужает список
    // опций "Шкаф" по data-dc (как в storagesystemsedit.tpl).
    (function () {
        var dcSelect = document.getElementById('tlDatacenterSite');
        var rackSelect = document.getElementById('RackCabinetId');
        if (!dcSelect || !rackSelect) return;
        var allOptions = Array.prototype.slice.call(rackSelect.options);

        function applyFilter() {
            var dc = dcSelect.value;
            var current = rackSelect.value;
            rackSelect.innerHTML = '';
            allOptions.forEach(function (opt) {
                var optDc = opt.getAttribute('data-dc') || '0';
                if (opt.value === '0' || dc === '0' || optDc === dc) rackSelect.appendChild(opt);
            });
            var present = Array.prototype.some.call(rackSelect.options, function (o) { return o.value === current; });
            rackSelect.value = present ? current : '0';
        }

        var sel = allOptions.find(function (opt) { return opt.selected; });
        if (sel && sel.value !== '0') dcSelect.value = sel.getAttribute('data-dc') || '0';
        applyFilter();
        dcSelect.addEventListener('change', applyFilter);
    })();
</script>
