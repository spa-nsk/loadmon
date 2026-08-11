<div class="hero-text mb-3">
    <h2>Список кластеров</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form id="clusterFilterForm" class="vm-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-4">
            <label for="filterVcenter" class="form-label">vCenter</label>
            <select id="filterVcenter" class="form-select" onchange="clusterFilterNavigate('vcenter')">
                <option value="all" {{if eq .CurVcenter "all"}}selected{{end}}>Все</option>
                {{range $v := .VCenterList}}
                <option value="{{$v}}" {{if eq $.CurVcenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-4">
            <label for="filterDatacenter" class="form-label">Датацентр</label>
            <select id="filterDatacenter" class="form-select" onchange="clusterFilterNavigate('datacenter')">
                <option value="all" {{if eq .CurDatacenter "all"}}selected{{end}}>Все</option>
                {{range $v := .DatacenterList}}
                <option value="{{$v}}" {{if eq $.CurDatacenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-4">
            <label for="filterCluster" class="form-label">Кластер</label>
            <select id="filterCluster" class="form-select" onchange="clusterFilterNavigate('cluster')">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $v := .ClusterList}}
                <option value="{{$v}}" {{if eq $.CurCluster $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-between align-items-center">
            <span class="text-muted small">Найдено кластеров: {{.ClusterCount}}</span>
            <a href="/sprsrv/listclusterview" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
        </div>
    </div>
</form>

<div class="table-responsive">
    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table vm-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>vCenter / Датацентр / Кластер</th>
                <th class="text-center">Хосты</th>
                <th class="text-center">Процессоры / ядра / потоки</th>
                <th class="text-center">Память, ГБ</th>
                <th class="text-center">Хранилище, ГБ</th>
                <th class="text-center">Виртуальные машины</th>
            </tr>
        </thead>
        <tbody>
            {{range $cl := .ClusterFullList}}
            <tr class="row-context-target" data-menu-id="clusterRowMenu{{$cl.ID}}" title="Правая кнопка мыши - меню действий">
                <td class="text-end">{{$cl.ID}}</td>
                <td style="min-width:220px;">
                    <span>
                        <b>{{$cl.Vcenter}}</b> / <b>{{$cl.Datacenter}}</b> / <b>{{$cl.Name}}</b>
                        {{if $cl.HostFlag}}<span class="badge text-bg-secondary ms-1">Отдельный хост</span>{{end}}
                    </span>
                </td>
                <td class="text-center">{{$cl.HostCount}}</td>

                {{if le $cl.CPUUsageMaximum 75.0}}
                <td class="text-center">{{$cl.AllNumCPUPkgs}} / {{$cl.AllNumCPUCores}} / {{$cl.AllNumCPUThreads}} ({{printf "%.0f" $cl.CPUUsageMaximum}}%)</td>
                {{else if le $cl.CPUUsageMaximum 90.0}}
                <td class="text-center table-warning">{{$cl.AllNumCPUPkgs}} / {{$cl.AllNumCPUCores}} / {{$cl.AllNumCPUThreads}} ({{printf "%.0f" $cl.CPUUsageMaximum}}%)</td>
                {{else}}
                <td class="text-center table-danger">{{$cl.AllNumCPUPkgs}} / {{$cl.AllNumCPUCores}} / {{$cl.AllNumCPUThreads}} ({{printf "%.0f" $cl.CPUUsageMaximum}}%)</td>
                {{end}}

                {{if le $cl.MemUsageMaximum 75.0}}
                <td class="text-center">{{printf "%.0f" $cl.AllMemorySizeGB}} ({{printf "%.0f" $cl.MemUsageMaximum}}%)</td>
                {{else if le $cl.MemUsageMaximum 90.0}}
                <td class="text-center table-warning">{{printf "%.0f" $cl.AllMemorySizeGB}} ({{printf "%.0f" $cl.MemUsageMaximum}}%)</td>
                {{else}}
                <td class="text-center table-danger">{{printf "%.0f" $cl.AllMemorySizeGB}} ({{printf "%.0f" $cl.MemUsageMaximum}}%)</td>
                {{end}}

                {{if gt $cl.AllCapacityGb 0.001}}
                {{if gt $cl.AllPercentUsed 90.0}}
                <td class="text-center table-danger">{{printf "%.0f" $cl.AllCapacityGb}} ({{printf "%.1f" $cl.AllPercentUsed}}%)</td>
                {{else}}
                <td class="text-center">{{printf "%.0f" $cl.AllCapacityGb}} ({{printf "%.1f" $cl.AllPercentUsed}}%)</td>
                {{end}}
                {{else}}
                <td class="text-center text-muted">нет данных</td>
                {{end}}

                <td class="text-center">{{$cl.VMCount}}</td>

                <td id="clusterRowMenu{{$cl.ID}}" class="d-none">
                    <div class="ctx-menu-heading">Загрузка кластера</div>
                    <a class="ctx-menu-item {{if eq $cl.PerfCount 0}}disabled{{end}}"
                        href="/sprsrv/clusterperfday/{{$cl.Vcenter}}/{{$cl.Datacenter}}/{{$cl.Name}}" target="_blank">
                        Загрузка за текущий день
                    </a>
                    <a class="ctx-menu-item"
                        href="/sprsrv/clusterperfperiod/{{$cl.Vcenter}}/{{$cl.Datacenter}}/{{$cl.Name}}" target="_blank">
                        Загрузка за период
                    </a>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="7" class="text-center text-muted py-4">По выбранным условиям фильтра кластеры не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<!-- всплывающее меню строки (правая кнопка мыши) -->
<div id="vmRowContextMenu"></div>

<div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
    <div class="text-muted small">
        Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.ClusterCount}}
    </div>
    <div class="d-flex align-items-center gap-2">
        <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
        <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="clusterPagerChangePageSize(this.value)">
            <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
            <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
            <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
        </select>
        <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clusterPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clusterPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
            <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clusterPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clusterPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
        </div>
    </div>
</div>

<style>
    .vm-filter-bar {
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

    .spr-classic-table tbody tr.row-context-target {
        cursor: context-menu;
    }

    #vmRowContextMenu {
        position: fixed;
        display: none;
        min-width: 260px;
        max-width: 380px;
        max-height: 80vh;
        overflow-y: auto;
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        box-shadow: 0 0.5rem 1.5rem rgba(0, 0, 0, 0.3);
        padding: 0.5rem;
        z-index: 2000;
    }

    #vmRowContextMenu.show {
        display: block;
    }

    #vmRowContextMenu .ctx-menu-heading {
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        font-weight: 600;
        color: var(--bs-secondary-color);
        padding: 0.35rem 0.6rem 0.15rem;
    }

    #vmRowContextMenu a.ctx-menu-item {
        display: block;
        padding: 0.35rem 0.6rem;
        border-radius: 0.4rem;
        font-size: 0.9rem;
        text-decoration: none;
        color: var(--bs-emphasis-color);
        transition: background-color .15s ease, color .15s ease;
    }

    #vmRowContextMenu a.ctx-menu-item.disabled {
        pointer-events: none;
        opacity: 0.5;
    }

    #vmRowContextMenu a.ctx-menu-item:hover,
    #vmRowContextMenu a.ctx-menu-item:focus {
        background-color: var(--bs-primary);
        color: #fff;
    }

    /* панель постраничной навигации - оформление идентично /sprsrv/vm (vmlist.tpl) */
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
</style>

