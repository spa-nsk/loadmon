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
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-bs-modules-btn" data-bs-toggle="tab" data-bs-target="#tab-bs-modules" type="button" role="tab" aria-controls="tab-bs-modules" aria-selected="false">Модули</button>
        </li>
        {{range $tab := .BsComponentTabs}}
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-bs-comp-{{$tab.Type}}-btn" data-bs-toggle="tab" data-bs-target="#tab-bs-comp-{{$tab.Type}}" type="button" role="tab" aria-controls="tab-bs-comp-{{$tab.Type}}" aria-selected="false">{{$tab.Name}}{{if $tab.Items}} <span class="badge text-bg-secondary">{{len $tab.Items}}</span>{{end}}</button>
        </li>
        {{end}}
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
                    <label for="StatusHistory" class="form-label">Дата статуса Итилиум</label>
                    <input type="date" id="StatusHistory" name="StatusHistory" class="form-control" value="{{.StatusHistoryStr}}">
                </div>
                <div class="col-12">
                    <label for="Description" class="form-label">Описание (Итилиум)</label>
                    <textarea id="Description" name="Description" class="form-control" rows="2">{{.BS.Description}}</textarea>
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

        <div class="tab-pane fade" id="tab-bs-modules" role="tabpanel" aria-labelledby="tab-bs-modules-btn" tabindex="0">
            <div class="mb-2">
                <h6 class="mb-1">Добавить модуль корзины</h6>
                <div class="text-muted small">Interconnect, OA/management, блоки питания, вентиляторы (и прочие). После «Добавить» строка попадёт на закладку своего типа; сохранение формы запишет весь набор модулей.</div>
            </div>
            <div class="row g-2 align-items-end" id="bsCompAddPanel">
                <div class="col-6 col-lg-2">
                    <label for="newCompType" class="form-label">Тип</label>
                    <select id="newCompType" class="form-select form-select-sm">
                        <option value="0">-</option>
                        {{range $t := .BsComponentTypes}}
                        <option value="{{$t.Value}}">{{$t.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompSlot" class="form-label">Отсек / слот</label>
                    <input type="text" id="newCompSlot" class="form-control form-control-sm" value="" placeholder="Bay 1, OK">
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompVendor" class="form-label">Производитель</label>
                    <input type="text" id="newCompVendor" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-3">
                    <label for="newCompModel" class="form-label">Модель</label>
                    <input type="text" id="newCompModel" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompSerial" class="form-label">Серийный №</label>
                    <input type="text" id="newCompSerial" class="form-control form-control-sm" value="">
                </div>
                <div class="col-12 col-lg-auto">
                    <button type="button" id="bsCompAddSubmit" class="btn btn-outline-secondary btn-sm">Добавить</button>
                </div>
            </div>
            <div id="bsCompAddError" class="text-danger small mt-2 d-none"></div>
        </div>

        {{range $tab := .BsComponentTabs}}
        <div class="tab-pane fade" id="tab-bs-comp-{{$tab.Type}}" role="tabpanel" aria-labelledby="tab-bs-comp-{{$tab.Type}}-btn" tabindex="0">
            <div class="text-muted small mb-2">Модули типа «{{$tab.Name}}». Удаление строки действует после сохранения формы. Данные обычно приходят из OneView.{{if or $tab.ShowRole $tab.ShowStatus}} Для interconnect/OA: роль и статус сохраняются отдельно; IP OA указывается в «Отсек / слот» (например «Bay 1, IP 10.0.0.1»).{{end}}</div>
            <div class="table-responsive">
                <table class="table table-sm align-middle mb-0">
                    <thead>
                        <tr>
                            {{if eq $tab.Type 0}}<th style="min-width:9rem">Тип</th>{{end}}
                            <th style="min-width:8rem">Отсек / слот</th>
                            <th style="min-width:7rem">Производитель</th>
                            <th style="min-width:10rem">Модель</th>
                            <th style="min-width:8rem">Серийный №</th>
                            {{if $tab.ShowCapacity}}<th style="min-width:6rem">Ёмкость, ГБ</th>{{end}}
                            {{if $tab.ShowSubType}}<th style="min-width:6rem">Подтип</th>{{end}}
                            {{if $tab.ShowRole}}<th style="min-width:7rem" title="Для OA/FLM — Active/Standby; IP указывается в поле «Отсек / слот»">Роль</th>{{end}}
                            {{if $tab.ShowStatus}}<th style="min-width:7rem">Статус</th>{{end}}
                            <th></th>
                        </tr>
                    </thead>
                    <tbody id="bsCompRows-{{$tab.Type}}" data-comp-type="{{$tab.Type}}" data-show-capacity="{{if $tab.ShowCapacity}}1{{else}}0{{end}}" data-show-subtype="{{if $tab.ShowSubType}}1{{else}}0{{end}}" data-show-role="{{if $tab.ShowRole}}1{{else}}0{{end}}" data-show-status="{{if $tab.ShowStatus}}1{{else}}0{{end}}">
                        {{range $c := $tab.Items}}
                        <tr class="bs-comp-row">
                            {{if eq $tab.Type 0}}
                            <td>
                                <select name="CompType" class="form-select form-select-sm">
                                    <option value="0">-</option>
                                    {{range $t := $.BsComponentTypes}}
                                    <option value="{{$t.Value}}" {{if eq $c.Type $t.Value}}selected{{end}}>{{$t.Name}}</option>
                                    {{end}}
                                </select>
                            </td>
                            <td><input type="text" name="CompSlot" class="form-control form-control-sm" value="{{$c.Slot}}"></td>
                            {{else}}
                            <td>
                                <input type="hidden" name="CompType" value="{{$tab.Type}}">
                                <input type="text" name="CompSlot" class="form-control form-control-sm" value="{{$c.Slot}}">
                            </td>
                            {{end}}
                            <td><input type="text" name="CompVendor" class="form-control form-control-sm" value="{{$c.Vendor}}"></td>
                            <td><input type="text" name="CompModel" class="form-control form-control-sm" value="{{$c.Model}}"></td>
                            <td><input type="text" name="CompSerialNum" class="form-control form-control-sm" value="{{$c.SerialNum}}"></td>
                            {{if $tab.ShowCapacity}}
                            <td><input type="number" step="0.01" name="CompCapacityGb" class="form-control form-control-sm" value="{{printf "%.2f" $c.CapacityGb}}"></td>
                            {{else}}
                            <td class="d-none"><input type="hidden" name="CompCapacityGb" value="{{printf "%.2f" $c.CapacityGb}}"></td>
                            {{end}}
                            {{if $tab.ShowSubType}}
                            <td>
                                <select name="CompSubType" class="form-select form-select-sm">
                                    {{range $s := $.HwComponentDiskSubTypes}}
                                    <option value="{{$s.Value}}" {{if eq $c.SubType $s.Value}}selected{{end}}>{{$s.Name}}</option>
                                    {{end}}
                                </select>
                            </td>
                            {{else}}
                            <td class="d-none"><input type="hidden" name="CompSubType" value="{{$c.SubType}}"></td>
                            {{end}}
                            {{if $tab.ShowRole}}
                            <td><input type="text" name="CompRole" class="form-control form-control-sm" value="{{$c.Role}}" placeholder="Active / Standby"></td>
                            {{else}}
                            <td class="d-none"><input type="hidden" name="CompRole" value="{{$c.Role}}"></td>
                            {{end}}
                            {{if $tab.ShowStatus}}
                            <td><input type="text" name="CompStatus" class="form-control form-control-sm" value="{{$c.Status}}" placeholder="OK / Present"></td>
                            {{else}}
                            <td class="d-none"><input type="hidden" name="CompStatus" value="{{$c.Status}}"></td>
                            {{end}}
                            <td><button type="button" class="btn btn-outline-danger btn-sm bs-comp-remove-row">Удалить</button></td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
        {{end}}

        <template id="bsCompRowTemplate">
            <tr class="bs-comp-row">
                <td>
                    <input type="hidden" name="CompType" value="">
                    <input type="text" name="CompSlot" class="form-control form-control-sm" value="">
                </td>
                <td><input type="text" name="CompVendor" class="form-control form-control-sm" value=""></td>
                <td><input type="text" name="CompModel" class="form-control form-control-sm" value=""></td>
                <td><input type="text" name="CompSerialNum" class="form-control form-control-sm" value=""></td>
                <td class="d-none"><input type="hidden" name="CompCapacityGb" value="0"></td>
                <td class="d-none"><input type="hidden" name="CompSubType" value="0"></td>
                <td class="bs-comp-role-cell"><input type="text" name="CompRole" class="form-control form-control-sm" value="" placeholder="Active / Standby"></td>
                <td class="d-none bs-comp-role-hidden-cell"><input type="hidden" name="CompRole" value="" class="bs-comp-role-hidden"></td>
                <td class="bs-comp-status-cell"><input type="text" name="CompStatus" class="form-control form-control-sm" value="" placeholder="OK / Present"></td>
                <td class="d-none bs-comp-status-hidden-cell"><input type="hidden" name="CompStatus" value="" class="bs-comp-status-hidden"></td>
                <td><button type="button" class="btn btn-outline-danger btn-sm bs-comp-remove-row">Удалить</button></td>
            </tr>
        </template>

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

    // Ручное добавление модулей корзины с раскладкой по закладкам типов.
    (function () {
        var tpl = document.getElementById('bsCompRowTemplate');
        var addBtn = document.getElementById('bsCompAddSubmit');
        var typeSelect = document.getElementById('newCompType');
        var errBox = document.getElementById('bsCompAddError');
        if (!tpl || !addBtn || !typeSelect) return;

        function bindRemove(row) {
            var btn = row.querySelector('.bs-comp-remove-row');
            if (!btn) return;
            btn.addEventListener('click', function () { row.remove(); });
        }
        Array.prototype.forEach.call(document.querySelectorAll('.bs-comp-row'), bindRemove);

        function showError(msg) {
            if (!errBox) return;
            if (!msg) { errBox.classList.add('d-none'); errBox.textContent = ''; return; }
            errBox.textContent = msg;
            errBox.classList.remove('d-none');
        }

        function clearAddForm() {
            typeSelect.value = '0';
            ['newCompSlot', 'newCompVendor', 'newCompModel', 'newCompSerial'].forEach(function (id) {
                var el = document.getElementById(id);
                if (el) el.value = '';
            });
        }

        addBtn.addEventListener('click', function () {
            showError('');
            var type = typeSelect.value;
            if (!type || type === '0') {
                showError('Выберите тип модуля.');
                return;
            }
            var tbody = document.getElementById('bsCompRows-' + type);
            if (!tbody) {
                showError('Для выбранного типа нет закладки.');
                return;
            }
            var node = tpl.content.cloneNode(true);
            var row = node.querySelector('.bs-comp-row');
            if (!row) return;
            var typeInput = row.querySelector('input[name="CompType"]');
            if (typeInput) typeInput.value = type;
            row.querySelector('input[name="CompSlot"]').value = (document.getElementById('newCompSlot') || {}).value || '';
            row.querySelector('input[name="CompVendor"]').value = (document.getElementById('newCompVendor') || {}).value || '';
            row.querySelector('input[name="CompModel"]').value = (document.getElementById('newCompModel') || {}).value || '';
            row.querySelector('input[name="CompSerialNum"]').value = (document.getElementById('newCompSerial') || {}).value || '';
            var showRole = tbody.getAttribute('data-show-role') === '1';
            var showStatus = tbody.getAttribute('data-show-status') === '1';
            var roleCell = row.querySelector('.bs-comp-role-cell');
            var roleHidden = row.querySelector('.bs-comp-role-hidden-cell');
            var statusCell = row.querySelector('.bs-comp-status-cell');
            var statusHidden = row.querySelector('.bs-comp-status-hidden-cell');
            if (roleCell && roleHidden) {
                if (showRole) { roleHidden.remove(); }
                else { roleCell.remove(); }
            }
            if (statusCell && statusHidden) {
                if (showStatus) { statusHidden.remove(); }
                else { statusCell.remove(); }
            }
            bindRemove(row);
            tbody.appendChild(node);
            clearAddForm();
            var tabBtn = document.getElementById('tab-bs-comp-' + type + '-btn');
            if (tabBtn && window.bootstrap && bootstrap.Tab) {
                bootstrap.Tab.getOrCreateInstance(tabBtn).show();
            }
        });
    })();
</script>
