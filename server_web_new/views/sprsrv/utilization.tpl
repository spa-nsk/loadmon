<!-- Контроллер: controllers/sprutilization.go (SprUtilization). Оформление -
     собственный набор util-* стилей (карточки/дерево/бейджи), графики -
     ApexCharts по общему паттерну /sprsrv/zabbixperfperiod (тёмная тема,
     подключение apexcharts.min.js). -->

<div class="hero-text mb-3">
    <h2>Утилизация виртуальной инфраструктуры</h2>
    <div class="text-muted">Сводная оценка загрузки кластеров, хостов, виртуальных машин, СХД и LUN по последним суточным данным. Риски перегруза и неиспользуемые ресурсы - в таблице рекомендаций внизу страницы.</div>
</div>

<div class="util-kpi-row mb-4">
    <div class="util-kpi-card">
        <div class="util-kpi-icon util-kpi-icon-cluster">⬡</div>
        <div><div class="util-kpi-value">{{.KPI.ClusterCount}}</div><div class="util-kpi-label">Кластеров</div></div>
    </div>
    <div class="util-kpi-card">
        <div class="util-kpi-icon util-kpi-icon-host">▤</div>
        <div><div class="util-kpi-value">{{.KPI.HostCount}}</div><div class="util-kpi-label">Хостов</div></div>
    </div>
    <div class="util-kpi-card">
        <div class="util-kpi-icon util-kpi-icon-vm">▣</div>
        <div><div class="util-kpi-value">{{.KPI.VMCount}}</div><div class="util-kpi-label">Виртуальных машин</div></div>
    </div>
    <div class="util-kpi-card">
        <div class="util-kpi-icon util-kpi-icon-storage">▦</div>
        <div><div class="util-kpi-value">{{.KPI.StorageSystemCount}}</div><div class="util-kpi-label">СХД</div></div>
    </div>
    <div class="util-kpi-card">
        <div class="util-kpi-icon util-kpi-icon-lun">▥</div>
        <div><div class="util-kpi-value">{{.KPI.LUNCount}}</div><div class="util-kpi-label">LUN</div></div>
    </div>
    <div class="util-kpi-card util-kpi-card-danger">
        <div class="util-kpi-icon util-kpi-icon-danger">⚠</div>
        <div><div class="util-kpi-value">{{.KPI.OverloadedCount}}</div><div class="util-kpi-label">Риски перегруза</div></div>
    </div>
    <div class="util-kpi-card util-kpi-card-warning">
        <div class="util-kpi-icon util-kpi-icon-warning">↓</div>
        <div><div class="util-kpi-value">{{.KPI.UnderusedCount}}</div><div class="util-kpi-label">Недоиспользуется</div></div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-4">
        <div class="util-card">
            <h6>Средняя загрузка CPU</h6>
            <div class="util-donut-row">
                <div class="util-donut-item"><div id="utilDonutClusterCpu"></div><div class="util-donut-caption">Кластеры</div></div>
                <div class="util-donut-item"><div id="utilDonutHostCpu"></div><div class="util-donut-caption">Хосты</div></div>
                <div class="util-donut-item"><div id="utilDonutVmCpu"></div><div class="util-donut-caption">VM</div></div>
            </div>
        </div>
    </div>
    <div class="col-12 col-xl-4">
        <div class="util-card">
            <h6>Средняя загрузка памяти</h6>
            <div class="util-donut-row">
                <div class="util-donut-item"><div id="utilDonutClusterMem"></div><div class="util-donut-caption">Кластеры</div></div>
                <div class="util-donut-item"><div id="utilDonutHostMem"></div><div class="util-donut-caption">Хосты</div></div>
                <div class="util-donut-item"><div id="utilDonutVmMem"></div><div class="util-donut-caption">VM</div></div>
            </div>
        </div>
    </div>
    <div class="col-12 col-xl-4">
        <div class="util-card">
            <h6>Хранилище</h6>
            <div class="util-donut-row">
                <div class="util-donut-item"><div id="utilDonutStorage"></div><div class="util-donut-caption">СХД занято</div></div>
                <div class="util-donut-item util-donut-item-text">
                    <div class="util-kpi-value">{{printf "%.1f" .KPI.TotalStorageRawTB}}&nbsp;ТБ</div>
                    <div class="util-donut-caption">Суммарная ёмкость СХД (raw)</div>
                </div>
            </div>
        </div>
    </div>
