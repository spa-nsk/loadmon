<div class="hero-text mb-3">
    <h2>Список виртуальных машин</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form id="vmFilterForm" class="vm-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="filterVcenter" class="form-label">vCenter</label>
            <select id="filterVcenter" class="form-select" onchange="vmFilterNavigate('vcenter')">
                <option value="all" {{if eq .CurVcenter "all"}}selected{{end}}>Все</option>
                {{range $v := .VCenterList}}
                <option value="{{$v}}" {{if eq $.CurVcenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="filterDatacenter" class="form-label">Датацентр</label>
            <select id="filterDatacenter" class="form-select" onchange="vmFilterNavigate('datacenter')">
                <option value="all" {{if eq .CurDatacenter "all"}}selected{{end}}>Все</option>
                {{range $v := .DatacenterList}}
                <option value="{{$v}}" {{if eq $.CurDatacenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="filterCluster" class="form-label">Кластер</label>
            <select id="filterCluster" class="form-select" onchange="vmFilterNavigate('cluster')">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $v := .ClusterList}}
                <option value="{{$v}}" {{if eq $.CurCluster $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3 position-relative">
            <label for="filterVM" class="form-label">Виртуальная машина</label>
            <input type="text" id="filterVM" class="form-control"
                placeholder="Все (введите часть имени и нажмите Enter)"
                value="{{if ne .CurVM "all"}}{{.CurVM}}{{end}}"
                autocomplete="off" />
            <div id="filterVMResults" class="list-group position-absolute w-100 d-none it-system-dropdown"></div>
            <!-- Скрытый источник имён ВМ для JS-подсказок (см. initVMNamePicker
                 ниже) - вместо нативного <datalist>, чей выпадающий список
                 браузер всегда рисует в системной светлой теме и не подчиняется
                 CSS/data-bs-theme страницы. -->
            <div id="filterVMDataList" class="d-none">
                {{range $v := .VMNameList}}
                <div class="vm-name-item" data-name="{{$v}}"></div>
                {{end}}
            </div>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-between align-items-center">
            <span class="text-muted small">Найдено виртуальных машин: {{.VMCount}}</span>
            <div class="d-flex gap-2">
                <a href="?export=xlsx" class="btn btn-outline-success btn-sm rounded-pill px-3" title="Экспорт в XLSX всех виртуальных машин, найденных по текущему фильтру (не только текущей страницы)">Экспорт в XLSX</a>
                <a href="/sprsrv/vm/all/all/all/all" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
            </div>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-end">
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" id="vmFilterStickyToggle" onchange="vmToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted" for="vmFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
</form>

<!-- Без обёртки .table-responsive: у неё overflow-x:auto, из-за чего браузер
     вынужден трактовать и overflow-y как auto - это создаёт собственный
     скролл-контекст, внутри которого position:sticky у thead "прилипает" не
     к верху окна, а к верху этого контейнера, и на глаз выглядит как
     заголовок таблицы, уехавший в середину экрана. Остальные страницы с
     закрепляемым заголовком (/sprsrv/hwviewlist, /sprsrv/cluster и т.п.) по
     той же причине тоже не используют .table-responsive для основной таблицы. -->
<div>
    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table vm-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>vCenter / Датацентр / Кластер / Имя</th>
                <th class="text-center">Процессоры / ядра</th>
                <th class="text-center">Память, ГБ</th>
                <th>Сеть</th>
                <th>IP адрес</th>
                <th>Диски (путь, размер, занято %, свободно)</th>
            </tr>
        </thead>
        <tbody>
            {{range $vm := .VMList}}
            <tr class="row-context-target {{if $vm.PowerState}}vm-row-on{{end}}" data-menu-id="vmRowMenu{{$vm.ID}}" title="Правая кнопка мыши - меню действий">
                <td class="text-end">{{$vm.ID}}</td>
                <td style="min-width:220px;">
                    <svg class="bi power-led {{if $vm.PowerState}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if $vm.PowerState}}Включена{{else}}Выключена{{end}}"><use href="#power"></use></svg>
                    <span>
                        <b>{{$vm.Vcenter}}</b> / <b>{{$vm.Datacenter}}</b> / <b>{{$vm.Cluster}}</b> / <b>{{$vm.Name}}</b>
                        {{if ne $vm.HostName ""}}({{$vm.HostName}}){{end}}
                        {{if $vm.WorkStation}}<span class="badge text-bg-info ms-1">Рабочая станция Windows</span>{{end}}
                    </span>
                </td>

                {{if le $vm.CPUUsageMaximum 75.0}}
                <td class="text-center">{{$vm.NumCPU}} / {{$vm.NumCores}} ({{printf "%.0f" $vm.CPUUsageMaximum}}%)</td>
                {{else if le $vm.CPUUsageMaximum 90.0}}
                <td class="text-center table-warning">{{$vm.NumCPU}} / {{$vm.NumCores}} ({{printf "%.0f" $vm.CPUUsageMaximum}}%)</td>
                {{else}}
                <td class="text-center table-danger">{{$vm.NumCPU}} / {{$vm.NumCores}} ({{printf "%.0f" $vm.CPUUsageMaximum}}%)</td>
                {{end}}

                {{if le $vm.MemUsageMaximum 75.0}}
                <td class="text-center">{{printf "%.0f" $vm.MemorySizeGB}} ({{printf "%.0f" $vm.MemUsageMaximum}}%)</td>
                {{else if le $vm.MemUsageMaximum 90.0}}
                <td class="text-center table-warning">{{printf "%.0f" $vm.MemorySizeGB}} ({{printf "%.0f" $vm.MemUsageMaximum}}%)</td>
                {{else}}
                <td class="text-center table-danger">{{printf "%.0f" $vm.MemorySizeGB}} ({{printf "%.0f" $vm.MemUsageMaximum}}%)</td>
                {{end}}

                <td><div style="max-width:160px; word-wrap:break-word;">{{$vm.Network}}</div></td>
                <td><div style="max-width:160px; word-wrap:break-word;">{{$vm.IPAddress}}</div></td>

                <td style="vertical-align:top;">
                    <div style="max-width:520px;">
                        <div class="small text-muted mb-1">Путь: {{$vm.PathName}}</div>
                        {{if gt $vm.CapacityGb 0.001}}
                        <table class="table table-sm table-borderless mb-0 vm-disk-table">
                            <tr class="fw-semibold">
                                <td>все диски</td>
                                <td class="text-end">{{printf "%.2f" $vm.CapacityGb}} ГБ</td>
                                {{if gt $vm.PercentUsed 90.0}}
                                <td class="text-end table-danger">{{printf "%.2f" $vm.PercentUsed}}%</td>
                                {{else if lt $vm.PercentUsed 1.0}}
                                <td class="text-end table-info">{{printf "%.2f" $vm.PercentUsed}}%</td>
                                {{else}}
                                <td class="text-end">{{printf "%.2f" $vm.PercentUsed}}%</td>
                                {{end}}
                                <td class="text-end">{{printf "%.2f" $vm.FreeSpaceGb}} ГБ своб.</td>
                            </tr>
                            {{range $dsk := $vm.Disk}}
                            <tr>
                                <td class="vm-disk-path" title="{{$dsk.DiskPath}}">{{$dsk.DiskPath}}</td>
                                <td class="text-end">{{printf "%.2f" $dsk.CapacityGb}} ГБ</td>
                                {{if gt $dsk.PercentUsed 90.0}}
                                <td class="text-end table-danger">{{printf "%.2f" $dsk.PercentUsed}}%</td>
                                {{else if lt $dsk.PercentUsed 1.0}}
                                <td class="text-end table-info">{{printf "%.2f" $dsk.PercentUsed}}%</td>
                                {{else}}
                                <td class="text-end">{{printf "%.2f" $dsk.PercentUsed}}%</td>
                                {{end}}
                                <td class="text-end">{{printf "%.2f" $dsk.FreeSpaceGb}} ГБ своб.</td>
                            </tr>
                            {{end}}

                            {{if $vm.UsageDatastore}}
                            <tr class="fw-semibold small">
                                <td>Хранилища и RDM диски</td>
                                <td class="text-end">Выделено</td>
                                <td class="text-end">% от общего</td>
                                <td></td>
                            </tr>
                            {{end}}
                            {{range $rdm := $vm.UsageDatastore}}
                            <tr>
                                {{if gt $rdm.PercentAllocateStorage 0.0}}
                                <td>{{$rdm.StorageName}}</td>
                                {{else}}
                                <td>{{$rdm.StorageName}} {{$rdm.Datastore}}</td>
                                {{end}}
                                <td class="text-end">{{printf "%.2f" $rdm.CommittedGb}} ГБ</td>
                                {{if gt $rdm.PercentAllocateStorage 0.0}}
                                <td class="text-end">{{printf "%.2f" $rdm.PercentAllocateStorage}}%</td>
                                {{else}}
                                <td></td>
                                {{end}}
                                <td></td>
                            </tr>
                            {{end}}

                            {{if $vm.Files}}
                            <tr class="fw-semibold small table-warning">
                                <td>Снапшоты</td>
                                <td class="text-end">Размер</td>
                                <td></td>
                                <td></td>
                            </tr>
                            {{end}}
                            {{range $vmf := $vm.Files}}
                            {{if eq $vmf.Type "snapshotData"}}
                            <tr class="table-warning">
                                <td>{{$vmf.Name}}</td>
                                <td class="text-end">{{printf "%.2f" $vmf.Size}} КБ</td>
                                <td></td>
                                <td></td>
                            </tr>
                            {{end}}
                            {{end}}
                        </table>
                        {{else}}
                        <div class="small text-muted">Нет данных о структуре дисков в виртуальной машине. Выделено {{printf "%.2f" $vm.DiskCommittedGB}} ГБ</div>
                        {{end}}
                    </div>
                </td>

                <td id="vmRowMenu{{$vm.ID}}" class="d-none">
                    <div class="ctx-menu-heading">Загрузка</div>
                    <a class="ctx-menu-item {{if eq $vm.PerfCount 0}}disabled{{end}}"
                        href="/sprsrv/vmperfday/{{$vm.Vcenter}}/{{$vm.Datacenter}}/{{$vm.Cluster}}/{{$vm.Name}}" target="_blank">
                        Загрузка за текущий день
                    </a>
                    <a class="ctx-menu-item"
                        href="/sprsrv/vmperfperiod/{{$vm.Vcenter}}/{{$vm.Datacenter}}/{{$vm.Cluster}}/{{$vm.Name}}" target="_blank">
                        Загрузка за период
                    </a>
                    <!-- Служебные пункты "тест1"/"тест2" (/sprsrv/vmperfkeytest,
                         /sprsrv/vmdelperfkey) убраны - методы не реализованы в controllers,
                         маршруты закомментированы в routers/routers.go. -->
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="7" class="text-center text-muted py-4">По выбранным условиям фильтра виртуальные машины не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<!-- всплывающее меню строки (правая кнопка мыши или кнопка "Действия") -->
<div id="vmRowContextMenu"></div>

<div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
    <div class="text-muted small">
        Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.VMCount}}
    </div>
    <div class="d-flex align-items-center gap-2">
        <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
        <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="vmPagerChangePageSize(this.value)">
            <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
            <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
            <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
        </select>
        <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
            <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
        </div>
    </div>
</div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, индикатора
     питания, всплывающего контекстного меню строки и панели пагинации
     вынесены в общий server_web_new/static/css/loadmon.css (подключён в
     header.tpl) - образец оформления для всего этого /sprsrv/view2. Здесь
     остаются только правила, специфичные для этой страницы. -->
<style>
    /* Стиль выпадающего списка подсказок по имени ВМ - скопирован из
       .it-system-dropdown (views/sprsrv/view2.tpl, updatebase.tpl): элементы
       Bootstrap list-group сами подчиняются теме (data-bs-theme), в отличие
       от нативного <datalist>, который браузер всегда рисует в системной
       светлой теме. */
    .it-system-dropdown {
        z-index: 1000;
        max-height: 280px;
        overflow-y: auto;
        box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, .25);
    }

    .it-system-dropdown .list-group-item {
        cursor: pointer;
    }

    .vm-disk-table td {
        padding: 0.15rem 0.4rem;
    }

    /* Полное имя диска (важно для unix-путей вида /opt/app/very/long/mount) -
       без этого длинное имя без пробелов не переносится и обрезается/вылезает
       за пределы ячейки; break-all переносит по любому символу, если слово
       не помещается, min-width:0 разрешает столбцу таблицы сжиматься под
       перенос вместо того чтобы расширять всю таблицу. */
    .vm-disk-table .vm-disk-path {
        white-space: normal;
        word-break: break-all;
        overflow-wrap: break-word;
        max-width: 260px;
        min-width: 0;
    }

    /* подсветка строки включённой ВМ - лёгкая заливка фона; цвет текста
       не меняется, статус питания уже показывает индикатор (power-led) */
    tr.vm-row-on {
        background-color: rgba(47, 179, 68, 0.08);
    }

    tr.vm-row-on:hover {
        --bs-table-hover-bg: rgba(47, 179, 68, 0.16);
    }

    [data-bs-theme="dark"] tr.vm-row-on {
        background-color: rgba(77, 171, 247, 0.10);
    }

    [data-bs-theme="dark"] tr.vm-row-on:hover {
        --bs-table-hover-bg: rgba(77, 171, 247, 0.18);
    }
</style>

<script>
    document.addEventListener('DOMContentLoaded', function () {
        initVMNamePicker();
    });

    // Поле "Виртуальная машина" - подсказки по частично введённому имени,
    // отрисованные вручную (Bootstrap list-group), а не через нативный
    // <datalist>: выпадающий список datalist рисуется самим браузером в
    // системной светлой теме и не подчиняется теме страницы (light/dark) -
    // приём идентичен initITSystemPicker() в views/sprsrv/view2.tpl.
    function initVMNamePicker() {
        var input = document.getElementById('filterVM');
        var results = document.getElementById('filterVMResults');
        var dataList = document.getElementById('filterVMDataList');
        if (!input || !results || !dataList) return;

        var names = Array.prototype.map.call(dataList.querySelectorAll('.vm-name-item'), function (el) {
            return el.getAttribute('data-name') || '';
        });

        function hideResults() {
            results.classList.add('d-none');
            results.innerHTML = '';
        }

        function renderResults(query) {
            var q = query.trim().toLowerCase();
            if (q === '') {
                hideResults();
                return;
            }
            var matches = names.filter(function (name) {
                return name.toLowerCase().indexOf(q) !== -1;
            }).slice(0, 50);

            if (matches.length === 0) {
                hideResults();
                return;
            }

            results.innerHTML = '';
            matches.forEach(function (name) {
                var row = document.createElement('button');
                row.type = 'button';
                row.className = 'list-group-item list-group-item-action';
                row.textContent = name;
                row.addEventListener('click', function () {
                    input.value = name;
                    hideResults();
                    vmFilterNavigate('vm');
                });
                results.appendChild(row);
            });
            results.classList.remove('d-none');
        }

        input.addEventListener('input', function () {
            renderResults(input.value);
        });
        input.addEventListener('focus', function () {
            if (input.value.trim() !== '') renderResults(input.value);
        });
        document.addEventListener('click', function (evt) {
            if (evt.target !== input && !results.contains(evt.target)) hideResults();
        });
        input.addEventListener('keydown', function (evt) {
            if (evt.key === 'Enter') {
                evt.preventDefault();
                hideResults();
                vmFilterNavigate('vm');
            }
        });
    }

    function vmFilterNavigate(changed) {
        var vc = document.getElementById('filterVcenter').value || 'all';
        var dc = document.getElementById('filterDatacenter').value || 'all';
        var cl = document.getElementById('filterCluster').value || 'all';
        var vm = document.getElementById('filterVM').value || 'all';

        // при смене вышестоящего уровня фильтра нижестоящие сбрасываются в "Все"
        if (changed === 'vcenter') { dc = 'all'; cl = 'all'; vm = 'all'; }
        if (changed === 'datacenter') { cl = 'all'; vm = 'all'; }
        if (changed === 'cluster') { vm = 'all'; }

        var enc = encodeURIComponent;
        window.location.href = '/sprsrv/vm/' + enc(vc) + '/' + enc(dc) + '/' + enc(cl) + '/' + enc(vm);
    }

    // Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
    // приём идентичен /sprsrv/hwviewlist, выбор запоминается отдельным
    // ключом в localStorage. Заголовок таблицы (.spr-classic-table thead th)
    // закреплён всегда, но его отступ сверху (top) подстраивается под текущее
    // состояние панели фильтра: если фильтр закреплён - заголовок прилипает
    // сразу под ним (на высоту панели), если фильтр не закреплён (прокручивается
    // вместе со страницей) - заголовок прилипает к самому верху экрана.
    (function () {
        var STORAGE_KEY = 'vmlist.filterSticky';
        var bar = document.querySelector('.vm-filter-bar');
        var toggle = document.getElementById('vmFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('vm-filter-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('vm-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.vmToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function vmPagerGoTo(page) {
        if (page < 1) page = 1;
        var url = new URL(window.location.href);
        url.searchParams.set('page', page);
        window.location.href = url.toString();
    }

    function vmPagerChangePageSize(size) {
        var url = new URL(window.location.href);
        url.searchParams.set('pagesize', size);
        url.searchParams.set('page', 1);
        window.location.href = url.toString();
    }

    // Всплывающее меню строки (аналог /sprsrv/view2): скрытый td с
    // содержимым меню (vmRowMenu + Id ВМ) переносится в #vmRowContextMenu и
    // позиционируется у места клика/правой кнопки мыши.
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

        window.vmShowRowMenu = function (evt, menuId) {
            evt.preventDefault();
            evt.stopPropagation();
            var rect = evt.currentTarget.getBoundingClientRect();
            showMenuById(menuId, rect.left, rect.bottom + 4);
        };

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