<script>
    function clusterFilterNavigate(changed) {
        var vc = document.getElementById('filterVcenter').value || 'all';
        var dc = document.getElementById('filterDatacenter').value || 'all';
        var cl = document.getElementById('filterCluster').value || 'all';

        if (changed === 'vcenter') { dc = 'all'; cl = 'all'; }
        if (changed === 'datacenter') { cl = 'all'; }

        // смена фильтра всегда возвращает на первую страницу списка
        var enc = encodeURIComponent;
        window.location.href = '/sprsrv/listclusterview?vcenter=' + enc(vc) + '&datacenter=' + enc(dc) + '&cluster=' + enc(cl) + '&page=1';
    }

    function clusterPagerGoTo(page) {
        if (page < 1) page = 1;
        var url = new URL(window.location.href);
        url.searchParams.set('page', page);
        window.location.href = url.toString();
    }

    function clusterPagerChangePageSize(size) {
        var url = new URL(window.location.href);
        url.searchParams.set('pagesize', size);
        url.searchParams.set('page', 1);
        window.location.href = url.toString();
    }

    // Всплывающее меню строки - тот же приём, что и в /sprsrv/vm (vmlist.tpl)
    (function () {
        var menu = document.getElementById('vmRowContextMenu');
        if (!menu) return;

        function hideMenu() {
            menu.classList.remove('show');
        }

        function positionMenu(x, y) {
            menu.style.left = x + 'px';
            menu.style.top = y + 'px';
            menu.classList.add('show');
            requestAnimationFrame(function () {
                var rect = menu.getBoundingClientRect();
                var newLeft = x, newTop = y;
                if (rect.right > window.innerWidth) {
                    newLeft = Math.max(0, window.innerWidth - rect.width - 4);
                }
                if (rect.bottom > window.innerHeight) {
                    newTop = Math.max(0, window.innerHeight - rect.height - 4);
                }
                menu.style.left = newLeft + 'px';
                menu.style.top = newTop + 'px';
            });
        }

        function showMenuById(menuId, x, y) {
            var src = document.getElementById(menuId);
            if (!src) return;
            menu.innerHTML = src.innerHTML;
            positionMenu(x, y);
        }

        var table = document.querySelector('.vm-table');
        if (table) {
            table.addEventListener('contextmenu', function (e) {
                var tr = e.target.closest('tr.row-context-target');
                if (!tr) return;
                e.preventDefault();
                showMenuById(tr.getAttribute('data-menu-id'), e.clientX, e.clientY);
            });
        }

        document.addEventListener('click', hideMenu);
        document.addEventListener('scroll', hideMenu, true);
        window.addEventListener('resize', hideMenu);
        document.addEventListener('contextmenu', function (e) {
            if (!e.target.closest('tr.row-context-target') && !e.target.closest('#vmRowContextMenu')) {
                hideMenu();
            }
        });
        menu.addEventListener('click', function (e) {
            e.stopPropagation();
        });
    })();
</script>
