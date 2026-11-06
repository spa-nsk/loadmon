<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Том СХД</h2>
        <div class="text-muted">Id записи: <b>{{.Id}}</b> &middot; только просмотр, запись заполняется автосинхронизацией</div>
    </div>
    <a href="/sprsrv/storagevolumeslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .ST}}
<ul class="nav nav-tabs" id="svViewTabs" role="tablist">
    <li class="nav-item" role="presentation">
        <button class="nav-link active" id="tab-sv-id-btn" data-bs-toggle="tab" data-bs-target="#tab-sv-id" type="button" role="tab" aria-controls="tab-sv-id" aria-selected="true">Идентификация</button>
    </li>
    <li class="nav-item" role="presentation">
        <button class="nav-link" id="tab-sv-virt-btn" data-bs-toggle="tab" data-bs-target="#tab-sv-virt" type="button" role="tab" aria-controls="tab-sv-virt" aria-selected="false">Виртуализация</button>
    </li>
    <li class="nav-item" role="presentation">
        <button class="nav-link" id="tab-sv-shd-btn" data-bs-toggle="tab" data-bs-target="#tab-sv-shd" type="button" role="tab" aria-controls="tab-sv-shd" aria-selected="false">СХД</button>
    </li>
    <li class="nav-item" role="presentation">
        <button class="nav-link" id="tab-sv-cap-btn" data-bs-toggle="tab" data-bs-target="#tab-sv-cap" type="button" role="tab" aria-controls="tab-sv-cap" aria-selected="false">Ёмкость</button>
    </li>
    <li class="nav-item" role="presentation">
        <button class="nav-link" id="tab-sv-misc-btn" data-bs-toggle="tab" data-bs-target="#tab-sv-misc" type="button" role="tab" aria-controls="tab-sv-misc" aria-selected="false">Прочее</button>
    </li>
</ul>

<div class="tab-content border border-top-0 rounded-bottom p-3 mb-3" id="svViewTabContent">

    <div class="tab-pane fade show active" id="tab-sv-id" role="tabpanel" aria-labelledby="tab-sv-id-btn" tabindex="0">
        <div class="row g-3">
            <div class="col-6 col-md-4">
                <div class="form-label mb-0 text-muted small">Статус</div>
                <div>
                    <span class="badge {{if eq .ST.Status 1}}text-bg-success{{else if eq .ST.Status 2}}text-bg-info{{else}}text-bg-secondary{{end}}">
                        {{if eq .ST.Status 1}}В виртуализации{{else if eq .ST.Status 2}}Активен{{else}}Выведен/не найден{{end}}
                    </span>
                </div>
            </div>
            <div class="col-6 col-md-4">
                <div class="form-label mb-0 text-muted small">UID</div>
                <div>{{if .ST.UID}}{{.ST.UID}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6 col-md-4">
                <div class="form-label mb-0 text-muted small">Тип дисков</div>
                <div>{{if .ST.DiskType}}{{.ST.DiskType}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Имя тома на СХД (StorageName)</div>
                <div>{{if .ST.StorageName}}{{.ST.StorageName}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Имя хранилища в vSphere (VirtualName)</div>
                <div>{{if .ST.VirtualName}}{{.ST.VirtualName}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Мнемоническое имя СХД (ShortName)</div>
                <div>{{if .ST.ShortName}}{{.ST.ShortName}}{{else}}&mdash;{{end}}</div>
            </div>
        </div>
    </div>

    <div class="tab-pane fade" id="tab-sv-virt" role="tabpanel" aria-labelledby="tab-sv-virt-btn" tabindex="0">
        <div class="row g-3">
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">vCenter</div>
                <div>{{if .ST.Vcenter}}{{.ST.Vcenter}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Датацентр</div>
                <div>{{if .ST.Datacenter}}{{.ST.Datacenter}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Кластер</div>
                <div>{{if .ST.Cluster}}{{.ST.Cluster}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Сервер(-а)</div>
                <div style="word-break:break-word;">{{if .ST.Server}}{{.ST.Server}}{{else}}&mdash;{{end}}</div>
            </div>
        </div>
    </div>

    <div class="tab-pane fade" id="tab-sv-shd" role="tabpanel" aria-labelledby="tab-sv-shd-btn" tabindex="0">
        {{if .SHD}}
        <div class="row g-3">
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Мнемоническое имя</div>
                <div><a href="/sprsrv/storagesystemsedit-tab/{{.SHD.Id}}" target="_blank">{{.SHD.ShortName}}</a></div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Производитель / модель</div>
                <div>{{.SHD.Vendor}} {{.SHD.Model}}</div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Серийный №</div>
                <div>{{if .SHD.SerialNum}}{{.SHD.SerialNum}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6 col-md-3">
                <div class="form-label mb-0 text-muted small">Размещение</div>
                <div>{{if .SHDPlacementText}}{{.SHDPlacementText}}{{else}}&mdash;{{end}}</div>
            </div>
        </div>
        {{else}}
        <div class="text-muted">СХД для этого тома не определена (см. журнал событий СХД - /admin/storageeventlog, тип "Неизвестный LUN").</div>
        {{end}}
    </div>

    <div class="tab-pane fade" id="tab-sv-cap" role="tabpanel" aria-labelledby="tab-sv-cap-btn" tabindex="0">
        <div class="row g-3">
            <div class="col-4">
                <div class="form-label mb-0 text-muted small">Ёмкость, Gb</div>
                <div>{{printf "%.1f" .ST.CapacityGb}}</div>
            </div>
            <div class="col-4">
                <div class="form-label mb-0 text-muted small">Свободно, Gb</div>
                <div>{{printf "%.1f" .ST.FreeSpaceGb}}</div>
            </div>
            <div class="col-4">
                <div class="form-label mb-0 text-muted small">Занято, %</div>
                <div>{{printf "%.1f" .ST.PercentUsed}}</div>
            </div>
        </div>
    </div>

    <div class="tab-pane fade" id="tab-sv-misc" role="tabpanel" aria-labelledby="tab-sv-misc-btn" tabindex="0">
        <div class="row g-3">
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Владелец</div>
                <div>{{if .ST.Owner}}{{.ST.Owner}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Аллокация</div>
                <div>{{if .ST.Allocation}}{{.ST.Allocation}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-12">
                <div class="form-label mb-0 text-muted small">Описание</div>
                <div style="word-break:break-word;">{{if .ST.Annotation}}{{.ST.Annotation}}{{else}}&mdash;{{end}}</div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Источник последнего обновления</div>
                <div>
                    {{if eq .ST.SourceUpdateType 1}}Вручную
                    {{else if eq .ST.SourceUpdateType 2}}vSphere (автосинхронизация)
                    {{else if eq .ST.SourceUpdateType 3}}API СХД (автосинхронизация)
                    {{else}}Не указано{{end}}
                </div>
            </div>
            <div class="col-6">
                <div class="form-label mb-0 text-muted small">Время последнего обновления</div>
                <div>{{.ST.Time_Mfy.Format "2006-01-02 15:04:05"}}</div>
            </div>
        </div>
    </div>

</div>
{{end}}
