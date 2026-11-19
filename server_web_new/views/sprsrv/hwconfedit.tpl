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
<form role="form" id="hwEditForm" method="POST">
    <div class="row g-3 align-items-start">
        <div class="col-12">
            <div class="hw-card">
                <h6>Общие сведения</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="Vendor" class="form-label">Производитель</label>
                        <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.HW.Vendor}}">
                    </div>
                    <div class="col-6">
                        <label for="Model" class="form-label">Модель</label>
                        <input type="text" id="Model" name="Model" class="form-control" value="{{.HW.Model}}">
                    </div>
                    <div class="col-6">
                        <label for="PartNum" class="form-label">Номер продукта</label>
                        <input type="text" id="PartNum" name="PartNum" class="form-control" value="{{.HW.PartNum}}">
                    </div>
                    <div class="col-6">
                        <label for="FormFactor" class="form-label">Тип сервера</label>
                        <select id="FormFactor" name="FormFactor" class="form-select">
                            <option value="0" {{if eq .HW.FormFactor 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .HW.FormFactor 1}}selected{{end}}>Rack</option>
                            <option value="2" {{if eq .HW.FormFactor 2}}selected{{end}}>Blade</option>
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="PowerState" class="form-label">Состояние питания</label>
                        <select id="PowerState" name="PowerState" class="form-select">
                            <option value="0" {{if eq .HW.PowerState 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .HW.PowerState 1}}selected{{end}}>Включен</option>
                            <option value="2" {{if eq .HW.PowerState 2}}selected{{end}}>Выключен</option>
                        </select>
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
                    <div class="col-6">
                        <label for="Active" class="form-label">Статус</label>
                        <select id="Active" name="Active" class="form-select">
                            <option value="0" {{if eq .HW.Active 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .HW.Active 1}}selected{{end}}>Активен</option>
                            <option value="2" {{if eq .HW.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                            <option value="3" {{if eq .HW.Active 3}}selected{{end}}>Списан</option>
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                        <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                            {{range $s := .ItiliumStatuses}}
                            <option value="{{$s.Value}}" {{if eq $.HW.ItiliumStatus $s.Value}}selected{{end}}>{{$s.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-6">
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
                    <div class="col-6">
                        <label for="SerialNum" class="form-label">Серийный номер</label>
                        <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.HW.SerialNum}}">
                    </div>
                    <div class="col-6">
                        <label for="VirtualSerialNum" class="form-label">Виртуальный серийный номер</label>
                        <input type="text" id="VirtualSerialNum" name="VirtualSerialNum" class="form-control" value="{{.HW.VirtualSerialNum}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Процессор и память</h6>
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

                <h6>Размещение</h6>
                <div class="row g-3">
                    <!-- ИЗМЕНЕНО (2026-09-10, по запросу пользователя "удали
                         поля ЦОД (текст, Итиллиум)/Номер стойки (текст,
                         Итиллиум)/Помещение (OneView)/Шкаф (OneView). По
                         другим полям блока приведи в соответствие с новой
                         концепцией размещения"): 4 старых read-only текстовых
                         поля (последние известные значения из Итиллиум/
                         OneView, до перехода на справочник) убраны -
                         единственный источник истины теперь справочники
                         models.DatacenterSite/models.RackCabinet ниже.
                         Подпись "ЦОД" переименована в "Площадка/ЦОД"
                         (согласуется с переименованием самого справочника,
                         см. CHANGELOG - в нём есть не только настоящие ЦОД, но
                         и склады/виртуальные площадки для списанного); опции
                         помечают площадки без признака IsDatacenter как
                         "(не ЦОД)", чтобы при выборе было видно, что это не
                         серверная. Каскад ЦОД -> Шкаф - тот же приём, что уже
                         в bladesystemsedit.tpl, см. JS внизу файла. Площадка
                         (DatacenterSiteId) сама никуда не сохраняется (нет
                         name), сохраняется только RackCabinetId. -->
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
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Учёт и договор</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                        <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.HW.InventoryNum}}">
                    </div>
                    <div class="col-6">
                        <label for="InventoryNumBuh" class="form-label">Инв. номер (бух.)</label>
                        <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.HW.InventoryNumBuh}}">
                    </div>
                    <div class="col-6">
                        <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                        <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.HW.TechnicalNumberIT}}">
                    </div>
                    <div class="col-6">
                        <label for="Counterparty" class="form-label">Контрагент</label>
                        <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.HW.Counterparty}}">
                    </div>
                    <div class="col-6">
                        <label for="ContractDoc" class="form-label">Номер договора</label>
                        <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.HW.ContractDoc}}">
                    </div>
                    <div class="col-6">
                        <label for="DateContractDoc" class="form-label">Дата договора</label>
                        <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
                    </div>
                    <div class="col-6">
                        <label for="Invoice" class="form-label">Накладная</label>
                        <input type="text" id="Invoice" name="Invoice" class="form-control" value="{{.HW.Invoice}}">
                    </div>
                    <div class="col-6">
                        <label for="DateInvoice" class="form-label">Дата получения</label>
                        <input type="date" id="DateInvoice" name="DateInvoice" class="form-control" value="{{.DateInvoiceStr}}">
                    </div>
                    <div class="col-6">
                        <label for="ServiceContract" class="form-label">Сервисный контракт</label>
                        <input type="text" id="ServiceContract" name="ServiceContract" class="form-control" value="{{.HW.ServiceContract}}">
                    </div>
                    <div class="col-6">
                        <label for="DateService" class="form-label">Окончание гарантии</label>
                        <input type="date" id="DateService" name="DateService" class="form-control" value="{{.DateServiceStr}}">
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
        </div>

        <div class="col-12">
            <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
            <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
        </div>
    </div>
</form>
{{end}}

<!-- Базовые стили .hw-card/.hw-card hr вынесены в общий
     server_web_new/static/css/loadmon.css - здесь остаётся только
     дополнение для заголовков карточек (заглавными буквами), которое
     присуще именно этой форме. -->
<style>
    .hw-card h6 {
        text-transform: uppercase;
        font-size: 0.8rem;
        letter-spacing: 0.03em;
    }
</style>

<script>
    // Каскадный выбор "ЦОД -> Шкаф" (2026-09-08, по запросу пользователя
    // "исправь ... в самой карточке редактирования") - select ЦОД
    // (hwEditDatacenterSiteId) сам никуда не сохраняется (нет атрибута name),
    // он только сужает список опций select "Шкаф" (RackCabinetId,
    // единственное поле, которое реально сохраняется) по атрибуту data-dc
    // каждой опции - идентично bladesystemsedit.tpl
    // (sprbladesystems.go:SprBladeSystemsEdit) и фильтру /sprsrv/hwconflist
    // (hwconflist.tpl).
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

        // При открытии карточки с уже сохранённым шкафом - подставляем в
        // select ЦОД тот ЦОД, которому принадлежит выбранный шкаф, чтобы
        // список шкафов сразу был сужен корректно.
        var selectedOption = allRackOptions.find(function (opt) { return opt.selected; });
        if (selectedOption && selectedOption.value !== '0') {
            dcSelect.value = selectedOption.getAttribute('data-dc') || '0';
        }
        applyFilter(true);

        dcSelect.addEventListener('change', function () { applyFilter(true); });
    })();
</script>
