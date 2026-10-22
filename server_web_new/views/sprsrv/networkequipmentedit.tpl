<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новое сетевое оборудование{{else}}Сетевое оборудование{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/networkequipmentlist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
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
                <label for="Name" class="form-label">Наименование (hostname)</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.SW.Name}}">
            </div>
            <div class="col-3">
                <label for="TypeId" class="form-label">Тип оборудования</label>
                <select id="TypeId" name="TypeId" class="form-select">
                    <option value="0">-</option>
                    {{range $t := .EquipmentTypes}}
                    <option value="{{$t.Id}}" {{if eq $.SW.TypeId $t.Id}}selected{{end}}>{{$t.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-3">
                <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                    <option value="0" {{if eq .SW.ItiliumStatus 0}}selected{{end}}>Не указано</option>
                    <option value="1" {{if eq .SW.ItiliumStatus 1}}selected{{end}}>В работе</option>
                    <option value="2" {{if eq .SW.ItiliumStatus 2}}selected{{end}}>Выведен</option>
                    <option value="3" {{if eq .SW.ItiliumStatus 3}}selected{{end}}>Списан</option>
                    <option value="4" {{if eq .SW.ItiliumStatus 4}}selected{{end}}>Свободно</option>
                    <option value="5" {{if eq .SW.ItiliumStatus 5}}selected{{end}}>На складе</option>
                    <option value="6" {{if eq .SW.ItiliumStatus 6}}selected{{end}}>Зарезервировано</option>
                    <option value="7" {{if eq .SW.ItiliumStatus 7}}selected{{end}}>В ремонте</option>
                    <option value="8" {{if eq .SW.ItiliumStatus 8}}selected{{end}}>В пути</option>
                    <option value="9" {{if eq .SW.ItiliumStatus 9}}selected{{end}}>Черновик</option>
                </select>
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
            <div class="col-3">
                <label for="ManagementIp" class="form-label">IP управления</label>
                <input type="text" id="ManagementIp" name="ManagementIp" class="form-control" value="{{.SW.ManagementIp}}">
            </div>
            <div class="col-3">
                <label for="MacAddress" class="form-label">MAC адрес</label>
                <input type="text" id="MacAddress" name="MacAddress" class="form-control" value="{{.SW.MacAddress}}">
            </div>
            <div class="col-6">
                <label for="NetworkRole" class="form-label">Принадлежность к прибору</label>
                <input type="text" id="NetworkRole" name="NetworkRole" class="form-control" value="{{.SW.NetworkRole}}" placeholder="Core / Access ...">
            </div>
            <div class="col-6">
                <label for="ReplacementModel" class="form-label">Модель предполагаемой замены</label>
                <input type="text" id="ReplacementModel" name="ReplacementModel" class="form-control" value="{{.SW.ReplacementModel}}">
            </div>
        </div>
    </div>

    <div class="hw-card mt-3">
        <h6>Размещение</h6>
        <div class="row g-3">
            <div class="col-3">
                <label for="DatacenterSiteId" class="form-label">ЦОД</label>
                <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                    <option value="0">-</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" {{if eq $.SW.DatacenterSiteId $dc.Id}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-3">
                <label for="RackCabinetId" class="form-label">Шкаф</label>
                <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                    <option value="0">-</option>
                    {{range $rc := .RackCabinets}}
                    <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.SW.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-3">
                <label for="StartUnitPos" class="form-label">Начальный юнит</label>
                <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{if gt .SW.StartUnitPos 0}}{{.SW.StartUnitPos}}{{end}}">
            </div>
            <div class="col-3">
                <label for="SizeUnit" class="form-label">Размер, юнитов</label>
                <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{if gt .SW.SizeUnit 0}}{{.SW.SizeUnit}}{{end}}">
            </div>
        </div>
    </div>

    <div class="hw-card mt-3">
        <h6>Учёт и закупка</h6>
        <div class="row g-3">
            <div class="col-4">
                <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.SW.TechnicalNumberIT}}">
            </div>
            <div class="col-4">
                <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.SW.InventoryNum}}">
            </div>
            <div class="col-4">
                <label for="InventoryNumBuh" class="form-label">Инв. номер (бух.)</label>
                <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.SW.InventoryNumBuh}}">
            </div>
            <div class="col-4">
                <label for="Owner" class="form-label">Автор / ответственный</label>
                <input type="text" id="Owner" name="Owner" class="form-control" value="{{.SW.Owner}}">
            </div>
            <div class="col-4">
                <label for="Counterparty" class="form-label">Поставщик</label>
                <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.SW.Counterparty}}">
            </div>
            <div class="col-4">
                <label for="ContractDoc" class="form-label">Документы закупки</label>
                <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.SW.ContractDoc}}">
            </div>
            <div class="col-4">
                <label for="DateContractDoc" class="form-label">Дата документа закупки</label>
                <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
            </div>
            <div class="col-4">
                <label for="Cost" class="form-label">Стоимость</label>
                <input type="text" id="Cost" name="Cost" class="form-control" value="{{printf "%.2f" .SW.Cost}}">
            </div>
            <div class="col-4">
                <label for="CostDiscount" class="form-label">Стоимость со скидкой</label>
                <input type="text" id="CostDiscount" name="CostDiscount" class="form-control" value="{{printf "%.2f" .SW.CostDiscount}}">
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
        <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
        <a href="/sprsrv/networkequipmentlist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
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
    // Выбор ЦОД сужает список шкафов по data-dc (шкаф сохраняется, если он из выбранного ЦОД).
    (function () {
        var dcSelect = document.getElementById('DatacenterSiteId');
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

        applyFilter();
        dcSelect.addEventListener('change', applyFilter);
    })();
</script>
