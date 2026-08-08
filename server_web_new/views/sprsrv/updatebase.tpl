<style>
    /* поля, недоступные для редактирования при текущем типе сервера */
    .field-view-only {
        background-color: var(--bs-secondary-bg);
        color: var(--bs-secondary-color);
        cursor: not-allowed;
    }
</style>

<script>
    // Серийный номер редактируется только для физического сервера (0) и
    // гипервизора (2); vCenter/Датацентр/Кластер - только для виртуальной
    // машины (1) и гипервизора (2). Для остальных сочетаний поля переводятся
    // в режим "только для просмотра" (readonly), значение при этом видно и
    // может быть скопировано, но не редактируется.
    // Дублируется проверкой на сервере (controllers/sprsrvupdatebase.go,
    // controllers/sprsrvaddserver.go) - JS здесь только для удобства ввода.
    function applyServerTypeFieldRules() {
        var typeSelect = document.getElementById('Type');
        if (!typeSelect) return;
        var type = parseInt(typeSelect.value, 10);

        var allowSerial = (type === 0 || type === 2);
        var allowVCenterFields = (type === 1 || type === 2);

        setFieldViewOnly('SerialNumber', !allowSerial);
        setFieldViewOnly('Vcenter', !allowVCenterFields);
        setFieldViewOnly('Datacenter', !allowVCenterFields);
        setFieldViewOnly('Cluster', !allowVCenterFields);
    }

    function setFieldViewOnly(id, viewOnly) {
        var el = document.getElementById(id);
        if (!el) return;
        el.readOnly = viewOnly;
        el.classList.toggle('field-view-only', viewOnly);
        el.title = viewOnly ? 'Поле недоступно для редактирования при текущем типе сервера' : '';
    }

    document.addEventListener('DOMContentLoaded', function () {
        applyServerTypeFieldRules();
        var typeSelect = document.getElementById('Type');
        if (typeSelect) {
            typeSelect.addEventListener('change', applyServerTypeFieldRules);
        }
    });
</script>

<div class="hero-text mb-3">
    <h2>{{if .IsNew}}Новый сервер{{else}}Общие данные сервера{{end}}</h2>
    {{if .flash.error}}
    <blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
    {{end}}
</div>

