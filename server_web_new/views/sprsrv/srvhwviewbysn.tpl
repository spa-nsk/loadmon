<div class="hero-text mb-3">
    <h2>Конфигурация сервера</h2>
    <div class="text-muted">Серийный №: <b>{{.SN}}</b></div>
</div>

<form id="hwSnSearchForm" class="hw-filter-bar mb-3 p-3 rounded border" onsubmit="return hwSnSearchSubmit(event)">
    <div class="row g-3 align-items-end">
        <div class="col-12 col-md-6 col-lg-4">
            <label for="hwSnInput" class="form-label">Серийный номер сервера</label>
            <input type="text" id="hwSnInput" class="form-control" placeholder="Например, CZ12345678" value="{{.SN}}" autofocus>
        </div>
        <div class="col-auto">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Найти</button>
        </div>
        <div class="col-auto">
            <a href="/sprsrv/hwviewlist" class="btn btn-outline-secondary rounded-pill px-3">К списку серверов</a>
        </div>
    </div>
</form>

<script>
    function hwSnSearchSubmit(evt) {
        evt.preventDefault();
        var sn = document.getElementById('hwSnInput').value.trim();
        if (sn === '') return false;
        window.location.href = '/sprsrv/srvhwviewbysn/' + encodeURIComponent(sn);
        return false;
    }
</script>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

