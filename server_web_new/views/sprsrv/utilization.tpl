<!-- Контроллер: controllers/sprutilization.go (SprUtilization). Оформление -
     "grafana-style" stat-панели и bar gauge на чистом CSS (без ApexCharts
     radialBar - он даёт наложение подписей на маленьких донатах), плюс
     ApexCharts только для горизонтальных bar-графиков, где он ведёт себя
     предсказуемо. Общий паттерн подключения ApexCharts/тёмной темы - как в
     /sprsrv/zabbixperfperiod. -->

<div class="hero-text mb-3">
    <h2>Утилизация виртуальной инфраструктуры</h2>
    <div class="text-muted">Сводная оценка загрузки кластеров, хостов, виртуальных машин, СХД и LUN по последним суточным данным. Риски перегруза и неиспользуемые ресурсы - в таблице рекомендаций внизу страницы.</div>
</div>

<div class="util-stat-row mb-3">
    <div class="util-stat">
        <div class="util-stat-label">Кластеров</div>
        <div class="util-stat-value">{{.KPI.ClusterCount}}</div>
    </div>
    <div class="util-stat">
        <div class="util-stat-label">Хостов</div>
        <div class="util-stat-value">{{.KPI.HostCount}}</div>
    </div>
    <div class="util-stat">
        <div class="util-stat-label">Виртуальных машин</div>
        <div class="util-stat-value">{{.KPI.VMCount}}</div>
    </div>
    <div class="util-stat">
        <div class="util-stat-label">СХД</div>
        <div class="util-stat-value">{{.KPI.StorageSystemCount}}</div>
    </div>
    <div class="util-stat">
        <div class="util-stat-label">LUN</div>
        <div class="util-stat-value">{{.KPI.LUNCount}}</div>
    </div>
    <div class="util-stat util-stat-accent-danger">
        <div class="util-stat-label">Риски перегруза</div>
        <div class="util-stat-value">{{.KPI.OverloadedCount}}</div>
    </div>
    <div class="util-stat util-stat-accent-warning">
        <div class="util-stat-label">Недоиспользуется</div>
        <div class="util-stat-value">{{.KPI.UnderusedCount}}</div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-8">
        <div class="util-card">
            <h6>Средняя загрузка, %</h6>
            <div class="util-gauge-grid" id="utilGaugeGrid"></div>
        </div>
    </div>
    <div class="col-12 col-xl-4">
        <div class="util-card">
            <h6>Хранилище</h6>
            <div class="util-gauge-grid util-gauge-grid-single" id="utilGaugeStorage"></div>
            <div class="util-storage-total">
                <div class="util-stat-value">{{printf "%.1f" .KPI.TotalStorageRawTB}}&nbsp;ТБ</div>
                <div class="util-stat-label">Суммарная ёмкость СХД (raw)</div>
            </div>
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
            <h6>Заполненность СХД и LUN, %</h6>
            <div id="utilBarStorage" class="util-chart-box"></div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="util-card">
            <h6>Перегруженные VM (CPU avg ≥ 85%)</h6>
            <div id="utilBarVmTop" class="util-chart-box"></div>
        </div>
    </div>
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

    /* --- Bar gauge (полоска с подписью значения сверху, без наложения текста) --- */
    .util-gauge-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
        gap: 1rem 1.5rem;
    }

    .util-gauge-grid-single { grid-template-columns: 1fr; }

    .util-gauge {
        display: flex;
        flex-direction: column;
        gap: 0.3rem;
    }

    .util-gauge-head {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
        font-size: 0.8rem;
    }

    .util-gauge-name { color: var(--bs-secondary-color); }

    .util-gauge-value {
        font-weight: 700;
        color: var(--bs-emphasis-color);
        font-variant-numeric: tabular-nums;
    }

    .util-gauge-track {
        height: 0.55rem;
        border-radius: 0.3rem;
        background: var(--bs-secondary-bg);
        overflow: hidden;
    }

    .util-gauge-fill {
        height: 100%;
        border-radius: 0.3rem;
        transition: width 0.3s ease;
    }

    .util-storage-total {
        text-align: center;
        margin-top: 1.25rem;
        padding-top: 1rem;
        border-top: 1px solid var(--bs-border-color);
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

    .util-risk-badge {
        display: inline-block;
        padding: 0.15rem 0.55rem;
        border-radius: 0.25rem;
        font-size: 0.72rem;
        font-weight: 600;
    }

    .util-risk-badge-danger { background: rgba(230, 73, 128, 0.15); color: #e64980; }
    .util-risk-badge-warning { background: rgba(245, 159, 0, 0.15); color: #f59f00; }

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

    var isDark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
    var apexTheme = isDark ? 'dark' : 'light';
    var gridColor = isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)';
    var COLOR_DANGER = '#e64980', COLOR_WARNING = '#f59f00', COLOR_NORMAL = '#4dabf7', COLOR_OK = '#2fb344';

    function statusColor(status) {
        if (status === 'overload') return COLOR_DANGER;
        if (status === 'underused') return COLOR_WARNING;
        return COLOR_NORMAL;
    }

    function gaugeColor(pct) {
        if (pct >= 80) return COLOR_DANGER;
        if (pct <= 15) return COLOR_WARNING;
        return COLOR_OK;
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

    // Простая CSS bar-gauge вместо ApexCharts radialBar - не даёт наложения
    // подписи значения на график на маленьких размерах, читается однозначно.
    function renderGauge(containerEl, name, pct) {
        var clamped = Math.max(0, Math.min(100, pct));
        var wrap = document.createElement('div');
        wrap.className = 'util-gauge';
        wrap.innerHTML =
            '<div class="util-gauge-head"><span class="util-gauge-name">' + name + '</span><span class="util-gauge-value">' + pct.toFixed(1) + '%</span></div>' +
            '<div class="util-gauge-track"><div class="util-gauge-fill" style="width:' + clamped + '%;background:' + gaugeColor(pct) + '"></div></div>';
        containerEl.appendChild(wrap);
    }

    var loadGrid = document.getElementById('utilGaugeGrid');
    if (loadGrid) {
        renderGauge(loadGrid, 'Кластеры · CPU', avg(data.clusters, 'cpuPct'));
        renderGauge(loadGrid, 'Кластеры · MEM', avg(data.clusters, 'memPct'));
        renderGauge(loadGrid, 'Хосты · CPU', avg(data.hosts, 'cpuPct'));
        renderGauge(loadGrid, 'Хосты · MEM', avg(data.hosts, 'memPct'));
        renderGauge(loadGrid, 'VM · CPU', avg(data.vms, 'cpuPct'));
        renderGauge(loadGrid, 'VM · MEM', avg(data.vms, 'memPct'));
    }

    var storageGrid = document.getElementById('utilGaugeStorage');
    if (storageGrid) {
        renderGauge(storageGrid, 'СХД занято', avg(data.storages, 'diskPct'));
        renderGauge(storageGrid, 'LUN занято', avg(data.luns, 'diskPct'));
    }

    function horizontalBar(elId, items, opts) {
        // items: [{name, value, color}]
        var el = document.getElementById(elId);
        if (!el) return;
        if (!items.length) { emptyState(elId, (opts && opts.emptyText) || 'Нет данных для отображения'); return; }
        var height = Math.max(220, items.length * 34 + 60);
        new ApexCharts(el, {
            chart: { type: 'bar', height: height, background: 'transparent', toolbar: { show: false }, animations: { enabled: false } },
            theme: { mode: apexTheme },
            plotOptions: { bar: { horizontal: true, distributed: true, barHeight: '62%', borderRadius: 3 } },
            series: [{ name: (opts && opts.unit) || '%', data: items.map(function (x) { return x.value; }) }],
            legend: { show: false },
            dataLabels: {
                enabled: true,
                formatter: function (v) { return v.toFixed(1) + ((opts && opts.unit === '%') ? '%' : ''); },
                style: { colors: ['var(--bs-emphasis-color)'] },
                offsetX: 6
            },
            xaxis: {
                categories: items.map(function (x) { return x.name; }),
                max: (opts && opts.unit === '%') ? 100 : undefined,
                labels: { style: { colors: 'var(--bs-secondary-color)' } }
            },
            yaxis: { labels: { style: { colors: 'var(--bs-emphasis-color)' } } },
            grid: { borderColor: gridColor },
            colors: items.map(function (x) { return x.color || COLOR_NORMAL; })
        }).render();
    }

    // Кластеры и хосты по CPU avg %
    (function () {
        var items = (data.clusters || []).map(function (x) { return { name: 'Кластер · ' + x.name, value: x.cpuPct, color: statusColor(x.status) }; })
            .concat((data.hosts || []).map(function (x) { return { name: 'Хост · ' + x.name, value: x.cpuPct, color: statusColor(x.status) }; }));
        items.sort(function (a, b) { return b.value - a.value; });
        horizontalBar('utilBarClustersHosts', items.slice(0, 15), { unit: '%', emptyText: 'Нет суточных данных по кластерам/хостам за последние 7 дней' });
    })();

    // СХД и LUN по заполненности
    (function () {
        var items = (data.storages || []).map(function (x) { return { name: 'СХД · ' + x.name, value: x.diskPct, color: statusColor(x.status) }; })
            .concat((data.luns || []).map(function (x) { return { name: 'LUN · ' + x.name, value: x.diskPct, color: statusColor(x.status) }; }));
        items.sort(function (a, b) { return b.value - a.value; });
        horizontalBar('utilBarStorage', items.slice(0, 15), { unit: '%', emptyText: 'Справочники СХД/LUN пусты' });
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
            var badgeClass = r.severity === 'danger' ? 'util-risk-badge-danger' : 'util-risk-badge-warning';
            var sevLabel = r.severity === 'danger' ? 'Перегруз' : 'Недоиспользование';
            tr.innerHTML = '<td>' + r.category + '</td>' +
                '<td>' + r.object + '</td>' +
                '<td>' + r.metric + '</td>' +
                '<td>' + r.value.toFixed(1) + '</td>' +
                '<td><span class="util-risk-badge ' + badgeClass + '">' + sevLabel + '</span></td>' +
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
