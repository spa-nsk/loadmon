<!-- Контроллер: controllers/sprutilitsystem.go (SprUtilizationItSystem).
     По прямому запросу пользователя (2026-09-18, "сделай аналогичный отчет
     но в другом разрезе по ИТ Системам. Теперь каждая страница по отдельной
     ИТ системе, без учёта размещения. В ИТ системах может быть часть
     виртуальной инфраструктуры и аппаратные серверы. Сделай на каждой
     странице 2 блока 1 по виртуальной инфраструктуре, второй по аппаратным
     серверам. Утилизацию по аппаратным серверам выбирай из
     models.ServerMetric. В каждом блоке укажи список серверов с их
     индивидуальной загрузкой."): страница по структуре повторяет
     /sprsrv/utildatacenter (те же nav-pills "листы", тот же 4-уровневый
     цветовой порог индикаторов, тот же стиль карточек) - НО группировка НЕ
     по физическому размещению, а по ServerSprRecords.ITSystemKey (ИТ
     система из Jira CMDB, см. itsystem/jira.go), и на каждой вкладке ДВА
     раздельных блока по источнику данных: "Виртуальная инфраструктура"
     (models.VMRecord, метрики с уровня vCenter) и "Аппаратные серверы"
     (models.ServerMetric, метрики с уровня ОС через Zabbix-агента - прямое
     требование пользователя по источнику). В каждом блоке - не только
     агрегат, но и построчный список серверов с их ИНДИВИДУАЛЬНОЙ загрузкой
     (см. .uis-server-table ниже) - тоже прямое требование пользователя.

     ИТ систем в организации может быть существенно больше, чем Площадок
     ЦОД на /sprsrv/utildatacenter - добавлено поле поиска по вкладкам
     (.uis-search), фильтрующее список "листов" на клиенте по имени/ключу,
     иначе при большом количестве систем список вкладок неудобно
     пролистывать. -->

<div class="hero-text mb-3">
    <h2>Утилизация инфраструктуры по ИТ системам</h2>
    <div class="text-muted">Загрузка виртуальной инфраструктуры (данные vCenter) и аппаратных серверов (данные Zabbix, models.ServerMetric) в разрезе ИТ систем - без учёта физического размещения оборудования, усреднено за выбранный диапазон дат (см. фильтр ниже). Определено ИТ систем с серверами: <b>{{.ItSysCount}}</b>.</div>
</div>

<!-- Диапазон дат отчёта - ДОБАВЛЕНО 2026-09-18, затем уточнено ("нужно
     усреднение за период") - метрики CPU/RAM каждого сервера/ВМ
     усредняются по всем его записям в выбранном диапазоне дат (не берётся
     только последняя запись), см. vmAccum/metricAccum в
     controllers/sprutilitsystem.go. Тот же паттерн формы, что на
     utildatacenter.tpl - см. controllers/utildaterange.go. -->
<form id="uisPeriodFilterForm" class="mb-3 p-3 rounded border" method="get">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="uisFilterFrom" class="form-label">Дата начала</label>
            <input type="date" id="uisFilterFrom" name="from" class="form-control" value="{{.UtilFromDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="uisFilterTo" class="form-label">Дата окончания</label>
            <input type="date" id="uisFilterTo" name="to" class="form-control" value="{{.UtilToDate}}">
        </div>
        <div class="col-6 col-md-3">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Показать</button>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 text-muted small">Показатели CPU/RAM по каждому серверу/ВМ усредняются по всем записям в выбранном диапазоне дат (среднее за период).</div>
    </div>
</form>

<div id="uisEmpty" class="uis-empty-state d-none">
    <div>Не найдено ни одной ИТ системы с привязанными активными серверами/ВМ в выбранном диапазоне дат (поле "ИТ система" не заполнено ни у одного сервера, либо нет данных за период).</div>
</div>

<div id="uisSearchWrap" class="mb-2 d-none">
    <input type="text" id="uisSearch" class="form-control form-control-sm" style="max-width: 340px;" placeholder="Поиск ИТ системы по имени или ключу...">
</div>

<ul class="nav nav-pills uis-tabs mb-3" id="uisTabNav" role="tablist"></ul>
<div class="tab-content" id="uisTabContent"></div>

<style>
    .uis-tabs { gap: 0.35rem; flex-wrap: wrap; }

    .uis-tabs .nav-link {
        color: var(--bs-emphasis-color);
        background: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 2rem;
        padding: 0.4rem 1rem;
        font-size: 0.85rem;
        font-weight: 600;
    }

    .uis-tabs .nav-link.active {
        background: linear-gradient(135deg, #4dabf7, #3b82f6);
        border-color: transparent;
        color: #fff;
        box-shadow: 0 2px 10px rgba(59, 130, 246, 0.35);
    }

    .uis-card {
        background: linear-gradient(160deg, var(--bs-tertiary-bg), var(--bs-secondary-bg));
        border: 1px solid var(--bs-border-color);
        border-radius: 0.75rem;
        padding: 1.1rem 1.3rem;
        box-shadow: 0 4px 18px rgba(0, 0, 0, 0.12);
        backdrop-filter: blur(6px);
        margin-bottom: 1rem;
    }

    .uis-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 700;
        font-size: 0.82rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        margin-bottom: 0.9rem;
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.5rem;
    }

    .uis-block-count {
        font-size: 0.68rem;
        text-transform: none;
        letter-spacing: 0;
        color: var(--bs-secondary-color);
        font-weight: 500;
    }

    .uis-gauge-row {
        display: flex;
        gap: 1.5rem;
        margin-bottom: 1rem;
    }

    .uis-gauge-card {
        display: flex;
        align-items: center;
        gap: 0.8rem;
    }

    .uis-gauge-wrap { position: relative; flex: 0 0 auto; }

    .uis-gauge-label-inside {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        pointer-events: none;
        font-weight: 800;
        font-size: 15px;
    }

    .uis-gauge-title {
        font-size: 0.72rem;
        text-transform: uppercase;
        letter-spacing: 0.04em;
        color: var(--bs-secondary-color);
    }

    .uis-legend {
        display: flex;
        flex-wrap: wrap;
        gap: 0.9rem;
        font-size: 0.78rem;
        color: var(--bs-secondary-color);
        margin-bottom: 1rem;
    }

    .uis-legend-dot {
        display: inline-block;
        width: 0.65rem;
        height: 0.65rem;
        border-radius: 50%;
        margin-right: 0.35rem;
        vertical-align: middle;
    }

    .uis-server-table th, .uis-server-table td { vertical-align: middle; font-size: 0.82rem; }
    .uis-server-name { font-weight: 600; color: var(--bs-emphasis-color); }
    .uis-server-detail { font-size: 0.72rem; color: var(--bs-secondary-color); }

    .uis-pct-cell { font-weight: 700; text-align: right; white-space: nowrap; }
    .uis-pct-crit { color: #c4162a; }
    .uis-pct-warn { color: #d9822b; }
    .uis-pct-ok { color: #2fb344; }
    .uis-pct-low { color: #4dabf7; }

    .uis-empty-block {
        color: var(--bs-secondary-color);
        font-size: 0.85rem;
        padding: 0.75rem 0;
    }

    .uis-empty-state {
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

    .uis-key-badge {
        font-size: 0.68rem;
        font-weight: 500;
        color: var(--bs-secondary-color);
        font-family: monospace;
    }
</style>

<script>
(function () {
    var itSysData = {{.ItSysDataJSON}};
    var itSysNames = {{.ItSysNamesJSON}};

    var COLOR = { crit: '#c4162a', warn: '#d9822b', ok: '#2fb344', low: '#4dabf7' };
    var LABEL = { crit: 'Критично', warn: 'Высокая', ok: 'Норма', low: 'Простой' };

    // Пороги ИДЕНТИЧНЫ /sprsrv/utildatacenter (controllers/sprutildatacenter.go:utilDcTier) -
    // тот же принятый в проекте 4-уровневый порог для утилизации.
    function tier(pct) {
        if (pct >= 95) return 'crit';
        if (pct >= 75) return 'warn';
        if (pct >= 5) return 'ok';
        return 'low';
    }

    function fmtPct(v) { return (v || 0).toFixed(1) + '%'; }

    function drawGauge(canvas, pct, size) {
        var dpr = window.devicePixelRatio || 1;
        canvas.width = size * dpr;
        canvas.height = size * dpr;
        canvas.style.width = size + 'px';
        canvas.style.height = size + 'px';
        var ctx = canvas.getContext('2d');
        ctx.scale(dpr, dpr);
        var cx = size / 2, cy = size / 2, r = size / 2 - 6;
        var start = -Math.PI / 2;
        var clamped = Math.max(0, Math.min(100, pct || 0));
        var end = start + (clamped / 100) * Math.PI * 2;
        var trackColor = getComputedStyle(document.documentElement).getPropertyValue('--bs-border-color') || '#444';

        ctx.clearRect(0, 0, size, size);

        ctx.beginPath();
        ctx.arc(cx, cy, r, 0, Math.PI * 2);
        ctx.lineWidth = 8;
        ctx.strokeStyle = trackColor.trim() || 'rgba(128,128,128,0.25)';
        ctx.stroke();

        ctx.beginPath();
        ctx.arc(cx, cy, r, start, end);
        ctx.lineWidth = 8;
        ctx.lineCap = 'round';
        ctx.strokeStyle = COLOR[tier(clamped)];
        ctx.stroke();
    }

    function gaugeBlock(pct, title) {
        var t = tier(pct);
        var wrap = document.createElement('div');
        wrap.className = 'uis-gauge-card';

        var gw = document.createElement('div');
        gw.className = 'uis-gauge-wrap';
        var canvas = document.createElement('canvas');
        gw.appendChild(canvas);
        var inside = document.createElement('div');
        inside.className = 'uis-gauge-label-inside';
        inside.style.color = COLOR[t];
        inside.textContent = fmtPct(pct);
        gw.appendChild(inside);
        wrap.appendChild(gw);

        var meta = document.createElement('div');
        meta.innerHTML = '<div class="uis-gauge-title">' + title + '</div>';
        wrap.appendChild(meta);

        canvas._redraw = function () { drawGauge(canvas, pct, 76); };
        setTimeout(canvas._redraw, 0);
        return wrap;
    }

    function legend() {
        var div = document.createElement('div');
        div.className = 'uis-legend';
        div.innerHTML =
            '<span><span class="uis-legend-dot" style="background:' + COLOR.crit + '"></span>≥95% - критично</span>' +
            '<span><span class="uis-legend-dot" style="background:' + COLOR.warn + '"></span>75-94% - высокая</span>' +
            '<span><span class="uis-legend-dot" style="background:' + COLOR.ok + '"></span>5-74% - норма</span>' +
            '<span><span class="uis-legend-dot" style="background:' + COLOR.low + '"></span>&lt;5% - простой</span>';
        return div;
    }

    // buildServerTable - список серверов блока с ИНДИВИДУАЛЬНОЙ загрузкой
    // (прямое требование пользователя) - без отдельного canvas на каждую
    // строку (при десятках серверов в ИТ системе это было бы избыточно) -
    // проценты выделены цветом текста по тому же порогу, что и гейджи.
    function buildServerTable(servers) {
        if (!servers || !servers.length) {
            var empty = document.createElement('div');
            empty.className = 'uis-empty-block';
            empty.textContent = 'Нет серверов этого типа в данной ИТ системе (за последние 7 суток).';
            return empty;
        }
        var wrap = document.createElement('div');
        wrap.className = 'table-responsive';
        var table = document.createElement('table');
        table.className = 'table table-sm align-middle uis-server-table mb-0';
        var rows = '';
        servers.forEach(function (s) {
            rows += '<tr>' +
                '<td><div class="uis-server-name">' + s.name + '</div>' + (s.detail ? '<div class="uis-server-detail">' + s.detail + '</div>' : '') + '</td>' +
                '<td class="uis-pct-cell uis-pct-' + tier(s.cpuPct) + '">' + fmtPct(s.cpuPct) + '</td>' +
                '<td class="uis-pct-cell uis-pct-' + tier(s.ramPct) + '">' + fmtPct(s.ramPct) + '</td>' +
                '</tr>';
        });
        table.innerHTML = '<thead><tr><th>Сервер</th><th class="text-end">CPU</th><th class="text-end">RAM</th></tr></thead><tbody>' + rows + '</tbody>';
        wrap.appendChild(table);
        return wrap;
    }

    function buildBlockCard(title, block) {
        var card = document.createElement('div');
        card.className = 'uis-card';
        var head = document.createElement('h6');
        head.innerHTML = '<span>' + title + '</span><span class="uis-block-count">серверов: ' + block.count + '</span>';
        card.appendChild(head);

        if (block.count > 0) {
            var gaugeRow = document.createElement('div');
            gaugeRow.className = 'uis-gauge-row';
            gaugeRow.appendChild(gaugeBlock(block.cpuPct, 'CPU, среднее'));
            gaugeRow.appendChild(gaugeBlock(block.ramPct, 'RAM, среднее'));
            card.appendChild(gaugeRow);
        }

        card.appendChild(buildServerTable(block.servers));
        return card;
    }

    function buildItSysPane(key, sys, isFirst) {
        var pane = document.createElement('div');
        pane.className = 'tab-pane fade' + (isFirst ? ' show active' : '');
        pane.id = 'uisPane_' + slug(key);
        pane.setAttribute('role', 'tabpanel');

        var titleRow = document.createElement('div');
        titleRow.className = 'mb-2';
        titleRow.innerHTML = '<span class="uis-key-badge">' + key + '</span>';
        pane.appendChild(titleRow);

        pane.appendChild(legend());
        pane.appendChild(buildBlockCard('Виртуальная инфраструктура (vCenter)', sys.vm));
        pane.appendChild(buildBlockCard('Аппаратные серверы (Zabbix, ServerMetric)', sys.hw));

        return pane;
    }

    function slug(s) {
        return (s || '').replace(/[^a-zA-Zа-яА-Я0-9]+/g, '_');
    }

    var tabNav = document.getElementById('uisTabNav');
    var tabContent = document.getElementById('uisTabContent');

    if (!itSysNames.length) {
        document.getElementById('uisEmpty').classList.remove('d-none');
        return;
    }

    itSysNames.forEach(function (key, i) {
        var sys = itSysData[key];
        var li = document.createElement('li');
        li.className = 'nav-item';
        li.dataset.search = (sys.name + ' ' + key).toLowerCase();
        li.innerHTML = '<button class="nav-link' + (i === 0 ? ' active' : '') + '" data-bs-toggle="tab" data-bs-target="#uisPane_' + slug(key) + '" type="button" role="tab">' + sys.name + '</button>';
        tabNav.appendChild(li);
        tabContent.appendChild(buildItSysPane(key, sys, i === 0));
    });

    // Поиск по вкладкам - см. комментарий вверху файла про большое число
    // ИТ систем по сравнению с Площадками ЦОД.
    if (itSysNames.length > 8) {
        document.getElementById('uisSearchWrap').classList.remove('d-none');
        document.getElementById('uisSearch').addEventListener('input', function (e) {
            var q = e.target.value.trim().toLowerCase();
            tabNav.querySelectorAll('li.nav-item').forEach(function (li) {
                li.classList.toggle('d-none', q !== '' && li.dataset.search.indexOf(q) === -1);
            });
        });
    }

    document.querySelectorAll('#uisTabNav button[data-bs-toggle="tab"]').forEach(function (btn) {
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
