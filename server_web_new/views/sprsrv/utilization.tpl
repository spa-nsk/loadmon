<!-- Контроллер: controllers/sprutilization.go (SprUtilization). Оформление -
     по образцу дашбордов Grafana/Zabbix (референсы присланы пользователем):
     сплошные цветные "singlestat"-панели вместо тонких полосок прогресса
     (.util-stat + renderSinglestat), heatmap-полоса заполненности СХД/LUN
     в духе панели "Datastores - Usage Capacity" (.util-heatmap +
     renderHeatmap), таблица рисков с заливкой всей ячейки severity-цветом
     как в Zabbix "Detected problems" (.util-risk-cell-*). ApexCharts
     оставлен только для горизонтальных bar-списков (кластеры/хосты/VM),
     где он ведёт себя предсказуемо. Общий паттерн подключения ApexCharts/
     тёмной темы - как в /sprsrv/zabbixperfperiod. -->

<div class="hero-text mb-3">
    <h2>Утилизация виртуальной инфраструктуры</h2>
    <div class="text-muted">Сводная оценка загрузки кластеров, хостов, виртуальных машин, СХД и LUN по последним суточным данным. Риски перегруза и неиспользуемые ресурсы - в таблице рекомендаций внизу страницы.</div>
</div>

<!-- Верхняя строка - "singlestat"-панели, как в grafana (сплошной цветной
     фон вместо тонкой полоски прогресса - количества нейтральные тёмно-серые,
     проценты загрузки и счётчики рисков красятся по порогу). Цвет проставляет
     JS (renderSinglestat) - см. ниже. -->
<div class="util-stat-row mb-3" id="utilStatCounts">
    <div class="util-stat util-stat-neutral"><div class="util-stat-label">Кластеров</div><div class="util-stat-value">{{.KPI.ClusterCount}}</div></div>
    <div class="util-stat util-stat-neutral"><div class="util-stat-label">Хостов</div><div class="util-stat-value">{{.KPI.HostCount}}</div></div>
    <div class="util-stat util-stat-neutral"><div class="util-stat-label">Виртуальных машин</div><div class="util-stat-value">{{.KPI.VMCount}}</div></div>
    <div class="util-stat util-stat-neutral"><div class="util-stat-label">СХД</div><div class="util-stat-value">{{.KPI.StorageSystemCount}}</div></div>
    <div class="util-stat util-stat-neutral"><div class="util-stat-label">LUN</div><div class="util-stat-value">{{.KPI.LUNCount}}</div></div>
    <div class="util-stat" id="utilStatOverload"><div class="util-stat-label">Риски перегруза</div><div class="util-stat-value">{{.KPI.OverloadedCount}}</div></div>
    <div class="util-stat" id="utilStatUnderused"><div class="util-stat-label">Недоиспользуется</div><div class="util-stat-value">{{.KPI.UnderusedCount}}</div></div>
</div>

<div class="util-stat-row mb-3" id="utilStatLoad"></div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>СХД - заполненность, %</h6>
            <div id="utilHeatmapStorage" class="util-heatmap"></div>
        </div>
    </div>
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>LUN - заполненность, % (топ-30 самых занятых)</h6>
            <div id="utilHeatmapLun" class="util-heatmap"></div>
        </div>
    </div>
</div>

<div class="util-card mb-3">
    <h6>Структура оборудования: vCenter → кластер → хост / VM</h6>
    <div class="text-muted small mb-2"><span class="util-legend-dot util-legend-danger"></span>перегруз &nbsp;<span class="util-legend-dot util-legend-warning"></span>недоиспользование &nbsp;<span class="util-legend-dot util-legend-normal"></span>норма</div>
    <div id="utilTree" class="util-tree"></div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>Кластеры и хосты по загрузке CPU, %</h6>
            <div id="utilBarClustersHosts" class="util-chart-box"></div>
        </div>
    </div>
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>Перегруженные VM (CPU avg ≥ 85%)</h6>
            <div id="utilBarVmTop" class="util-chart-box"></div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>Кандидаты на right-sizing / отключение (CPU max &lt; 5%)</h6>
            <div id="utilBarVmIdle" class="util-chart-box"></div>
        </div>
    </div>
</div>

