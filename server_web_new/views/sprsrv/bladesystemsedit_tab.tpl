<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая блейд-корзина{{else}}Блейд-корзина{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/bladesystemslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .BS}}
<form role="form" id="bsEditForm" method="POST" class="pb-4">
    <ul class="nav nav-tabs" id="bsEditTabs" role="tablist">
        <li class="nav-item" role="presentation">
            <button class="nav-link active" id="tab-bs-general-btn" data-bs-toggle="tab" data-bs-target="#tab-bs-general" type="button" role="tab" aria-controls="tab-bs-general" aria-selected="true">Общие сведения</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-bs-account-btn" data-bs-toggle="tab" data-bs-target="#tab-bs-account" type="button" role="tab" aria-controls="tab-bs-account" aria-selected="false">Учетные данные и размещение</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-bs-contract-btn" data-bs-toggle="tab" data-bs-target="#tab-bs-contract" type="button" role="tab" aria-controls="tab-bs-contract" aria-selected="false">Учёт и договор</button>
        </li>
    </ul>

    <div class="tab-content border border-top-0 rounded-bottom p-3 mb-3" id="bsEditTabContent">

        <div class="tab-pane fade show active" id="tab-bs-general" role="tabpanel" aria-labelledby="tab-bs-general-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-4">
                    <label for="Name" class="form-label">Наименование</label>
                    <input type="text" id="Name" name="Name" class="form-control" value="{{.BS.Name}}">
                </div>
                <div class="col-4">
                    <label for="Type" class="form-label">Тип</label>
                    <input type="text" id="Type" name="Type" class="form-control" value="{{.BS.Type}}" placeholder="например, c7000">
                </div>
                <div class="col-4">
                    <label for="Vendor" class="form-label">Производитель</label>
                    <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.BS.Vendor}}">
                </div>
                <div class="col-4">
                    <label for="Model" class="form-label">Модель</label>
                    <input type="text" id="Model" name="Model" class="form-control" value="{{.BS.Model}}">
                </div>
                <div class="col-4">
                    <label for="PartNum" class="form-label">Продуктовый номер</label>
                    <input type="text" id="PartNum" name="PartNum" class="form-control" value="{{.BS.PartNum}}">
                </div>
                <div class="col-4">
                    <label for="SerialNum" class="form-label">Серийный номер</label>
                    <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.BS.SerialNum}}">
                </div>
                <div class="col-4">
                    <label class="form-label">Ср. мощность 24ч, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .BS.PowerAvg24hW}}" readonly title="OneView enclosure или оценка по блейдам">
                </div>
                <div class="col-4">
                    <label class="form-label">Пик 24ч, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .BS.PowerPeak24hW}}" readonly>
                </div>
                <div class="col-4">
                    <label class="form-label">Паспорт max, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .BS.PowerRatedMaxW}}" readonly>
                </div>
                <div class="col-3">
                    <label for="PowerState" class="form-label">Состояние питания</label>
                    <select id="PowerState" name="PowerState" class="form-select">
                        <option value="0" {{if eq .BS.PowerState 0}}selected{{end}}>Не указано</option>
                        <option value="1" {{if eq .BS.PowerState 1}}selected{{end}}>Включен</option>
                        <option value="2" {{if eq .BS.PowerState 2}}selected{{end}}>Выключен</option>
                    </select>
                </div>
                <div class="col-3">
                    <label for="Active" class="form-label">Статус</label>
                    <select id="Active" name="Active" class="form-select">
                        <option value="0" {{if eq .BS.Active 0}}selected{{end}}>Не указано</option>
                        <option value="1" {{if eq .BS.Active 1}}selected{{end}}>Активен</option>
                        <option value="2" {{if eq .BS.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                    </select>
                </div>
                <div class="col-3">
                    <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                    <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                        {{range $s := .ItiliumStatuses}}
                        <option value="{{$s.Value}}" {{if eq $.BS.ItiliumStatus $s.Value}}selected{{end}}>{{$s.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-3">
                    <label for="Size" class="form-label">Количество слотов</label>
                    <input type="number" id="Size" name="Size" class="form-control" value="{{.BS.Size}}">
                </div>
                <div class="col-3">
                    <label for="Used" class="form-label">Занято слотов</label>
                    <input type="number" id="Used" name="Used" class="form-control" value="{{.BS.Used}}">
                </div>
                <div class="col-4">
                    <label for="Owner" class="form-label">Владелец</label>
                    <input type="text" id="Owner" name="Owner" class="form-control" value="{{.BS.Owner}}">
                </div>
                <div class="col-5">
                    <label for="Anotation" class="form-label">Примечание</label>
                    <input type="text" id="Anotation" name="Anotation" class="form-control" value="{{.BS.Anotation}}">
                </div>
            </div>
        </div>

        <div class="tab-pane fade" id="tab-bs-account" role="tabpanel" aria-labelledby="tab-bs-account-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-4">
                    <label for="TechnicalNumberIT" class="form-label">Технический номер</label>
                    <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.BS.TechnicalNumberIT}}">
                </div>
                <div class="col-4">
                    <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                    <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.BS.InventoryNum}}">
                </div>
                <div class="col-4">
                    <label for="InventoryNumBuh" class="form-label">Инв. номер (бух.)</label>
                    <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.BS.InventoryNumBuh}}">
                </div>
                <div class="col-4">
                    <label for="DatacenterSiteId" class="form-label">ЦОД</label>
                    <select id="DatacenterSiteId" class="form-select">
                        <option value="0">-</option>
                        {{range $dc := .DatacenterSites}}
                        <option value="{{$dc.Id}}">{{$dc.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-4">
                    <label for="RackCabinetId" class="form-label">Шкаф (справочник)</label>
                    <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                        <option value="0">-</option>
                        {{range $rc := .RackCabinets}}
                        <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.BS.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-2">
                    <label for="StartUnitPos" class="form-label">Юнит начала</label>
                    <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{.BS.StartUnitPos}}">
                </div>
                <div class="col-2">
                    <label for="EndUnitPos" class="form-label">Юнит конца</label>
                    <input type="number" id="EndUnitPos" name="EndUnitPos" class="form-control" value="{{.BS.EndUnitPos}}">
                </div>
            </div>
        </div>

        <div class="tab-pane fade" id="tab-bs-contract" role="tabpanel" aria-labelledby="tab-bs-contract-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-6">
                    <label for="Counterparty" class="form-label">Контрагент</label>
                    <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.BS.Counterparty}}">
                </div>
                <div class="col-6">
                    <label for="ContractDoc" class="form-label">Номер договора</label>
                    <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.BS.ContractDoc}}">
                </div>
                <div class="col-6">
                    <label for="DateContractDoc" class="form-label">Дата договора</label>
                    <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
                </div>
                <div class="col-6">
                    <label for="Invoice" class="form-label">Накладная</label>
                    <input type="text" id="Invoice" name="Invoice" class="form-control" value="{{.BS.Invoice}}">
                </div>
                <div class="col-6">
                    <label for="DateInvoice" class="form-label">Дата получения</label>
                    <input type="date" id="DateInvoice" name="DateInvoice" class="form-control" value="{{.DateInvoiceStr}}">
                </div>
                <div class="col-6">
                    <label for="ServiceContract" class="form-label">Сервисный контракт</label>
                    <input type="text" id="ServiceContract" name="ServiceContract" class="form-control" value="{{.BS.ServiceContract}}">
                </div>
                <div class="col-6">
                    <label for="DateService" class="form-label">Окончание гарантии</label>
                    <input type="date" id="DateService" name="DateService" class="form-control" value="{{.DateServiceStr}}">
                </div>
            </div>
        </div>

    </div>

    <div class="d-flex gap-2">
        <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
        <a href="/sprsrv/bladesystemslist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
{{end}}

