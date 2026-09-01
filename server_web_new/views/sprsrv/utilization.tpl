<!-- Стили карточек/графиков переиспользуют паттерн /sprsrv/zabbixperfperiod
     (vm-perf-card, apexcharts-* тёмная тема) - см. views/sprsrv/zabbixperfperiod.tpl.
     Контроллер: controllers/sprutilization.go (SprUtilization). -->

<div class="hero-text mb-3">
    <h2>Утилизация виртуальной инфраструктуры</h2>
    <div class="text-muted">Сводная оценка загрузки кластеров, хостов, виртуальных машин, СХД и LUN по последним суточным данным. Выявленные риски перегруза и неиспользуемые ресурсы - в разделе рекомендаций ниже.</div>
</div>

<div class="row g-3 mb-3" id="utilKpiRow">
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card">
            <div class="util-kpi-value">{{.KPI.ClusterCount}}</div>
            <div class="util-kpi-label">Кластеров</div>
        </div>
    </div>
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card">
            <div class="util-kpi-value">{{.KPI.HostCount}}</div>
            <div class="util-kpi-label">Хостов</div>
        </div>
    </div>
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card">
            <div class="util-kpi-value">{{.KPI.VMCount}}</div>
            <div class="util-kpi-label">Виртуальных машин</div>
        </div>
    </div>
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card">
            <div class="util-kpi-value">{{.KPI.StorageSystemCount}}</div>
            <div class="util-kpi-label">СХД</div>
        </div>
    </div>
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card">
            <div class="util-kpi-value">{{.KPI.LUNCount}}</div>
            <div class="util-kpi-label">LUN</div>
        </div>
    </div>
    <div class="col-6 col-md-3 col-xl-2">
        <div class="util-kpi-card util-kpi-card-risk">
            <div class="util-kpi-value text-danger">{{.KPI.OverloadedCount}}</div>
            <div class="util-kpi-label">Риски перегруза</div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-4">
        <div class="vm-perf-card">
            <h6>Средняя загрузка CPU/Memory</h6>
            <div id="utilAvgDonuts" class="row text-center g-2">
                <div class="col-4"><div id="utilDonutClusterCpu"></div><div class="small text-muted">Кластеры CPU</div></div>
                <div class="col-4"><div id="utilDonutHostCpu"></div><div class="small text-muted">Хосты CPU</div></div>
                <div class="col-4"><div id="utilDonutVmCpu"></div><div class="small text-muted">VM CPU</div></div>
            </div>
        </div>
    </div>
    <div class="col-12 col-xl-4">
        <div class="vm-perf-card">
            <h6>Средняя загрузка памяти</h6>
            <div class="row text-center g-2">
                <div class="col-4"><div id="utilDonutClusterMem"></div><div class="small text-muted">Кластеры MEM</div></div>
                <div class="col-4"><div id="utilDonutHostMem"></div><div class="small text-muted">Хосты MEM</div></div>
                <div class="col-4"><div id="utilDonutVmMem"></div><div class="small text-muted">VM MEM</div></div>
            </div>
        </div>
    </div>
    <div class="col-12 col-xl-4">
        <div class="vm-perf-card">
            <h6>Хранилище</h6>
            <div class="row text-center g-2">
                <div class="col-6"><div id="utilDonutStorage"></div><div class="small text-muted">СХД занято</div></div>
                <div class="col-6">
                    <div class="util-kpi-value">{{printf "%.1f" .KPI.TotalStorageRawTB}} ТБ</div>
                    <div class="small text-muted">Суммарная ёмкость СХД (raw)</div>
                </div>
            </div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Структура: vCenter → кластер → хост → VM</h6>
            <div id="utilTreeChart" style="min-height:360px;"></div>
        </div>
    </div>
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Топ загруженных/недоиспользуемых кластеров и хостов (CPU avg %)</h6>
            <div id="utilBarClustersHosts" style="min-height:360px;"></div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Топ загруженных VM (CPU avg %)</h6>
            <div id="utilBarVmTop"></div>
        </div>
    </div>
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Кандидаты на right-sizing / отключение (простаивающие VM)</h6>
            <div id="utilBarVmIdle"></div>
        </div>
    </div>
</div>

<div class="row g-3 mb-3">
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Заполненность СХД, %</h6>
            <div id="utilBarStorage"></div>
        </div>
    </div>
    <div class="col-12 col-xl-6">
        <div class="vm-perf-card">
            <h6>Заполненность LUN, % (топ-15)</h6>
            <div id="utilBarLun"></div>
        </div>
    </div>
</div>

<div class="vm-perf-card mb-3">
    <h6>Риски и рекомендации по оптимизации</h6>
    <div class="table-responsive">
        <table class="table table-sm align-middle mb-0" id="utilRisksTable">
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
    <div id="utilRisksEmpty" class="text-muted small d-none">Рисков перегруза или существенного недоиспользования по текущим данным не выявлено.</div>
</div>

