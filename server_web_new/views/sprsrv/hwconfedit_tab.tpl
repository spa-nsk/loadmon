<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая конфигурация сервера{{else}}Конфигурация сервера{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .HW}}
<form role="form" id="hwEditForm" method="POST" class="pb-4">
    <ul class="nav nav-tabs flex-wrap" id="hwEditTabs" role="tablist">
        <li class="nav-item" role="presentation">
            <button class="nav-link active" id="tab-hw-general-btn" data-bs-toggle="tab" data-bs-target="#tab-hw-general" type="button" role="tab" aria-controls="tab-hw-general" aria-selected="true">Общие сведения</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-hw-account-btn" data-bs-toggle="tab" data-bs-target="#tab-hw-account" type="button" role="tab" aria-controls="tab-hw-account" aria-selected="false">Учетные данные и размещение</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-hw-base-btn" data-bs-toggle="tab" data-bs-target="#tab-hw-base" type="button" role="tab" aria-controls="tab-hw-base" aria-selected="false">Базовая конфигурация</button>
        </li>
        {{range $tab := .HwComponentTabs}}
        <li class="nav-item" role="presentation">
            <button class="nav-link" id="tab-hw-comp-{{$tab.Type}}-btn" data-bs-toggle="tab" data-bs-target="#tab-hw-comp-{{$tab.Type}}" type="button" role="tab" aria-controls="tab-hw-comp-{{$tab.Type}}" aria-selected="false">{{$tab.Name}}{{if $tab.Items}} <span class="badge text-bg-secondary">{{len $tab.Items}}</span>{{end}}</button>
        </li>
        {{end}}
    </ul>

    <div class="tab-content border border-top-0 rounded-bottom p-3 mb-3" id="hwEditTabContent">

        <div class="tab-pane fade show active" id="tab-hw-general" role="tabpanel" aria-labelledby="tab-hw-general-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-4">
                    <label for="Vendor" class="form-label">Производитель</label>
                    <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.HW.Vendor}}">
                </div>
                <div class="col-4">
                    <label for="Model" class="form-label">Модель</label>
                    <input type="text" id="Model" name="Model" class="form-control" value="{{.HW.Model}}">
                </div>
                <div class="col-4">
                    <label for="PartNum" class="form-label">Номер продукта</label>
                    <input type="text" id="PartNum" name="PartNum" class="form-control" value="{{.HW.PartNum}}">
                </div>
                <div class="col-4">
                    <label for="SerialNum" class="form-label">Серийный номер</label>
                    <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.HW.SerialNum}}">
                </div>
                <div class="col-4">
                    <label for="VirtualSerialNum" class="form-label">Виртуальный серийный номер</label>
                    <input type="text" id="VirtualSerialNum" name="VirtualSerialNum" class="form-control" value="{{.HW.VirtualSerialNum}}">
                </div>
                <div class="col-4">
                    <label for="FormFactor" class="form-label">Тип сервера</label>
                    <select id="FormFactor" name="FormFactor" class="form-select">
                        <option value="0" {{if eq .HW.FormFactor 0}}selected{{end}}>Не указано</option>
                        <option value="1" {{if eq .HW.FormFactor 1}}selected{{end}}>Rack</option>
                        <option value="2" {{if eq .HW.FormFactor 2}}selected{{end}}>Blade</option>
                    </select>
                </div>
                <div class="col-4">
                    <label for="PowerState" class="form-label">Состояние питания</label>
                    <select id="PowerState" name="PowerState" class="form-select">
                        <option value="0" {{if eq .HW.PowerState 0}}selected{{end}}>Не указано</option>
                        <option value="1" {{if eq .HW.PowerState 1}}selected{{end}}>Включен</option>
                        <option value="2" {{if eq .HW.PowerState 2}}selected{{end}}>Выключен</option>
                    </select>
                </div>
                <div class="col-4">
                    <label for="Active" class="form-label">Статус</label>
                    <select id="Active" name="Active" class="form-select">
                        <option value="0" {{if eq .HW.Active 0}}selected{{end}}>Не указано</option>
                        <option value="1" {{if eq .HW.Active 1}}selected{{end}}>Активен</option>
                        <option value="2" {{if eq .HW.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                        <option value="3" {{if eq .HW.Active 3}}selected{{end}}>Списан</option>
                    </select>
                </div>
                <div class="col-4">
                    <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                    <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                        {{range $s := .ItiliumStatuses}}
                        <option value="{{$s.Value}}" {{if eq $.HW.ItiliumStatus $s.Value}}selected{{end}}>{{$s.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-4">
                    <label for="StatusHistory" class="form-label">Дата статуса Итилиум</label>
                    <input type="date" id="StatusHistory" name="StatusHistory" class="form-control" value="{{.StatusHistoryStr}}">
                </div>
                <div class="col-12">
                    <label for="Description" class="form-label">Описание (Итилиум)</label>
                    <textarea id="Description" name="Description" class="form-control" rows="3">{{.HW.Description}}</textarea>
                </div>
                <div class="col-12">
                    <label for="DeActivateInfo" class="form-label">Причина деактивации/списания</label>
                    <input type="text" id="DeActivateInfo" name="DeActivateInfo" class="form-control" value="{{.HW.DeActivateInfo}}" placeholder="Например: списан по акту №..., заменён на ...">
                </div>
                <div class="col-4">
                    <label class="form-label">Ср. мощность 24ч, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .HW.PowerAvg24hW}}" readonly title="OneView environmentalConfiguration; не редактируется вручную">
                </div>
                <div class="col-4">
                    <label class="form-label">Пик 24ч, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .HW.PowerPeak24hW}}" readonly>
                </div>
                <div class="col-4">
                    <label class="form-label">Паспорт max, Вт</label>
                    <input type="text" class="form-control" value="{{printf "%.0f" .HW.PowerRatedMaxW}}" readonly>
                </div>
            </div>
        </div>

        <div class="tab-pane fade" id="tab-hw-account" role="tabpanel" aria-labelledby="tab-hw-account-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-4">
                    <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                    <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.HW.TechnicalNumberIT}}">
                </div>
                <div class="col-4">
                    <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                    <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.HW.InventoryNum}}">
                </div>
                <div class="col-4">
                    <label for="InventoryNumBuh" class="form-label">Инвентарный номер (Бух)</label>
                    <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.HW.InventoryNumBuh}}">
                </div>
                <div class="col-4">
                    <label for="Counterparty" class="form-label">Контрагент</label>
                    <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.HW.Counterparty}}">
                </div>
                <div class="col-4">
                    <label for="ContractDoc" class="form-label">Номер договора</label>
                    <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.HW.ContractDoc}}">
                </div>
                <div class="col-4">
                    <label for="DateContractDoc" class="form-label">Дата договора</label>
                    <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
                </div>
                <div class="col-3">
                    <label for="Invoice" class="form-label">Накладная</label>
                    <input type="text" id="Invoice" name="Invoice" class="form-control" value="{{.HW.Invoice}}">
                </div>
                <div class="col-3">
                    <label for="DateInvoice" class="form-label">Дата получения</label>
                    <input type="date" id="DateInvoice" name="DateInvoice" class="form-control" value="{{.DateInvoiceStr}}">
                </div>
                <div class="col-3">
                    <label for="ServiceContract" class="form-label">Сервисный контракт</label>
                    <input type="text" id="ServiceContract" name="ServiceContract" class="form-control" value="{{.HW.ServiceContract}}">
                </div>
                <div class="col-3">
                    <label for="DateService" class="form-label">Дата окончания</label>
                    <input type="date" id="DateService" name="DateService" class="form-control" value="{{.DateServiceStr}}">
                </div>
                <div class="col-6">
                    <label for="hwEditDatacenterSiteId" class="form-label">Площадка/ЦОД</label>
                    <select id="hwEditDatacenterSiteId" class="form-select"{{if .RackLockedByChassis}} disabled{{end}}>
                        <option value="0">-</option>
                        {{range $dc := .DatacenterSites}}
                        <option value="{{$dc.Id}}">{{$dc.Name}}{{if not $dc.IsDatacenter}} (не ЦОД){{end}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-6">
                    <label for="RackCabinetId" class="form-label">Шкаф</label>
                    <select id="RackCabinetId" name="RackCabinetId" class="form-select"{{if .RackLockedByChassis}} disabled{{end}}>
                        <option value="0">-</option>
                        {{range $rc := .RackCabinets}}
                        <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.HW.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                        {{end}}
                    </select>
                    {{if .RackLockedByChassis}}<div class="form-text">Шкаф задаётся блейд-корзиной {{.RackLockedByChassis}} и здесь не редактируется.</div>{{end}}
                </div>
                <div class="col-6">
                    <label for="StartUnitPos" class="form-label">Юнит начала</label>
                    <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{.HW.StartUnitPos}}">
                </div>
                <div class="col-6">
                    <label for="SizeUnit" class="form-label">Занимаемых юнитов</label>
                    <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{.HW.SizeUnit}}">
                </div>
                <div class="col-12">
                    <label for="SourceUpdate" class="form-label">Источник данных</label>
                    <input type="text" id="SourceUpdate" name="SourceUpdate" class="form-control" value="{{.HW.SourceUpdate}}">
                </div>
                <div class="col-12">
                    <label class="form-label">Источник последнего обновления</label>
                    <div>
                        <span class="badge {{if eq .HW.SourceUpdateType 1}}text-bg-secondary{{else if eq .HW.SourceUpdateType 2}}text-bg-info{{else if eq .HW.SourceUpdateType 3}}text-bg-primary{{else if eq .HW.SourceUpdateType 4}}text-bg-warning{{else}}text-bg-light{{end}}">
                            {{if eq .HW.SourceUpdateType 1}}Вручную{{else if eq .HW.SourceUpdateType 2}}VMware{{else if eq .HW.SourceUpdateType 3}}HPE OneView{{else if eq .HW.SourceUpdateType 4}}Zabbix{{else}}Не указано{{end}}
                        </span>
                        <span class="small text-muted ms-1">проставляется автоматически при синхронизации/сохранении, вручную не редактируется</span>
                    </div>
                </div>
            </div>
        </div>

        <div class="tab-pane fade" id="tab-hw-base" role="tabpanel" aria-labelledby="tab-hw-base-btn" tabindex="0">
            <div class="row g-3">
                <div class="col-6">
                    <label for="CPUModel" class="form-label">Модель процессора</label>
                    <input type="text" id="CPUModel" name="CPUModel" class="form-control" value="{{.HW.CPUModel}}">
                </div>
                <div class="col-6">
                    <label for="MemoryMb" class="form-label">Память, МБ</label>
                    <input type="number" id="MemoryMb" name="MemoryMb" class="form-control" value="{{.HW.MemoryMb}}">
                </div>
                <div class="col-4">
                    <label for="CPUCount" class="form-label">Кол-во процессоров</label>
                    <input type="number" id="CPUCount" name="CPUCount" class="form-control" value="{{.HW.CPUCount}}">
                </div>
                <div class="col-4">
                    <label for="CorePerCPU" class="form-label">Ядер на процессор</label>
                    <input type="number" id="CorePerCPU" name="CorePerCPU" class="form-control" value="{{.HW.CorePerCPU}}">
                </div>
                <div class="col-4">
                    <label for="CoreMHz" class="form-label">Частота, МГц</label>
                    <input type="number" id="CoreMHz" name="CoreMHz" class="form-control" value="{{.HW.CoreMHz}}">
                </div>
            </div>

            <hr class="my-3">
            <div class="mb-2">
                <h6 class="mb-1">Контроллер управления (iLO / iDRAC / IPMI)</h6>
                <div class="text-muted small">Данные с OneView и ручная правка; отображаются в 3D-виде сервера (модуль iLO).</div>
            </div>
            <div class="row g-3 mb-3">
                <div class="col-6 col-lg-3">
                    <label for="MpIPAddress" class="form-label">IP адрес</label>
                    <input type="text" id="MpIPAddress" name="MpIPAddress" class="form-control" value="{{.HW.MpIPAddress}}" autocomplete="off">
                </div>
                <div class="col-6 col-lg-3">
                    <label for="MpHostName" class="form-label">Hostname</label>
                    <input type="text" id="MpHostName" name="MpHostName" class="form-control" value="{{.HW.MpHostName}}" autocomplete="off">
                </div>
                <div class="col-6 col-lg-3">
                    <label for="MpModel" class="form-label">Модель контроллера</label>
                    <input type="text" id="MpModel" name="MpModel" class="form-control" value="{{.HW.MpModel}}">
                </div>
                <div class="col-6 col-lg-3">
                    <label for="MpFirmwareVersion" class="form-label">Прошивка контроллера</label>
                    <input type="text" id="MpFirmwareVersion" name="MpFirmwareVersion" class="form-control" value="{{.HW.MpFirmwareVersion}}">
                </div>
                <div class="col-6 col-lg-3">
                    <label for="RomVersion" class="form-label">BIOS / ROM</label>
                    <input type="text" id="RomVersion" name="RomVersion" class="form-control" value="{{.HW.RomVersion}}">
                </div>
            </div>

            <hr class="my-3">

            <div class="mb-2">
                <h6 class="mb-1">Добавить компонент</h6>
                <div class="text-muted small">Укажите тип и параметры, затем нажмите «Добавить» — строка сразу попадёт на закладку своего типа. Сохранение формы запишет весь набор компонентов.</div>
            </div>
            <div class="row g-2 align-items-end" id="hwCompAddPanel">
                <div class="col-6 col-lg-2">
                    <label for="newCompType" class="form-label">Тип</label>
                    <select id="newCompType" class="form-select form-select-sm">
                        <option value="0">-</option>
                        {{range $t := .HwComponentTypes}}
                        <option value="{{$t.Value}}">{{$t.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompSlot" class="form-label">Слот</label>
                    <input type="text" id="newCompSlot" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompVendor" class="form-label">Производитель</label>
                    <input type="text" id="newCompVendor" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompModel" class="form-label">Модель</label>
                    <input type="text" id="newCompModel" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-2">
                    <label for="newCompSerial" class="form-label">Серийный №</label>
                    <input type="text" id="newCompSerial" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-1" id="newCompCapacityWrap" style="display:none;">
                    <label for="newCompCapacity" class="form-label">Ёмкость, ГБ</label>
                    <input type="number" step="0.01" id="newCompCapacity" class="form-control form-control-sm" value="">
                </div>
                <div class="col-6 col-lg-1" id="newCompSubTypeWrap" style="display:none;">
                    <label for="newCompSubType" class="form-label">Подтип</label>
                    <select id="newCompSubType" class="form-select form-select-sm">
                        {{range $s := .HwComponentDiskSubTypes}}
                        <option value="{{$s.Value}}">{{$s.Name}}</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-12 col-lg-auto">
                    <button type="button" id="hwCompAddSubmit" class="btn btn-outline-secondary btn-sm">Добавить</button>
                </div>
            </div>
            <div id="hwCompAddError" class="text-danger small mt-2 d-none"></div>
        </div>

        {{range $tab := .HwComponentTabs}}
        <div class="tab-pane fade" id="tab-hw-comp-{{$tab.Type}}" role="tabpanel" aria-labelledby="tab-hw-comp-{{$tab.Type}}-btn" tabindex="0">
            <div class="d-flex justify-content-between align-items-center mb-2 gap-2 flex-wrap">
                <div class="text-muted small">Компоненты типа «{{$tab.Name}}». Удаление строки действует после сохранения формы.</div>
                {{if eq $tab.Type 1}}
                <div class="d-flex align-items-center gap-2">
                    <label for="hwMemFillFilter" class="small text-muted mb-0">Показать:</label>
                    <select id="hwMemFillFilter" class="form-select form-select-sm" style="width:auto;" title="Фильтр по заполненности производителя, модели и ёмкости">
                        <option value="all">Все модули</option>
                        <option value="filled" selected>Заполненные (производитель, модель, ёмкость)</option>
                        <option value="empty">Незаполненные</option>
                    </select>
                    <span id="hwMemFillFilterStat" class="small text-muted"></span>
                </div>
                {{end}}
            </div>
            <div class="table-responsive">
                <table class="table table-sm align-middle mb-0">
                    <thead>
                        <tr>
                            {{if eq $tab.Type 0}}<th style="min-width:9rem">Тип</th>{{end}}
                            <th style="min-width:6rem">Слот</th>
                            <th style="min-width:7rem">Производитель</th>
                            <th style="min-width:10rem">Модель</th>
                            <th style="min-width:8rem">Серийный №</th>
                            {{if $tab.ShowCapacity}}<th style="min-width:6rem">Ёмкость, ГБ</th>{{end}}
                            {{if $tab.ShowSubType}}<th style="min-width:6rem">Подтип диска</th>{{end}}
                            <th></th>
                        </tr>
                    </thead>
                    <tbody id="hwCompRows-{{$tab.Type}}" data-comp-type="{{$tab.Type}}" data-show-capacity="{{if $tab.ShowCapacity}}1{{else}}0{{end}}" data-show-subtype="{{if $tab.ShowSubType}}1{{else}}0{{end}}">
                        {{range $c := $tab.Items}}
                        <tr class="hw-comp-row">
                            {{if eq $tab.Type 0}}
                            <td>
                                <select name="CompType" class="form-select form-select-sm">
                                    <option value="0">-</option>
                                    {{range $t := $.HwComponentTypes}}
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
                            <td>
                                <input type="hidden" name="CompRole" value="{{$c.Role}}">
                                <input type="hidden" name="CompStatus" value="{{$c.Status}}">
                                <button type="button" class="btn btn-outline-danger btn-sm hw-comp-remove-row">Удалить</button>
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
        {{end}}

        <template id="hwCompRowTemplate">
            <tr class="hw-comp-row">
                <td>
                    <input type="hidden" name="CompType" value="">
                    <input type="text" name="CompSlot" class="form-control form-control-sm" value="">
                </td>
                <td><input type="text" name="CompVendor" class="form-control form-control-sm" value=""></td>
                <td><input type="text" name="CompModel" class="form-control form-control-sm" value=""></td>
                <td><input type="text" name="CompSerialNum" class="form-control form-control-sm" value=""></td>
                <td class="hw-comp-capacity-cell">
                    <input type="number" step="0.01" name="CompCapacityGb" class="form-control form-control-sm" value="">
                </td>
                <td class="d-none hw-comp-capacity-hidden-cell"><input type="hidden" name="CompCapacityGb" value="0" class="hw-comp-capacity-hidden"></td>
                <td class="hw-comp-subtype-cell">
                    <select name="CompSubType" class="form-select form-select-sm">
                        {{range $s := .HwComponentDiskSubTypes}}
                        <option value="{{$s.Value}}">{{$s.Name}}</option>
                        {{end}}
                    </select>
                </td>
                <td class="d-none hw-comp-subtype-hidden-cell"><input type="hidden" name="CompSubType" value="0" class="hw-comp-subtype-hidden"></td>
                <td>
                    <input type="hidden" name="CompRole" value="">
                    <input type="hidden" name="CompStatus" value="">
                    <button type="button" class="btn btn-outline-danger btn-sm hw-comp-remove-row">Удалить</button>
                </td>
            </tr>
        </template>

    </div>

    <div class="d-flex gap-2">
        <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
        <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
{{end}}

<script>
    // Каскадный выбор "ЦОД -> Шкаф" - как в hwconfedit.tpl.
    (function () {
        var dcSelect = document.getElementById('hwEditDatacenterSiteId');
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

    // Ручное добавление компонента на "Базовой конфигурации" с раскладкой
    // по закладкам типов; редактирование/удаление - на закладках типов.
    (function () {
        var tpl = document.getElementById('hwCompRowTemplate');
        var addBtn = document.getElementById('hwCompAddSubmit');
        var typeSelect = document.getElementById('newCompType');
        var capWrap = document.getElementById('newCompCapacityWrap');
        var subWrap = document.getElementById('newCompSubTypeWrap');
        var errBox = document.getElementById('hwCompAddError');
        if (!tpl || !addBtn || !typeSelect) return;

        // Типы с ёмкостью ГБ: 1 память, 2 диск, 5 RAID (кэш). Подтип - только диск (2).
        function typeShowsCapacity(t) { return t === '1' || t === '2' || t === '5'; }
        function typeShowsSubType(t) { return t === '2'; }

        function bindRemove(row) {
            var btn = row.querySelector('.hw-comp-remove-row');
            if (!btn) return;
            btn.addEventListener('click', function () { row.remove(); });
        }

        Array.prototype.forEach.call(document.querySelectorAll('.hw-comp-row'), bindRemove);

        function toggleAddFields() {
            var t = typeSelect.value;
            if (capWrap) capWrap.style.display = typeShowsCapacity(t) ? '' : 'none';
            if (subWrap) subWrap.style.display = typeShowsSubType(t) ? '' : 'none';
        }
        typeSelect.addEventListener('change', toggleAddFields);
        toggleAddFields();

        function showError(msg) {
            if (!errBox) return;
            if (!msg) {
                errBox.classList.add('d-none');
                errBox.textContent = '';
                return;
            }
            errBox.textContent = msg;
            errBox.classList.remove('d-none');
        }

        function clearAddForm() {
            typeSelect.value = '0';
            ['newCompSlot', 'newCompVendor', 'newCompModel', 'newCompSerial', 'newCompCapacity'].forEach(function (id) {
                var el = document.getElementById(id);
                if (el) el.value = '';
            });
            var sub = document.getElementById('newCompSubType');
            if (sub) sub.value = '0';
            toggleAddFields();
        }

        addBtn.addEventListener('click', function () {
            showError('');
            var type = typeSelect.value;
            if (!type || type === '0') {
                showError('Выберите тип компонента.');
                return;
            }
            var tbody = document.getElementById('hwCompRows-' + type);
            if (!tbody) {
                showError('Для выбранного типа нет закладки.');
                return;
            }

            var showCap = tbody.getAttribute('data-show-capacity') === '1';
            var showSub = tbody.getAttribute('data-show-subtype') === '1';

            var node = tpl.content.cloneNode(true);
            var row = node.querySelector('.hw-comp-row');
            if (!row) return;

            var typeInput = row.querySelector('input[name="CompType"]');
            if (typeInput) typeInput.value = type;

            row.querySelector('input[name="CompSlot"]').value = (document.getElementById('newCompSlot') || {}).value || '';
            row.querySelector('input[name="CompVendor"]').value = (document.getElementById('newCompVendor') || {}).value || '';
            row.querySelector('input[name="CompModel"]').value = (document.getElementById('newCompModel') || {}).value || '';
            row.querySelector('input[name="CompSerialNum"]').value = (document.getElementById('newCompSerial') || {}).value || '';

            var capCell = row.querySelector('.hw-comp-capacity-cell');
            var capHiddenCell = row.querySelector('.hw-comp-capacity-hidden-cell');
            var capVal = (document.getElementById('newCompCapacity') || {}).value || '';
            // Ровно одно поле CompCapacityGb на строку.
            if (showCap) {
                if (capHiddenCell) capHiddenCell.remove();
                if (capCell) {
                    var capInput = capCell.querySelector('input[name="CompCapacityGb"]');
                    if (capInput) capInput.value = capVal;
                }
            } else {
                if (capCell) capCell.remove();
                if (capHiddenCell) {
                    var capHidden = capHiddenCell.querySelector('.hw-comp-capacity-hidden');
                    if (capHidden) capHidden.value = '0';
                }
            }

            var subtypeCell = row.querySelector('.hw-comp-subtype-cell');
            var subtypeHiddenCell = row.querySelector('.hw-comp-subtype-hidden-cell');
            var subtypeHidden = row.querySelector('.hw-comp-subtype-hidden');
            var newSub = document.getElementById('newCompSubType');
            var subVal = newSub ? newSub.value : '0';
            // Ровно одно поле CompSubType на строку.
            if (showSub) {
                if (subtypeHiddenCell) subtypeHiddenCell.remove();
                if (subtypeCell) {
                    var sel = subtypeCell.querySelector('select[name="CompSubType"]');
                    if (sel) sel.value = subVal;
                }
            } else {
                if (subtypeCell) subtypeCell.remove();
                if (subtypeHidden) subtypeHidden.value = '0';
            }

            bindRemove(row);
            tbody.appendChild(node);
            clearAddForm();

            if (type === '1' && typeof window.hwApplyMemFillFilter === 'function') {
                window.hwApplyMemFillFilter();
            }

            var tabBtn = document.getElementById('tab-hw-comp-' + type + '-btn');
            if (tabBtn && window.bootstrap && bootstrap.Tab) {
                bootstrap.Tab.getOrCreateInstance(tabBtn).show();
            }
        });
    })();

    // Фильтр отображения модулей памяти по заполненности Vendor/Model/CapacityGb.
    // Скрытые строки остаются в форме и сохраняются — фильтр только визуальный.
    (function () {
        var tbody = document.getElementById('hwCompRows-1');
        var filter = document.getElementById('hwMemFillFilter');
        var stat = document.getElementById('hwMemFillFilterStat');
        if (!tbody || !filter) return;

        function isFilled(row) {
            var vendor = ((row.querySelector('input[name="CompVendor"]') || {}).value || '').trim();
            var model = ((row.querySelector('input[name="CompModel"]') || {}).value || '').trim();
            var capRaw = ((row.querySelector('input[name="CompCapacityGb"]') || {}).value || '').trim();
            var cap = parseFloat(capRaw.replace(',', '.'));
            return vendor !== '' && model !== '' && !isNaN(cap) && cap > 0;
        }

        function apply() {
            var mode = filter.value;
            var rows = tbody.querySelectorAll('.hw-comp-row');
            var total = 0, shown = 0, filled = 0;
            Array.prototype.forEach.call(rows, function (row) {
                total += 1;
                var ok = isFilled(row);
                if (ok) filled += 1;
                var visible = mode === 'all' || (mode === 'filled' && ok) || (mode === 'empty' && !ok);
                row.classList.toggle('d-none', !visible);
                if (visible) shown += 1;
            });
            if (stat) {
                stat.textContent = shown + ' из ' + total + (mode !== 'all' ? ' (заполнено: ' + filled + ')' : '');
            }
        }

        window.hwApplyMemFillFilter = apply;
        filter.addEventListener('change', apply);
        tbody.addEventListener('input', function (evt) {
            var name = (evt.target && evt.target.name) || '';
            if (name === 'CompVendor' || name === 'CompModel' || name === 'CompCapacityGb') {
                apply();
            }
        });
        apply();
    })();
</script>
