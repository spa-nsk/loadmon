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
                <!-- ИЗМЕНЕНО (2026-09-08, по запросу пользователя "Проверь
                     структуру StorageSystems, она изменилась"): у модели
                     models.StorageSystems больше нет текстовых полей City/
                     DatacenterRoom/RackNum/Allocation/StartUnit (string) -
                     размещение теперь описывается ТОЛЬКО через каскадный
                     select ЦОД -> Шкаф (RackCabinetId, FK на
                     models.RackCabinet) и числовые StartUnitPos/EndUnitPos/
                     SizeUnit - тот же набор полей, что у BladeSystems, см.
                     bladesystemsedit.tpl. Старая версия этого блока
                     ссылалась на уже не существующие поля модели (баг,
                     оставшийся с прошлой правки models_add.go) - исправлено
                     здесь заодно. -->
                    <div class="row g-3">
                    <div class="col-6">
                        <label for="stEditDatacenterSiteId" class="form-label">ЦОД</label>
                        <select id="stEditDatacenterSiteId" class="form-select">
                            <option value="0">-</option>
                            {{range $dc := .DatacenterSites}}
                            <option value="{{$dc.Id}}">{{$dc.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-6">
                        <label for="RackCabinetId" class="form-label">Шкаф (справочник)</label>
                        <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                            <option value="0">-</option>
                            {{range $rc := .RackCabinets}}
                            <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.ST.RackCabinetId $rc.Id}}selected{{end}}>{{$rc.Name}}</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-4">
                        <label for="StartUnitPos" class="form-label">Юнит начала</label>
                        <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{.ST.StartUnitPos}}">
                    </div>
                    <div class="col-4">
                        <label for="EndUnitPos" class="form-label">Юнит окончания</label>
                        <input type="number" id="EndUnitPos" name="EndUnitPos" class="form-control" value="{{.ST.EndUnitPos}}">
                    </div>
                    <div class="col-4">
                        <label for="SizeUnit" class="form-label">Размер, юнитов</label>
                        <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{.ST.SizeUnit}}">
                    </div>
                    <div class="col-6">
                        <label for="TechnicalNumberIT" class="form-label">Технический номер ИТ</label>
                        <input type="text" id="TechnicalNumberIT" name="TechnicalNumberIT" class="form-control" value="{{.ST.TechnicalNumberIT}}">
                    </div>
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
    // Каскадный select "ЦОД -> Шкаф" (2026-09-08, по запросу пользователя
    // "добавь в models.StorageSystems поле RackCabinetId и сделай
    // соответствующую обработку и отображение в справочнике СХД и форме
    // редактирования СХД") - select ЦОД (stEditDatacenterSiteId) сам никуда
    // не отправляется (нет атрибута name), он только сужает список опций
    // select "Шкаф" (RackCabinetId, единственное поле, которое реально уходит
    // в форму) по атрибуту data-dc каждой опции - тот же приём, что уже
    // используется в hwconfedit.tpl / hwconflist.tpl / view2.tpl.
    (function () {
        var dcSelect = document.getElementById('stEditDatacenterSiteId');
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
