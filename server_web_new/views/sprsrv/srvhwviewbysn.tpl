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
        {{if .HW}}
        <div class="col-auto">
            <a href="/sprsrv/hwconfeditbysn/{{.HW.Identity.SerialNumber}}" target="_blank" title="Открыть конфигурацию этого сервера из справочника в базе данных loadmon" class="btn btn-outline-secondary rounded-pill px-3">Конфигурация сервера (БД)</a>
        </div>
        {{end}}
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
        <div class="hw-card hw-card-fill">
            <h6>Общие сведения</h6>
            <table class="table table-sm table-borderless hw-kv">
                <tr><td>Имя сервера</td><td>{{.HW.Identity.ServerName}}</td></tr>
                <tr><td>Hostname</td><td>{{.HW.Identity.MpHostName}}</td></tr>
                <tr><td>Модель</td><td>{{.HW.Identity.Model}}{{if ne .HW.Identity.ShortModel ""}} ({{.HW.Identity.ShortModel}}){{end}}</td></tr>
                <tr><td>Форм-фактор</td><td>{{.HW.Identity.FormFactor}}</td></tr>
                <tr><td>Серийный номер</td><td>{{.HW.Identity.SerialNumber}}</td></tr>
                <tr><td>UUID</td><td>{{.HW.Identity.UUID}}</td></tr>
                <tr><td>Партномер</td><td>{{.HW.Identity.PartNumber}}</td></tr>
                <tr>
                    <td>Состояние эксплуатации</td>
                    <td>
                        <span class="badge {{if eq .HW.Identity.Status "OK"}}text-bg-success{{else if eq .HW.Identity.Status "Warning"}}text-bg-warning{{else if eq .HW.Identity.Status "Critical"}}text-bg-danger{{else}}text-bg-secondary{{end}}">{{.HW.Identity.Status}}</span>
                    </td>
                </tr>
                <tr>
                    <td>Питание</td>
                    <td>
                        <span class="badge {{if eq .HW.Identity.PowerState "On"}}text-bg-success{{else if eq .HW.Identity.PowerState "Off"}}text-bg-secondary{{else}}text-bg-warning{{end}}">{{.HW.Identity.PowerState}}</span>
                    </td>
                </tr>
                <tr><td>Позиция в отсеке (bay)</td><td>{{.HW.Identity.Position}}</td></tr>
                <tr><td>ОС</td><td>{{.HW.Identity.OperatingSystem}}</td></tr>
            </table>
        </div>
    </div>

    <div class="col-12 col-xl-6">
        <div class="hw-card">
            <h6>Процессоры и память</h6>
            <table class="table table-sm table-borderless hw-kv">
                {{if .HW.Processors}}
                {{with index .HW.Processors 0}}
                <tr><td>Тип процессора</td><td>{{.Model}}</td></tr>
                <tr><td>Частота, МГц</td><td>{{.OperatingSpeedMHz}}</td></tr>
                {{end}}
                {{end}}
                <tr><td>Количество процессоров</td><td>{{len .HW.Processors}}</td></tr>
                <tr><td>Память, МБ</td><td>{{.TotalMemoryMb}}</td></tr>
                <tr><td>Модулей памяти</td><td>{{len .HW.Memory}}</td></tr>
            </table>
        </div>

        <div class="hw-card">
            <h6>Размещение</h6>
            <table class="table table-sm table-borderless hw-kv">
                <tr><td>Датацентр</td><td>{{if ne .HW.Datacenter.Name ""}}{{.HW.Datacenter.Name}}{{else}}&mdash;{{end}}</td></tr>
                <tr><td>Шкаф</td><td>{{if ne .HW.Rack.Name ""}}{{.HW.Rack.Name}}{{else}}&mdash;{{end}}</td></tr>
                <tr>
                    <td>Размещение в шкафу</td>
                    <td>
                        {{if ne .HW.Rack.MountLocation ""}}{{.HW.Rack.MountLocation}}
                        {{else if ne .RackSlotRange ""}}{{.RackSlotRange}}
                        {{else}}&mdash;{{end}}
                    </td>
                </tr>
                <tr><td>Корзина (enclosure)</td><td>{{if ne .HW.EnclosureInfo.Name ""}}{{.HW.EnclosureInfo.Name}}{{else}}&mdash;{{end}}</td></tr>
                <tr><td>Отсек в корзине (bay)</td><td>{{if gt .HW.BayInfo.BayNumber 0}}{{.HW.BayInfo.BayNumber}}{{else}}&mdash;{{end}}</td></tr>
            </table>
        </div>
    </div>

    {{if .HW.Memory}}
    <div class="col-12">
        <div class="hw-card">
            <h6>Модули памяти ({{len .HW.Memory}})</h6>
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
                        {{range $m := .HW.Memory}}
                        <tr>
                            <td>{{$m.DeviceLocator}}</td>
                            <td>{{$m.Name}}</td>
                            <td>{{$m.MemoryDeviceType}} ({{$m.BaseModuleType}})</td>
                            <td class="text-end">{{$m.CapacityMiB}}</td>
                            <td class="text-end">{{$m.OperatingSpeedMHz}}</td>
                            <td>{{$m.Manufacturer}}</td>
                            <td>{{$m.PartNumber}}</td>
                            <td>{{$m.Health}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
    </div>
    {{end}}

    {{if .HW.Controllers}}
    <div class="col-12">
        <div class="hw-card">
            <h6>Локальные RAID-контроллеры ({{len .HW.Controllers}})</h6>
            {{range $s := .HW.Controllers}}
            <div class="mb-3">
                <div class="fw-semibold">{{$s.Name}} — {{$s.Model}} ({{$s.Location}}), серийный № {{$s.SerialNumber}}, состояние: {{$s.Health}}</div>
                {{if $s.Volumes}}
                <div class="table-responsive mt-2">
                    <table class="table table-sm table-bordered table-striped align-middle">
                        <thead>
                            <tr>
                                <th class="text-center">№</th>
                                <th>RAID</th>
                                <th>Тип носителя</th>
                                <th class="text-end">Ёмкость, МБ</th>
                                <th>Состояние</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range $ld := $s.Volumes}}
                            <tr>
                                <td class="text-center">{{$ld.Number}}</td>
                                <td>{{$ld.RAID}}</td>
                                <td>{{$ld.MediaType}}</td>
                                <td class="text-end">{{$ld.CapacityMiB}}</td>
                                <td>{{$ld.Health}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
                {{end}}
                {{if $s.Drives}}
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
                            {{range $pd := $s.Drives}}
                            <tr>
                                <td>{{$pd.Location}}</td>
                                <td>{{$pd.Model}}</td>
                                <td>{{$pd.SerialNumber}}</td>
                                <td>{{$pd.MediaType}}</td>
                                <td class="text-end">{{$pd.CapacityMiB}}</td>
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

<!-- Стили закрепляемой панели фильтра, карточки данных (.hw-card,
     .hw-card-fill - height:100% только там, где в колонке ровно одна
     карточка, см. комментарий в loadmon.css) и пар "ключ - значение"
     (.hw-kv) вынесены в общий server_web_new/static/css/loadmon.css
     (подключён в header.tpl). -->