<div class="vm-perf-card mb-3">
    <h6>Данных недостаточно? Предложения по дополнительным методам сбора</h6>
    <div class="text-muted small mb-2">Оценка построена на данных, которые уже собирает loadmon (суточные агрегаты ClusterRecord/HostRecord/VMRecord, справочники StorageSystems/StorageVolumes). Ниже - выявленные пробелы, для устранения которых потребуются доработки сбора метрик.</div>
    <ul id="utilDataGapsList" class="small mb-0"></ul>
</div>

<script src="/static/js/apexcharts.min.js?v={{.StaticVer}}"></script>

<style>
    .util-kpi-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 0.9rem 0.5rem;
        text-align: center;
        height: 100%;
    }

    .util-kpi-card-risk {
        border-color: var(--bs-danger-border-subtle, var(--bs-border-color));
    }

    .util-kpi-value {
        font-size: 1.6rem;
        font-weight: 700;
        line-height: 1.2;
    }

    .util-kpi-label {
        color: var(--bs-secondary-color);
        font-size: 0.8rem;
    }

    .vm-perf-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 1rem;
        height: 100%;
    }

    .vm-perf-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        margin-bottom: 0.75rem;
    }

    .apexcharts-toolbar { z-index: 5; }

    .apexcharts-menu {
        background: var(--bs-tertiary-bg) !important;
        border-color: var(--bs-border-color) !important;
        color: var(--bs-emphasis-color) !important;
    }

    .apexcharts-menu-item:hover { background: var(--bs-secondary-bg) !important; }

    .util-risk-danger { color: var(--bs-danger); font-weight: 600; }
    .util-risk-warning { color: var(--bs-warning); font-weight: 600; }

    .util-tree-node {
        font-size: 0.78rem;
        border: 1px solid var(--bs-border-color);
        border-radius: 0.4rem;
        padding: 0.15rem 0.4rem;
        margin: 0.15rem;
        display: inline-block;
    }

    .util-tree-overload { border-color: var(--bs-danger); color: var(--bs-danger); }
    .util-tree-underused { border-color: var(--bs-warning); color: var(--bs-warning); }
</style>