<div class="util-card mb-3">
    <h6>Риски и рекомендации по оптимизации</h6>
    {{if gt .RisksTotal .RisksShown}}
    <div class="text-muted small mb-2">Показаны {{.RisksShown}} наиболее критичных записей из {{.RisksTotal}} - отсортировано по степени риска, затем по значению метрики.</div>
    {{end}}
    <div class="table-responsive util-risks-scroll">
        <table class="table table-sm align-middle mb-0 util-risks-table" id="utilRisksTable">
            <thead>
                <tr>
                    <th>Категория</th>
                    <th>Объект</th>
                    <th>Метрика</th>
                    <th>Значение</th>
                    <th>Риск</th>
                    <th>Рекомендация</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>
    <div id="utilRisksEmpty" class="util-empty-state d-none">
        <div>Рисков перегруза или существенного недоиспользования по текущим данным не выявлено.</div>
    </div>
</div>

<div class="util-card mb-3">
    <h6>Данных недостаточно? Предложения по дополнительным методам сбора</h6>
    <div class="text-muted small mb-2">Оценка построена на данных, которые уже собирает loadmon (суточные агрегаты ClusterRecord/HostRecord/VMRecord, справочники StorageSystems/StorageVolumes). Ниже - выявленные пробелы, для устранения которых потребуются доработки сбора метрик.</div>
    <ul id="utilDataGapsList" class="util-gaps-list mb-0"></ul>
</div>

<script src="/static/js/apexcharts.min.js?v={{.StaticVer}}"></script>

