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
                    <div class="col-6">
                        <label for="DatacenterRoom" class="form-label">ЦОД</label>
                        <input type="text" id="DatacenterRoom" name="DatacenterRoom" class="form-control" value="{{.HW.DatacenterRoom}}">
                    </div>
                    <div class="col-6">
                        <label for="RackNum" class="form-label">Номер стойки</label>
                        <input type="text" id="RackNum" name="RackNum" class="form-control" value="{{.HW.RackNum}}">
                    </div>
                    <div class="col-6">
                        <label for="StartUnitPos" class="form-label">Юнит начала</label>
                        <input type="number" id="StartUnitPos" name="StartUnitPos" class="form-control" value="{{.HW.StartUnitPos}}">
                    </div>
                    <div class="col-6">
                        <label for="SizeUnit" class="form-label">Занимаемых юнитов</label>
                        <input type="number" id="SizeUnit" name="SizeUnit" class="form-control" value="{{.HW.SizeUnit}}">
                    </div>
                    <div class="col-6">
                        <label for="DatacenterName" class="form-label">Помещение (OneView)</label>
                        <input type="text" id="DatacenterName" name="DatacenterName" class="form-control" value="{{.HW.DatacenterName}}">
                    </div>
                    <div class="col-6">
                        <label for="RackName" class="form-label">Шкаф (OneView)</label>
                        <input type="text" id="RackName" name="RackName" class="form-control" value="{{.HW.RackName}}">
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

<style>
    .hw-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 1.1rem;
    }

    .hw-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        margin-bottom: 0.75rem;
        text-transform: uppercase;
        font-size: 0.8rem;
        letter-spacing: 0.03em;
    }

    .hw-card hr {
        border-color: var(--bs-border-color);
        opacity: 0.5;
    }
</style>