<script>
(function () {
    var data = {{.ChartsJSON}};
    var risks = {{.RisksJSON}};
    var dataGaps = {{.DataGapsJSON}};

    var isDark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
    var apexTheme = isDark ? 'dark' : 'light';
    var gridColor = isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)';

    function statusColor(status) {
        if (status === 'overload') return '#e64980';
        if (status === 'underused') return '#f59f00';
        return '#4dabf7';
    }

    function avg(list, field) {
        if (!list || !list.length) return 0;
        var sum = 0;
        list.forEach(function (x) { sum += (x[field] || 0); });
        return sum / list.length;
    }

    function donut(elId, pct, label) {
        var el = document.getElementById(elId);
        if (!el) return;
        var color = pct >= 80 ? '#e64980' : (pct <= 15 ? '#f59f00' : '#2fb344');
        new ApexCharts(el, {
            chart: { type: 'radialBar', height: 140, background: 'transparent' },
            theme: { mode: apexTheme },
            series: [Math.round(pct * 10) / 10],
            colors: [color],
            plotOptions: { radialBar: { hollow: { size: '55%' }, dataLabels: { value: { formatter: function (v) { return v + '%'; } } } } },
            labels: [label || '']
        }).render();
    }

    donut('utilDonutClusterCpu', avg(data.clusters, 'cpuPct'));
    donut('utilDonutHostCpu', avg(data.hosts, 'cpuPct'));
    donut('utilDonutVmCpu', avg(data.vms, 'cpuPct'));
    donut('utilDonutClusterMem', avg(data.clusters, 'memPct'));
    donut('utilDonutHostMem', avg(data.hosts, 'memPct'));
    donut('utilDonutVmMem', avg(data.vms, 'memPct'));
    donut('utilDonutStorage', avg(data.storages, 'diskPct'));

    function barChart(elId, categories, series, unit, height) {
        var el = document.getElementById(elId);
        if (!el) return;
        if (!categories.length) { el.innerHTML = '<div class="text-muted small">Нет данных</div>'; return; }
        new ApexCharts(el, {
            chart: { type: 'bar', height: height || 340, background: 'transparent', toolbar: { show: false }, animations: { enabled: false } },
            theme: { mode: apexTheme },
            plotOptions: { bar: { horizontal: true, distributed: true, barHeight: '65%' } },
            series: [{ name: unit || '%', data: series }],
            legend: { show: false },
            dataLabels: { enabled: true, formatter: function (v) { return v.toFixed(1); }, style: { colors: ['var(--bs-emphasis-color)'] } },
            xaxis: { categories: categories, max: unit === '%' ? 100 : undefined, labels: { style: { colors: 'var(--bs-emphasis-color)' } } },
            yaxis: { labels: { style: { colors: 'var(--bs-emphasis-color)' } } },
            grid: { borderColor: gridColor },
            colors: series.map(function () { return '#4dabf7'; })
        }).render();
    }

    // Топ кластеров и хостов по CPU avg (перегруженные сверху, недоиспользуемые снизу)
    (function () {
        var items = (data.clusters || []).map(function (x) { return { name: 'Кластер: ' + x.name, v: x.cpuPct, status: x.status, color: statusColor(x.status) }; })
            .concat((data.hosts || []).map(function (x) { return { name: 'Хост: ' + x.name, v: x.cpuPct, status: x.status, color: statusColor(x.status) }; }));
        items.sort(function (a, b) { return b.v - a.v; });
        items = items.slice(0, 15);
        var el = document.getElementById('utilBarClustersHosts');
        if (!el) return;
        if (!items.length) { el.innerHTML = '<div class="text-muted small">Нет данных</div>'; return; }
        new ApexCharts(el, {
            chart: { type: 'bar', height: 380, background: 'transparent', toolbar: { show: false }, animations: { enabled: false } },
            theme: { mode: apexTheme },
            plotOptions: { bar: { horizontal: true, distributed: true, barHeight: '70%' } },
            series: [{ name: 'CPU avg %', data: items.map(function (x) { return x.v; }) }],
            legend: { show: false },
            dataLabels: { enabled: true, formatter: function (v) { return v.toFixed(1); } },
            xaxis: { categories: items.map(function (x) { return x.name; }), max: 100 },
            colors: items.map(function (x) { return x.color; }),
            grid: { borderColor: gridColor }
        }).render();
    })();

    // Топ загруженных VM
    (function () {
        var items = (data.vms || []).filter(function (x) { return x.status === 'overload'; })
            .sort(function (a, b) { return b.cpuPct - a.cpuPct; }).slice(0, 15);
        barChart('utilBarVmTop', items.map(function (x) { return x.name; }), items.map(function (x) { return x.cpuPct; }), '%', 380);
    })();

    // Простаивающие VM
    (function () {
        var items = (data.vms || []).filter(function (x) { return x.status === 'underused'; })
            .sort(function (a, b) { return a.cpuPct - b.cpuPct; }).slice(0, 15);
        barChart('utilBarVmIdle', items.map(function (x) { return x.name; }), items.map(function (x) { return x.cpuPct; }), '%', 380);
    })();

    // СХД
    (function () {
        var items = (data.storages || []).sort(function (a, b) { return b.diskPct - a.diskPct; });
        barChart('utilBarStorage', items.map(function (x) { return x.name; }), items.map(function (x) { return x.diskPct; }), '%', Math.max(200, items.length * 32));
    })();

    // LUN (топ-15 по занятости)
    (function () {
        var items = (data.luns || []).sort(function (a, b) { return b.diskPct - a.diskPct; }).slice(0, 15);
        barChart('utilBarLun', items.map(function (x) { return x.name; }), items.map(function (x) { return x.diskPct; }), '%', 380);
    })();

    // Структура оборудования - простое дерево "vCenter/кластер -> хосты -> VM count"
    (function () {
        var el = document.getElementById('utilTreeChart');
        if (!el) return;
        if (!(data.clusters || []).length) { el.innerHTML = '<div class="text-muted small">Нет данных для построения структуры</div>'; return; }

        var hostsByCluster = {};
        (data.hosts || []).forEach(function (h) {
            var key = h.parent || '(без кластера)';
            (hostsByCluster[key] = hostsByCluster[key] || []).push(h);
        });
        var vmsByCluster = {};
        (data.vms || []).forEach(function (v) {
            var key = v.parent || '(без кластера)';
            (vmsByCluster[key] = vmsByCluster[key] || []).push(v);
        });

        var html = '';
        data.clusters.forEach(function (c) {
            var hosts = hostsByCluster[c.name] || [];
            var vms = vmsByCluster[c.name] || [];
            html += '<div class="mb-2">';
            html += '<span class="util-tree-node' + (c.status === 'overload' ? ' util-tree-overload' : (c.status === 'underused' ? ' util-tree-underused' : '')) + '">' +
                '<b>' + c.parent + ' / ' + c.name + '</b> - CPU ' + c.cpuPct.toFixed(0) + '% / MEM ' + c.memPct.toFixed(0) + '%</span><br/>';
            html += '<span class="text-muted small">Хостов: ' + hosts.length + ', VM: ' + vms.length + '</span><br/>';
            hosts.forEach(function (h) {
                html += '<span class="util-tree-node' + (h.status === 'overload' ? ' util-tree-overload' : (h.status === 'underused' ? ' util-tree-underused' : '')) + '">' +
                    h.name + ' ' + h.cpuPct.toFixed(0) + '%/' + h.memPct.toFixed(0) + '%</span>';
            });
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
            var sevClass = r.severity === 'danger' ? 'util-risk-danger' : 'util-risk-warning';
            var sevLabel = r.severity === 'danger' ? 'Перегруз' : 'Недоиспользование';
            tr.innerHTML = '<td>' + r.category + '</td>' +
                '<td>' + r.object + '</td>' +
                '<td>' + r.metric + '</td>' +
                '<td>' + r.value.toFixed(1) + '</td>' +
                '<td class="' + sevClass + '">' + sevLabel + '</td>' +
                '<td>' + r.recommendation + '</td>';
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
