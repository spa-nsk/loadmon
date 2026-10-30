<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая запись СХД{{else}}Запись СХД{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/storagesystemslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .ST}}
<form role="form" id="stEditForm" method="POST">
    <div class="row g-3 align-items-start">
        <div class="col-12">
            <div class="hw-card">
                <h6>Общие сведения</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="ShortName" class="form-label">Мнемоническое имя</label>
                        <input type="text" id="ShortName" name="ShortName" class="form-control" value="{{.ST.ShortName}}">
                    </div>
                    <div class="col-6">
                        <label for="TagName" class="form-label">Мнемонические имена в LUN</label>
                        <input type="text" id="TagName" name="TagName" class="form-control" value="{{.ST.TagName}}">
                    </div>
                    <div class="col-6">
                        <label for="Vendor" class="form-label">Производитель</label>
                        <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.ST.Vendor}}">
                    </div>
                    <div class="col-6">
                        <label for="Model" class="form-label">Модель</label>
                        <input type="text" id="Model" name="Model" class="form-control" value="{{.ST.Model}}">
                    </div>
                    <div class="col-6">
                        <label for="PartNum" class="form-label">Код продукта у вендора</label>
                        <input type="text" id="PartNum" name="PartNum" class="form-control" value="{{.ST.PartNum}}">
                    </div>
                    <div class="col-6">
                        <label for="SerialNum" class="form-label">Серийный номер</label>
                        <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.ST.SerialNum}}">
                    </div>
                    <div class="col-6">
                        <label for="UID" class="form-label">UID</label>
                        <input type="text" id="UID" name="UID" class="form-control" value="{{.ST.UID}}">
                    </div>
                    <div class="col-6">
                        <label for="Firmware" class="form-label">Версия прошивки</label>
                        <input type="text" id="Firmware" name="Firmware" class="form-control" value="{{.ST.Firmware}}">
                    </div>
                    <div class="col-4">
                        <label for="TypeWork" class="form-label">Среда</label>
                        <select id="TypeWork" name="TypeWork" class="form-select">
                            <option value="0" {{if eq .ST.TypeWork 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .ST.TypeWork 1}}selected{{end}}>Продуктовая</option>
                            <option value="2" {{if eq .ST.TypeWork 2}}selected{{end}}>Внутренний тест</option>
                            <option value="3" {{if eq .ST.TypeWork 3}}selected{{end}}>Внешний тест / Бэкап</option>
                        </select>
                    </div>
                    <div class="col-4">
                        <label for="Status" class="form-label">Статус</label>
                        <select id="Status" name="Status" class="form-select">
                            <option value="0" {{if eq .ST.Status 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .ST.Status 1}}selected{{end}}>Используется</option>
                            <option value="2" {{if eq .ST.Status 2}}selected{{end}}>Выключен</option>
                            <option value="3" {{if eq .ST.Status 3}}selected{{end}}>Не используется</option>
                            <option value="4" {{if eq .ST.Status 4}}selected{{end}}>Тестирование</option>
                            <option value="5" {{if eq .ST.Status 5}}selected{{end}}>Приёмка</option>
                            <option value="6" {{if eq .ST.Status 6}}selected{{end}}>Демонтирован</option>
                        </select>
                    </div>
                    <div class="col-4">
                        <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                        <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                            {{range $s := .ItiliumStatuses}}
                            <option value="{{$s.Value}}" {{if eq $.ST.ItiliumStatus $s.Value}}selected{{end}}>{{$s.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-4">
                        <label for="Power" class="form-label">Питание</label>
                        <select id="Power" name="Power" class="form-select">
                            <option value="0" {{if eq .ST.Power 0}}selected{{end}}>Выключен</option>
                            <option value="1" {{if eq .ST.Power 1}}selected{{end}}>Включен</option>
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="ApiType" class="form-label">Тип API</label>
                        <select id="ApiType" name="ApiType" class="form-select">
                            <option value="0" {{if eq .ST.ApiType 0}}selected{{end}}>Не указано</option>
                            <option value="1" {{if eq .ST.ApiType 1}}selected{{end}}>VNX CLI</option>
                            <option value="2" {{if eq .ST.ApiType 2}}selected{{end}}>Unity REST</option>
                            <option value="3" {{if eq .ST.ApiType 3}}selected{{end}}>Dorado REST</option>
                            <option value="4" {{if eq .ST.ApiType 4}}selected{{end}}>OceanStor</option>
                            <option value="5" {{if eq .ST.ApiType 5}}selected{{end}}>Lenovo REST</option>
                        </select>
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Ёмкость и сеть</h6>
                <div class="row g-3">
                    <div class="col-4">
                        <label for="Size" class="form-label">Максимум дисков</label>
                        <input type="number" id="Size" name="Size" class="form-control" value="{{.ST.Size}}">
                    </div>
                    <div class="col-4">
                        <label for="Used" class="form-label">Установлено дисков</label>
                        <input type="number" id="Used" name="Used" class="form-control" value="{{.ST.Used}}">
                    </div>
                    <div class="col-4">
                        <label for="TotalSizeRAW" class="form-label">Ёмкость (сырая), Гб</label>
                        <input type="number" step="0.01" id="TotalSizeRAW" name="TotalSizeRAW" class="form-control" value="{{printf "%.2f" .ST.TotalSizeRAW}}">
                    </div>
                    <div class="col-4">
                        <label for="UsedPercent" class="form-label">Занято, %</label>
                        <input type="number" step="0.01" id="UsedPercent" name="UsedPercent" class="form-control" value="{{printf "%.2f" .ST.UsedPercent}}">
                    </div>
                    <div class="col-4">
                        <label for="IpAddress" class="form-label">IP адрес</label>
                        <input type="text" id="IpAddress" name="IpAddress" class="form-control" value="{{.ST.IpAddress}}">
                    </div>
                    <div class="col-4">
                        <label for="IpAddressNext" class="form-label">IP адрес 2</label>
                        <input type="text" id="IpAddressNext" name="IpAddressNext" class="form-control" value="{{.ST.IpAddressNext}}">
                    </div>
                    <div class="col-12">
                        <label for="WWNMask" class="form-label">WWN маска дисков</label>
                        <input type="text" id="WWNMask" name="WWNMask" class="form-control" value="{{.ST.WWNMask}}">
                    </div>
                </div>

                <hr class="my-3">

                <h6>AD / безопасность</h6>
                <div class="row g-3 align-items-center">
                    <div class="col-auto">
                        <div class="form-check">
                            <input class="form-check-input" type="checkbox" id="Ad" name="Ad" {{if .ST.Ad}}checked{{end}}>
                            <label class="form-check-label" for="Ad">В AD</label>
                        </div>
                    </div>
                    <div class="col-auto">
                        <div class="form-check">
                            <input class="form-check-input" type="checkbox" id="Ldaps" name="Ldaps" {{if .ST.Ldaps}}checked{{end}}>
                            <label class="form-check-label" for="Ldaps">Ldaps</label>
                        </div>
                    </div>
                    <div class="col-auto">
                        <div class="form-check">
                            <input class="form-check-input" type="checkbox" id="AdCert" name="AdCert" {{if .ST.AdCert}}checked{{end}}>
                            <label class="form-check-label" for="AdCert">Сертификат AD</label>
                        </div>
                    </div>
                    <div class="col-3">
                        <label for="AdCertDate" class="form-label">Дата истечения сертификата</label>
                        <input type="date" id="AdCertDate" name="AdCertDate" class="form-control" value="{{.AdCertDateStr}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <div class="hw-card">
                <h6>Размещение</h6>
                <!-- ИЗМЕНЕНО (2026-09-09, по запросу пользователя "СХД может
                     размещаться в нескольких шкафах и схема со ссылкой
                     StorageSystems.RackCabinetId не позволяет реализовать
                     учет размещения"): одиночный select "Шкаф" +
                     StartUnitPos/EndUnitPos/SizeUnit ПРЯМО на StorageSystems
                     заменены на динамический список строк "шкаф" - каждая
                     строка это одна запись models.StorageSystemPlacement
                     (см. models_add.go). Строки повторяют один и тот же
                     набор полей формы (name="PlacementRackCabinetId" и
                     т.д. без индексов в имени) - контроллер
                     (SprStorageSystemsEdit/saveStorageSystemPlacements в
                     controllers/sprstoragesystems.go) читает их как
                     параллельные массивы через GetStrings и сопоставляет по
                     порядку. SizeUnit не показывается отдельным полем -
                     вычисляется на сервере из StartUnitPos/EndUnitPos (тот
                     же приём, что и в bladesystemsedit.tpl). -->
                <div class="row g-3">
                    <div class="col-6">
                        <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                        <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.ST.TechnicalNumberIT}}">
                    </div>
                </div>

                <hr class="my-3">

                <div class="d-flex justify-content-between align-items-center mb-2">
                    <div class="text-muted small">
                        Одна СХД может занимать несколько шкафов одновременно (например, контроллеры в одном шкафу, дисковые полки расширения - в другом) - добавьте по одной строке на каждый шкаф.
                    </div>
                    <button type="button" id="placementAddRow" class="btn btn-outline-secondary btn-sm">+ Добавить шкаф</button>
                </div>

                <div id="placementRows">
                    {{range $i, $p := .Placements}}
                    <div class="row g-3 align-items-end placement-row mb-2">
                        <div class="col-3">
                            <label class="form-label">ЦОД</label>
                            <!-- ИЗМЕНЕНО (2026-09-10, по запросу пользователя про IT-0000041888
                                 "Местоположение указано, но данных по шкафу нет и информация по
                                 размещению в ЦОД пропала"): добавлен name="PlacementDatacenterSiteId",
                                 чтобы значение отправлялось на сервер - теперь строка может задавать
                                 ЦОД БЕЗ выбора шкафа (site-only размещение, см.
                                 models.StorageSystemPlacement.DatacenterSiteId и
                                 saveStorageSystemPlacements в controllers/sprstoragesystems.go).
                                 Если у строки уже выбран шкаф, каскадная JS-логика ниже всё равно
                                 держит это поле синхронизированным с data-dc выбранного шкафа. -->
                            <select name="PlacementDatacenterSiteId" class="form-select placement-dc-select">
                                <option value="0">-</option>
                                {{range $dc := $.DatacenterSites}}
                                <option value="{{$dc.Id}}" {{if and (eq $p.RackCabinetId 0) (eq $p.DatacenterSiteId $dc.Id)}}selected{{end}}>{{$dc.Name}}</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="col-3">
                            <label class="form-label">Шкаф</label>
                            <select name="PlacementRackCabinetId" class="form-select placement-rack-select">
                                <option value="0">-</option>
                                {{range $rc := $.RackCabinets}}
                                <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $p.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                                {{end}}
                            </select>
                        </div>
                        <div class="col-2">
                            <label class="form-label">Юнит начала</label>
                            <input type="number" name="PlacementStartUnitPos" class="form-control" value="{{if gt $p.StartUnitPos 0}}{{$p.StartUnitPos}}{{end}}">
                        </div>
                        <div class="col-2">
                            <label class="form-label">Юнит окончания</label>
                            <input type="number" name="PlacementEndUnitPos" class="form-control" value="{{if gt $p.EndUnitPos 0}}{{$p.EndUnitPos}}{{end}}">
                        </div>
                        <div class="col-2">
                            <button type="button" class="btn btn-outline-danger btn-sm placement-remove-row">Удалить</button>
                        </div>
                    </div>
                    {{end}}
                </div>

                <hr class="my-3">

                <h6>Владелец и описание</h6>
                <div class="row g-3">
                    <div class="col-6">
                        <label for="Owner" class="form-label">Владелец</label>
                        <input type="text" id="Owner" name="Owner" class="form-control" value="{{.ST.Owner}}">
                    </div>
                    <div class="col-12">
                        <label for="Anotation" class="form-label">Описание</label>
                        <input type="text" id="Anotation" name="Anotation" class="form-control" value="{{.ST.Anotation}}">
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
                        <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.ST.InventoryNum}}">
                    </div>
                    <div class="col-6">
                        <label for="InventoryNumBuh" class="form-label">Инв. номер (бух.)</label>
                        <input type="text" id="InventoryNumBuh" name="InventoryNumBuh" class="form-control" value="{{.ST.InventoryNumBuh}}">
                    </div>
                    <div class="col-6">
                        <label for="Counterparty" class="form-label">Контрагент</label>
                        <input type="text" id="Counterparty" name="Counterparty" class="form-control" value="{{.ST.Counterparty}}">
                    </div>
                    <div class="col-6">
                        <label for="ContractDoc" class="form-label">Номер договора</label>
                        <input type="text" id="ContractDoc" name="ContractDoc" class="form-control" value="{{.ST.ContractDoc}}">
                    </div>
                    <div class="col-6">
                        <label for="DateContractDoc" class="form-label">Дата договора</label>
                        <input type="date" id="DateContractDoc" name="DateContractDoc" class="form-control" value="{{.DateContractDocStr}}">
                    </div>
                    <div class="col-6">
                        <label for="Invoice" class="form-label">Накладная</label>
                        <input type="text" id="Invoice" name="Invoice" class="form-control" value="{{.ST.Invoice}}">
                    </div>
                    <div class="col-6">
                        <label for="DateInvoice" class="form-label">Дата получения</label>
                        <input type="date" id="DateInvoice" name="DateInvoice" class="form-control" value="{{.DateInvoiceStr}}">
                    </div>
                    <div class="col-6">
                        <label for="ServiceContract" class="form-label">Сервисный контракт</label>
                        <input type="text" id="ServiceContract" name="ServiceContract" class="form-control" value="{{.ST.ServiceContract}}">
                    </div>
                    <div class="col-6">
                        <label for="DateService" class="form-label">Окончание гарантии</label>
                        <input type="date" id="DateService" name="DateService" class="form-control" value="{{.DateServiceStr}}">
                    </div>
                </div>
            </div>
        </div>

        <div class="col-12">
            <button type="submit" class="btn btn-primary rounded-pill px-4">{{if .IsNew}}Создать{{else}}Сохранить{{end}}</button>
            <a href="/sprsrv/storagesystemslist/0" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
        </div>
    </div>
