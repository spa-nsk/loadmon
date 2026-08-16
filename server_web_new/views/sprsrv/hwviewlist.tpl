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
    <input type="hidden" name="page" id="pageField" value="{{.Page}}">

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
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" title="Экспорт данных в CSV (utf8)" formtarget="_blank" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" title="Экспорт данных в XLSX" formtarget="_blank" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                {{if eq .OnView 1}}
                <span class="text-muted small ms-auto">Найдено серверов: {{.HWListCount}}</span>
                {{end}}
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex justify-content-end">
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="hwViewFilterStickyToggle" onchange="hwViewToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="hwViewFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    {{if eq .OnView 1}}
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

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.HWListCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="hwvPageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="hwvPageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="hwvPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwvPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwvPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwvPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwvPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
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

    .hw-filter-bar.hw-filter-bar-unpinned {
        position: static;
        box-shadow: none;
    }

    .pager-bar {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        padding: 0.5rem 0.75rem;
    }

    .pager-bar-sticky {
        position: sticky;
        bottom: 0;
        z-index: 6;
        margin-left: -1rem;
        margin-right: -1rem;
        margin-bottom: -1rem;
        border-radius: 0;
        border-left: none;
        border-right: none;
        border-bottom: none;
        box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.15);
    }

    @media (min-width: 768px) {
        .pager-bar-sticky {
            margin-left: -1.5rem;
            margin-right: -1.5rem;
        }
    }

    .pager-nav .btn:disabled {
        color: var(--bs-secondary-color);
        opacity: 0.55;
    }

    .pager-nav .btn:not(:disabled):not(.disabled):hover {
        background-color: var(--bs-primary);
        color: #fff;
    }

    .pager-pagesize {
        min-width: 4.5rem;
    }

    .spr-classic-table {
        --bs-table-border-color: var(--bs-border-color);
    }

    .spr-classic-table thead th {
        position: sticky;
        top: var(--hwv-thead-top, 0px);
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
    // Смена любого из фильтров начинает выборку заново - со страницы 1,
    // иначе можно было бы остаться на, скажем, 5-й странице нового, гораздо
    // более узкого результата и увидеть пустую таблицу.
    function hwvResetPageField() {
        var field = document.getElementById('pageField');
        if (field) field.value = 1;
    }

    function selectdatacenter(val) {
        hwvResetPageField();
        document.getElementById("Update").submit();
    }

    function selectrack(val) {
        hwvResetPageField();
        document.getElementById("Update").submit();
    }

    function selectstatus(val) {
        hwvResetPageField();
        document.getElementById("Update").submit();
    }

    function selectmaintenance(val) {
        hwvResetPageField();
        document.getElementById("Update").submit();
    }

    function selectpowerstate(val) {
        hwvResetPageField();
        document.getElementById("Update").submit();
    }

    // Переход по странице без потери значений фильтра - номер страницы
    // пишется в скрытое поле формы, после чего форма отправляется тем же
    // POST-ом, что и обычный "Найти" (приём идентичен /sprsrv/hwconflist).
    function hwvPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function hwvPagerChangePageSize(size) {
        var pageSizeField = document.createElement('input');
        pageSizeField.type = 'hidden';
        pageSizeField.name = 'pagesize';
        pageSizeField.value = size;
        var form = document.getElementById('Update');
        form.appendChild(pageSizeField);
        hwvResetPageField();
        form.submit();
    }

    // Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
    // приём идентичен /sprsrv/hwconflist, выбор запоминается отдельным
    // ключом в localStorage. Заголовок таблицы закреплён всегда ("в
    // постоянном режиме"), но его отступ сверху (top) подстраивается под
    // текущее состояние панели фильтра: если фильтр закреплён - заголовок
    // прилипает сразу под ним (на высоту панели), если фильтр не закреплён
    // (прокручивается вместе со страницей) - заголовок прилипает к самому
    // верху экрана.
    (function () {
        var STORAGE_KEY = 'hwviewlist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('hwViewFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('hw-filter-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--hwv-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('hw-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.hwViewToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();
</script>