<style>
    /* --- Верхняя строка stat-панелей (в духе Grafana stat panel) --- */
    .util-stat-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
        gap: 1px;
        background: var(--bs-border-color);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        overflow: hidden;
    }

    .util-stat {
        background: var(--bs-tertiary-bg);
        padding: 0.85rem 1rem;
        border-left: 3px solid transparent;
    }

    .util-stat-accent-danger { border-left-color: #e64980; }
    .util-stat-accent-warning { border-left-color: #f59f00; }

    .util-stat-label {
        color: var(--bs-secondary-color);
        font-size: 0.72rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        margin-bottom: 0.25rem;
    }

    .util-stat-value {
        font-size: 1.75rem;
        font-weight: 700;
        line-height: 1.1;
        color: var(--bs-emphasis-color);
    }

    .util-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        padding: 1rem 1.2rem;
        height: 100%;
        /* изолируем каждую карточку в собственный слой отрисовки - защита от
           артефактов перерисовки браузера (наложение старого кадра поверх
           нового) при большом количестве DOM-узлов, которые JS вставляет
           в соседние карточки (например, сотни строк в таблице рисков) */
        contain: content;
        position: relative;
        z-index: 0;
    }

    /* таблица рисков может быть длинной даже после ограничения в 60 строк -
       держим её в собственной прокручиваемой области фиксированной высоты,
       а не даём карточке растягиваться на всю длину таблицы */
    .util-risks-scroll {
        max-height: 640px;
        overflow-y: auto;
    }

    .util-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        font-size: 0.85rem;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        margin-bottom: 0.9rem;
    }

    .util-chart-box { min-height: 320px; }

    /* --- Singlestat-панели (сплошной цветной фон, крупное число) - в духе
       grafana stat panel (см. пример "CPU Usage 14.55%" зелёным фоном) --- */
    .util-stat-row + .util-stat-row { margin-top: -0.75rem; }

    .util-stat {
        background: var(--bs-secondary-bg);
        padding: 0.7rem 1rem;
        border-radius: 0.35rem;
        color: #fff;
    }

    .util-stat-neutral { background: #3a3f4b; }

    .util-stat-ok { background: #299a3e; }
    .util-stat-watch { background: #d9822b; }
    .util-stat-crit { background: #c4162a; }

    .util-stat-label {
        font-size: 0.72rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        opacity: 0.85;
        margin-bottom: 0.2rem;
    }

    .util-stat-value {
        font-size: 1.7rem;
        font-weight: 700;
        line-height: 1.1;
    }

    /* --- Heatmap-полоса заполненности (в духе grafana "Datastores - Usage
       Capacity": числа сверху цветом по порогу, вертикальный столбик снизу,
       имя объекта под столбиком) --- */
    .util-heatmap {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-end;
        gap: 0.5rem;
        min-height: 220px;
        padding-top: 0.5rem;
    }

    .util-heatmap-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        width: 2.4rem;
    }

    .util-heatmap-pct {
        font-size: 0.68rem;
        font-weight: 700;
        margin-bottom: 0.3rem;
        white-space: nowrap;
    }

    .util-heatmap-bar {
        width: 1.1rem;
        height: 140px;
        border-radius: 0.2rem;
        background: var(--bs-secondary-bg);
        display: flex;
        align-items: flex-end;
        overflow: hidden;
    }

    .util-heatmap-fill { width: 100%; border-radius: 0.2rem 0.2rem 0 0; }

    .util-heatmap-name {
        font-size: 0.62rem;
        color: var(--bs-secondary-color);
        margin-top: 0.3rem;
        max-width: 3.4rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        writing-mode: vertical-rl;
        transform: rotate(180deg);
        height: 3.4rem;
    }

    .util-legend-dot {
        display: inline-block;
        width: 0.6rem;
        height: 0.6rem;
        border-radius: 50%;
    }

    .util-legend-danger { background: #e64980; }
    .util-legend-warning { background: #f59f00; }
    .util-legend-normal { background: #4dabf7; }

    /* --- Дерево структуры оборудования --- */
    .util-tree {
        max-height: 480px;
        overflow-y: auto;
        border: 1px solid var(--bs-border-color);
        border-radius: 0.4rem;
    }

    .util-tree-cluster {
        border-bottom: 1px solid var(--bs-border-color);
        padding: 0.6rem 0.8rem;
    }

    .util-tree-cluster:last-child { border-bottom: none; }

    .util-tree-cluster-head {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 0.5rem;
        margin-bottom: 0.45rem;
        padding-left: 0.6rem;
        border-left: 3px solid #4dabf7;
    }

    .util-tree-cluster.util-status-overload .util-tree-cluster-head { border-left-color: #e64980; }
    .util-tree-cluster.util-status-underused .util-tree-cluster-head { border-left-color: #f59f00; }

    .util-tree-cluster-name {
        font-weight: 600;
        color: var(--bs-emphasis-color);
        font-size: 0.9rem;
    }

    .util-tree-vcenter {
        color: var(--bs-secondary-color);
        font-size: 0.75rem;
    }

    .util-tree-metrics {
        margin-left: auto;
        font-size: 0.75rem;
        color: var(--bs-secondary-color);
        white-space: nowrap;
    }

    .util-tree-metrics b { color: var(--bs-emphasis-color); }

    .util-tree-children {
        display: flex;
        flex-wrap: wrap;
        gap: 0.35rem;
        padding-left: 0.6rem;
    }

    .util-chip {
        font-size: 0.72rem;
        border: 1px solid var(--bs-border-color);
        border-left: 3px solid #4dabf7;
        border-radius: 0.25rem;
        padding: 0.2rem 0.55rem;
        background: var(--bs-secondary-bg);
        color: var(--bs-emphasis-color);
        white-space: nowrap;
    }

    .util-chip-overload { border-left-color: #e64980; color: #e64980; }
    .util-chip-underused { border-left-color: #f59f00; color: #f59f00; }

    /* --- Пустые состояния графиков --- */
    .util-empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        min-height: 200px;
        color: var(--bs-secondary-color);
        text-align: center;
        font-size: 0.85rem;
        padding: 1rem;
    }

    /* --- Таблица рисков --- */
    .util-risks-table thead th {
        color: var(--bs-secondary-color);
        font-weight: 600;
        font-size: 0.72rem;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        border-bottom-width: 1px;
    }

    /* Заливка всей ячейки цветом по severity - в духе таблицы "Detected
       problems" в Zabbix, где цвет строки/ячейки сразу считывается взглядом,
       без необходимости искать отдельный маленький бейдж */
    .util-risk-cell-danger { background: #c4162a; color: #fff; font-weight: 600; }
    .util-risk-cell-warning { background: #d9822b; color: #fff; font-weight: 600; }

    .util-gaps-list {
        padding-left: 1.1rem;
        color: var(--bs-secondary-color);
        font-size: 0.85rem;
    }

    .util-gaps-list li { margin-bottom: 0.35rem; }

    .apexcharts-toolbar { z-index: 5; }

    .apexcharts-menu {
        background: var(--bs-tertiary-bg) !important;
        border-color: var(--bs-border-color) !important;
        color: var(--bs-emphasis-color) !important;
    }

    .apexcharts-menu-item:hover { background: var(--bs-secondary-bg) !important; }
</style>

<script>
(function () {
    var data = {{.ChartsJSON}};
    var risks = {{.RisksJSON}};
    var dataGaps = {{.DataGapsJSON}};

    var c = (window.LoadmonApexTheme && LoadmonApexTheme.colors()) || { mode: 'light', foreColor: '#334155', grid: 'rgba(15,23,42,0.12)' };
    var apexTheme = c.mode;
    var gridColor = c.grid;
    var COLOR_DANGER = '#e64980', COLOR_WARNING = '#f59f00', COLOR_NORMAL = '#4dabf7', COLOR_OK = '#2fb344';

    function statusColor(status) {
        if (status === 'overload') return COLOR_DANGER;
        if (status === 'underused') return COLOR_WARNING;
        return COLOR_NORMAL;
    }

    function avg(list, field) {
        if (!list || !list.length) return 0;
        var sum = 0;
        list.forEach(function (x) { sum += (x[field] || 0); });
        return sum / list.length;
    }

    function emptyState(elId, text) {
        var el = document.getElementById(elId);
        if (!el) return;
        el.innerHTML = '<div class="util-empty-state"><div>' + text + '</div></div>';
    }

    // Singlestat-панель со сплошной цветной заливкой по порогу значения -
    // как "CPU Usage"/"RAM Usage" в примере grafana (зелёный/жёлтый/красный
    // фон целиком, а не тонкая полоска прогресса).
    function statClass(pct) {
        if (pct >= 85) return 'util-stat-crit';
        if (pct >= 65) return 'util-stat-watch';
        return 'util-stat-ok';
    }

    function renderSinglestat(containerEl, name, pct) {
        var div = document.createElement('div');
        div.className = 'util-stat ' + statClass(pct);
        div.innerHTML = '<div class="util-stat-label">' + name + '</div><div class="util-stat-value">' + pct.toFixed(1) + '%</div>';
        containerEl.appendChild(div);
    }

    var loadRow = document.getElementById('utilStatLoad');
    if (loadRow) {
        renderSinglestat(loadRow, 'Кластеры · CPU', avg(data.clusters, 'cpuPct'));
        renderSinglestat(loadRow, 'Кластеры · MEM', avg(data.clusters, 'memPct'));
        renderSinglestat(loadRow, 'Хосты · CPU', avg(data.hosts, 'cpuPct'));
        renderSinglestat(loadRow, 'Хосты · MEM', avg(data.hosts, 'memPct'));
        renderSinglestat(loadRow, 'VM · CPU', avg(data.vms, 'cpuPct'));
        renderSinglestat(loadRow, 'VM · MEM', avg(data.vms, 'memPct'));
        renderSinglestat(loadRow, 'СХД занято', avg(data.storages, 'diskPct'));
        renderSinglestat(loadRow, 'LUN занято', avg(data.luns, 'diskPct'));
    }

    // Счётчики рисков в верхней строке красим по факту наличия (>0 - тревожный
    // цвет, 0 - зелёный "всё ок"), как status-плитки в примерах grafana/zabbix.
    (function () {
        var overloadEl = document.getElementById('utilStatOverload');
        var underusedEl = document.getElementById('utilStatUnderused');
        var overloadCount = {{.KPI.OverloadedCount}};
        var underusedCount = {{.KPI.UnderusedCount}};
        if (overloadEl) overloadEl.classList.add(overloadCount > 0 ? 'util-stat-crit' : 'util-stat-ok');
        if (underusedEl) underusedEl.classList.add(underusedCount > 0 ? 'util-stat-watch' : 'util-stat-ok');
    })();

    // Heatmap-полоса заполненности СХД/LUN - вертикальные столбики с числом
    // сверху и именем снизу, цвет столбика по тому же порогу, что и
    // singlestat-панели (в духе grafana "Datastores - Usage Capacity").
    function renderHeatmap(elId, items, emptyText) {
        var el = document.getElementById(elId);
        if (!el) return;
        if (!items.length) { emptyState(elId, emptyText); return; }
        var html = '';
        items.forEach(function (x) {
            var pct = Math.max(0, Math.min(100, x.diskPct));
            var color = pct >= 85 ? '#c4162a' : (pct >= 65 ? '#d9822b' : '#299a3e');
            html += '<div class="util-heatmap-item">' +
                '<div class="util-heatmap-pct" style="color:' + color + '">' + x.diskPct.toFixed(1) + '%</div>' +
                '<div class="util-heatmap-bar"><div class="util-heatmap-fill" style="height:' + pct + '%;background:' + color + '"></div></div>' +
                '<div class="util-heatmap-name" title="' + x.name + '">' + x.name + '</div>' +
                '</div>';
        });
        el.innerHTML = html;
    }

    renderHeatmap('utilHeatmapStorage', (data.storages || []).slice().sort(function (a, b) { return b.diskPct - a.diskPct; }), 'Справочник СХД пуст');
    renderHeatmap('utilHeatmapLun', (data.luns || []).slice().sort(function (a, b) { return b.diskPct - a.diskPct; }).slice(0, 30), 'Справочник LUN пуст');

    function horizontalBar(elId, items, opts) {
        // items: [{name, value, color}]
        var el = document.getElementById(elId);
        if (!el) return;
        if (!items.length) { emptyState(elId, (opts && opts.emptyText) || 'Нет данных для отображения'); return; }
        var height = Math.max(220, items.length * 34 + 60);
        var barChart = new ApexCharts(el, {
            chart: { type: 'bar', height: height, background: 'transparent', foreColor: c.foreColor, toolbar: { show: false }, animations: { enabled: false } },
            theme: { mode: apexTheme },
            plotOptions: { bar: { horizontal: true, distributed: true, barHeight: '62%', borderRadius: 3 } },
            series: [{ name: (opts && opts.unit) || '%', data: items.map(function (x) { return x.value; }) }],
            legend: { show: false },
            dataLabels: {
                enabled: true,
                formatter: function (v) { return v.toFixed(1) + ((opts && opts.unit === '%') ? '%' : ''); },
                style: { colors: [c.foreColor] },
                offsetX: 6
            },
            xaxis: {
                categories: items.map(function (x) { return x.name; }),
                max: (opts && opts.unit === '%') ? 100 : undefined,
                labels: { style: { colors: c.muted || c.foreColor } }
            },
            yaxis: { labels: { style: { colors: c.foreColor } } },
            grid: { borderColor: gridColor },
            colors: items.map(function (x) { return x.color || COLOR_NORMAL; })
        });
        barChart.render();
        if (window.LoadmonApexTheme) LoadmonApexTheme.register(barChart);
    }

    // Кластеры и хосты по CPU avg %
    (function () {
        var items = (data.clusters || []).map(function (x) { return { name: 'Кластер · ' + x.name, value: x.cpuPct, color: statusColor(x.status) }; })
            .concat((data.hosts || []).map(function (x) { return { name: 'Хост · ' + x.name, value: x.cpuPct, color: statusColor(x.status) }; }));
        items.sort(function (a, b) { return b.value - a.value; });
        horizontalBar('utilBarClustersHosts', items.slice(0, 15), { unit: '%', emptyText: 'Нет суточных данных по кластерам/хостам за последние 7 дней' });
    })();

    // Перегруженные VM
    (function () {
        var items = (data.vms || []).filter(function (x) { return x.status === 'overload'; })
            .sort(function (a, b) { return b.cpuPct - a.cpuPct; }).slice(0, 15)
            .map(function (x) { return { name: x.name, value: x.cpuPct, color: COLOR_DANGER }; });
        horizontalBar('utilBarVmTop', items, { unit: '%', emptyText: 'Перегруженных VM (CPU avg ≥ 85%) не обнаружено' });
    })();

    // Простаивающие VM
    (function () {
        var items = (data.vms || []).filter(function (x) { return x.status === 'underused'; })
            .sort(function (a, b) { return a.cpuPct - b.cpuPct; }).slice(0, 15)
            .map(function (x) { return { name: x.name, value: x.cpuPct, color: COLOR_WARNING }; });
        horizontalBar('utilBarVmIdle', items, { unit: '%', emptyText: 'VM с признаками простоя (CPU max < 5% при ≥2 vCPU или ≥4 ГБ RAM) не обнаружено' });
    })();

    // Структура оборудования: карточка на кластер + чипы хостов
    (function () {
        var el = document.getElementById('utilTree');
        if (!el) return;
        var clusters = (data.clusters || []).slice();
        if (!clusters.length) {
            emptyState('utilTree', 'Нет суточных данных ClusterRecord за последние 7 дней - структуру построить не из чего');
            return;
        }

        var hostsByCluster = {};
        (data.hosts || []).forEach(function (h) {
            var key = h.parent || '(без кластера)';
            (hostsByCluster[key] = hostsByCluster[key] || []).push(h);
        });
        var vmCountByCluster = {};
        (data.vms || []).forEach(function (v) {
            var key = v.parent || '(без кластера)';
            vmCountByCluster[key] = (vmCountByCluster[key] || 0) + 1;
        });

        function statusClass(status, prefix) {
            if (status === 'overload') return prefix + '-overload';
            if (status === 'underused') return prefix + '-underused';
            return '';
        }

        var html = '';
        clusters.sort(function (a, b) { return b.cpuPct - a.cpuPct; }).forEach(function (c) {
            var hosts = (hostsByCluster[c.name] || []).slice().sort(function (a, b) { return b.cpuPct - a.cpuPct; });
            var vmCount = vmCountByCluster[c.name] || 0;

            html += '<div class="util-tree-cluster ' + statusClass(c.status, 'util-status') + '">';
            html += '  <div class="util-tree-cluster-head">';
            html += '    <span class="util-tree-cluster-name">' + c.name + '</span>';
            html += '    <span class="util-tree-vcenter">' + (c.parent || '') + '</span>';
            html += '    <span class="util-tree-metrics">CPU <b>' + c.cpuPct.toFixed(0) + '%</b> · MEM <b>' + c.memPct.toFixed(0) + '%</b> · хостов <b>' + hosts.length + '</b> · VM <b>' + vmCount + '</b></span>';
            html += '  </div>';
            html += '  <div class="util-tree-children">';
            if (!hosts.length) {
                html += '<span class="util-chip">нет данных по хостам</span>';
            } else {
                hosts.forEach(function (h) {
                    html += '<span class="util-chip ' + statusClass(h.status, 'util-chip') + '">' + h.name + ' — ' + h.cpuPct.toFixed(0) + '% / ' + h.memPct.toFixed(0) + '%</span>';
                });
            }
            html += '  </div>';
            html += '</div>';
        });
        el.innerHTML = html;
    })();

    // Таблица рисков/рекомендаций
    (function () {
        var tbody = document.querySelector('#utilRisksTable tbody');
        var empty = document.getElementById('utilRisksEmpty');
        if (!tbody) return;
        if (!risks || !risks.length) {
            if (empty) empty.classList.remove('d-none');
            document.getElementById('utilRisksTable').classList.add('d-none');
            return;
        }
        risks.forEach(function (r) {
            var tr = document.createElement('tr');
            var cellClass = r.severity === 'danger' ? 'util-risk-cell-danger' : 'util-risk-cell-warning';
            var sevLabel = r.severity === 'danger' ? 'Перегруз' : 'Недоиспользование';
            tr.innerHTML = '<td>' + r.category + '</td>' +
                '<td>' + r.object + '</td>' +
                '<td>' + r.metric + '</td>' +
                '<td>' + r.value.toFixed(1) + '</td>' +
                '<td class="' + cellClass + '">' + sevLabel + '</td>' +
                '<td class="text-muted small">' + r.recommendation + '</td>';
            tbody.appendChild(tr);
        });
    })();

    // Пробелы в данных
    (function () {
        var list = document.getElementById('utilDataGapsList');
        if (!list) return;
        (dataGaps || []).forEach(function (g) {
            var li = document.createElement('li');
            li.textContent = g;
            list.appendChild(li);
        });
    })();
})();
</script>