{{if .HW}}
<div class="row g-3">
    <div class="col-12 col-xl-6">
        <div class="hw-card">
            <h6>Общие сведения</h6>
            <table class="table table-sm table-borderless hw-kv">
                <tr><td>Имя сервера</td><td>{{.HW.Base.ServerName}}</td></tr>
                <tr><td>Hostname</td><td>{{.HW.Base.Hostname}}</td></tr>
                <tr><td>Модель</td><td>{{.HW.Base.Model}}{{if ne .HW.Base.ShortModel ""}} ({{.HW.Base.ShortModel}}){{end}}</td></tr>
                <tr><td>Форм-фактор</td><td>{{.HW.Base.FormFactor}}</td></tr>
                <tr><td>Серийный номер</td><td>{{.HW.Base.SerialNumber}}</td></tr>
                <tr><td>Виртуальный серийный номер</td><td>{{.HW.Base.VirtualSerialNumber}}</td></tr>
                <tr><td>UUID</td><td>{{.HW.Base.UUID}}</td></tr>
                <tr><td>Партномер</td><td>{{.HW.Base.PartNumber}}</td></tr>
                <tr>
                    <td>Состояние эксплуатации</td>
                    <td>
                        <span class="badge {{if eq .HW.Base.Status "OK"}}text-bg-success{{else if eq .HW.Base.Status "Warning"}}text-bg-warning{{else if eq .HW.Base.Status "Critical"}}text-bg-danger{{else}}text-bg-secondary{{end}}">{{.HW.Base.Status}}</span>
                    </td>
                </tr>
                <tr>
                    <td>Питание</td>
                    <td>
                        <span class="badge {{if eq .HW.Base.PowerState "On"}}text-bg-success{{else if eq .HW.Base.PowerState "Off"}}text-bg-secondary{{else}}text-bg-warning{{end}}">{{.HW.Base.PowerState}}</span>
                    </td>
                </tr>
                <tr><td>Режим обслуживания</td><td>{{if .HW.Base.MaintenanceMode}}<span class="badge text-bg-warning">Да</span>{{else}}Нет{{end}}</td></tr>
                <tr><td>Позиция в стойке</td><td>{{.HW.Base.Position}}</td></tr>
                <tr><td>ОС</td><td>{{.HW.OperatingSystem}}</td></tr>
            </table>
        </div>
    </div>

    <div class="col-12 col-xl-6">
        <div class="hw-card">
            <h6>Процессоры и память</h6>
            <table class="table table-sm table-borderless hw-kv">
                <tr><td>Тип процессора</td><td>{{.HW.Base.ProcessorType}}</td></tr>
                <tr><td>Частота, МГц</td><td>{{.HW.Base.ProcessorSpeedMhz}}</td></tr>
                <tr><td>Количество процессоров</td><td>{{.HW.Base.ProcessorCount}}</td></tr>
                <tr><td>Ядер на процессор</td><td>{{.HW.Base.ProcessorCoreCount}}</td></tr>
                <tr><td>Память, МБ</td><td>{{.HW.Base.MemoryMb}}</td></tr>
                <tr><td>Модулей памяти</td><td>{{.HW.Memory.Count}}</td></tr>
            </table>
        </div>

        <div class="hw-card mt-3">
            <h6>Размещение</h6>
            <table class="table table-sm table-borderless hw-kv">
                <tr><td>Датацентр</td><td>{{.HW.Datacenter.Name}}</td></tr>
                <tr><td>Стойка</td><td>{{.HW.Rack.Name}}</td></tr>
                <tr><td>Отсек (bay)</td><td>{{.HW.BayInfo.BayNumber}}</td></tr>
                <tr><td>Корзина (enclosure)</td><td>{{.HW.EnclosureInfo.Name}}</td></tr>
            </table>
        </div>
    </div>

    {{if .HW.Memory.Data}}
    <div class="col-12">
        <div class="hw-card">
            <h6>Модули памяти ({{.HW.Memory.Count}})</h6>
            <div class="table-responsive">
                <table class="table table-sm table-bordered table-striped align-middle">
                    <thead>
                        <tr>
                            <th>Расположение</th>
                            <th>Наименование</th>
                            <th>Тип</th>
                            <th class="text-end">Ёмкость, МБ</th>
                            <th class="text-end">Частота, МГц</th>
                            <th>Производитель</th>
                            <th>Партномер</th>
                            <th>Состояние</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $m := .HW.Memory.Data}}
                        <tr>
                            <td>{{$m.DeviceLocator}}</td>
                            <td>{{$m.Name}}</td>
                            <td>{{$m.MemoryDeviceType}} ({{$m.BaseModuleType}})</td>
                            <td class="text-end">{{$m.CapacityMiB}}</td>
                            <td class="text-end">{{$m.OperatingSpeedMhz}}</td>
                            <td>{{$m.Manufacturer}}</td>
                            <td>{{$m.PartNumber}}</td>
                            <td>{{$m.Status.Health}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
    </div>
    {{end}}

    {{if .HW.Storage.Data}}
    <div class="col-12">
        <div class="hw-card">
            <h6>Локальные RAID-контроллеры ({{.HW.Storage.Count}})</h6>
            {{range $s := .HW.Storage.Data}}
            <div class="mb-3">
                <div class="fw-semibold">{{$s.Name}} — {{$s.Model}} ({{$s.Location}}), серийный № {{$s.SerialNumber}}, состояние: {{$s.Status.Health}}</div>
                {{if $s.LogicalDrives}}
                <div class="table-responsive mt-2">
                    <table class="table table-sm table-bordered table-striped align-middle">
                        <thead>
                            <tr>
                                <th class="text-center">№</th>
                                <th>RAID</th>
                                <th>Тип</th>
                                <th class="text-end">Ёмкость, МБ</th>
                                <th>Шифрование</th>
                                <th>Состояние</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range $ld := $s.LogicalDrives}}
                            <tr>
                                <td class="text-center">{{$ld.LogicalDriveNumber}}</td>
                                <td>{{$ld.Raid}}</td>
                                <td>{{$ld.LogicalDriveType}}</td>
                                <td class="text-end">{{printf "%.0f" $ld.CapacityMiB}}</td>
                                <td>{{if $ld.LogicalDriveEncryption}}Да{{else}}Нет{{end}}</td>
                                <td>{{$ld.Status.Health}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
                {{end}}
                {{if $s.PhysicalDrives}}
                <div class="table-responsive mt-2">
                    <table class="table table-sm table-bordered table-striped align-middle">
                        <thead>
                            <tr>
                                <th>Расположение</th>
                                <th>Модель</th>
                                <th>Серийный №</th>
                                <th>Тип носителя</th>
                                <th class="text-end">Ёмкость, МБ</th>
                                <th>Использование</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range $pd := $s.PhysicalDrives}}
                            <tr>
                                <td>{{$pd.Location}}</td>
                                <td>{{$pd.Model}}</td>
                                <td>{{$pd.SerialNumber}}</td>
                                <td>{{$pd.MediaType}}</td>
                                <td class="text-end">{{printf "%.0f" $pd.CapacityMiB}}</td>
                                <td>{{$pd.DiskDriveUse}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
    </div>
    {{end}}
</div>
{{end}}

<style>
    .hw-filter-bar {
        position: sticky;
        top: 0;
        z-index: 5;
        background-color: var(--bs-body-bg);
        border-color: var(--bs-border-color) !important;
        box-shadow: 0 2px 6px rgba(0, 0, 0, .08);
    }

    .hw-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 1rem;
        height: 100%;
    }

    .hw-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        margin-bottom: 0.75rem;
    }

    .hw-kv td:first-child {
        color: var(--bs-secondary-color);
        width: 45%;
    }

    .hw-kv td {
        padding: 0.2rem 0.4rem;
    }
</style>