<script>
    // Каскадный выбор "ЦОД -> Шкаф" - как в bladesystemsedit.tpl.
    (function () {
        var dcSelect = document.getElementById('DatacenterSiteId');
        var rackSelect = document.getElementById('RackCabinetId');
        if (!dcSelect || !rackSelect) return;

        var allRackOptions = Array.prototype.slice.call(rackSelect.options);

        function applyFilter(preserveSelection) {
            var dc = dcSelect.value;
            var currentValue = rackSelect.value;
            rackSelect.innerHTML = '';
            allRackOptions.forEach(function (opt) {
                var optDc = opt.getAttribute('data-dc') || '0';
                if (opt.value === '0' || dc === '0' || optDc === dc) {
                    rackSelect.appendChild(opt);
                }
            });
            if (preserveSelection) {
                var stillPresent = Array.prototype.some.call(rackSelect.options, function (opt) {
                    return opt.value === currentValue;
                });
                rackSelect.value = stillPresent ? currentValue : '0';
            }
        }

        var selectedOption = allRackOptions.find(function (opt) { return opt.selected; });
        if (selectedOption && selectedOption.value !== '0') {
            dcSelect.value = selectedOption.getAttribute('data-dc') || '0';
        }
        applyFilter(true);

        dcSelect.addEventListener('change', function () { applyFilter(true); });
    })();
</script>