</form>
{{end}}

<!-- Базовые стили .hw-card/.hw-card hr вынесены в общий
     server_web_new/static/css/loadmon.css - здесь остаётся только
     дополнение для заголовков карточек (заглавными буквами), идентичное
     /sprsrv/hwconfedit. -->
<style>
    .hw-card h6 {
        text-transform: uppercase;
        font-size: 0.8rem;
        letter-spacing: 0.03em;
    }
</style>

<script>
    // Динамический список размещений СХД в нескольких шкафах (2026-09-09, по
    // запросу пользователя "СХД может размещаться в нескольких шкафах и
    // схема со ссылкой StorageSystems.RackCabinetId не позволяет
    // реализовать учет размещения") - каждая строка .placement-row это одна
    // будущая запись models.StorageSystemPlacement. Внутри строки работает
    // тот же каскад "ЦОД -> Шкаф", что и раньше в этой форме (select ЦОД сам
    // никуда не отправляется, он только сужает список опций select "Шкаф" по
    // атрибуту data-dc), но теперь применяется к КАЖДОЙ строке независимо
    // через closest/querySelector вместо фиксированных id (id не могут
    // повторяться при клонировании строк).
    (function () {
        var container = document.getElementById('placementRows');
        var addBtn = document.getElementById('placementAddRow');
        if (!container) return;

        function bindCascade(row) {
            var dcSelect = row.querySelector('.placement-dc-select');
            var rackSelect = row.querySelector('.placement-rack-select');
            if (!dcSelect || !rackSelect) return;

            var allOptions = Array.prototype.slice.call(rackSelect.options);

            function applyFilter(preserveSelection) {
                var dc = dcSelect.value;
                var currentValue = rackSelect.value;
                rackSelect.innerHTML = '';
                allOptions.forEach(function (opt) {
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

            var selectedOption = allOptions.find(function (opt) { return opt.selected; });
            if (selectedOption && selectedOption.value !== '0') {
                dcSelect.value = selectedOption.getAttribute('data-dc') || '0';
            }
            applyFilter(true);

            dcSelect.addEventListener('change', function () { applyFilter(true); });
        }

        function bindRemove(row) {
            var btn = row.querySelector('.placement-remove-row');
            if (!btn) return;
            btn.addEventListener('click', function () {
                if (container.querySelectorAll('.placement-row').length <= 1) {
                    // Последнюю строку не удаляем совсем (иначе нечего будет
                    // клонировать для "+ Добавить шкаф") - просто очищаем её.
                    row.querySelectorAll('select').forEach(function (s) { s.value = '0'; });
                    row.querySelectorAll('input').forEach(function (i) { i.value = ''; });
                    return;
                }
                row.remove();
            });
        }

        Array.prototype.forEach.call(container.querySelectorAll('.placement-row'), function (row) {
            bindCascade(row);
            bindRemove(row);
        });

        if (addBtn) {
            addBtn.addEventListener('click', function () {
                var rows = container.querySelectorAll('.placement-row');
                var template = rows[rows.length - 1];
                if (!template) return;
                var clone = template.cloneNode(true);
                clone.querySelectorAll('select').forEach(function (s) { s.value = '0'; });
                clone.querySelectorAll('input').forEach(function (i) { i.value = ''; });
                container.appendChild(clone);
                bindCascade(clone);
                bindRemove(clone);
            });
        }
    })();
</script>
