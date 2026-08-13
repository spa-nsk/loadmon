<div class="hero-text mb-3">
    <h2>Список конфигураций серверов</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="Update" method="POST">
    <div class="hw-filter-bar mb-3 p-3 rounded border">
        <div class="row g-3 align-items-end">
            <div class="col-6 col-md-4">
                <label for="PowerState" class="form-label">Состояние питания</label>
                <select id="PowerState" name="PowerState" size="1" class="form-select" onchange="selectpowerstate(this.value)">
                    <option value="all" {{if eq .HWViewFilter.PowerState "all"}}selected{{end}}>Все</option>
                    <option value="On" {{if eq .HWViewFilter.PowerState "On"}}selected{{end}}>Включен</option>
                    <option value="Off" {{if eq .HWViewFilter.PowerState "Off"}}selected{{end}}>Выключен</option>
                    <option value="PoweringOff" {{if eq .HWViewFilter.PowerState "PoweringOff"}}selected{{end}}>Выключение</option>
                    <option value="PoweringOn" {{if eq .HWViewFilter.PowerState "PoweringOn"}}selected{{end}}>Включение</option>
                    <option value="Resetting" {{if eq .HWViewFilter.PowerState "Resetting"}}selected{{end}}>Перезагрузка (сброс)</option>
                    <option value="Unknown" {{if eq .HWViewFilter.PowerState "Unknown"}}selected{{end}}>Не определено</option>
                </select>
            </div>
            <div class="col-6 col-md-4">
                <label for="Status" class="form-label">Состояние эксплуатации</label>
                <select id="Status" name="Status" size="1" class="form-select" onchange="selectstatus(this.value)">
                    <option value="all" {{if eq .HWViewFilter.Status "all"}}selected{{end}}>Все</option>
                    <option value="OK" {{if eq .HWViewFilter.Status "OK"}}selected{{end}}>Нормально</option>
                    <option value="Disabled" {{if eq .HWViewFilter.Status "Disabled"}}selected{{end}}>Отключен</option>
                    <option value="Warning" {{if eq .HWViewFilter.Status "Warning"}}selected{{end}}>Требует внимания</option>
                    <option value="Critical" {{if eq .HWViewFilter.Status "Critical"}}selected{{end}}>Критичное</option>
                    <option value="Unknown" {{if eq .HWViewFilter.Status "Unknown"}}selected{{end}}>Не определено</option>
                </select>
            </div>
            <div class="col-6 col-md-4">
                <label for="MaintenanceMode" class="form-label">Режим обслуживания</label>
                <select id="MaintenanceMode" name="MaintenanceMode" size="1" class="form-select" onchange="selectmaintenance(this.value)">
                    <option value="all" {{if eq .HWViewFilter.MaintenanceMode "all"}}selected{{end}}>Все</option>
                    <option value="false" {{if eq .HWViewFilter.MaintenanceMode "false"}}selected{{end}}>В работе</option>
                    <option value="true" {{if eq .HWViewFilter.MaintenanceMode "true"}}selected{{end}}>В обслуживании</option>
                </select>
            </div>
            <div class="col-6 col-md-6">
                <label for="DatacenterName" class="form-label">Датацентр</label>
                <select id="DatacenterName" name="DatacenterName" size="1" class="form-select" onchange="selectdatacenter(this.value)">
                    <option value="all" {{if eq .HWViewFilter.DatacenterName "all"}}selected{{end}}>Все</option>
                    {{range $room := .Datacenters}}
                    <option value="{{$room}}" {{if eq $room $.HWViewFilter.DatacenterName}}selected{{end}}>{{$room}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-6">
                <label for="RackName" class="form-label">Шкаф</label>
                <select id="RackName" name="RackName" size="1" class="form-select" onchange="selectrack(this.value)">
                    <option value="all" {{if eq .HWViewFilter.RackName "all"}}selected{{end}}>Все</option>
                    {{range $rack := .Racks}}
                    <option value="{{$rack}}" {{if eq $rack $.HWViewFilter.RackName}}selected{{end}}>{{$rack}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex align-items-center gap-2">
                <button type="submit" name="FindHW" value="Найти" class="btn btn-primary rounded-pill px-3">Найти</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" title="Экспорт данных в CSV (utf8)" class="btn btn-outline-secondary rounded-pill px-3">Экспорт в CSV</button>
                {{if eq .OnView 1}}
                <span class="text-muted small ms-auto">Найдено серверов: {{.HWListCount}}</span>
                {{end}}
            </div>
        </div>
    </div>

    {{if eq .OnView 1}}
    <div class="table-responsive">
        <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
            <thead>
                <tr>
                    <th>Состояние эксплуатации</th>
                    <th>Серийный №</th>
                    <th>Имя</th>
                    <th>Модель</th>
                    <th>Размещение</th>
                </tr>
            </thead>
            <tbody id="myTable">
                {{range $ind, $hw := .HWList}}
                <tr>
                    <td>
                        <span class="badge {{if eq $hw.Identity.Status "OK"}}text-bg-success{{else if eq $hw.Identity.Status "Warning"}}text-bg-warning{{else if eq $hw.Identity.Status "Critical"}}text-bg-danger{{else}}text-bg-secondary{{end}}">{{$hw.Identity.Status}}</span>
                    </td>
                    <td>
                        <svg class="bi power-led {{if eq $hw.Identity.PowerState "On"}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{$hw.Identity.PowerState}}"><use href="#power"></use></svg>
                        <a href="/sprsrv/srvhwviewbysn/{{$hw.Identity.SerialNumber}}" target="_blank" title="Подробнее">{{$hw.Identity.SerialNumber}}</a>
                    </td>
                    <td>{{$hw.Identity.ServerName}}</td>
                    <td>
                        <div style="max-width:340px; word-wrap:break-word;">{{$hw.Identity.Model}} {{if ne $hw.Identity.PartNumber ""}}({{$hw.Identity.PartNumber}}){{end}}</div>
                    </td>
                    <td>{{if ne $hw.Datacenter.Name ""}}{{$hw.Datacenter.Name}}{{end}}{{if ne $hw.Rack.Name ""}} ({{$hw.Rack.Name}}){{end}}</td>
                </tr>
                {{else}}
                <tr>
                    <td colspan="5" class="text-center text-muted py-4">По выбранным условиям фильтра серверы не найдены</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    {{end}}
</form>

<style>
    .hw-filter-bar {
        position: sticky;
        top: 0;
        z-index: 5;
        background-color: var(--bs-body-bg);
        border-color: var(--bs-border-color) !important;
        box-shadow: 0 2px 6px rgba(0, 0, 0, .08);
    }

    .spr-classic-table {
        --bs-table-border-color: var(--bs-border-color);
    }

    .spr-classic-table thead th {
        position: sticky;
        top: 0;
        z-index: 3;
        background-color: var(--bs-tertiary-bg);
        color: var(--bs-emphasis-color);
        font-weight: 600;
        border-bottom: 2px solid var(--bs-border-color);
        vertical-align: middle;
    }

    .spr-classic-table td,
    .spr-classic-table th {
        border-color: var(--bs-border-color);
    }

    .spr-classic-table tbody tr:hover {
        --bs-table-hover-bg: var(--bs-secondary-bg);
    }

    .power-led {
        width: 1.2em;
        height: 1.2em;
        vertical-align: -0.25em;
        margin-right: 0.3rem;
        flex-shrink: 0;
    }

    .power-led.is-on {
        color: #2fb344;
        filter: drop-shadow(0 0 2px rgba(47, 179, 68, 0.65));
    }

    [data-bs-theme="dark"] .power-led.is-on {
        color: #4dabf7;
        filter: drop-shadow(0 0 3px rgba(77, 171, 247, 0.75));
    }

    .power-led.is-off {
        color: var(--bs-secondary-color);
        opacity: 0.45;
        filter: none;
    }
</style>

<script>
    function selectdatacenter(val) {
        document.getElementById("Update").submit();
    }

    function selectrack(val) {
        document.getElementById("Update").submit();
    }

    function selectstatus(val) {
        document.getElementById("Update").submit();
    }

    function selectmaintenance(val) {
        document.getElementById("Update").submit();
    }

    function selectpowerstate(val) {
        document.getElementById("Update").submit();
    }
</script>