<form role="form" id="UpdateBase" method="POST" class="pb-4">

    <div class="card mb-3">
        <div class="card-header fw-semibold">Общие данные</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-3">
                    <label for="Type" class="form-label">Тип сервера</label>
                    <select name="Type" id="Type" class="form-select">
                        <option value="0" {{if eq .record.Type 0}}selected{{end}}>Физический сервер</option>
                        <option value="1" {{if eq .record.Type 1}}selected{{end}}>Виртуальная машина</option>
                        <option value="2" {{if eq .record.Type 2}}selected{{end}}>Гипервизор VMware</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SerialNumber" class="form-label">Серийный номер</label>
                    <input name="SerialNumber" id="SerialNumber" class="form-control" type="text" value="{{.record.SerialNumber}}" title="Заполняется для физического сервера и гипервизора" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Vcenter" class="form-label">vCenter</label>
                    <input name="Vcenter" id="Vcenter" class="form-control" type="text" value="{{.record.Vcenter}}" title="Заполняется для виртуальной машины и гипервизора" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Datacenter" class="form-label">Датацентр</label>
                    <input name="Datacenter" id="Datacenter" class="form-control" type="text" value="{{.record.Datacenter}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Cluster" class="form-label">Кластер</label>
                    <input name="Cluster" id="Cluster" class="form-control" type="text" value="{{.record.Cluster}}" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Классификация</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-3">
                    <label for="TypeMicroservice" class="form-label">Роль в микросервисах</label>
                    <select name="TypeMicroservice" id="TypeMicroservice" class="form-select">
                        <option value="0" {{if eq .record.TypeMicroservice 0}}selected{{end}}>не микросервис</option>
                        <option value="1" {{if eq .record.TypeMicroservice 1}}selected{{end}}>Docker</option>
                        <option value="2" {{if eq .record.TypeMicroservice 2}}selected{{end}}>Kubernetes</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="RoleType" class="form-label">APP/DB сервер</label>
                    <select name="RoleType" id="RoleType" class="form-select">
                        <option value="0" {{if eq .record.RoleType 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.RoleType 1}}selected{{end}}>APP сервер</option>
                        <option value="2" {{if eq .record.RoleType 2}}selected{{end}}>DB сервер</option>
                        <option value="3" {{if eq .record.RoleType 3}}selected{{end}}>DB+APP сервер</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="TypeWork" class="form-label">Тип среды</label>
                    <select name="TypeWork" id="TypeWork" class="form-select">
                        <option value="0" {{if eq .record.TypeWork 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.TypeWork 1}}selected{{end}}>Продуктовая</option>
                        <option value="2" {{if eq .record.TypeWork 2}}selected{{end}}>Внутренний тест</option>
                        <option value="3" {{if eq .record.TypeWork 3}}selected{{end}}>Внешний тест</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="CriticalLevel" class="form-label">Уровень критичности</label>
                    <select name="CriticalLevel" id="CriticalLevel" class="form-select">
                        <option value="0" {{if eq .record.CriticalLevel 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.CriticalLevel 1}}selected{{end}}>Vital</option>
                        <option value="2" {{if eq .record.CriticalLevel 2}}selected{{end}}>Critical</option>
                        <option value="3" {{if eq .record.CriticalLevel 3}}selected{{end}}>Secondary</option>
                        <option value="4" {{if eq .record.CriticalLevel 4}}selected{{end}}>Non Priority</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Active" class="form-label">Состояние</label>
                    <select name="Active" id="Active" class="form-select">
                        <option value="1" {{if eq .record.Active 1}}selected{{end}}>Активен</option>
                        <option value="2" {{if eq .record.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                    </select>
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Идентификация и сеть</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-4">
                    <label for="Name" class="form-label">Имя сервера</label>
                    <input name="Name" id="Name" class="form-control" type="text" value="{{.record.Name}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="NameFQDN" class="form-label">Полное имя (FQDN)</label>
                    <input name="NameFQDN" id="NameFQDN" class="form-control" type="text" value="{{.record.NameFQDN}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="NameInfo" class="form-label">Альтернативное имя</label>
                    <input name="NameInfo" id="NameInfo" class="form-control" type="text" value="{{.record.NameInfo}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="IPAddress" class="form-label">IP адрес(а)</label>
                    <input name="IPAddress" id="IPAddress" class="form-control" type="text" value="{{.record.IPAddress}}" title="Список через разделитель ;" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="MacAddress" class="form-label">MAC адрес(а)</label>
                    <input name="MacAddress" id="MacAddress" class="form-control" type="text" value="{{.record.MacAddress}}" title="Список через разделитель ;" />
                </div>
                <div class="col-sm-6 col-md-2">
                    <label for="Network" class="form-label">Сеть</label>
                    <input name="Network" id="Network" class="form-control" type="text" value="{{.record.Network}}" title="Например: vlan144" />
                </div>
                <div class="col-sm-6 col-md-2">
                    <label for="NetworkName" class="form-label">Наименование сети</label>
                    <input name="NetworkName" id="NetworkName" class="form-control" type="text" value="{{.record.NetworkName}}" title="Например: DMZ, Prom" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Ответственные</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-4">
                    <label for="Owner" class="form-label">Ответственный</label>
                    <input name="Owner" id="Owner" class="form-control" type="text" value="{{.record.Owner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="BusinessOwner" class="form-label">Владелец от бизнеса</label>
                    <input name="BusinessOwner" id="BusinessOwner" class="form-control" type="text" value="{{.record.BusinessOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ITOwner" class="form-label">Владелец со стороны ИТ</label>
                    <input name="ITOwner" id="ITOwner" class="form-control" type="text" value="{{.record.ITOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ProductOwner" class="form-label">Владелец продукта</label>
                    <input name="ProductOwner" id="ProductOwner" class="form-control" type="text" value="{{.record.ProductOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Support" class="form-label">Ответственные за поддержку</label>
                    <input name="Support" id="Support" class="form-control" type="text" value="{{.record.Support}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="SupportTeam" class="form-label">Подразделение поддержки</label>
                    <input name="SupportTeam" id="SupportTeam" class="form-control" type="text" value="{{.record.SupportTeam}}" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Система и назначение</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-12">
                    <label for="ITSystemKey" class="form-label">ИТ система</label>
                    <select name="ITSystemKey" id="ITSystemKey" class="form-select">
                        <option value="" {{if eq .record.ITSystemKey "" }}selected{{end}}>не выбрано</option>
                        {{range $its := .ITSystemSpr}}
                        <option value="{{$its.ObjectKey}}" {{if eq $its.ObjectKey $.record.ITSystemKey}}selected{{end}}>
                            {{$its.ObjectKey}} {{$its.Label}}: {{$its.FullName}} {{$its.Status}} {{$its.SystemType}} {{$its.BusinessColumn}} {{$its.CFO}}
                        </option>
                        {{end}}
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="OS" class="form-label">Операционная система</label>
                    <input name="OS" id="OS" class="form-control" type="text" value="{{.record.OS}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SystemType" class="form-label">Тип системы</label>
                    <input name="SystemType" id="SystemType" class="form-control" type="text" value="{{.record.SystemType}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="LicensingType" class="form-label">Тип лицензирования</label>
                    <input name="LicensingType" id="LicensingType" class="form-control" type="text" value="{{.record.LicensingType}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SolutionType" class="form-label">Вариант решения</label>
                    <input name="SolutionType" id="SolutionType" class="form-control" type="text" value="{{.record.SolutionType}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Users" class="form-label">Пользователи системы</label>
                    <input name="Users" id="Users" class="form-control" type="text" value="{{.record.Users}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Allocation" class="form-label">Аллокация (подразделение)</label>
                    <input name="Allocation" id="Allocation" class="form-control" type="text" value="{{.record.Allocation}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Role" class="form-label">Роль сервера</label>
                    <input name="Role" id="Role" class="form-control" type="text" value="{{.record.Role}}" title="Например: сервер баз данных, сервер приложений, хост" />
                </div>
                <div class="col-12">
                    <label for="Anotation" class="form-label">Описание сервера</label>
                    <textarea name="Anotation" id="Anotation" class="form-control" rows="2">{{.record.Anotation}}</textarea>
                </div>
                <div class="col-12">
                    <label for="Details" class="form-label">Примечание</label>
                    <textarea name="Details" id="Details" class="form-control" rows="2">{{.record.Details}}</textarea>
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Расположение и обслуживание</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-8 col-md-6">
                    <label for="DPCOffice" class="form-label">ЦОД: адрес и площадка</label>
                    <select name="DPCOffice" id="DPCOffice" class="form-select">
                        <option value="0" {{if eq .record.DPCOffice 0}}selected{{end}}>не выбрано</option>
                        <option value="101" {{if eq .record.DPCOffice 101}}selected{{end}}>623к, г. Новосибирск, ул. Советская 37</option>
                        <option value="102" {{if eq .record.DPCOffice 102}}selected{{end}}>624к, г. Новосибирск, ул. Советская 37</option>
                        <option value="103" {{if eq .record.DPCOffice 103}}selected{{end}}>316к, г. Новосибирск, ул. Советская 37</option>
                        <option value="121" {{if eq .record.DPCOffice 121}}selected{{end}}>Мегафон, г. Новосибирск, ул. Толмачевская 19а к3</option>
                        <option value="131" {{if eq .record.DPCOffice 131}}selected{{end}}>Ростелеком, г. Новосибирск, ул. Менделеева 1</option>
                        <option value="201" {{if eq .record.DPCOffice 201}}selected{{end}}>3-02 г. Москва, ул. Мира 69</option>
                        <option value="202" {{if eq .record.DPCOffice 202}}selected{{end}}>4-19 г. Москва, ул. Мира 69</option>
                        <option value="281" {{if eq .record.DPCOffice 281}}selected{{end}}>DSP г. Москва, ул. Шарикоподшипниковская улица 11/9</option>
                        <option value="291" {{if eq .record.DPCOffice 291}}selected{{end}}>М1 г. Москва, ул. Варшавское шоссе 125</option>
                    </select>
                </div>
                <div class="col-sm-4 col-md-6">
                    <label for="SZHistory" class="form-label">Номера СЗ</label>
                    <input name="SZHistory" id="SZHistory" class="form-control" type="text" value="{{.record.SZHistory}}" title="Через пробел, в хронологическом порядке" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Monitoring" class="form-label">Мониторинг</label>
                    <select name="Monitoring" id="Monitoring" class="form-select">
                        <option value="0" {{if eq .record.Monitoring 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.Monitoring 1}}selected{{end}}>включен</option>
                        <option value="2" {{if eq .record.Monitoring 2}}selected{{end}}>отключен</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Backuped" class="form-label">Резервное копирование</label>
                    <select name="Backuped" id="Backuped" class="form-select">
                        <option value="0" {{if eq .record.Backuped 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.Backuped 1}}selected{{end}}>включено</option>
                        <option value="2" {{if eq .record.Backuped 2}}selected{{end}}>отключено</option>
                    </select>
                </div>
                <div class="col-sm-12 col-md-6">
                    <label for="MonitoringSZ" class="form-label">Основание постановки на мониторинг</label>
                    <input name="MonitoringSZ" id="MonitoringSZ" class="form-control" type="text" value="{{.record.MonitoringSZ}}" />
                </div>
            </div>
        </div>
    </div>

    <div class="d-flex gap-2">
        <input type="submit" title="{{if .IsNew}}Создать сервер{{else}}Сохранить изменения{{end}}" value="{{if .IsNew}}Создать{{else}}Сохранить{{end}}" class="btn btn-primary rounded-pill px-4" />
        <a href="/sprsrv/view2/{{.backmode}}/0" title="Вернуться без сохранения" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
