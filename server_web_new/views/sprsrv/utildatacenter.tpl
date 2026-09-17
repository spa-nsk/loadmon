<!-- Контроллер: controllers/sprutildatacenter.go (SprUtilizationDatacenter).
     По прямому запросу пользователя (2026-09-17, "сделай отчет по утилизации
     виртуальной инфраструктуры по ЦОД... с canvas и стильными индикаторами...
     предварительно просмотри в интернете аналогичные представления"):
     круговые индикаторы загрузки нарисованы на <canvas> (не SVG/ApexCharts -
     явное требование пользователя), по образцу современных capacity-
     дашбордов (Grafana/ManageEngine style: сплошная цветная заливка по
     порогу вместо тонкой линии прогресса, см. поиск "datacenter capacity
     utilization dashboard design" - источники в CHANGELOG). Отдельная
     вкладка на каждый ЦОД (где есть гипервизоры) + сводная вкладка
     сравнения ЦОД между собой - "листы" реализованы как Bootstrap
     nav-tabs (полноценные отдельные HTML-страницы/листы Excel здесь не при
     чём, это веб-отчёт, но принцип "один раздел на один ЦОД" соблюдён). -->

<div class="hero-text mb-3">
    <h2>Утилизация виртуальной инфраструктуры по ЦОД</h2>
    <div class="text-muted">Загрузка CPU, RAM и СХД по кластерам в разрезе ЦОД (логический объект vCenter Datacenter, где есть гипервизоры) - по последним суточным данным. Определено ЦОД: <b>{{.DcCount}}</b>.</div>
</div>

<div id="udcEmpty" class="udc-empty-state d-none">
    <div>Не найдено ни одного ЦОД с гипервизорами (нет свежих суточных записей ClusterRecord с заполненным Datacenter за последние 7 дней).</div>
</div>

<ul class="nav nav-pills udc-tabs mb-3" id="udcTabNav" role="tablist"></ul>
<div class="tab-content" id="udcTabContent"></div>

<style>
    .udc-tabs {
        gap: 0.35rem;
        flex-wrap: wrap;
    }

    .udc-tabs .nav-link {
        color: var(--bs-emphasis-color);
        background: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 2rem;
        padding: 0.4rem 1rem;
        font-size: 0.85rem;
        font-weight: 600;
    }

    .udc-tabs .nav-link.active {
        background: linear-gradient(135deg, #4dabf7, #3b82f6);
        border-color: transparent;
        color: #fff;
        box-shadow: 0 2px 10px rgba(59, 130, 246, 0.35);
    }

    .udc-tabs .nav-link.udc-tab-compare.active {
        background: linear-gradient(135deg, #845ef7, #5f3dc4);
        box-shadow: 0 2px 10px rgba(95, 61, 196, 0.35);
    }

    /* --- "Стеклянные" карточки (glassmorphism, тренд 2026 capacity-дашбордов
       по итогам обзора аналогичных представлений) --- */
    .udc-card {
        background: linear-gradient(160deg, var(--bs-tertiary-bg), var(--bs-secondary-bg));
        border: 1px solid var(--bs-border-color);
        border-radius: 0.75rem;
        padding: 1.1rem 1.3rem;
        box-shadow: 0 4px 18px rgba(0, 0, 0, 0.12);
        backdrop-filter: blur(6px);
    }

    .udc-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 700;
        font-size: 0.82rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        margin-bottom: 0.9rem;
    }

    .udc-dc-summary-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 1rem;
        margin-bottom: 1.25rem;
    }

    /* --- Большой круговой индикатор (canvas) с числом по центру --- */
    .udc-gauge-card {
        display: flex;
        align-items: center;
        gap: 1rem;
    }

    .udc-gauge-wrap {
        position: relative;
        flex: 0 0 auto;
    }

    .udc-gauge-label-inside {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        pointer-events: none;
    }

    .udc-gauge-pct {
        font-weight: 800;
        line-height: 1;
    }

    .udc-gauge-meta {
        flex: 1 1 auto;
        min-width: 0;
    }

    .udc-gauge-title {
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        color: var(--bs-secondary-color);
        margin-bottom: 0.25rem;
    }

    .udc-gauge-figures {
        font-size: 0.85rem;
        color: var(--bs-emphasis-color);
        font-weight: 600;
    }

    .udc-gauge-figures small {
        color: var(--bs-secondary-color);
        font-weight: 400;
    }

    .udc-badge {
        display: inline-block;
        font-size: 0.68rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        padding: 0.12rem 0.5rem;
        border-radius: 1rem;
        color: #fff;
        margin-top: 0.3rem;
    }

    .udc-badge-crit { background: #c4162a; }
    .udc-badge-warn { background: #d9822b; }
    .udc-badge-ok { background: #2fb344; }
    .udc-badge-low { background: #4dabf7; }

    .udc-cluster-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 1rem;
    }

    .udc-cluster-card {
        background: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 0.9rem 1rem;
    }

    .udc-cluster-head {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 0.5rem;
        margin-bottom: 0.6rem;
    }

    .udc-cluster-name {
        font-weight: 700;
        color: var(--bs-emphasis-color);
        font-size: 0.92rem;
    }

    .udc-cluster-sub {
        font-size: 0.72rem;
        color: var(--bs-secondary-color);
    }

    .udc-mini-gauges {
        display: flex;
        justify-content: space-between;
        gap: 0.5rem;
    }

    .udc-mini-gauge {
        display: flex;
        flex-direction: column;
        align-items: center;
        flex: 1 1 0;
    }

    .udc-mini-gauge-name {
        font-size: 0.68rem;
        color: var(--bs-secondary-color);
        text-transform: uppercase;
        letter-spacing: 0.03em;
        margin-top: 0.3rem;
    }

    .udc-mini-gauge-figures {
        font-size: 0.66rem;
        color: var(--bs-secondary-color);
        text-align: center;
        margin-top: 0.1rem;
        white-space: nowrap;
    }

    .udc-legend {
        display: flex;
        flex-wrap: wrap;
        gap: 0.9rem;
        font-size: 0.78rem;
        color: var(--bs-secondary-color);
        margin-bottom: 1rem;
    }

    .udc-legend-dot {
        display: inline-block;
        width: 0.65rem;
        height: 0.65rem;
        border-radius: 50%;
        margin-right: 0.35rem;
        vertical-align: middle;
    }

    .udc-compare-table th, .udc-compare-table td { vertical-align: middle; }

    .udc-compare-cell-crit { background: #c4162a; color: #fff; font-weight: 700; }
    .udc-compare-cell-warn { background: #d9822b; color: #fff; font-weight: 700; }
    .udc-compare-cell-ok { background: #2fb344; color: #fff; font-weight: 700; }
    .udc-compare-cell-low { background: #4dabf7; color: #fff; font-weight: 700; }

    .udc-empty-state {
        display: flex;
        align-items: center;
        justify-content: center;
        min-height: 200px;
        color: var(--bs-secondary-color);
        text-align: center;
        padding: 1.5rem;
        background: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
    }

    .udc-bar-chart-box canvas { width: 100%; }
</style>

<script>
(function () {
    var dcData = {{.DcDataJSON}};
    var dcNames = {{.DcNamesJSON}};
    var dcCompare = {{.DcCompareJSON}};

    var COLOR = { crit: '#c4162a', warn: '#d9822b', ok: '#2fb344', low: '#4dabf7' };
    var LABEL = { crit: 'Критично', warn: 'Высокая', ok: 'Норма', low: 'Простой' };

    // Пороги ИДЕНТИЧНЫ серверным (controllers/sprutildatacenter.go:utilDcTier) -
    // дублируются на клиенте, т.к. раскраска идёт по числу из JSON, а не по
    // заранее вычисленному на сервере классу (упрощает добавление новых
    // виджетов без правки Go-кода).
    function tier(pct) {
        if (pct >= 95) return 'crit';
        if (pct >= 75) return 'warn';
        if (pct >= 5) return 'ok';
        return 'low';
    }

    function fmtPct(v) { return (v || 0).toFixed(1) + '%'; }
    function fmtGb(v) {
        v = v || 0;
        if (v >= 1024) return (v / 1024).toFixed(1) + ' ТБ';
        return v.toFixed(0) + ' ГБ';
    }
    function fmtMhz(v) {
        v = v || 0;
        if (v >= 1000) return (v / 1000).toFixed(1) + ' ГГц';
        return v.toFixed(0) + ' МГц';
    }

    // --- Круговой индикатор на canvas: кольцо прогресса + число по центру.
    // Стиль - "сплошное цветное кольцо по порогу", как в современных
    // capacity-дашбордах (Grafana gauge panel / ManageEngine OpManager) -
    // см. обзор перед разработкой, источники в CHANGELOG.
    function drawGauge(canvas, pct, size) {
        var dpr = window.devicePixelRatio || 1;
        canvas.width = size * dpr;
        canvas.height = size * dpr;
        canvas.style.width = size + 'px';
        canvas.style.height = size + 'px';
        var ctx = canvas.getContext('2d');
        ctx.scale(dpr, dpr);
        var cx = size / 2, cy = size / 2, r = size / 2 - 7;
        var start = -Math.PI / 2;
        var clamped = Math.max(0, Math.min(100, pct || 0));
        var end = start + (clamped / 100) * Math.PI * 2;
        var trackColor = getComputedStyle(document.documentElement).getPropertyValue('--bs-border-color') || '#444';

        ctx.clearRect(0, 0, size, size);

        // фоновое кольцо (полная окружность, приглушённый цвет темы)
        ctx.beginPath();
        ctx.arc(cx, cy, r, 0, Math.PI * 2);
        ctx.lineWidth = 9;
        ctx.strokeStyle = trackColor.trim() || 'rgba(128,128,128,0.25)';
        ctx.stroke();

        // цветная дуга прогресса
        ctx.beginPath();
        ctx.arc(cx, cy, r, start, end);
        ctx.lineWidth = 9;
        ctx.lineCap = 'round';
        ctx.strokeStyle = COLOR[tier(clamped)];
        ctx.stroke();
    }

    function gaugeBlock(pct, title, totalText, usedText, size) {
        size = size || 96;
        var t = tier(pct);
        var wrap = document.createElement('div');
        wrap.className = 'udc-gauge-card';

        var gw = document.createElement('div');
        gw.className = 'udc-gauge-wrap';
        var canvas = document.createElement('canvas');
        gw.appendChild(canvas);
        var inside = document.createElement('div');
        inside.className = 'udc-gauge-label-inside';
        var pctEl = document.createElement('div');
        pctEl.className = 'udc-gauge-pct';
        pctEl.style.fontSize = (size * 0.22) + 'px';
        pctEl.style.color = COLOR[t];
        pctEl.textContent = fmtPct(pct);
        inside.appendChild(pctEl);
        gw.appendChild(inside);
        wrap.appendChild(gw);

        var meta = document.createElement('div');
        meta.className = 'udc-gauge-meta';
        meta.innerHTML =
            '<div class="udc-gauge-title">' + title + '</div>' +
            '<div class="udc-gauge-figures">' + usedText + ' <small>из ' + totalText + '</small></div>' +
            '<span class="udc-badge udc-badge-' + t + '">' + LABEL[t] + '</span>';
        wrap.appendChild(meta);

        canvas._redraw = function () { drawGauge(canvas, pct, size); };
        setTimeout(canvas._redraw, 0);
        return wrap;
    }

    function miniGauge(pct, name, figures) {
        var t = tier(pct);
        var wrap = document.createElement('div');
        wrap.className = 'udc-mini-gauge';
        var gw = document.createElement('div');
        gw.className = 'udc-gauge-wrap';
        var canvas = document.createElement('canvas');
        gw.appendChild(canvas);
        var inside = document.createElement('div');
        inside.className = 'udc-gauge-label-inside';
        var pctEl = document.createElement('div');
        pctEl.className = 'udc-gauge-pct';
        pctEl.style.fontSize = '11px';
        pctEl.style.color = COLOR[t];
        pctEl.textContent = Math.round(pct || 0) + '%';
        inside.appendChild(pctEl);
        gw.appendChild(inside);
        wrap.appendChild(gw);
        var nameEl = document.createElement('div');
        nameEl.className = 'udc-mini-gauge-name';
        nameEl.textContent = name;
        wrap.appendChild(nameEl);
        var figEl = document.createElement('div');
        figEl.className = 'udc-mini-gauge-figures';
        figEl.textContent = figures;
        wrap.appendChild(figEl);
        canvas._redraw = function () { drawGauge(canvas, pct, 56); };
        setTimeout(canvas._redraw, 0);
        return wrap;
    }

    function legend() {
        var div = document.createElement('div');
        div.className = 'udc-legend';
        div.innerHTML =
            '<span><span class="udc-legend-dot" style="background:' + COLOR.crit + '"></span>≥95% - критично</span>' +
            '<span><span class="udc-legend-dot" style="background:' + COLOR.warn + '"></span>75-94% - высокая</span>' +
            '<span><span class="udc-legend-dot" style="background:' + COLOR.ok + '"></span>5-74% - норма</span>' +
            '<span><span class="udc-legend-dot" style="background:' + COLOR.low + '"></span>&lt;5% - простой</span>';
        return div;
    }

    function buildDcPane(name, dc, isFirst) {
        var pane = document.createElement('div');
        pane.className = 'tab-pane fade' + (isFirst ? ' show active' : '');
        pane.id = 'udcPane_' + slug(name);
        pane.setAttribute('role', 'tabpanel');

        pane.appendChild(legend());

        var summaryRow = document.createElement('div');
        summaryRow.className = 'udc-dc-summary-row';
        var cpuCard = document.createElement('div');
        cpuCard.className = 'udc-card';
        cpuCard.appendChild(gaugeBlock(dc.cpuPct, 'CPU по ЦОД', fmtMhz(dc.cpuTotalMhz), fmtMhz(dc.cpuUsedMhz), 108));
        var ramCard = document.createElement('div');
        ramCard.className = 'udc-card';
        ramCard.appendChild(gaugeBlock(dc.ramPct, 'RAM по ЦОД', fmtGb(dc.ramTotalGb), fmtGb(dc.ramUsedGb), 108));
        var stCard = document.createElement('div');
        stCard.className = 'udc-card';
        stCard.appendChild(gaugeBlock(dc.storagePct, 'СХД по ЦОД', fmtGb(dc.storageTotalGb), fmtGb(dc.storageUsedGb), 108));
        summaryRow.appendChild(cpuCard);
        summaryRow.appendChild(ramCard);
        summaryRow.appendChild(stCard);
        pane.appendChild(summaryRow);

        var clustersCard = document.createElement('div');
        clustersCard.className = 'udc-card';
        clustersCard.innerHTML = '<h6>Кластеры ЦОД «' + name + '» (' + dc.clusterCount + '), хостов ' + dc.hostCount + ', VM ' + dc.vmCount + '</h6>';
        var grid = document.createElement('div');
        grid.className = 'udc-cluster-grid';
        (dc.clusters || []).forEach(function (c) {
            var card = document.createElement('div');
            card.className = 'udc-cluster-card';
            var head = document.createElement('div');
            head.className = 'udc-cluster-head';
            head.innerHTML = '<span class="udc-cluster-name">' + c.name + '</span>' +
                '<span class="udc-cluster-sub">' + (c.vcenter || '') + ' · хостов ' + c.hostCount + ' · VM ' + c.vmCount + '</span>';
            card.appendChild(head);
            var mini = document.createElement('div');
            mini.className = 'udc-mini-gauges';
            mini.appendChild(miniGauge(c.cpuPct, 'CPU', fmtMhz(c.cpuUsedMhz) + ' / ' + fmtMhz(c.cpuTotalMhz)));
            mini.appendChild(miniGauge(c.ramPct, 'RAM', fmtGb(c.ramUsedGb) + ' / ' + fmtGb(c.ramTotalGb)));
            mini.appendChild(miniGauge(c.storagePct, 'СХД', fmtGb(c.storageUsedGb) + ' / ' + fmtGb(c.storageTotalGb)));
            card.appendChild(mini);
            grid.appendChild(card);
        });
        clustersCard.appendChild(grid);
        pane.appendChild(clustersCard);

        return pane;
    }

    // --- Сравнительная вкладка: сгруппированная столбчатая диаграмма на
    // canvas (CPU/RAM/СХД по каждому ЦОД) + таблица с разницей по
    // отношению к минимальному значению среди ЦОД (наглядно показывает,
    // насколько неравномерно распределена нагрузка между площадками). ---
    function drawCompareChart(canvas, dcs) {
        var dpr = window.devicePixelRatio || 1;
        var cssW = canvas.parentElement.clientWidth || 600;
        var cssH = Math.max(260, dcs.length * 70 + 60);
        canvas.width = cssW * dpr;
        canvas.height = cssH * dpr;
        canvas.style.width = cssW + 'px';
        canvas.style.height = cssH + 'px';
        var ctx = canvas.getContext('2d');
        ctx.scale(dpr, dpr);
        ctx.clearRect(0, 0, cssW, cssH);

        var metrics = [
            { key: 'cpuPct', name: 'CPU' },
            { key: 'ramPct', name: 'RAM' },
            { key: 'storagePct', name: 'СХД' }
        ];
        var leftPad = 90, rightPad = 20, topPad = 10;
        var rowH = (cssH - topPad) / dcs.length;
        var barH = Math.min(16, rowH / (metrics.length + 1.4));
        var maxW = cssW - leftPad - rightPad;

        ctx.font = '12px sans-serif';
        ctx.textBaseline = 'middle';

        dcs.forEach(function (dc, i) {
            var rowTop = topPad + i * rowH;
            ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue('--bs-emphasis-color').trim() || '#ccc';
            ctx.textAlign = 'right';
            ctx.fillText(dc.name, leftPad - 10, rowTop + rowH / 2);

            metrics.forEach(function (m, mi) {
                var y = rowTop + 8 + mi * (barH + 4);
                var pct = Math.max(0, Math.min(100, dc[m.key] || 0));
                var w = (pct / 100) * maxW;
                // трек
                ctx.fillStyle = 'rgba(128,128,128,0.18)';
                ctx.fillRect(leftPad, y, maxW, barH);
                // значение
                ctx.fillStyle = COLOR[tier(pct)];
                ctx.fillRect(leftPad, y, w, barH);
                ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue('--bs-secondary-color').trim() || '#999';
                ctx.textAlign = 'left';
                ctx.font = '10px sans-serif';
                ctx.fillText(m.name + ' ' + pct.toFixed(1) + '%', leftPad + maxW + 6, y + barH / 2);
                ctx.font = '12px sans-serif';
            });
        });
    }

    function compareCellClass(pct) {
        return 'udc-compare-cell-' + tier(pct);
    }

    function buildComparePane(dcs) {
        var pane = document.createElement('div');
        pane.className = 'tab-pane fade';
        pane.id = 'udcPane_compare';
        pane.setAttribute('role', 'tabpanel');

        pane.appendChild(legend());

        if (!dcs.length) {
            var empty = document.createElement('div');
            empty.className = 'udc-empty-state';
            empty.innerHTML = '<div>Недостаточно данных для сравнения ЦОД.</div>';
            pane.appendChild(empty);
            return pane;
        }

        var chartCard = document.createElement('div');
        chartCard.className = 'udc-card mb-3';
        chartCard.innerHTML = '<h6>Сравнение загрузки ЦОД по CPU / RAM / СХД, %</h6><div class="udc-bar-chart-box"></div>';
        var box = chartCard.querySelector('.udc-bar-chart-box');
        var canvas = document.createElement('canvas');
        box.appendChild(canvas);
        pane.appendChild(chartCard);
        canvas._redraw = function () { drawCompareChart(canvas, dcs); };
        setTimeout(canvas._redraw, 0);
        window.addEventListener('resize', function () { canvas._redraw(); });

        var tableCard = document.createElement('div');
        tableCard.className = 'udc-card';
        tableCard.innerHTML = '<h6>Разница утилизации между ЦОД</h6>' +
            '<div class="text-muted small mb-2">Ячейка окрашена по собственному порогу значения (как индикаторы выше). Колонки "Разница с мин." показывают, насколько сильнее загружен этот ЦОД по сравнению с наименее загруженным - помогает быстро увидеть перекос нагрузки между площадками.</div>';
        var wrap = document.createElement('div');
        wrap.className = 'table-responsive';
        var table = document.createElement('table');
        table.className = 'table table-sm align-middle udc-compare-table mb-0';
        var minCpu = Math.min.apply(null, dcs.map(function (d) { return d.cpuPct || 0; }));
        var minRam = Math.min.apply(null, dcs.map(function (d) { return d.ramPct || 0; }));
        var minSt = Math.min.apply(null, dcs.map(function (d) { return d.storagePct || 0; }));
        var thead = '<thead><tr>' +
            '<th>ЦОД</th><th>Кластеров</th><th>Хостов</th><th>VM</th>' +
            '<th>CPU %</th><th>Разница с мин.</th>' +
            '<th>RAM %</th><th>Разница с мин.</th>' +
            '<th>СХД %</th><th>Разница с мин.</th>' +
            '</tr></thead>';
        var rows = '';
        dcs.forEach(function (d) {
            rows += '<tr>' +
                '<td><b>' + d.name + '</b></td>' +
                '<td>' + d.clusterCount + '</td>' +
                '<td>' + d.hostCount + '</td>' +
                '<td>' + d.vmCount + '</td>' +
                '<td class="' + compareCellClass(d.cpuPct) + '">' + fmtPct(d.cpuPct) + '</td>' +
                '<td>+' + (d.cpuPct - minCpu).toFixed(1) + ' п.п.</td>' +
                '<td class="' + compareCellClass(d.ramPct) + '">' + fmtPct(d.ramPct) + '</td>' +
                '<td>+' + (d.ramPct - minRam).toFixed(1) + ' п.п.</td>' +
                '<td class="' + compareCellClass(d.storagePct) + '">' + fmtPct(d.storagePct) + '</td>' +
                '<td>+' + (d.storagePct - minSt).toFixed(1) + ' п.п.</td>' +
                '</tr>';
        });
        table.innerHTML = thead + '<tbody>' + rows + '</tbody>';
        wrap.appendChild(table);
        tableCard.appendChild(wrap);
        pane.appendChild(tableCard);

        return pane;
    }

    function slug(s) {
        return (s || '').replace(/[^a-zA-Zа-яА-Я0-9]+/g, '_');
    }

    // --- Точка входа: строим вкладки-табы по dcNames + вкладку "Сравнение" ---
    var tabNav = document.getElementById('udcTabNav');
    var tabContent = document.getElementById('udcTabContent');

    if (!dcNames.length) {
        document.getElementById('udcEmpty').classList.remove('d-none');
        return;
    }

    dcNames.forEach(function (name, i) {
        var li = document.createElement('li');
        li.className = 'nav-item';
        li.innerHTML = '<button class="nav-link' + (i === 0 ? ' active' : '') + '" data-bs-toggle="tab" data-bs-target="#udcPane_' + slug(name) + '" type="button" role="tab">' + name + '</button>';
        tabNav.appendChild(li);
        tabContent.appendChild(buildDcPane(name, dcData[name], i === 0));
    });

    var compareLi = document.createElement('li');
    compareLi.className = 'nav-item';
    compareLi.innerHTML = '<button class="nav-link udc-tab-compare" data-bs-toggle="tab" data-bs-target="#udcPane_compare" type="button" role="tab">📊 Сравнение ЦОД</button>';
    tabNav.appendChild(compareLi);
    tabContent.appendChild(buildComparePane(dcCompare));

    // Перерисовка canvas при показе вкладки - критично для сравнительной
    // диаграммы (её ширина считается от clientWidth контейнера в момент
    // отрисовки; пока вкладка скрыта Bootstrap-табом (display:none), этот
    // размер равен 0 и диаграмма осталась бы пустой, если бы отрисовалась
    // только один раз сразу при построении DOM, до первого показа вкладки).
    // Каждый canvas, для которого это актуально, хранит свою функцию
    // перерисовки в cv._redraw (проставляется в gaugeBlock/miniGauge/
    // buildComparePane выше) - здесь она просто вызывается повторно.
    document.querySelectorAll('#udcTabNav button[data-bs-toggle="tab"]').forEach(function (btn) {
        btn.addEventListener('shown.bs.tab', function (e) {
            var pane = document.querySelector(e.target.getAttribute('data-bs-target'));
            if (!pane) return;
            pane.querySelectorAll('canvas').forEach(function (cv) {
                if (typeof cv._redraw === 'function') cv._redraw();
            });
        });
    });
})();
</script>
