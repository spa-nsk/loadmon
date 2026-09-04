<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Конфигурация СХД</h2>
        <div class="text-muted">Id записи: <b>{{.Id}}</b> &middot; только просмотр, данные заполняются автосинхронизацией</div>
    </div>
    <a href="/sprsrv/storagesystemslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .SHD}}
<div class="row g-3 align-items-start">
    <div class="col-12 col-lg-6">
        <div class="hw-card">
            <h6>Идентификация</h6>
            <div class="row g-3">
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Питание</div>
                    <div>
                        <svg class="bi power-led {{if eq .SHD.Power 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq .SHD.Power 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg>
                        {{if eq .SHD.Power 1}}Включен{{else}}Выключен{{end}}
                    </div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Статус</div>
                    <div>
                        <span class="badge {{if eq .SHD.Status 1}}text-bg-success{{else if eq .SHD.Status 3}}text-bg-secondary{{else if eq .SHD.Status 6}}text-bg-dark{{else}}text-bg-light{{end}}">
                            {{if eq .SHD.Status 1}}Используется{{else if eq .SHD.Status 2}}Выключен{{else if eq .SHD.Status 3}}Не используется{{else if eq .SHD.Status 4}}Тестирование{{else if eq .SHD.Status 5}}Приёмка{{else if eq .SHD.Status 6}}Демонтирован{{else}}Не указано{{end}}
                        </span>
                    </div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Мнемоническое имя</div>
                    <div><a href="/sprsrv/storagesystemsedit/{{.SHD.Id}}" target="_blank">{{.SHD.ShortName}}</a></div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Производитель / модель</div>
                    <div>{{.SHD.Vendor}} {{.SHD.Model}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Серийный №</div>
                    <div>{{if .SHD.SerialNum}}{{.SHD.SerialNum}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Код продукта (PartNum)</div>
                    <div>{{if .SHD.PartNum}}{{.SHD.PartNum}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Версия прошивки</div>
                    <div>{{if .SHD.Firmware}}{{.SHD.Firmware}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Тип API опроса</div>
                    <div>
                        {{if eq .SHD.ApiType 1}}VNX CLI
                        {{else if eq .SHD.ApiType 2}}Unity REST
                        {{else if eq .SHD.ApiType 3}}Huawei Dorado REST
                        {{else if eq .SHD.ApiType 4}}Huawei OceanStor REST
                        {{else if eq .SHD.ApiType 5}}Lenovo REST
                        {{else}}Не указано{{end}}
                    </div>
                </div>
            </div>
        </div>

        <div class="hw-card mt-3">
            <h6>Сеть</h6>
            <div class="row g-3">
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">IP-адрес</div>
                    <div>{{if .SHD.IpAddress}}{{.SHD.IpAddress}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">IP-адрес 2</div>
                    <div>{{if .SHD.IpAddressNext}}{{.SHD.IpAddressNext}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">WWN-маска дисков</div>
                    <div>{{if .SHD.WWNMask}}{{.SHD.WWNMask}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Список имён в LUN (TagName)</div>
                    <div>{{if .SHD.TagName}}{{.SHD.TagName}}{{else}}&mdash;{{end}}</div>
                </div>
            </div>
        </div>

        <div class="hw-card mt-3">
            <h6>Размещение</h6>
            <div class="row g-3">
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Город</div>
                    <div>{{if .SHD.City}}{{.SHD.City}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">ЦОД / серверная</div>
                    <div>{{if .SHD.DatacenterRoom}}{{.SHD.DatacenterRoom}}{{else}}&mdash;{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Стойка / юнит</div>
                    <div>{{if .SHD.RackNum}}{{.SHD.RackNum}}{{else}}&mdash;{{end}} {{if .SHD.StartUnit}}/ {{.SHD.StartUnit}}{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Размер в юнитах</div>
                    <div>{{if .SHD.SizeUnit}}{{.SHD.SizeUnit}}{{else}}&mdash;{{end}}</div>
                </div>
            </div>
        </div>
    </div>

    <div class="col-12 col-lg-6">
        <div class="hw-card">
            <h6>Ёмкость и опрос</h6>
            <div class="row g-3">
                <div class="col-4">
                    <div class="form-label mb-0 text-muted small">Дисков (макс/факт)</div>
                    <div>{{.SHD.Size}} / {{.SHD.Used}}</div>
                </div>
                <div class="col-4">
                    <div class="form-label mb-0 text-muted small">Ёмкость (сырая), Gb</div>
                    <div>{{printf "%.1f" .SHD.TotalSizeRAW}}</div>
                </div>
                <div class="col-4">
                    <div class="form-label mb-0 text-muted small">Занято, %</div>
                    <div>{{printf "%.1f" .SHD.UsedPercent}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Последний успешный опрос</div>
                    <div>{{if .SHD.LastScanOK}}{{.SHD.LastScanTime.Format "2006-01-02 15:04:05"}}{{else}}<span class="text-danger">Ошибка опроса</span>{{end}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Ошибка последнего опроса</div>
                    <div style="word-break:break-word;">{{if .SHD.LastScanError}}{{.SHD.LastScanError}}{{else}}&mdash;{{end}}</div>
                </div>
            </div>
        </div>

        <div class="hw-card mt-3">
            <h6>Найдено при синхронизации</h6>
            <div class="row g-3">
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Полок / сторадж-процессоров всего</div>
                    <div>{{.EnclosureCount}}</div>
                </div>
                <div class="col-6">
                    <div class="form-label mb-0 text-muted small">Дисков всего</div>
                    <div>{{.DiskCount}}</div>
                </div>
            </div>
        </div>
    </div>
</div>

<div class="hw-card mt-3">
    <h6>Сторадж-процессоры / полки управления ({{len .Processors}})</h6>
    {{if .Processors}}
    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table mb-0">
        <thead>
            <tr>
                <th>Имя</th>
                <th>Производитель / модель</th>
                <th>Серийный №</th>
                <th class="text-end">Слот</th>
                <th class="text-end">Bus Id</th>
                <th>Статус</th>
                <th class="text-end">Дисков</th>
            </tr>
        </thead>
        <tbody>
            {{range $p := .Processors}}
            <tr>
                <td>{{$p.Enclosure.Name}}</td>
                <td>{{$p.Enclosure.Vendor}} {{$p.Enclosure.Model}}</td>
                <td>{{if $p.Enclosure.SerialNum}}{{$p.Enclosure.SerialNum}}{{else}}&mdash;{{end}}</td>
                <td class="text-end">{{$p.Enclosure.SlotNumber}}</td>
                <td class="text-end">{{$p.Enclosure.BusId}}</td>
                <td>
                    <span class="badge {{if eq $p.Enclosure.Status 1}}text-bg-success{{else}}text-bg-secondary{{end}}">
                        {{if eq $p.Enclosure.Status 1}}Активна{{else}}Выведена{{end}}
                    </span>
                </td>
                <td class="text-end">{{len $p.Disks}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <div class="text-muted">Сторадж-процессоры/полки управления не найдены (не синхронизированы для этого вендора либо СХД не опрашивалась - см. "Последний успешный опрос" выше).</div>
    {{end}}
</div>

<div class="hw-card mt-3">
    <h6>Полки с данными ({{len .DataShelves}})</h6>
    {{if .DataShelves}}
    {{range $s := .DataShelves}}
    <div class="mb-3">
        <div class="d-flex flex-wrap align-items-center gap-2 mb-1">
            <b>{{$s.Enclosure.Name}}</b>
            <span class="text-muted small">{{$s.Enclosure.Vendor}} {{$s.Enclosure.Model}}</span>
            {{if $s.Enclosure.SerialNum}}<span class="text-muted small">S/N: {{$s.Enclosure.SerialNum}}</span>{{end}}
            <span class="badge {{if eq $s.Enclosure.Status 1}}text-bg-success{{else}}text-bg-secondary{{end}}">
                {{if eq $s.Enclosure.Status 1}}Активна{{else}}Выведена{{end}}
            </span>
            <span class="text-muted small ms-auto">Слот {{$s.Enclosure.SlotNumber}} &middot; Bus {{$s.Enclosure.BusId}} &middot; дисков: {{len $s.Disks}}</span>
        </div>
        {{if $s.Disks}}
        <table class="table table-sm table-bordered table-striped align-middle spr-classic-table mb-0">
            <thead>
                <tr>
                    <th class="text-end">Слот</th>
                    <th>Производитель / модель</th>
                    <th>Тип</th>
                    <th>Серийный №</th>
                    <th class="text-end">Ёмкость, Gb</th>
                    <th>RAID-группа</th>
                    <th>Статус</th>
                </tr>
            </thead>
            <tbody>
                {{range $d := $s.Disks}}
                <tr>
                    <td class="text-end">{{$d.SlotNumber}}</td>
                    <td>{{$d.Vendor}} {{$d.Model}}</td>
                    <td>{{if $d.DiskType}}{{$d.DiskType}}{{else}}&mdash;{{end}}</td>
                    <td>{{if $d.SerialNum}}{{$d.SerialNum}}{{else}}&mdash;{{end}}</td>
                    <td class="text-end">{{printf "%.1f" $d.CapacityGb}}</td>
                    <td>{{if $d.RaidGroupId}}{{$d.RaidGroupId}}{{else}}&mdash;{{end}}</td>
                    <td>
                        {{if eq $d.Status 1}}<span class="badge text-bg-success">В использовании</span>
                        {{else if eq $d.Status 2}}<span class="badge text-bg-info">Резерв (spare)</span>
                        {{else}}<span class="badge text-bg-secondary">Выведен</span>{{end}}
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <div class="text-muted small">Диски на этой полке не найдены.</div>
        {{end}}
    </div>
    {{end}}
    {{else}}
    <div class="text-muted">Полки с данными не найдены (не синхронизированы для этого вендора либо СХД не опрашивалась).</div>
    {{end}}
</div>

{{if .UnassignedDisks}}
<div class="hw-card mt-3">
    <h6>Диски без привязки к полке ({{len .UnassignedDisks}})</h6>
    <div class="text-muted small mb-2">Диски синхронизированы, но hardware-инвентарь полок для этого вендора не реализован либо полка ещё не привязана.</div>
    <table class="table table-sm table-bordered table-striped align-middle spr-classic-table mb-0">
        <thead>
            <tr>
                <th class="text-end">Полка №</th>
                <th class="text-end">Слот</th>
                <th>Производитель / модель</th>
                <th>Тип</th>
                <th>Серийный №</th>
                <th class="text-end">Ёмкость, Gb</th>
                <th>Статус</th>
            </tr>
        </thead>
        <tbody>
            {{range $d := .UnassignedDisks}}
            <tr>
                <td class="text-end">{{$d.EnclosureNum}}</td>
                <td class="text-end">{{$d.SlotNumber}}</td>
                <td>{{$d.Vendor}} {{$d.Model}}</td>
                <td>{{if $d.DiskType}}{{$d.DiskType}}{{else}}&mdash;{{end}}</td>
                <td>{{if $d.SerialNum}}{{$d.SerialNum}}{{else}}&mdash;{{end}}</td>
                <td class="text-end">{{printf "%.1f" $d.CapacityGb}}</td>
                <td>
                    {{if eq $d.Status 1}}<span class="badge text-bg-success">В использовании</span>
                    {{else if eq $d.Status 2}}<span class="badge text-bg-info">Резерв (spare)</span>
                    {{else}}<span class="badge text-bg-secondary">Выведен</span>{{end}}
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}
{{end}}

<!-- Стиль карточек (.hw-card) и таблиц (.spr-classic-table) - общий с
     /sprsrv/storagevolumeview, server_web_new/static/css/loadmon.css.
     Страница строго read-only - нет ни одного <input>/<select>, нет
     POST-обработчика в контроллере
     (controllers/sprstoragesystems.go:SprStorageSystemConfigView) - все
     данные (СХД, сторадж-процессоры/полки, диски) заполняются
     автосинхронизацией (cmdb.SyncStorageHardwareForSystem). -->