</div>

<div class="util-card mb-3">
    <h6>Структура оборудования: vCenter → кластер → хост / VM</h6>
    <div class="text-muted small mb-2">Цвет рамки узла отражает статус: <span class="util-legend-dot util-legend-danger"></span>перегруз, <span class="util-legend-dot util-legend-warning"></span>недоиспользование, <span class="util-legend-dot util-legend-normal"></span>норма.</div>
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
            <h6>Перегруженные VM (CPU avg ≥ {{printf "%.0f" 85.0}}%)</h6>
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
    <div class="table-responsive">
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
        <div class="util-empty-icon">✓</div>
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
    .util-kpi-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
        gap: 0.75rem;
    }

    .util-kpi-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.75rem;
        padding: 0.9rem 1rem;
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }

    .util-kpi-card-danger { border-color: rgba(230, 73, 128, 0.45); }
    .util-kpi-card-warning { border-color: rgba(245, 159, 0, 0.45); }

    .util-kpi-icon {
        width: 2.4rem;
        height: 2.4rem;
        border-radius: 0.6rem;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1.2rem;
        flex-shrink: 0;
        background: var(--bs-secondary-bg);
        color: var(--bs-emphasis-color);
    }

    .util-kpi-icon-danger { background: rgba(230, 73, 128, 0.15); color: #e64980; }
    .util-kpi-icon-warning { background: rgba(245, 159, 0, 0.15); color: #f59f00; }

    .util-kpi-value {
        font-size: 1.5rem;
        font-weight: 700;
        line-height: 1.15;
    }

    .util-kpi-label {
        color: var(--bs-secondary-color);
        font-size: 0.78rem;
    }

    .util-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.75rem;
        padding: 1.1rem 1.2rem;
        height: 100%;
    }

    .util-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        margin-bottom: 0.85rem;
    }

    .util-chart-box { min-height: 320px; }

    .util-donut-row {
        display: flex;
        justify-content: space-around;
        text-align: center;
        flex-wrap: wrap;
    }

    .util-donut-item { flex: 1 1 0; min-width: 90px; }

    .util-donut-item-text {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
    }

    .util-donut-caption {
        color: var(--bs-secondary-color);
        font-size: 0.78rem;
        margin-top: -0.25rem;
    }

    .util-legend-dot {
        display: inline-block;
        width: 0.6rem;
        height: 0.6rem;
        border-radius: 50%;
        margin: 0 0.3rem 0 0.6rem;
    }

    .util-legend-danger { background: #e64980; }
    .util-legend-warning { background: #f59f00; }
    .util-legend-normal { background: #4dabf7; }

    /* --- Дерево структуры оборудования --- */
    .util-tree {
        display: flex;
        flex-direction: column;
        gap: 0.9rem;
        max-height: 520px;
        overflow-y: auto;
        padding-right: 0.25rem;
    }

    .util-tree-cluster {
        border: 1px solid var(--bs-border-color);
        border-left: 4px solid #4dabf7;
        border-radius: 0.6rem;
        padding: 0.7rem 0.9rem;
        background: var(--bs-secondary-bg);
    }

    .util-tree-cluster.util-status-overload { border-left-color: #e64980; }
    .util-tree-cluster.util-status-underused { border-left-color: #f59f00; }

    .util-tree-cluster-head {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 0.5rem;
        margin-bottom: 0.5rem;
    }

    .util-tree-cluster-name {
        font-weight: 600;
        color: var(--bs-emphasis-color);
    }

    .util-tree-vcenter {
        color: var(--bs-secondary-color);
        font-size: 0.78rem;
    }

    .util-tree-metrics {
        margin-left: auto;
        font-size: 0.78rem;
        color: var(--bs-secondary-color);
        white-space: nowrap;
    }

    .util-tree-metrics b { color: var(--bs-emphasis-color); }

    .util-tree-children {
        display: flex;
        flex-wrap: wrap;
        gap: 0.4rem;
    }

    .util-chip {
        font-size: 0.74rem;
        border: 1px solid var(--bs-border-color);
        border-radius: 1rem;
        padding: 0.2rem 0.6rem;
        background: var(--bs-tertiary-bg);
        color: var(--bs-emphasis-color);
        white-space: nowrap;
    }

    .util-chip-host { border-left: 3px solid #4dabf7; }
    .util-chip-overload { border-color: #e64980; color: #e64980; }
    .util-chip-underused { border-color: #f59f00; color: #f59f00; }

    .util-chip-vm-count {
        font-size: 0.74rem;
        color: var(--bs-secondary-color);
        align-self: center;
    }

    /* --- Пустые состояния графиков --- */
    .util-empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        min-height: 220px;
        color: var(--bs-secondary-color);
        text-align: center;
        font-size: 0.85rem;
        padding: 1rem;
    }

    .util-empty-icon {
        width: 2.2rem;
        height: 2.2rem;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bs-secondary-bg);
        font-size: 1.1rem;
    }

    /* --- Таблица рисков --- */
    .util-risks-table thead th {
        color: var(--bs-secondary-color);
        font-weight: 600;
        font-size: 0.78rem;
        text-transform: uppercase;
        letter-spacing: 0.02em;
        border-bottom-width: 1px;
    }

    .util-risk-badge {
        display: inline-block;
        padding: 0.15rem 0.55rem;
        border-radius: 1rem;
        font-size: 0.75rem;
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

    function avg(list, field) {
        if (!list || !list.length) return 0;
        var sum = 0;
        list.forEach(function (x) { sum += (x[field] || 0); });
        return sum / list.length;
    }

    function emptyState(elId, text) {
        var el = document.getElementById(elId);
        if (!el) return;
        el.innerHTML = '<div class="util-empty-state"><div class="util-empty-icon">–</div><div>' + text + '</div></div>';
    }

    function donut(elId, pct, captionElSelector) {
        var el = document.getElementById(elId);
        if (!el) return;
        var color = pct >= 80 ? COLOR_DANGER : (pct <= 15 ? COLOR_WARNING : COLOR_OK);
        new ApexCharts(el, {
            chart: { type: 'radialBar', height: 130, background: 'transparent' },
            theme: { mode: apexTheme },
            series: [Math.round(pct * 10) / 10],
            colors: [color],
            plotOptions: { radialBar: { hollow: { size: '55%' }, dataLabels: { value: { fontSize: '0.95rem', formatter: function (v) { return v + '%'; } } } } },
            labels: ['']
        }).render();
    }

    donut('utilDonutClusterCpu', avg(data.clusters, 'cpuPct'));
    donut('utilDonutHostCpu', avg(data.hosts, 'cpuPct'));
    donut('utilDonutVmCpu', avg(data.vms, 'cpuPct'));
    donut('utilDonutClusterMem', avg(data.clusters, 'memPct'));
    donut('utilDonutHostMem', avg(data.hosts, 'memPct'));
    donut('utilDonutVmMem', avg(data.vms, 'memPct'));
    donut('utilDonutStorage', avg(data.storages, 'diskPct'));

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
        horizontalBar('utilBarVmTop', items, { unit: '%', emptyText: 'Перегруженных VM (CPU avg ≥ 85%) не обнаружено - хороший знак' });
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
        var clusters = data.clusters || [];
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
            var hosts = (hostsByCluster[c.name] || []).sort(function (a, b) { return b.cpuPct - a.cpuPct; });
            var vmCount = vmCountByCluster[c.name] || 0;

            html += '<div class="util-tree-cluster ' + statusClass(c.status, 'util-status') + '">';
            html += '  <div class="util-tree-cluster-head">';
            html += '    <span class="util-tree-cluster-name">⬡ ' + c.name + '</span>';
            html += '    <span class="util-tree-vcenter">' + (c.parent || '') + '</span>';
            html += '    <span class="util-tree-metrics">CPU <b>' + c.cpuPct.toFixed(0) + '%</b> · MEM <b>' + c.memPct.toFixed(0) + '%</b> · хостов <b>' + hosts.length + '</b> · VM <b>' + vmCount + '</b></span>';
            html += '  </div>';
            html += '  <div class="util-tree-children">';
            if (!hosts.length) {
                html += '<span class="util-chip">нет данных по хостам</span>';
            } else {
                hosts.forEach(function (h) {
                    html += '<span class="util-chip util-chip-host ' + statusClass(h.status, 'util-chip') + '">▤ ' + h.name + ' · ' + h.cpuPct.toFixed(0) + '%/' + h.memPct.toFixed(0) + '%</span>';
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
