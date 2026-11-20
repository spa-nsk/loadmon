<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .ITMode}}Оборудование ИТ-системы{{if .HasSystem}} {{.SystemName}}{{end}}{{else if .ShowEquipFilters}}Оборудование площадки/ЦОД{{else}}Оборудование ЦОД {{.SiteName}}{{end}}</h2>
        <div class="text-muted" id="dvSummary">{{if .SiteName}}{{.SiteName}} &middot; {{end}}Шкафов: {{.RackCount}}</div>
    </div>
    {{if .ITMode}}
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacenterequipview/0">Оборудование площадки/ЦОД</a>
    {{else if .ShowEquipFilters}}
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacentersitemap">Карта площадок/ЦОД</a>
    {{else}}
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacentersitelist/0">К справочнику площадок</a>
    {{end}}
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

{{if .ITMode}}
<form id="itFilter" method="GET" action="/sprsrv/itsystemequipview" class="hw-filter-bar mb-3 p-3 rounded border">
    <label class="form-label">ИТ система</label>
    <div class="row g-2 align-items-start">
        <div class="col-md-3">
            <input type="text" name="ITSystemKey" id="ITSystemKey" class="form-control"
                value="{{.ITSystemKey}}" placeholder="CMDB-" autocomplete="off">
            <div id="ITSystemKeyHint" class="form-text"></div>
        </div>
        <div class="col-md-6 position-relative">
            <input type="text" id="ITSystemSearch" class="form-control"
                placeholder="Поиск по справочнику: код, название, статус..." autocomplete="off">
            <div id="ITSystemSearchResults" class="list-group position-absolute w-100 d-none it-system-dropdown"></div>
        </div>
        <div class="col-md-3 d-flex gap-2">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Показать</button>
            <a href="/sprsrv/itsystemequipview" class="btn btn-outline-secondary rounded-pill px-3">Сброс</a>
        </div>
    </div>
    <div id="ITSystemDataList" class="d-none">
        {{range $its := .ITSystemSpr}}
        <div class="it-system-item"
            data-key="{{$its.ObjectKey}}"
            data-label="{{$its.Label}}"
            data-full="{{$its.FullName}}"
            data-status="{{$its.Status}}"
            data-type="{{$its.SystemType}}"
            data-biz="{{$its.BusinessColumn}}"
            data-cfo="{{$its.CFO}}"></div>
        {{end}}
    </div>
    <div id="itSiteChips" class="mt-2 d-flex flex-wrap align-items-center gap-2"></div>
</form>
{{else if .ShowEquipFilters}}
<div class="hw-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-12 col-md-4 position-relative">
            <label for="LocalitySearch" class="form-label">Населённый пункт</label>
            <input type="text" id="LocalitySearch" class="form-control" autocomplete="off"
                   placeholder="начните вводить название, например: Новосиб" value="{{.FilterLocalityName}}">
            <input type="hidden" id="LocalityCode" name="LocalityCode" value="{{.FilterLocalityCode}}">
            <div id="LocalityList" class="list-group position-absolute w-100 shadow d-none" style="z-index: 1050; max-height: 320px; overflow-y: auto;"></div>
        </div>
        <div class="col-12 col-md-5">
            <label for="DatacenterSiteId" class="form-label">Площадка/ЦОД</label>
            <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                <option value="0">Выберите площадку/ЦОД</option>
                {{range $dc := .DatacenterSites}}
                <option value="{{$dc.Id}}" data-locality="{{$dc.LocalityCode}}" {{if eq $.Id (printf "%d" $dc.Id)}}selected{{end}}>{{$dc.Name}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-12 col-md-3 d-flex gap-2">
            <button type="button" id="dvEquipApply" class="btn btn-primary rounded-pill px-3">Показать</button>
            <button type="button" id="dvEquipReset" class="btn btn-outline-secondary rounded-pill px-3" title="Сбросить фильтр">Сброс</button>
        </div>
    </div>
</div>
{{end}}

<div id="dvEquipRoot">
<div class="dv-layout{{if or .ShowEquipFilters .ITMode}} dv-layout-filters{{end}}{{if .ITMode}} dv-layout-it{{end}}" id="dvLayout">
    <div class="dv-left">
        <div class="dv-sel d-flex align-items-center justify-content-between gap-2">
            <div id="dvSel" class="dv-sel-text">{{if .ITMode}}Наведите курсор на оборудование ИТ-системы - откроется карточка со ссылками{{else}}Наведите курсор на шкаф{{end}}</div>
        </div>
        <div class="dv-stage" id="dvStage">
            <canvas id="dvCanvas"></canvas>
            <div id="dvTip" class="dv-tip" hidden></div>
            {{if .ITMode}}<div id="itCard" class="dv-tip it-card" hidden></div>{{end}}
            <div id="dvZoomInd" class="dv-zoom-ind">Масштаб: 100%</div>
            <button type="button" id="dvResetView" class="dv-reset-view" title="Сбросить масштаб и ракурс (двойной клик по пустому месту - то же)">&#8634; Вид</button>
            <div class="dv-view-hint">Колесо - масштаб (и справа у шкафа) &middot; тяните за угол основания - ракурс &middot; тяните пустое место - сдвиг</div>
        </div>
    </div>
    {{if .ITMode}}
    <div class="dv-right">
        <div class="dv-right-head">
            <div class="fw-semibold small">Используемое оборудование</div>
            <div class="it-legend small text-muted mt-1">
                <span><i style="background:#38bdf8"></i>сервер / гипервизор системы</span>
                <span><i style="background:#fb923c"></i>хост размещения ВМ</span>
                <span><i style="background:#94a3b8"></i>хост кластера без ВМ системы</span>
                <span><i style="background:#34d399"></i>блейд-корзина</span>
                <span><i style="background:#a78bfa"></i>СХД</span>
            </div>
        </div>
        <div id="itListBody" class="it-list-body"></div>
    </div>
    {{else}}
    <div class="dv-right">
        <iframe id="dvFrame" class="dv-frame" title="Шкаф" src="about:blank"></iframe>
    </div>
    {{end}}
</div>
{{if .ITMode}}
<div id="dvEmpty" class="dv-empty d-none">{{if .HasSystem}}Для ИТ-системы не найдено оборудования с размещением{{else}}Выберите ИТ-систему и нажмите «Показать»{{end}}</div>
{{else if .ShowEquipFilters}}
<div id="dvEmpty" class="dv-empty d-none">Выберите населённый пункт и площадку/ЦОД, затем нажмите «Показать»</div>
{{end}}
</div>

<style>
    .dv-layout { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr); gap: 0.75rem; height: calc(100vh - 170px); min-height: 520px; }
    .dv-layout-filters { height: calc(100vh - 280px); }
    @media (max-width: 1100px) { .dv-layout { grid-template-columns: 1fr; height: auto; } .dv-layout-filters { height: auto; } .dv-left { height: 60vh; } .dv-frame { height: 80vh; } }
    .dv-empty { display: flex; align-items: center; justify-content: center; min-height: 280px; color: var(--bs-secondary-color); border: 1px dashed var(--bs-border-color); border-radius: 0.6rem; background: var(--bs-tertiary-bg); }
    .dv-left { display: flex; flex-direction: column; gap: 0.4rem; }
    .dv-left, .dv-right { min-height: 0; }
    .dv-sel-text { min-width: 0; }
    .dv-sel { flex: 0 0 auto; padding: 0.35rem 0.7rem; border-radius: 0.5rem; border: 1px solid var(--bs-border-color); background: var(--bs-tertiary-bg); font-size: 0.9rem; }
    .dv-sel .dv-sep { color: var(--bs-secondary-color); margin: 0 0.35rem; }
    .dv-stage { position: relative; width: 100%; flex: 1 1 auto; min-height: 0; border-radius: 0.6rem; overflow: hidden; border: 1px solid var(--bs-border-color); background: linear-gradient(#171d25, #0e1318); }
    [data-bs-theme="light"] .dv-stage { background: linear-gradient(#f3f6fa, #dbe3ec); }
    #dvCanvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; }
    .dv-right { display: flex; flex-direction: column; border: 1px solid var(--bs-border-color); border-radius: 0.6rem; background: linear-gradient(#171d25, #0e1318); overflow: hidden; }
    [data-bs-theme="light"] .dv-right { background: linear-gradient(#f3f6fa, #dbe3ec); }
    .dv-right-head { padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--bs-border-color); }
    .dv-frame { flex: 1 1 auto; width: 100%; border: 0; min-height: 0; background: transparent; }
    /* всплывающие элементы поверх сцены - палитра тёмной и светлой тем */
    .dv-tip { position: absolute; z-index: 5; pointer-events: none; min-width: 190px; padding: 0.55rem 0.7rem; border-radius: 0.5rem; font-size: 0.82rem; line-height: 1.35;
              background: rgba(17, 23, 30, 0.96); color: #e6edf3; border: 1px solid rgba(110, 190, 190, 0.5); box-shadow: 0 6px 22px rgba(0, 0, 0, 0.45); }
    [data-bs-theme="light"] .dv-tip { background: rgba(255, 255, 255, 0.97); color: #1d2b3a; border-color: rgba(13, 148, 136, 0.45); box-shadow: 0 6px 22px rgba(30, 50, 80, 0.22); }
    .dv-zoom-ind { position: absolute; z-index: 6; top: 12px; left: 50%; transform: translateX(-50%); pointer-events: none; padding: 0.3rem 0.8rem; border-radius: 999px; font-size: 0.85rem; font-weight: 600;
                   background: rgba(17, 23, 30, 0.94); color: #e6edf3; border: 1px solid rgba(110, 190, 190, 0.55); box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4); opacity: 0; transition: opacity 0.25s; }
    [data-bs-theme="light"] .dv-zoom-ind { background: rgba(255, 255, 255, 0.96); color: #1d2b3a; border-color: rgba(13, 148, 136, 0.5); box-shadow: 0 4px 16px rgba(30, 50, 80, 0.2); }
    .dv-zoom-ind.dv-zi-on { opacity: 1; }
    .dv-reset-view { position: absolute; z-index: 6; top: 10px; right: 10px; padding: 0.15rem 0.6rem; border-radius: 999px; font-size: 0.8rem; cursor: pointer;
                     background: rgba(17, 23, 30, 0.88); color: #c3d4e2; border: 1px solid rgba(110, 190, 190, 0.5); }
    .dv-reset-view:hover { color: #fff; border-color: #2dd4bf; }
    [data-bs-theme="light"] .dv-reset-view { background: rgba(255, 255, 255, 0.92); color: #34506a; border-color: rgba(52, 96, 120, 0.5); }
    [data-bs-theme="light"] .dv-reset-view:hover { color: #0f766e; border-color: #0d9488; }
    .dv-layout-it { grid-template-columns: minmax(0, 1.5fr) minmax(300px, 0.9fr); }
    @media (max-width: 1100px) { .dv-layout-it { grid-template-columns: 1fr; } .it-list-body { max-height: 50vh; } }
    .it-list-body { flex: 1 1 auto; min-height: 0; overflow: auto; }
    .it-legend span { display: inline-flex; align-items: center; gap: 0.3rem; margin-right: 0.8rem; }
    .it-legend i { display: inline-block; width: 0.7rem; height: 0.7rem; border-radius: 2px; }
    .it-grp { padding: 0.3rem 0.75rem; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--bs-secondary-color); background: var(--bs-tertiary-bg); border-bottom: 1px solid var(--bs-border-color); position: sticky; top: 0; z-index: 1; }
    .it-li { padding: 0.45rem 0.75rem; border-bottom: 1px solid var(--bs-border-color); cursor: pointer; font-size: 0.85rem; }
    .it-li:hover, .it-li.it-on { background: rgba(45, 212, 191, 0.14); }
    .it-sub2 { padding: 0.2rem 0.75rem; font-size: 0.72rem; font-weight: 600; color: var(--bs-secondary-color); border-bottom: 1px solid var(--bs-border-color); }
    .it-sub2.it-cl { font-weight: 500; padding-left: 1.2rem; font-style: italic; }
    .it-li .it-nm { font-weight: 600; }
    .it-li .it-sub { color: var(--bs-secondary-color); font-size: 0.78rem; }
    .it-li .it-det { display: none; margin-top: 0.35rem; font-size: 0.78rem; }
    .it-li.it-open .it-det { display: block; }
    .it-li .it-det dl { margin: 0; }
    .it-li .it-det dt { color: var(--bs-secondary-color); font-weight: 500; }
    .it-li .it-det dd { margin: 0 0 0.2rem; word-break: break-all; }
    .it-li .it-det .it-lk { display: flex; flex-direction: column; gap: 0.15rem; margin-top: 0.3rem; }
    .it-card { pointer-events: auto; max-width: 340px; z-index: 8; }
    .it-card a { color: #5eead4; }
    [data-bs-theme="light"] .it-card a { color: #0f766e; }
    .it-card .it-lk { display: flex; flex-direction: column; gap: 0.15rem; margin-top: 0.4rem; }
    .it-card td:last-child { word-break: break-all; }
    .it-system-dropdown { z-index: 1000; max-height: 280px; overflow-y: auto; box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, .25); }
    .it-system-dropdown .list-group-item small { display: block; color: var(--bs-secondary-color); }
    .dv-view-hint { position: absolute; z-index: 4; left: 10px; bottom: 8px; pointer-events: none; font-size: 0.72rem; color: rgba(170, 190, 208, 0.75); }
    [data-bs-theme="light"] .dv-view-hint { color: #3f566c; }
    .dv-tip b.dv-name { font-size: 0.95rem; color: #f8fafc; }
    [data-bs-theme="light"] .dv-tip b.dv-name { color: #0f1b2a; }
    .dv-tip table { border-collapse: collapse; margin-top: 0.25rem; }
    .dv-tip td { padding: 1px 0; vertical-align: top; }
    .dv-tip td:first-child { color: #9fb0c2; padding-right: 0.7rem; white-space: nowrap; }
    [data-bs-theme="light"] .dv-tip td:first-child { color: #52667a; }
    .dv-tip .dv-hint { margin-top: 0.35rem; color: #8196aa; font-size: 0.75rem; }
    [data-bs-theme="light"] .dv-tip .dv-hint { color: #64788c; }
</style>

<script>
{{if .ITMode}}
(function () {
    // Тот же выбор ИТ-системы, что в справочнике серверов (view2.tpl / updatebase.tpl)
    var keyInput = document.getElementById('ITSystemKey');
    var hint = document.getElementById('ITSystemKeyHint');
    var searchInput = document.getElementById('ITSystemSearch');
    var results = document.getElementById('ITSystemSearchResults');
    var dataList = document.getElementById('ITSystemDataList');
    if (!keyInput || !searchInput || !results || !dataList) return;
    var items = Array.prototype.map.call(dataList.querySelectorAll('.it-system-item'), function (el) {
        return {
            key: el.getAttribute('data-key') || '', label: el.getAttribute('data-label') || '', full: el.getAttribute('data-full') || '',
            status: el.getAttribute('data-status') || '', type: el.getAttribute('data-type') || '',
            biz: el.getAttribute('data-biz') || '', cfo: el.getAttribute('data-cfo') || ''
        };
    });
    function findByKey(key) {
        if (!key) return null;
        var lower = key.toLowerCase();
        for (var i = 0; i < items.length; i++) if (items[i].key.toLowerCase() === lower) return items[i];
        return null;
    }
    function describe(item) {
        var parts = [item.key];
        if (item.label) parts.push(item.label);
        if (item.full) parts.push(item.full);
        return parts.join(' — ');
    }
    function updateHint() {
        var match = findByKey(keyInput.value.trim());
        if (match) { hint.textContent = describe(match) + (match.status ? ' (' + match.status + ')' : ''); hint.classList.remove('text-danger'); }
        else if (keyInput.value.trim() !== '') { hint.textContent = 'Код не найден в справочнике'; hint.classList.add('text-danger'); }
        else { hint.textContent = ''; hint.classList.remove('text-danger'); }
    }
    function hideResults() { results.classList.add('d-none'); results.innerHTML = ''; }
    function escapeHtml(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
    function renderResults(query) {
        var q = query.trim().toLowerCase();
        if (q === '') { hideResults(); return; }
        var matches = items.filter(function (item) {
            return [item.key, item.label, item.full, item.status, item.type, item.biz, item.cfo].some(function (v) { return v.toLowerCase().indexOf(q) !== -1; });
        }).slice(0, 50);
        if (matches.length === 0) {
            results.innerHTML = '<div class="list-group-item text-muted">Ничего не найдено</div>';
            results.classList.remove('d-none');
            return;
        }
        results.innerHTML = '';
        matches.forEach(function (item) {
            var row = document.createElement('button');
            row.type = 'button';
            row.className = 'list-group-item list-group-item-action';
            row.innerHTML = '<div>' + escapeHtml(item.key) + (item.label ? ' — ' + escapeHtml(item.label) : '') + '</div>' + (item.full ? '<small>' + escapeHtml(item.full) + '</small>' : '');
            row.addEventListener('click', function () {
                keyInput.value = item.key; searchInput.value = describe(item); updateHint(); hideResults();
            });
            results.appendChild(row);
        });
        results.classList.remove('d-none');
    }
    searchInput.addEventListener('input', function () { renderResults(searchInput.value); });
    searchInput.addEventListener('focus', function () { if (searchInput.value.trim() !== '') renderResults(searchInput.value); });
    document.addEventListener('click', function (evt) { if (evt.target !== searchInput && !results.contains(evt.target)) hideResults(); });
    keyInput.addEventListener('input', updateHint);
    keyInput.addEventListener('blur', updateHint);
    var initial = findByKey(keyInput.value.trim());
    if (initial) searchInput.value = describe(initial);
    updateHint();
})();
{{end}}
{{if .ShowEquipFilters}}
(function () {
    // Фильтр населённый пункт → площадка/ЦОД (как в справочнике шкафов / площадок)
    var search = document.getElementById('LocalitySearch');
    var list = document.getElementById('LocalityList');
    var code = document.getElementById('LocalityCode');
    var siteSelect = document.getElementById('DatacenterSiteId');
    if (!search || !list || !code || !siteSelect) return;
    var timer = null, seq = 0;
    var allSiteOptions = Array.prototype.slice.call(siteSelect.options);

    function filterSitesByLocality() {
        var loc = code.value || '';
        var cur = siteSelect.value;
        siteSelect.innerHTML = '';
        allSiteOptions.forEach(function (opt) {
            if (opt.value === '0' || opt.value === '') {
                siteSelect.appendChild(opt.cloneNode(true));
                return;
            }
            if (!loc || opt.getAttribute('data-locality') === loc) {
                siteSelect.appendChild(opt.cloneNode(true));
            }
        });
        var still = false;
        Array.prototype.forEach.call(siteSelect.options, function (o) { if (o.value === cur) still = true; });
        siteSelect.value = still ? cur : '0';
    }
    function hide() { list.classList.add('d-none'); list.innerHTML = ''; }
    function pick(item) {
        code.value = item.oktmo;
        search.value = item.label;
        hide();
        filterSitesByLocality();
    }
    function renderItems(items) {
        list.innerHTML = '';
        if (!items.length) {
            var empty = document.createElement('div');
            empty.className = 'list-group-item text-muted small';
            empty.textContent = 'Ничего не найдено';
            list.appendChild(empty);
        }
        items.forEach(function (it) {
            var a = document.createElement('button');
            a.type = 'button';
            a.className = 'list-group-item list-group-item-action small';
            a.textContent = it.label + '  [' + it.oktmo + ']';
            a.addEventListener('mousedown', function (e) { e.preventDefault(); pick(it); });
            list.appendChild(a);
        });
        list.classList.remove('d-none');
    }
    function query() {
        var q = search.value.trim();
        if (q.length < 2) { hide(); return; }
        var my = ++seq;
        fetch('/sprsrv/localitysearch?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
            .then(function (r) { return r.json(); })
            .then(function (items) { if (my === seq) renderItems(items); })
            .catch(function () { hide(); });
    }
    function go() {
        var sid = siteSelect.value || '0';
        var loc = code.value || '';
        var url = '/sprsrv/datacenterequipview/' + sid;
        if (loc) url += '?locality=' + encodeURIComponent(loc);
        location.href = url;
    }
    search.addEventListener('input', function () {
        if (search.value.trim() === '') { code.value = ''; filterSitesByLocality(); }
        clearTimeout(timer);
        timer = setTimeout(query, 200);
    });
    search.addEventListener('blur', function () { setTimeout(hide, 150); });
    search.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') hide();
        if (e.key === 'Enter') { e.preventDefault(); go(); }
    });
    siteSelect.addEventListener('change', go);
    var applyBtn = document.getElementById('dvEquipApply');
    var resetBtn = document.getElementById('dvEquipReset');
    if (applyBtn) applyBtn.addEventListener('click', go);
    if (resetBtn) resetBtn.addEventListener('click', function () { location.href = '/sprsrv/datacenterequipview/0'; });
    filterSitesByLocality();
})();
{{end}}
(function () {
    'use strict';

    var DATA = {{.DCJSON}};
    var canvas = document.getElementById('dvCanvas');
    var emptyEl = document.getElementById('dvEmpty');
    var layoutEl = document.getElementById('dvLayout');
    // IT - режим «Оборудование ИТ-системы» (/sprsrv/itsystemequipview): та же сцена, но шкафы
    // только с оборудованием ИТ-системы; юниты рисуются на лицевой стороне шкафа, справа - список.
    var IT = DATA && DATA.it ? DATA.it : null;
    if (!DATA || !canvas || (!IT && (!DATA.racks || !DATA.racks.length))) {
        if (emptyEl && layoutEl) {
            layoutEl.classList.add('d-none');
            emptyEl.classList.remove('d-none');
            if (DATA && DATA.siteName && !IT) emptyEl.textContent = 'В площадке «' + DATA.siteName + '» нет шкафов для отображения';
        }
        return;
    }
    if (emptyEl) emptyEl.classList.add('d-none');
    if (layoutEl) layoutEl.classList.remove('d-none');

    var stage = document.getElementById('dvStage');
    var tip = document.getElementById('dvTip');
    var ctx = canvas.getContext('2d');
    var allRacks = DATA.racks || [];
    var racks = allRacks.filter(function (r) { return !r.shelf; });   // шкафы
    var shelves = allRacks.filter(function (r) { return r.shelf; });  // стеллаж площадки без шкафов ("вне шкафа")
    var TAU = Math.PI * 2;

    // ---- цветовые схемы (светлая/тёмная тема сайта, атрибут data-bs-theme): приглушённая графитово-бирюзовая палитра ----
    // Изображения оборудования (шкафы APC, устройства вендоров, стойки стеллажа) от темы не зависят.
    var PALS = {
        dark:  { bgTop: '#171d25', bgBot: '#0e1318', floorTop: '#243242', floorBot: '#1b2733', floorEdge: 'rgba(110,190,190,0.5)', grid: 'rgba(150,175,195,0.14)',
                 sel: '#2dd4bf', hov: '#99f6e4', selTxt: '#5eead4', hovTxt: '#ccfbf1', labelText: '#e6e9f2', labelShadow: 'rgba(0,0,0,0.6)', deckEdge: 'rgba(120,190,200,0.5)', glowMode: 'lighter',
                 cornerBg: 'rgba(15,21,28,0.8)', cornerBgHot: 'rgba(45,212,191,0.3)', cornerEdge: 'rgba(175,208,228,0.95)', cornerGlyph: '#dbe8f2' },
        light: { bgTop: '#f3f6fa', bgBot: '#dbe3ec', floorTop: '#bccbdc', floorBot: '#a6b8cc', floorEdge: 'rgba(40,82,110,0.8)', grid: 'rgba(40,70,100,0.24)',
                 sel: '#0d9488', hov: '#14b8a6', selTxt: '#0f766e', hovTxt: '#0f766e', labelText: '#17324a', labelShadow: 'rgba(255,255,255,0.9)', deckEdge: 'rgba(52,96,120,0.5)', glowMode: 'source-over',
                 cornerBg: 'rgba(255,255,255,0.88)', cornerBgHot: 'rgba(13,148,136,0.2)', cornerEdge: 'rgba(40,80,110,0.95)', cornerGlyph: '#2b4a64' }
    };
    var PAL = PALS.dark;

    // оттенки янтаря для световых столбов: от светло-янтарного (малая мощность) к насыщенно-оранжевому (максимум)
    // PowerLight из app2.json: AnimatedFire | AnimatedLight | StaticLight
    var POWER_LIGHT = "{{.PowerLight}}";
    var glowT = 0;
    // мерцание светодиодов включённого оборудования на полках (спячка/выкл — статично)
    function ledBright(seed) {
        var t = glowT, s = (seed || 0) * 12.9898;
        var a = 0.62 + 0.38 * (0.5 + 0.5 * Math.sin(t * 2.15 + s));
        var b = 0.78 + 0.22 * (0.5 + 0.5 * Math.sin(t * 5.4 + s * 1.73));
        var dip = 1 - 0.12 * Math.max(0, Math.sin(t * 15.5 + s * 2.9));
        return Math.min(1, Math.max(0.48, a * b * dip));
    }
    function glowRGB(frac, dark) {
        var stops = [[253, 224, 120], [251, 191, 36], [245, 158, 11]];     // один и тот же светлый янтарный ряд в обеих темах
        var t = Math.max(0, Math.min(1, frac)) * 2, i = Math.min(1, Math.floor(t)), f = t - i;
        var a = stops[i], b = stops[i + 1];
        return Math.round(a[0] + (b[0] - a[0]) * f) + ',' + Math.round(a[1] + (b[1] - a[1]) * f) + ',' + Math.round(a[2] + (b[2] - a[2]) * f);
    }
    function fadeSafe(v) { return Math.max(0, Math.min(1, v)); }
    // исходный статический световой столб (прямоугольник + навершие)
    function drawGlowStatic(cx, cy, bw, bh, frac, dark, mode) {
        var rgb = glowRGB(frac, dark), a0 = (dark ? 0.32 : 0.5) + (dark ? 0.4 : 0.35) * frac;
        ctx.save();
        ctx.globalCompositeOperation = mode;
        var g = ctx.createLinearGradient(0, cy, 0, cy - bh);
        g.addColorStop(0, 'rgba(' + rgb + ',' + a0.toFixed(2) + ')');
        g.addColorStop(1, 'rgba(' + rgb + ',0)');
        ctx.fillStyle = g;
        ctx.fillRect(cx - bw / 2, cy - bh, bw, bh);
        ctx.fillStyle = 'rgba(' + rgb + ',' + Math.min(0.95, a0 + 0.3).toFixed(2) + ')';
        ctx.beginPath(); ctx.ellipse(cx, cy, bw * 0.52, bw * 0.24, 0, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }
    // анимированный световой столб без языков пламени (пульс яркости/высоты)
    function drawGlowAnimatedLight(cx, cy, bw, bh, frac, dark, mode, seed) {
        var t = glowT, ph0 = (seed || 0) * 0.618;
        var rgb = glowRGB(frac, dark), rgbHot = '255,244,200';
        var aBase = (dark ? 0.28 : 0.42) + (dark ? 0.35 : 0.3) * frac;
        var pulse = 0.94 + 0.06 * Math.sin(t * 2.1 + ph0);
        var H = bh * pulse, W = bw * (0.96 + 0.04 * Math.sin(t * 1.4 + ph0 * 1.3));
        var aMul = 0.88 + 0.12 * Math.sin(t * 2.6 + ph0);
        ctx.save();
        ctx.globalCompositeOperation = mode;
        var g0 = ctx.createLinearGradient(0, cy, 0, cy - H);
        g0.addColorStop(0, 'rgba(' + rgb + ',' + (aBase * aMul).toFixed(2) + ')');
        g0.addColorStop(0.55, 'rgba(' + rgb + ',' + (aBase * 0.45 * aMul).toFixed(2) + ')');
        g0.addColorStop(1, 'rgba(' + rgb + ',0)');
        ctx.fillStyle = g0;
        ctx.beginPath();
        ctx.moveTo(cx - W * 0.5, cy);
        ctx.quadraticCurveTo(cx - W * 0.18, cy - H * 0.55, cx, cy - H);
        ctx.quadraticCurveTo(cx + W * 0.18, cy - H * 0.55, cx + W * 0.5, cy);
        ctx.closePath();
        ctx.fill();
        var coreH = H * (0.6 + 0.15 * frac) * (0.94 + 0.06 * Math.sin(t * 3.2 + ph0));
        var coreW = W * (0.16 + 0.06 * frac);
        var cg = ctx.createLinearGradient(0, cy, 0, cy - coreH);
        cg.addColorStop(0, 'rgba(' + rgbHot + ',' + ((dark ? 0.7 : 0.6) * aMul).toFixed(2) + ')');
        cg.addColorStop(0.65, 'rgba(' + rgb + ',' + (0.32 * aMul).toFixed(2) + ')');
        cg.addColorStop(1, 'rgba(' + rgbHot + ',0)');
        ctx.fillStyle = cg;
        ctx.beginPath();
        ctx.moveTo(cx - coreW, cy);
        ctx.quadraticCurveTo(cx - coreW * 0.25, cy - coreH * 0.5, cx, cy - coreH);
        ctx.quadraticCurveTo(cx + coreW * 0.25, cy - coreH * 0.5, cx + coreW, cy);
        ctx.closePath();
        ctx.fill();
        ctx.fillStyle = 'rgba(' + rgb + ',' + Math.min(0.95, aBase * aMul + 0.28).toFixed(2) + ')';
        ctx.beginPath(); ctx.ellipse(cx, cy, W * 0.52, W * 0.24, 0, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }
    // анимированный световой столб с языками пламени; высота/ширина по-прежнему от мощности (frac)
    function drawGlowAnimatedFire(cx, cy, bw, bh, frac, dark, mode, seed) {
        var t = glowT, ph0 = (seed || 0) * 0.618;
        var rgb = glowRGB(frac, dark);
        var rgbHot = '255,244,200', rgbCool = '249,115,22';
        var aBase = (dark ? 0.28 : 0.42) + (dark ? 0.35 : 0.3) * frac;
        var flick = 0.92 + 0.08 * Math.sin(t * 3.1 + ph0) + 0.04 * Math.sin(t * 7.4 + ph0 * 2);
        var H = bh * flick, W = bw * (0.92 + 0.08 * Math.sin(t * 2.2 + ph0));
        ctx.save();
        ctx.globalCompositeOperation = mode;
        var g0 = ctx.createLinearGradient(0, cy, 0, cy - H);
        g0.addColorStop(0, 'rgba(' + rgb + ',' + aBase.toFixed(2) + ')');
        g0.addColorStop(0.55, 'rgba(' + rgb + ',' + (aBase * 0.45).toFixed(2) + ')');
        g0.addColorStop(1, 'rgba(' + rgb + ',0)');
        ctx.fillStyle = g0;
        ctx.beginPath();
        ctx.moveTo(cx - W * 0.48, cy);
        ctx.quadraticCurveTo(cx - W * 0.22, cy - H * 0.55, cx + Math.sin(t * 2.6 + ph0) * W * 0.06, cy - H);
        ctx.quadraticCurveTo(cx + W * 0.22, cy - H * 0.55, cx + W * 0.48, cy);
        ctx.closePath();
        ctx.fill();
        var n = 3 + Math.round(frac * 3);
        for (var i = 0; i < n; i++) {
            var u = n <= 1 ? 0.5 : i / (n - 1);
            var phase = t * (2.4 + i * 0.55) + ph0 + i * 1.7;
            var sway = Math.sin(phase) * W * (0.1 + 0.08 * frac);
            var lean = Math.sin(phase * 0.7 + 1.1) * W * 0.06;
            var hMul = (0.45 + 0.55 * (0.35 + 0.65 * (1 - Math.abs(u - 0.5) * 1.2))) * (0.82 + 0.18 * Math.sin(phase * 1.3));
            var h = H * hMul;
            var w = W * (0.16 + 0.14 * (1 - Math.abs(u - 0.5)) + 0.06 * frac) * (0.85 + 0.15 * Math.sin(phase * 1.8));
            var bx = cx + (u - 0.5) * W * 0.55 + sway * 0.35;
            var tipX = bx + sway + lean;
            var tipY = cy - h;
            var g = ctx.createLinearGradient(bx, cy, tipX, tipY);
            g.addColorStop(0, 'rgba(' + rgbCool + ',' + ((dark ? 0.55 : 0.48) * (0.5 + 0.5 * frac)).toFixed(2) + ')');
            g.addColorStop(0.35, 'rgba(' + rgb + ',' + ((dark ? 0.7 : 0.62) * (0.55 + 0.45 * frac)).toFixed(2) + ')');
            g.addColorStop(0.75, 'rgba(' + rgbHot + ',' + ((dark ? 0.55 : 0.5) * fadeSafe(0.4 + 0.4 * frac)).toFixed(2) + ')');
            g.addColorStop(1, 'rgba(' + rgbHot + ',0)');
            ctx.fillStyle = g;
            ctx.beginPath();
            ctx.moveTo(bx - w, cy);
            ctx.bezierCurveTo(bx - w * 0.85, cy - h * 0.35, tipX - w * 0.25, cy - h * 0.72, tipX, tipY);
            ctx.bezierCurveTo(tipX + w * 0.25, cy - h * 0.72, bx + w * 0.85, cy - h * 0.35, bx + w, cy);
            ctx.closePath();
            ctx.fill();
        }
        var coreH = H * (0.55 + 0.2 * frac) * (0.9 + 0.1 * Math.sin(t * 4.2 + ph0));
        var coreW = W * (0.14 + 0.08 * frac);
        var cg = ctx.createLinearGradient(0, cy, 0, cy - coreH);
        cg.addColorStop(0, 'rgba(' + rgbHot + ',' + (dark ? 0.75 : 0.65) + ')');
        cg.addColorStop(0.6, 'rgba(' + rgb + ',0.35)');
        cg.addColorStop(1, 'rgba(' + rgbHot + ',0)');
        ctx.fillStyle = cg;
        ctx.beginPath();
        ctx.moveTo(cx - coreW, cy);
        ctx.quadraticCurveTo(cx - coreW * 0.3, cy - coreH * 0.5, cx + Math.sin(t * 3.5 + ph0) * coreW * 0.4, cy - coreH);
        ctx.quadraticCurveTo(cx + coreW * 0.3, cy - coreH * 0.5, cx + coreW, cy);
        ctx.closePath();
        ctx.fill();
        ctx.fillStyle = 'rgba(' + rgb + ',' + Math.min(0.95, aBase + 0.28).toFixed(2) + ')';
        ctx.beginPath(); ctx.ellipse(cx, cy, W * 0.52, W * 0.24, 0, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }
    function drawGlow(cx, cy, bw, bh, frac, dark, mode, seed) {
        if (POWER_LIGHT === 'StaticLight') { drawGlowStatic(cx, cy, bw, bh, frac, dark, mode); return; }
        if (POWER_LIGHT === 'AnimatedLight') { drawGlowAnimatedLight(cx, cy, bw, bh, frac, dark, mode, seed); return; }
        drawGlowAnimatedFire(cx, cy, bw, bh, frac, dark, mode, seed);
    }
    var KIND_COLOR = { server: '#38bdf8', host: '#38bdf8', chassis: '#34d399', storage: '#a78bfa', vm: '#fbbf24' };
    var KIND_LABEL = { server: 'Сервер', host: 'Хост ESXi', chassis: 'Блейд-корзина', storage: 'СХД', vm: 'Виртуальная машина' };
    var ROLE_COLOR = { host: '#fb923c', cluster: '#94a3b8' };          // IT: хост размещения ВМ / хост кластера без ВМ системы
    var ROLE_LABEL = { host: 'Хост размещения ВМ', cluster: 'Хост кластера' };
    function eqColor(e) { return ROLE_COLOR[e.role || e.r] || KIND_COLOR[e.kind || e.k] || '#f472b6'; }
    function pushThemeToFrame() {                    // фрейм со шкафом - отдельная страница: передаём ей текущую тему
        var f = document.getElementById('dvFrame');
        if (f && f.contentWindow) { try { f.contentWindow.postMessage({ type: 'dvTheme', theme: document.documentElement.getAttribute('data-bs-theme') }, location.origin); } catch (e) { } }
    }
    function setPal() {
        PAL = document.documentElement.getAttribute('data-bs-theme') === 'light' ? PALS.light : PALS.dark;
        stage.style.background = 'linear-gradient(' + PAL.bgTop + ',' + PAL.bgBot + ')';
        pushThemeToFrame();
    }
    setPal();
    if (window.MutationObserver) new MutationObserver(function () { setPal(); if (typeof render === 'function' && G.w) render(); }).observe(document.documentElement, { attributes: true, attributeFilter: ['data-bs-theme'] });

    // ---- сводка ----
    var kw = function (w) { return w >= 1000 ? (w / 1000).toFixed(1) + ' кВт' : Math.round(w) + ' Вт'; };
    (function () {
        if (IT) {
            var st = IT.stats || {};
            var s0 = 'площадок: <b>' + (st.sites || 0) + '</b> &middot; шкафов: <b>' + (st.racks || 0) + '</b>'
                + ' &middot; серверов: <b>' + (st.servers || 0) + '</b> &middot; ВМ: <b>' + (st.vms || 0) + '</b>'
                + ' &middot; хостов ESXi: <b>' + (st.hosts || 0) + '</b>' + (st.clusters ? ' (кластеров: ' + st.clusters + ')' : '')
                + ' &middot; СХД: <b>' + (st.storages || 0) + '</b>';
            document.getElementById('dvSummary').innerHTML = s0;
            return;
        }
        var s = 'Шкафов: <b>' + racks.length + '</b>' + (shelves.length ? ' &middot; есть стеллаж (оборудование вне шкафов)' : '');
        if (DATA.powerAvgW > 0) s += ' &middot; мощность (средняя 24ч): <b>' + kw(DATA.powerAvgW) + '</b>';
        if (DATA.powerPeakW > 0) s += ' &middot; пик: <b>' + kw(DATA.powerPeakW) + '</b>';
        document.getElementById('dvSummary').innerHTML = s;
    })();


    // ---- оборудование стеллажа: раскладка по секциям/полкам (те же правила, что в просмотре шкафа справа) ----
    var SH_CLEAR = (42 - 0.5 - 0.8 * 6) / 5;        // свободная высота между полками, U
    function isAP(it) { return it.k === 'switch' && /aironet|\bair-|\bc91\d{2}|unifi|\buap|ubiquiti|точка доступа|access point|wi-?fi|\bwlan/i.test(it.vd || ''); }
    function shelfSize(it) { if (isAP(it)) return Math.max(2, it.u || 0); return it.u > 0 ? it.u : (it.k === 'chassis' ? 10 : (it.k === 'tape' ? 6 : (it.k === 'switch' ? 1 : 2))); }
    function packShelf(r) {
        var list = (r.items || []).map(function (it) {
            return {
                k: it.k, id: it.id || 0, nm: it.nm || '', vd: it.vd || '', p: it.p, u: it.u,
                slots: it.slots || 0, blades: it.blades || [],
                key: it.key || '', role: it.role || '', rows: it.rows || [], links: it.links || [],
                su: Math.min(shelfSize(it), SH_CLEAR)
            };
        });
        var rank = { storage: 0, tape: 1, chassis: 1, server: 2, 'switch': 3 };
        list.sort(function (a, b) { return (b.su - a.su) || ((rank[a.k] || 9) - (rank[b.k] || 9)) || String(a.nm || a.vd).localeCompare(String(b.nm || b.vd)); });
        var units = [];
        function newUnit() { var lv = []; for (var k = 0; k < 5; k++) lv.push({ items: [], used: 0 }); units.push(lv); }
        newUnit();
        list.forEach(function (it) {
            for (var ui = 0; ; ui++) {
                if (ui >= units.length) newUnit();
                for (var k = 0; k < 5; k++) {
                    var lv = units[ui][k];
                    if (lv.used + it.su <= SH_CLEAR + 0.001) { lv.items.push({ it: it, off: lv.used }); lv.used += it.su + 0.12; return; }
                }
            }
        });
        r.packed = units;
        r.units = units.length;
    }
    function shelfStyle(vd) {
        var s = String(vd).toLowerCase();
        if (/hpe|hp |hewlett|proliant|\bdl\d|synergy|c7000|3par|primera|nimble/.test(s)) return '#01a982';
        if (/dell|poweredge|emc|unity|vnx|powerstore|powervault/.test(s)) return '#2a8be0';
        if (/lenovo|thinksystem|\bsr\d{3}/.test(s)) return '#e2231a';
        if (/huawei|oceanstor|dorado/.test(s)) return '#e0334d';
        if (/\bdepo\b|\bstorm\s*[345]\d{3}/.test(s)) return '#6b67e9';
        if (/ubiquiti|unifi|\buap/.test(s)) return '#006fff';
        if (/cisco|catalyst|nexus|\bisr|\basr|\bmds|\basa\b|aironet|\bair-|\bc91\d{2}/.test(s)) return '#049fd9';
        return '#8b95a5';
    }

    // ---- раскладка шкафов по сетке; в режиме IT - отдельный блок (пол + подпись) на каждую площадку ----
    var maxPower = 0;
    var groups = [];
    if (IT) {
        var gById = {};
        (IT.sites || []).forEach(function (s) { var g = { id: s.id, name: s.name, equip: s.equip, racks: [], shelves: [] }; groups.push(g); gById[s.id] = g; });
        allRacks.forEach(function (r) { var g = gById[r.siteId]; if (g) (r.shelf ? g.shelves : g.racks).push(r); });
    } else {
        groups.push({ id: DATA.siteId || 0, name: '', equip: 0, racks: racks, shelves: shelves });
    }
    groups.forEach(function (g) {
        var n = g.racks.length;
        g.gridCols = n ? Math.max(1, Math.ceil(Math.sqrt(n * 1.3))) : 0;   // нет шкафов - стеллаж по левому краю
        g.rows = n ? Math.ceil(n / g.gridCols) : 1;
        g.racks.forEach(function (r, k) {
            r.gi = k % g.gridCols; r.gj = Math.floor(k / g.gridCols);
            if (r.powerAvgW > maxPower) maxPower = r.powerAvgW;
        });
        // стеллаж - справа от группы шкафов, габариты каждой секции как у шкафа 42U (на клетку по ширине)
        var su = 0;
        g.shelves.forEach(function (r) {
            packShelf(r);
            r.gi = (g.gridCols ? g.gridCols + 0.6 : 0.15) + su; r.gj = Math.floor((g.rows - 1) / 2);
            su += r.units;
        });
        g.w = g.gridCols ? g.gridCols + (su ? 0.6 + su : 0) : su + 0.3;   // ширина блока с учётом стеллажа
    });
    // блоки площадок: до 3 в ряд, между ними промежутки под линии связи; cols/rows - габариты всей сцены
    var GAP_X = 3.4, GAP_Y = 3.4, PER_ROW = 3;
    var cols = 0, rows = 0;
    (function () {
        var x = 0, y = 0, rowH = 0, cnt = 0;
        groups.forEach(function (g) {
            if (cnt && cnt % PER_ROW === 0) { x = 0; y += rowH + GAP_Y; rowH = 0; }
            g.ox = x; g.oy = y;
            x += g.w + GAP_X; rowH = Math.max(rowH, g.rows);
            cols = Math.max(cols, g.ox + g.w); rows = Math.max(rows, g.oy + g.rows);
            cnt++;
        });
        if (!rows) rows = 1;
    })();
    groups.forEach(function (g) { g.racks.concat(g.shelves).forEach(function (r) { r.gi += g.ox; r.gj += g.oy; }); });

    // отдельные рисуемые объекты: шкафы и секции стеллажа (у секции - свой центр для сортировки по глубине)
    var items3d = [];
    racks.forEach(function (r) { items3d.push({ r: r, shelfUnit: -1, ca: r.gi + 0.5, cb: r.gj + 0.5 }); });
    shelves.forEach(function (r) { for (var u = 0; u < r.units; u++) items3d.push({ r: r, shelfUnit: u, ca: r.gi + u + 0.5, cb: r.gj + 0.5 }); });

    var FW = 0.60, FD = 0.80;          // размеры шкафа в клетках (ширина 600, глубина 1000 при клетке 1.2 м)
    // выбранный/наведённый объект: { r, u } где u = -1 для шкафа, 0..N-1 для секции стеллажа
    var selected = null, hovered = null, hoverCorner = -1;
    var polys = [];                    // силуэты для hit-test: {r, u, pts}
    var G = { w: 0, h: 0, dpr: 1, s: 1, baseS: 1 };
    // eq - ключ оборудования ИТ-системы (режим IT): шкаф/секция + конкретное устройство
    function sameHit(a, b) { return !!a && !!b && a.r === b.r && a.u === b.u && (a.eq || '') === (b.eq || ''); }
    function hitOf(r, u, eq, item) { return { r: r, u: u == null ? -1 : u, eq: eq || '', item: item || null }; }
    var selEq = '';                    // IT: ключ оборудования, выбранного в списке/закреплённого кликом
    function eqActive(key) { return !!key && (key === selEq || (!!hovered && hovered.eq === key)); }
    function unitStats(r, u) {
        var s = { servers: 0, chassis: 0, storages: 0, tapes: 0, network: 0 }, pk = r.packed && r.packed[u];
        if (!pk) return s;
        pk.forEach(function (lv) {
            lv.items.forEach(function (pl) {
                var k = pl.it.k;
                if (k === 'server') s.servers++;
                else if (k === 'chassis') s.chassis++;
                else if (k === 'storage') s.storages++;
                else if (k === 'tape') s.tapes++;
                else if (k === 'switch') s.network++;
            });
        });
        return s;
    }

    // ---- камера: поворот (yaw) вокруг вертикали и наклон (pitch); по умолчанию - прежняя изометрия 45 / 30 градусов ----
    var YAW0 = Math.PI / 4, PITCH0 = Math.PI / 6;
    var cam = { yaw: YAW0, pitch: PITCH0, zoom: 1, panX: 0, panY: 0 };
    var CV = { sy: 0, cy: 0, sp: 0, zk: 1, K: 1, offX: 0, offY: 0, aPos: true, bPos: true };

    function hz(r) { return G.s * 1.55 * Math.max(0.5, Math.min(1.4, r.uHeight / 42)); }

    // проекция без смещения (для расчёта габаритов): ca, cb - координаты относительно центра сцены
    function projRaw(a, b, z, yaw, pitch, S) {
        var ca = a - cols / 2, cb = b - rows / 2, sy = Math.sin(yaw), cy = Math.cos(yaw);
        var X = ca * cy - cb * sy, Y = ca * sy + cb * cy, K = Math.SQRT2 * S;
        return [X * K, Y * K * Math.sin(pitch) - z * Math.cos(pitch) / Math.cos(PITCH0)];
    }
    var FLOOR_M = 0.35;
    function floorCorners() { return [[-FLOOR_M, -FLOOR_M], [cols + FLOOR_M, -FLOOR_M], [cols + FLOOR_M, rows + FLOOR_M], [-FLOOR_M, rows + FLOOR_M]]; }
    function sceneBBox(yaw, pitch, S) {
        var zTop = 1.55 * 1.4 * S + 0.3 * S, minX = 1e9, maxX = -1e9, minY = 1e9, maxY = -1e9;
        floorCorners().forEach(function (c) {
            [0, zTop].forEach(function (z) {
                var p = projRaw(c[0], c[1], z, yaw, pitch, S);
                minX = Math.min(minX, p[0]); maxX = Math.max(maxX, p[0]); minY = Math.min(minY, p[1]); maxY = Math.max(maxY, p[1]);
            });
        });
        return { minX: minX, maxX: maxX, minY: minY, maxY: maxY };
    }
    function setCamera() {
        var S = G.baseS * cam.zoom;
        G.s = S;
        CV.sy = Math.sin(cam.yaw); CV.cy = Math.cos(cam.yaw); CV.sp = Math.sin(cam.pitch);
        CV.zk = Math.cos(cam.pitch) / Math.cos(PITCH0); CV.K = Math.SQRT2 * S;
        CV.aPos = CV.sy > 0; CV.bPos = CV.cy > 0;           // какие боковые грани видны (нормаль к наблюдателю)
        var bb = sceneBBox(cam.yaw, cam.pitch, S);
        CV.offX = G.w / 2 + cam.panX - (bb.minX + bb.maxX) / 2;
        CV.offY = G.h / 2 + cam.panY - (bb.minY + bb.maxY) / 2;
    }
    function P(a, b, z) {
        var ca = a - cols / 2, cb = b - rows / 2;
        var X = ca * CV.cy - cb * CV.sy, Y = ca * CV.sy + cb * CV.cy;
        return [X * CV.K + CV.offX, Y * CV.K * CV.sp - z * CV.zk + CV.offY];
    }
    function depthOf(ca, cb) { return (ca - cols / 2) * CV.sy + (cb - rows / 2) * CV.cy; }

    function layout() {
        var rect = stage.getBoundingClientRect();
        G.dpr = window.devicePixelRatio || 1;
        G.w = Math.max(200, rect.width); G.h = Math.max(200, rect.height);
        canvas.width = Math.round(G.w * G.dpr); canvas.height = Math.round(G.h * G.dpr);
        var bb = sceneBBox(YAW0, PITCH0, 1);                   // базовый масштаб - вписать сцену в окно при стандартном ракурсе
        G.baseS = Math.min(G.w * 0.94 / (bb.maxX - bb.minX), G.h * 0.9 / (bb.maxY - bb.minY));
        setCamera();
    }

    // ---- примитивы ----
    function poly(pts, fill, stroke, lw) {
        ctx.beginPath();
        ctx.moveTo(pts[0][0], pts[0][1]);
        for (var i = 1; i < pts.length; i++) ctx.lineTo(pts[i][0], pts[i][1]);
        ctx.closePath();
        if (fill) { ctx.fillStyle = fill; ctx.fill(); }
        if (stroke) { ctx.strokeStyle = stroke; ctx.lineWidth = lw || 1; ctx.stroke(); }
    }
    function lin(x0, y0, x1, y1, stops) {
        var g = ctx.createLinearGradient(x0, y0, x1, y1);
        stops.forEach(function (s) { g.addColorStop(s[0], s[1]); });
        return g;
    }
    function hull(pts) {                                       // выпуклая оболочка (цепочка Эндрю) - силуэт для hit-test
        var p = pts.slice().sort(function (a, b) { return a[0] - b[0] || a[1] - b[1]; });
        function cross(o, a, b) { return (a[0] - o[0]) * (b[1] - o[1]) - (a[1] - o[1]) * (b[0] - o[0]); }
        var lo = [], up = [], i;
        for (i = 0; i < p.length; i++) { while (lo.length >= 2 && cross(lo[lo.length - 2], lo[lo.length - 1], p[i]) <= 0) lo.pop(); lo.push(p[i]); }
        for (i = p.length - 1; i >= 0; i--) { while (up.length >= 2 && cross(up[up.length - 2], up[up.length - 1], p[i]) <= 0) up.pop(); up.push(p[i]); }
        lo.pop(); up.pop();
        return lo.concat(up);
    }
    function boxHull(a0, a1, b0, b1, z0, z1) {
        var pts = [];
        [a0, a1].forEach(function (a) { [b0, b1].forEach(function (b) { pts.push(P(a, b, z0)); pts.push(P(a, b, z1)); }); });
        return hull(pts);
    }
    // локальная система грани: origin (screen), ux/uy - вектор на Wl, vx/vy - вектор на Hl
    function faceTransform(o, ux, uy, vx, vy, Wl, Hl) {
        ctx.transform(ux / Wl, uy / Wl, vx / Hl, vy / Hl, o[0], o[1]);
    }
    // вертикальная грань между двумя верхними точками pA, pB (экранные) высотой hpx: слева направо без зеркалирования
    function vertFaceXf(pA, pB, hpx, Wl, Hl) {
        var l = pA, r = pB;
        if (pA[0] > pB[0]) { l = pB; r = pA; }
        faceTransform(l, r[0] - l[0], r[1] - l[1], 0, hpx * CV.zk, Wl, Hl);
    }
    // горизонтальная грань (верх) прямоугольника: подбираем ориентацию, чтобы текст читался (без зеркала и вверх ногами)
    function topFaceXf(a0, a1, b0, b1, z, Wl, Hl) {
        var best = null, bestScore = -1e9;
        var cs = [[a0, b0], [a1, b0], [a1, b1], [a0, b1]];
        [[0, 1], [1, 0]].forEach(function (swap) {
            for (var ci = 0; ci < 4; ci++) for (var sgn = 0; sgn < 2; sgn++) {
                var o = cs[ci], pa = cs[(ci + (sgn ? 3 : 1)) % 4], pb = cs[(ci + (sgn ? 1 : 3)) % 4];
                var u = swap[0] ? pa : pb, v = swap[0] ? pb : pa;
                var po = P(o[0], o[1], z), pu = P(u[0], u[1], z), pv = P(v[0], v[1], z);
                var ux = pu[0] - po[0], uy = pu[1] - po[1], vx = pv[0] - po[0], vy = pv[1] - po[1];
                var det = ux * vy - uy * vx;
                if (det <= 0) continue;                         // зеркальная ориентация - пропускаем
                var len = Math.sqrt(ux * ux + uy * uy) || 1;
                var score = ux / len + (uy / len) * -0.2;       // u должен идти вправо; небольшое предпочтение вдоль экрана
                if (score > bestScore) { bestScore = score; best = [po, ux, uy, vx, vy]; }
            }
        });
        if (!best) return false;
        faceTransform(best[0], best[1], best[2], best[3], best[4], Wl, Hl);
        return true;
    }

    function drawFloor() {
        ctx.fillStyle = lin(0, 0, 0, G.h, [[0, PAL.bgTop], [1, PAL.bgBot]]);
        ctx.fillRect(0, 0, G.w, G.h);
        groups.forEach(function (g) {                         // пол каждой площадки (без IT - одна плита на всю сцену)
            var gw = g.w, gh = g.rows, ox = g.ox, oy = g.oy;
            var fc = [[ox - FLOOR_M, oy - FLOOR_M], [ox + gw + FLOOR_M, oy - FLOOR_M], [ox + gw + FLOOR_M, oy + gh + FLOOR_M], [ox - FLOOR_M, oy + gh + FLOOR_M]].map(function (c) { return P(c[0], c[1], 0); });
            poly(fc, lin(0, Math.min.apply(null, fc.map(function (p) { return p[1]; })), 0, Math.max.apply(null, fc.map(function (p) { return p[1]; })), [[0, PAL.floorTop], [1, PAL.floorBot]]), PAL.floorEdge, 1.5);
            ctx.strokeStyle = PAL.grid; ctx.lineWidth = 1;
            for (var i = 0; i <= gw; i++) { var a = P(ox + i, oy, 0), b = P(ox + i, oy + gh, 0); ctx.beginPath(); ctx.moveTo(a[0], a[1]); ctx.lineTo(b[0], b[1]); ctx.stroke(); }
            for (var j = 0; j <= gh; j++) { var c = P(ox, oy + j, 0), d = P(ox + gw, oy + j, 0); ctx.beginPath(); ctx.moveTo(c[0], c[1]); ctx.lineTo(d[0], d[1]); ctx.stroke(); }
        });
    }

    // ---- режим IT: линии связи между площадками (по полу) с анимацией передачи данных и подписи площадок ----
    function plateEdge(g, tx, ty) {                            // точка выхода луча из центра площадки к (tx, ty) за границу её плиты
        var cx = g.ox + g.w / 2, cy = g.oy + g.rows / 2, dx = tx - cx, dy = ty - cy;
        var hx = g.w / 2 + FLOOR_M, hy = g.rows / 2 + FLOOR_M;
        var t = Math.min(dx ? hx / Math.abs(dx) : 1e9, dy ? hy / Math.abs(dy) : 1e9);
        return [cx + dx * t, cy + dy * t];
    }
    var siteLinks = [];
    if (IT && groups.length > 1) {                              // до 5 площадок - каждая с каждой, иначе цепочкой
        if (groups.length <= 5) { for (var li = 0; li < groups.length; li++) for (var lj = li + 1; lj < groups.length; lj++) siteLinks.push([groups[li], groups[lj]]); }
        else { for (var lk = 0; lk + 1 < groups.length; lk++) siteLinks.push([groups[lk], groups[lk + 1]]); }
    }
    function drawSiteLinks() {
        if (!siteLinks.length) return;
        var dark = PAL === PALS.dark, base = dark ? '45,212,191' : '13,148,136', lw = Math.max(2, G.s * 0.045);
        siteLinks.forEach(function (lk, idx) {
            var A = lk[0], B = lk[1];
            var ea = plateEdge(A, B.ox + B.w / 2, B.oy + B.rows / 2), eb = plateEdge(B, A.ox + A.w / 2, A.oy + A.rows / 2);
            var pa = P(ea[0], ea[1], 0), pb = P(eb[0], eb[1], 0);
            ctx.save();
            ctx.lineCap = 'round';
            ctx.strokeStyle = 'rgba(' + base + ',' + (dark ? 0.22 : 0.3) + ')'; ctx.lineWidth = lw * 2.6;       // ореол канала
            ctx.beginPath(); ctx.moveTo(pa[0], pa[1]); ctx.lineTo(pb[0], pb[1]); ctx.stroke();
            ctx.strokeStyle = 'rgba(' + base + ',' + (dark ? 0.55 : 0.7) + ')'; ctx.lineWidth = lw;               // канал
            ctx.beginPath(); ctx.moveTo(pa[0], pa[1]); ctx.lineTo(pb[0], pb[1]); ctx.stroke();
            ctx.setLineDash([lw * 2.2, lw * 3.2]); ctx.lineDashOffset = -glowT * lw * 6;                          // бегущие штрихи
            ctx.strokeStyle = dark ? 'rgba(204,251,241,0.85)' : 'rgba(255,255,255,0.9)'; ctx.lineWidth = lw * 0.45;
            ctx.beginPath(); ctx.moveTo(pa[0], pa[1]); ctx.lineTo(pb[0], pb[1]); ctx.stroke();
            ctx.setLineDash([]);
            for (var d = 0; d < 2; d++) {                                                                         // пакеты данных в обе стороны
                for (var k = 0; k < 3; k++) {
                    var ph = (glowT * (d ? 0.17 : 0.23) + k / 3 + idx * 0.21 + d * 0.5) % 1, t = d ? 1 - ph : ph;
                    var x = pa[0] + (pb[0] - pa[0]) * t, y = pa[1] + (pb[1] - pa[1]) * t, rr = lw * (d ? 1.25 : 1.6);
                    var gr = ctx.createRadialGradient(x, y, 0, x, y, rr * 2.4);
                    gr.addColorStop(0, d ? 'rgba(253,224,120,0.95)' : 'rgba(167,243,208,0.95)');
                    gr.addColorStop(1, d ? 'rgba(253,224,120,0)' : 'rgba(167,243,208,0)');
                    ctx.globalCompositeOperation = dark ? 'lighter' : 'source-over';
                    ctx.fillStyle = gr; ctx.beginPath(); ctx.arc(x, y, rr * 2.4, 0, TAU); ctx.fill();
                    ctx.globalCompositeOperation = 'source-over';
                    ctx.fillStyle = d ? '#fbbf24' : '#34d399'; ctx.beginPath(); ctx.arc(x, y, rr * 0.7, 0, TAU); ctx.fill();
                }
            }
            [pa, pb].forEach(function (p) {                                                                       // порты каналов на границах площадок
                ctx.fillStyle = dark ? '#0f1a22' : '#ffffff'; ctx.strokeStyle = 'rgb(' + base + ')'; ctx.lineWidth = Math.max(1.5, lw * 0.6);
                ctx.beginPath(); ctx.arc(p[0], p[1], lw * 1.3, 0, TAU); ctx.fill(); ctx.stroke();
            });
            ctx.restore();
        });
    }
    var siteLabelBoxes = [];
    function drawSiteLabels() {
        siteLabelBoxes = [];
        if (!IT) return;
        groups.forEach(function (g) {
            var p = P(g.ox + g.w / 2, g.oy + g.rows / 2, G.s * 1.55 * 1.3);
            var racksN = g.racks.length, title = g.name || ('Площадка ' + g.id);
            var sub = 'шкафов ' + racksN + ' · оборудования ' + (g.equip || 0);
            ctx.save();
            ctx.font = '700 13px "Segoe UI", Arial, sans-serif';
            var tw = Math.max(ctx.measureText(title).width, 0), w = tw + 24, h = 36;
            ctx.font = '500 10.5px "Segoe UI", Arial, sans-serif';
            w = Math.max(w, ctx.measureText(sub).width + 24);
            var x = p[0] - w / 2, y = p[1] - h;
            var hot = focusedGroup === g;
            ctx.fillStyle = hot ? PAL.cornerBgHot : PAL.cornerBg; ctx.strokeStyle = hot ? PAL.sel : PAL.cornerEdge; ctx.lineWidth = hot ? 2 : 1.3;
            ctx.beginPath();
            if (ctx.roundRect) ctx.roundRect(x, y, w, h, 9); else ctx.rect(x, y, w, h);
            ctx.fill(); ctx.stroke();
            ctx.beginPath(); ctx.moveTo(p[0] - 5, y + h); ctx.lineTo(p[0] + 5, y + h); ctx.lineTo(p[0], y + h + 7); ctx.closePath(); ctx.fillStyle = ctx.strokeStyle; ctx.fill();
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.font = '700 13px "Segoe UI", Arial, sans-serif'; ctx.fillStyle = PAL.labelText; ctx.fillText(title, p[0], y + 12);
            ctx.font = '500 10.5px "Segoe UI", Arial, sans-serif'; ctx.fillStyle = PAL.cornerGlyph; ctx.fillText(sub, p[0], y + 26);
            ctx.restore();
            siteLabelBoxes.push({ g: g, x: x, y: y, w: w, h: h + 7 });
        });
    }
    function siteLabelAt(x, y) {
        for (var i = siteLabelBoxes.length - 1; i >= 0; i--) { var b = siteLabelBoxes[i]; if (x >= b.x && x <= b.x + b.w && y >= b.y && y <= b.y + b.h) return b.g; }
        return null;
    }
    var focusedGroup = null;
    function focusGroup(g) {                                    // камера на площадку: крупнее и по центру окна
        if (!g) return;
        focusedGroup = g;
        var span = Math.max(g.w, g.rows) + 2.5, tot = Math.max(cols, rows);
        cam.zoom = Math.max(1, Math.min(4, tot / span * 0.85));
        cam.panX = 0; cam.panY = 0; setCamera();
        var p = P(g.ox + g.w / 2, g.oy + g.rows / 2, G.s * 0.6);
        cam.panX += G.w / 2 - p[0]; cam.panY += G.h / 2 - p[1];
        setCamera(); tip.hidden = true; render(); showZoomInd();
    }
    function focusAll() { focusedGroup = null; resetView(); }

    // маркеры углов основания: за любой можно тянуть - меняется ракурс (курсор "move" как подсказка)
    function cornerPoints() { return floorCorners().map(function (c) { return P(c[0], c[1], 0); }); }
    function drawCorners() {
        var cp = cornerPoints();
        cp.forEach(function (p, i) {
            var hot = (hoverCorner === i) || (dragMode === 'rotate' && dragCorner === i);
            ctx.save();
            ctx.fillStyle = hot ? PAL.cornerBgHot : PAL.cornerBg;
            ctx.strokeStyle = hot ? PAL.sel : PAL.cornerEdge; ctx.lineWidth = hot ? 2.2 : 1.4;
            ctx.beginPath(); ctx.arc(p[0], p[1], hot ? 11 : 9, 0, TAU); ctx.fill(); ctx.stroke();
            // значок "перемещение во все стороны" - четыре стрелки
            ctx.strokeStyle = hot ? PAL.sel : PAL.cornerGlyph; ctx.lineWidth = 1.4; ctx.lineCap = 'round';
            var r = hot ? 6 : 5;
            ctx.beginPath();
            ctx.moveTo(p[0] - r, p[1]); ctx.lineTo(p[0] + r, p[1]); ctx.moveTo(p[0], p[1] - r); ctx.lineTo(p[0], p[1] + r);
            [[1, 0], [-1, 0], [0, 1], [0, -1]].forEach(function (d) {
                ctx.moveTo(p[0] + d[0] * r, p[1] + d[1] * r);
                ctx.lineTo(p[0] + d[0] * (r - 2.2) + d[1] * 2, p[1] + d[1] * (r - 2.2) + d[0] * 2);
                ctx.moveTo(p[0] + d[0] * r, p[1] + d[1] * r);
                ctx.lineTo(p[0] + d[0] * (r - 2.2) - d[1] * 2, p[1] + d[1] * (r - 2.2) - d[0] * 2);
            });
            ctx.stroke();
            ctx.restore();
        });
    }
    function cornerAt(x, y) {
        var cp = cornerPoints();
        for (var i = 0; i < cp.length; i++) { var dx = x - cp[i][0], dy = y - cp[i][1]; if (dx * dx + dy * dy <= 14 * 14) return i; }
        return -1;
    }

    function drawRack(r) {
        var a0 = r.gi + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = r.gj + (1 - FD) / 2, b1 = b0 + FD;
        var H = hz(r), s = G.s;
        var isSel = sameHit(selected, hitOf(r, -1)), isHov = sameHit(hovered, hitOf(r, -1)), lit = isSel || isHov;
        var aS = CV.aPos ? a1 : a0, bS = CV.bPos ? b1 : b0;

        // тень на полу
        poly([P(a0 - 0.05, b0 + 0.05, 0), P(a1 + 0.12, b0 + 0.05, 0), P(a1 + 0.12, b1 + 0.1, 0), P(a0 - 0.05, b1 + 0.1, 0)], 'rgba(0,0,0,0.28)');

        var fb = [P(a0, bS, H), P(a1, bS, H), P(a1, bS, 0), P(a0, bS, 0)];      // грань по оси b: дверь, если видна передняя сторона
        var fa = [P(aS, b0, H), P(aS, b1, H), P(aS, b1, 0), P(aS, b0, 0)];      // боковая грань по оси a
        var top = [P(a0, b0, H), P(a1, b0, H), P(a1, b1, H), P(a0, b1, H)];
        var hasDoor = CV.bPos;

        poly(fb, lin(0, Math.min(fb[0][1], fb[1][1]), 0, Math.max(fb[2][1], fb[3][1]), [[0, lit ? '#3a3e48' : '#2c2f37'], [1, lit ? '#262930' : '#1b1d22']]));
        poly(fa, lin(0, Math.min(fa[0][1], fa[1][1]), 0, Math.max(fa[2][1], fa[3][1]), [[0, lit ? '#2a2d34' : '#202228'], [1, '#121317']]));
        poly(top, lit ? '#4a4e5a' : '#3a3d47');

        // --- передняя перфорированная дверь (локальные координаты 120 x 300); в режиме IT дверь открыта - видно оборудование ---
        if (hasDoor && IT) {
            drawRackEq(r, fb, H);
        } else if (hasDoor) {
            ctx.save();
            vertFaceXf(fb[0], fb[1], H, 120, 300);
            ctx.strokeStyle = 'rgba(255,255,255,0.14)'; ctx.lineWidth = 1.4;
            ctx.strokeRect(5, 5, 110, 290);
            ctx.fillStyle = 'rgba(0,0,0,0.55)';
            for (var yy = 40; yy < 282; yy += 9) {
                for (var xx = 14; xx < 94; xx += 8) {
                    ctx.beginPath(); ctx.arc(xx + ((((yy - 40) / 9) | 0) % 2 ? 4 : 0), yy, 2.1, 0, TAU); ctx.fill();
                }
            }
            ctx.fillStyle = lin(98, 0, 108, 0, [[0, '#b8bcc6'], [1, '#6f7480']]);
            ctx.fillRect(100, 120, 7, 60);
            ctx.fillStyle = '#111'; ctx.beginPath(); ctx.arc(103.5, 150, 2.2, 0, TAU); ctx.fill();
            ctx.fillStyle = 'rgba(0,0,0,0.6)'; ctx.fillRect(12, 12, 96, 22);
            ctx.fillStyle = '#3dcd58'; ctx.font = '700 15px "Segoe UI", Arial, sans-serif'; ctx.textBaseline = 'middle'; ctx.textAlign = 'left';
            ctx.fillText('APC', 18, 24);
            ctx.fillStyle = 'rgba(230,232,240,0.6)'; ctx.font = '600 8px "Segoe UI", Arial, sans-serif';
            ctx.fillText('NetShelter', 58, 24);
            ctx.restore();
        } else {                                           // задняя стенка: сплошная панель с ребром жёсткости
            ctx.save();
            vertFaceXf(fb[0], fb[1], H, 120, 300);
            ctx.strokeStyle = 'rgba(255,255,255,0.08)'; ctx.lineWidth = 1.2;
            ctx.strokeRect(6, 6, 108, 288);
            ctx.fillStyle = 'rgba(255,255,255,0.05)'; ctx.fillRect(55, 6, 10, 288);
            ctx.restore();
        }

        // --- боковая панель: вертикальные рёбра ---
        ctx.save();
        vertFaceXf(fa[0], fa[1], H, 160, 300);
        ctx.strokeStyle = 'rgba(255,255,255,0.08)'; ctx.lineWidth = 1.2;
        ctx.strokeRect(8, 8, 144, 284);
        ctx.fillStyle = 'rgba(255,255,255,0.05)';
        ctx.fillRect(75, 8, 10, 284);
        ctx.restore();

        // --- верх: вентиляторная панель и название шкафа ---
        ctx.save();
        if (topFaceXf(a0, a1, b0, b1, H, 100, 130)) {
            ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillRect(12, 8, 76, 24);
            ctx.fillStyle = 'rgba(255,255,255,0.12)';
            for (var k = 0; k < 6; k++) ctx.fillRect(16 + k * 12, 11, 7, 18);
            ctx.fillStyle = 'rgba(0,0,0,0.45)'; ctx.fillRect(12, 40, 34, 28); ctx.fillRect(54, 40, 34, 28);
            var fsz = 26;
            ctx.font = '800 ' + fsz + 'px "Segoe UI", Arial, sans-serif';
            var tw = ctx.measureText(r.name).width;
            if (tw > 88) { fsz = Math.max(9, Math.floor(fsz * 88 / tw)); ctx.font = '800 ' + fsz + 'px "Segoe UI", Arial, sans-serif'; }
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillStyle = 'rgba(0,0,0,0.55)'; ctx.fillText(r.name, 51.5, 102.5);
            ctx.fillStyle = isSel ? PAL.selTxt : (isHov ? PAL.hovTxt : '#d9dcea');
            ctx.fillText(r.name, 50, 101);
        }
        ctx.restore();

        // кромки
        var edge = isSel ? PAL.sel : (isHov ? PAL.hov : 'rgba(255,255,255,0.16)');
        var elw = isSel ? 2.4 : (isHov ? 2 : 1);
        poly(fb, null, edge, elw); poly(fa, null, edge, elw); poly(top, null, edge, elw);

        // световой столб-пламя над шкафом пропорционально мощности
        if (maxPower > 0 && r.powerAvgW > 0) {
            var frac = r.powerAvgW / maxPower;
            var cx = (top[0][0] + top[2][0]) / 2, cy = (top[0][1] + top[2][1]) / 2;
            var bh = H * CV.zk * (0.35 + 0.9 * frac), bw = (a1 - a0) * s * 0.75;
            drawGlow(cx, cy, bw, bh, frac, PAL === PALS.dark, PAL.glowMode, r.id || 0);
        }

        polys.push({ r: r, u: -1, pts: boxHull(a0, a1, b0, b1, 0, H) });
        if (IT && hasDoor) eqHits(r, fb, H);
    }

    // ---- режим IT: оборудование ИТ-системы на лицевой стороне шкафа (локальные координаты грани 120 x 300) ----
    var EQ_X0 = 8, EQ_X1 = 112, EQ_Y0 = 10, EQ_Y1 = 290;
    function eqBand(r, e) {                                    // положение полосы оборудования в локальных координатах грани
        var uh = r.uHeight || 42, unit = (EQ_Y1 - EQ_Y0) / uh;
        var size = Math.max(1, e.sizeU || 1), start = Math.max(1, Math.min(uh, e.startU || 1));
        if (start + size - 1 > uh) size = uh - start + 1;
        return { y: EQ_Y0 + (uh - (start + size - 1)) * unit, h: size * unit, unit: unit };
    }
    function faceXY(fb, x, y, H) {                              // локальные (x, y) грани -> экран (с учётом зеркалирования vertFaceXf)
        var l = fb[0], rr = fb[1];
        if (l[0] > rr[0]) { l = fb[1]; rr = fb[0]; }
        return [l[0] + (rr[0] - l[0]) * x / 120, l[1] + (rr[1] - l[1]) * x / 120 + H * CV.zk * y / 300];
    }
    function drawRackEq(r, fb, H) {
        var uh = r.uHeight || 42, unit = (EQ_Y1 - EQ_Y0) / uh;
        ctx.save();
        vertFaceXf(fb[0], fb[1], H, 120, 300);
        ctx.fillStyle = '#0a0c10'; ctx.fillRect(EQ_X0, EQ_Y0, EQ_X1 - EQ_X0, EQ_Y1 - EQ_Y0);
        ctx.fillStyle = 'rgba(255,255,255,0.05)';
        if (uh <= 60) for (var q = 1; q < uh; q++) ctx.fillRect(EQ_X0, EQ_Y0 + q * unit - 0.2, EQ_X1 - EQ_X0, 0.4);
        ctx.fillStyle = '#6f7480'; ctx.fillRect(EQ_X0 - 3, EQ_Y0 - 2, 3, EQ_Y1 - EQ_Y0 + 4); ctx.fillRect(EQ_X1, EQ_Y0 - 2, 3, EQ_Y1 - EQ_Y0 + 4);
        (r.eq || []).forEach(function (e) {
            var bd = eqBand(r, e), col = eqColor(e), on = eqActive(e.key);
            var y = bd.y + 0.5, h = Math.max(1, bd.h - 1), x = EQ_X0 + 1.5, w = EQ_X1 - EQ_X0 - 3;
            ctx.globalAlpha = (e.role === 'cluster' && !on) ? 0.62 : 1;      // хосты кластера без ВМ системы - приглушены
            ctx.fillStyle = lin(0, y, 0, y + h, [[0, on ? col : '#3b4250'], [1, on ? col : '#242831']]);
            ctx.fillRect(x, y, w, h);
            ctx.fillStyle = col; ctx.fillRect(x, y, 3, h);                    // цветная метка типа слева
            if (e.kind === 'chassis') {                                       // слоты корзины и занятые блейды
                var slots = e.slots > 0 ? e.slots : 16, cols2 = slots > 8 ? 8 : slots, rows2 = Math.ceil(slots / cols2);
                var by = {}, nbl = (e.blades || []);
                nbl.forEach(function (b) { if (b && b.slot > 0) by[b.slot] = b; });
                var ax = x + 6, aw = w * 0.62, cw = aw / cols2, ch = (h - 3) / rows2;
                for (var si = 0; si < slots; si++) {
                    var bl = by[si + 1], cx2 = ax + (si % cols2) * cw, cy2 = y + 1.5 + Math.floor(si / cols2) * ch;
                    ctx.fillStyle = bl ? (on ? '#d1fae5' : '#34d399') : 'rgba(255,255,255,0.08)';
                    ctx.fillRect(cx2 + 0.4, cy2 + 0.4, Math.max(0.5, cw - 0.8), Math.max(0.5, ch - 0.8));
                }
            } else if (h > 5) {                                               // «морда» сервера/СХД: перфорация и отсеки
                ctx.fillStyle = 'rgba(0,0,0,0.38)';
                var bays = e.kind === 'storage' ? 8 : 4;
                for (var k2 = 0; k2 < bays; k2++) ctx.fillRect(x + 6 + k2 * 9, y + h * 0.2, 7, h * 0.6);
            }
            var lab = (e.name || '').replace(/\s+/g, ' ').trim();
            if (h >= 6 && lab) {
                var fs = Math.max(4, Math.min(h * 0.62, 9)), lx = x + w * (e.kind === 'chassis' ? 0.7 : 0.45);
                ctx.font = '700 ' + fs + 'px "Segoe UI", Arial, sans-serif'; ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
                var maxw = x + w - 9 - lx;
                while (lab.length > 3 && ctx.measureText(lab).width > maxw) lab = lab.slice(0, -2);
                ctx.fillStyle = 'rgba(0,0,0,0.6)'; ctx.fillText(lab, lx + 0.4, y + h / 2 + 0.4);
                ctx.fillStyle = '#f1f5f9'; ctx.fillText(lab, lx, y + h / 2);
            }
            ctx.fillStyle = e.power === 1 ? '#35e06a' : (e.power === 2 ? '#ffab1a' : '#8a919c');   // светодиод питания
            ctx.beginPath(); ctx.arc(x + w - 4, y + Math.min(h / 2, 5), Math.min(2.2, Math.max(0.9, h * 0.25)), 0, TAU); ctx.fill();
            if (on) { ctx.strokeStyle = '#fff'; ctx.lineWidth = 1.6; ctx.strokeRect(x, y, w, h); }
            else if (e.sizeGuess) { ctx.strokeStyle = 'rgba(255,255,255,0.35)'; ctx.setLineDash([2, 2]); ctx.lineWidth = 0.7; ctx.strokeRect(x, y, w, h); ctx.setLineDash([]); }
            ctx.globalAlpha = 1;
        });
        ctx.restore();
    }
    function eqHits(r, fb, H) {                                 // силуэты полос для наведения/клика (после шкафа - имеют приоритет)
        (r.eq || []).forEach(function (e) {
            var bd = eqBand(r, e);
            var pts = [faceXY(fb, EQ_X0, bd.y, H), faceXY(fb, EQ_X1, bd.y, H), faceXY(fb, EQ_X1, bd.y + bd.h, H), faceXY(fb, EQ_X0, bd.y + bd.h, H)];
            polys.push({ r: r, u: -1, eq: e.key || '', item: e, pts: pts });
        });
    }

    // ---- стеллаж: трёхмерная проекция, габариты секции как у шкафа 42U, 5 полок ----
    // Полки - тонкие рамки (без сплошной заливки), оборудование рисуется непрозрачными блоками поверх.
    // точка доступа на полке: белая "шайба" (UniFi) или малый белый корпус (Cisco), светодиод питания
    function drawShelfAP(a0, a1, b0, b1, z, h, it, lit) {
        var cx = (a0 + a1) / 2, cb = (b0 + b1) / 2, rad = Math.min(a1 - a0, b1 - b0) * 0.34, hh = Math.min(h * 0.45, G.s * 0.05);
        var round = /unifi|uap|ubiquiti/i.test(it.vd), on = it.p === 1, pts = [], q;
        for (q = 0; q < 20; q++) { var an = q / 20 * TAU; pts.push([cx + Math.cos(an) * rad * (round ? 1 : 1.25), cb + Math.sin(an) * rad * (round ? 1 : 0.95)]); }
        var top = pts.map(function (p) { return P(p[0], p[1], z + hh); });
        var bot = pts.map(function (p) { return P(p[0], p[1], z); });
        poly(bot.concat(top.slice().reverse()), '#aab1bb');
        for (q = 0; q < 20; q++) { var nn = (q + 1) % 20; poly([top[q], top[nn], bot[nn], bot[q]], '#b5bcc5'); }
        poly(top, lit ? '#ffffff' : '#eef0f3', 'rgba(0,0,0,0.3)', 0.7);
        var lc = P(cx, cb, z + hh);
        ctx.save();
        if (on) {
            var abr = ledBright((it.id || 0) * 0.37 + cx + cb);
            ctx.globalAlpha = 0.55 + 0.45 * abr;
            ctx.shadowColor = round ? '#3aa0ff' : '#35e06a';
            ctx.shadowBlur = Math.max(2, rad * G.s * 0.2 * abr);
        }
        ctx.fillStyle = on ? (round ? '#3aa0ff' : '#35e06a') : '#8a919c';
        ctx.beginPath(); ctx.arc(lc[0], lc[1], Math.max(1.5, rad * G.s * 0.14), 0, TAU); ctx.fill();
        ctx.restore();
    }
    function drawShelfItem(a0, a1, b0, b1, z, h, it, lit) {
        if (isAP(it)) { drawShelfAP(a0, a1, b0, b1, z, h, it, lit); return; }
        var acc = shelfStyle(it.vd), on = it.p === 1;
        var ia0 = a0 + 0.03, ia1 = a1 - 0.03, ib0 = b0 + 0.05, ib1 = b1 - 0.02, zt = z + h;
        var aS = CV.aPos ? ia1 : ia0, bS = CV.bPos ? ib1 : ib0;
        var front = [P(ia0, bS, zt), P(ia1, bS, zt), P(ia1, bS, z), P(ia0, bS, z)];
        var side = [P(aS, ib0, zt), P(aS, ib1, zt), P(aS, ib1, z), P(aS, ib0, z)];
        var top = [P(ia0, ib0, zt), P(ia1, ib0, zt), P(ia1, ib1, zt), P(ia0, ib1, zt)];
        poly(side, '#1e2026');
        poly(front, lin(0, Math.min(front[0][1], front[1][1]), 0, Math.max(front[2][1], front[3][1]), [[0, lit ? '#454a56' : '#353841'], [1, '#22252b']]), 'rgba(0,0,0,0.6)', 0.7);
        poly(top, lit ? '#6a7080' : '#575c69', 'rgba(255,255,255,0.25)', 0.7);
        ctx.save(); ctx.strokeStyle = acc; ctx.lineWidth = Math.max(1.4, G.s * 0.022); ctx.lineCap = 'butt';   // цветная полоса вендора по передней кромке
        ctx.beginPath(); ctx.moveTo(front[0][0], front[0][1]); ctx.lineTo(front[1][0], front[1][1]); ctx.stroke(); ctx.restore();
        if (CV.bPos) {                                   // лицевая панель видна только с передней стороны
            ctx.save();
            vertFaceXf(front[0], front[1], h, 100, 40);
            ctx.fillStyle = acc; ctx.fillRect(4, 3, 92, 4);
            if (it.k === 'chassis') {
                // слоты с блейдами (или пустые) + подпись корзины
                var slots = it.slots > 0 ? it.slots : 8;
                var cols = slots > 8 ? 8 : Math.min(slots, 4);
                var brow = Math.ceil(slots / cols);
                var byMap = {};
                (it.blades || []).forEach(function (bl) { if (bl && bl.s > 0) byMap[bl.s] = bl; });
                var gapX = 1.2, gapY = 1.2, ox = 6, oy = 9, aw = 72, ah = 22;
                var cw = (aw - gapX * (cols - 1)) / cols, chh = (ah - gapY * (brow - 1)) / brow;
                for (var si = 0; si < slots; si++) {
                    var col = si % cols, row = (si / cols) | 0;
                    var bx = ox + col * (cw + gapX), by = oy + row * (chh + gapY);
                    var bl = byMap[si + 1];
                    if (bl) {
                        ctx.fillStyle = bl.p === 1 ? '#3a4048' : '#2a2e34';
                        ctx.fillRect(bx, by, cw, chh);
                        ctx.fillStyle = bl.p === 1 ? '#35e06a' : (bl.p === 2 ? '#ffab1a' : '#6a7380');
                        ctx.fillRect(bx + cw * 0.15, by + chh * 0.2, Math.max(1.2, cw * 0.22), Math.max(1.2, chh * 0.55));
                    } else {
                        ctx.fillStyle = 'rgba(0,0,0,0.55)';
                        ctx.fillRect(bx, by, cw, chh);
                        ctx.fillStyle = 'rgba(255,255,255,0.08)';
                        for (var g = 1; g < 3; g++) ctx.fillRect(bx + cw * g / 3 - 0.4, by + 1, 0.8, chh - 2);
                    }
                }
                var label = (it.nm || it.vd || 'Корзина').replace(/\s+/g, ' ').trim();
                ctx.fillStyle = 'rgba(230,233,238,0.72)';
                ctx.font = '600 5.5px "Segoe UI", Arial, sans-serif';
                ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
                if (label.length > 18) label = label.slice(0, 17) + '…';
                ctx.fillText(label, 6, 34);
                var used = (it.blades || []).length;
                ctx.fillStyle = 'rgba(230,233,238,0.45)';
                ctx.font = '500 5px "Segoe UI", Arial, sans-serif';
                ctx.fillText(used + '/' + slots, 78, 34);
            } else {
                ctx.fillStyle = 'rgba(0,0,0,0.55)';
                var rows3 = it.k === 'storage' ? 4 : (it.k === 'tape' ? 3 : 1);
                for (var q = 0; q < rows3; q++) {
                    if (it.k === 'tape') { ctx.fillRect(8 + q * 22, 11, 20, 24); ctx.fillStyle = '#1f6fb5'; ctx.fillRect(74, 12, 18, 14); ctx.fillStyle = 'rgba(0,0,0,0.55)'; }
                    else if (it.k === 'storage') ctx.fillRect(8, 11 + q * 7, 70, 5);
                    else if (it.k === 'switch') { for (var pp = 0; pp < 16; pp++) ctx.fillRect(8 + pp * 5, 13 + (pp % 2) * 8, 4, 7); }
                    else ctx.fillRect(8, 13, 62, 20);
                }
                if (it.k === 'server') { ctx.fillStyle = 'rgba(255,255,255,0.12)'; for (var d = 0; d < 6; d++) ctx.fillRect(12 + d * 10, 18, 6, 10); }
                var nm = (it.nm || it.vd || '').replace(/\s+/g, ' ').trim();
                if (nm) {
                    ctx.fillStyle = 'rgba(230,233,238,0.55)';
                    ctx.font = '500 5px "Segoe UI", Arial, sans-serif';
                    ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
                    if (nm.length > 20) nm = nm.slice(0, 19) + '…';
                    ctx.fillText(nm, 8, 34);
                }
            }
            // p=1 включено — мерцает; p=2 спячка/выкл — статичный янтарь; иначе серый
            if (on) {
                var sbr = ledBright((it.id || 0) * 0.41 + a0 * 3 + b0);
                ctx.globalAlpha = 0.55 + 0.45 * sbr;
                ctx.shadowColor = '#35e06a'; ctx.shadowBlur = 3 + 4 * sbr;
            }
            ctx.fillStyle = on ? '#35e06a' : (it.p === 2 ? '#ffab1a' : '#8a919c');
            ctx.beginPath(); ctx.arc(88, 24, 4, 0, TAU); ctx.fill();
            ctx.restore();
        }
        poly(front, null, lit ? PAL.hov : 'rgba(255,255,255,0.22)', 0.8);
    }
    // замкнутый каркас выделения: видимые рёбра - сплошные, скрытые (за стеллажом) - тонкие полупрозрачные
    function boxWire(pj, a0, a1, b0, b1, z0, z1, aFar, bFar, color, w) {
        var aNear = aFar === a0 ? a1 : a0, bNear = bFar === b0 ? b1 : b0;
        function seg(p, q, hidden) {
            ctx.save(); ctx.strokeStyle = color; ctx.lineWidth = hidden ? Math.max(1, w * 0.5) : w; ctx.globalAlpha = hidden ? 0.45 : 1; ctx.lineCap = 'round';
            ctx.beginPath(); ctx.moveTo(p[0], p[1]); ctx.lineTo(q[0], q[1]); ctx.stroke(); ctx.restore();
        }
        var A = [a0, a1], B = [b0, b1], i, j;
        for (i = 0; i < 2; i++) for (j = 0; j < 2; j++) {                   // вертикальные рёбра
            var hv = (A[i] === aFar && B[j] === bFar);
            seg(pj(A[i], B[j], z0), pj(A[i], B[j], z1), hv);
        }
        [z0, z1].forEach(function (z) {                                    // рёбра нижнего и верхнего контура
            for (j = 0; j < 2; j++) seg(pj(a0, B[j], z), pj(a1, B[j], z), z === z0 && B[j] === bFar);
            for (i = 0; i < 2; i++) seg(pj(A[i], b0, z), pj(A[i], b1, z), z === z0 && A[i] === aFar);
        });
    }
    function drawShelfUnit(r, u, labels) {
        var isSel = sameHit(selected, hitOf(r, u)), isHov = sameHit(hovered, hitOf(r, u)), lit = isSel || isHov;
        var H = G.s * 1.55;                                      // высота как у шкафа 42U (hz)
        var post = Math.max(1.6, G.s * 0.05);
        var zs = H * 0.2, tb = H * 0.022;                         // шаг полок и толщина балки
        var a0 = r.gi + u + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = r.gj + (1 - FD) / 2, b1 = b0 + FD;
        poly([P(a0 - 0.05, b0 + 0.05, 0), P(a1 + 0.12, b0 + 0.05, 0), P(a1 + 0.12, b1 + 0.1, 0), P(a0 - 0.05, b1 + 0.1, 0)], 'rgba(0,0,0,0.28)');
        var line = function (x, y, z0, z1, col, w) {
            var p0 = P(x, y, z0), p1 = P(x, y, z1);
            ctx.strokeStyle = col; ctx.lineWidth = w; ctx.lineCap = 'round';
            ctx.beginPath(); ctx.moveTo(p0[0], p0[1]); ctx.lineTo(p1[0], p1[1]); ctx.stroke();
        };
        var pc = isSel ? PAL.sel : (isHov ? PAL.hov : '#5a7898');
        // четыре стойки: сначала дальние от наблюдателя, потом ближние
        var posts = [[a0, b0], [a1, b0], [a1, b1], [a0, b1]].map(function (p) { return { a: p[0], b: p[1], d: depthOf(p[0], p[1]) }; });
        posts.sort(function (x, y) { return x.d - y.d; });
        var medD = (posts[1].d + posts[2].d) / 2;
        posts.forEach(function (p) { if (p.d <= medD) line(p.a, p.b, 0, H, pc, post); });
        // диагональная связь на дальней стенке
        var far = posts[0], far2 = posts[1];
        var d0 = P(far.a, far.b, 0), d1 = P(far2.a, far2.b, H * 0.96);
        ctx.strokeStyle = 'rgba(70,92,118,0.6)'; ctx.lineWidth = Math.max(1, post * 0.6);
        ctx.beginPath(); ctx.moveTo(d0[0], d0[1]); ctx.lineTo(d1[0], d1[1]); ctx.stroke();
        var aS = CV.aPos ? a1 : a0, bS = CV.bPos ? b1 : b0;
        // полки снизу вверх: настил, балки, затем оборудование этой полки; вышележащий настил рисуется позже и частично прикрывает оборудование ниже
        var pk = r.packed && r.packed[u];
        if (IT) polys.push({ r: r, u: u, pts: boxHull(a0, a1, b0, b1, 0, H) });     // IT: силуэт секции - под силуэтами устройств
        for (var k = 0; k < 5; k++) {
            var z = H * 0.04 + zs * k;
            poly([P(a0, b0, z + tb), P(a1, b0, z + tb), P(a1, b1, z + tb), P(a0, b1, z + tb)], 'rgba(125,145,165,0.5)', PAL.deckEdge, 0.9);
            var fbm = [P(a0, bS, z + tb), P(a1, bS, z + tb), P(a1, bS, z), P(a0, bS, z)];
            poly(fbm, lin(0, Math.min(fbm[0][1], fbm[1][1]), 0, Math.max(fbm[2][1], fbm[3][1]), [[0, '#5a7898'], [1, '#2a3b4e']]));
            var fam = [P(aS, b0, z + tb), P(aS, b1, z + tb), P(aS, b1, z), P(aS, b0, z)];
            poly(fam, '#2f435a');
            if (pk) {
                var zb = z + tb, unitH = (zs - tb - H * 0.012) / SH_CLEAR;
                pk[k].items.forEach(function (pl) {
                    var zi = zb + pl.off * unitH, hi = pl.it.su * unitH * 0.86;
                    var litIt = lit || (IT && eqActive(pl.it.key));
                    drawShelfItem(a0, a1, b0, b1, zi, hi, pl.it, litIt);   // 0.86 - видимый зазор между устройствами
                    if (IT && pl.it.key) polys.push({ r: r, u: u, eq: pl.it.key, item: pl.it, pts: boxHull(a0 + 0.03, a1 - 0.03, b0 + 0.05, b1 - 0.02, zi, zi + hi) });
                });
            }
        }
        posts.forEach(function (p) { if (p.d > medD) line(p.a, p.b, 0, H, pc, post * 1.15); });
        labels.push({ a0: a0, a1: a1, b0: b0, b1: b1, H: H, sel: isSel, num: u + 1 });
        var hl = boxHull(a0, a1, b0, b1, 0, H);
        if (isSel || isHov) boxWire(P, a0, a1, b0, b1, 0, H, CV.aPos ? a0 : a1, CV.bPos ? b0 : b1, isSel ? PAL.sel : PAL.hov, isSel ? 2.6 : 2);
        if (!IT) polys.push({ r: r, u: u, pts: hl });
    }

    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        drawFloor();
        drawSiteLinks();
        polys = [];
        // порядок рисования - по глубине относительно наблюдателя (дальние первыми)
        var order = items3d.slice().sort(function (x, y) { return depthOf(x.ca, x.cb) - depthOf(y.ca, y.cb) || x.ca - y.ca; });
        var labels = [];
        order.forEach(function (it) { if (it.shelfUnit >= 0) drawShelfUnit(it.r, it.shelfUnit, labels); else drawRack(it.r); });
        // подпись "СТЕЛЛАЖ N" на верхнем ободе каждой секции (N - порядок наполнения), поверх всего
        labels.forEach(function (L) {
            var txt = 'СТЕЛЛАЖ ' + L.num;
            ctx.save();
            if (topFaceXf(L.a0, L.a1, L.b0, L.b1, L.H, 100, 130)) {
                ctx.font = '800 17px "Segoe UI", Arial, sans-serif'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                ctx.fillStyle = PAL.labelShadow; ctx.fillText(txt, 51, 66);
                ctx.fillStyle = L.sel ? PAL.selTxt : PAL.labelText; ctx.fillText(txt, 50, 65);
            }
            ctx.restore();
        });
        drawSiteLabels();
        drawCorners();
    }

    // ---- hit-test ----
    function inPoly(pt, pts) {
        var inside = false;
        for (var i = 0, j = pts.length - 1; i < pts.length; j = i++) {
            var xi = pts[i][0], yi = pts[i][1], xj = pts[j][0], yj = pts[j][1];
            if (((yi > pt[1]) !== (yj > pt[1])) && (pt[0] < (xj - xi) * (pt[1] - yi) / (yj - yi) + xi)) inside = !inside;
        }
        return inside;
    }
    function pick(x, y) {
        for (var i = polys.length - 1; i >= 0; i--) { // последние нарисованы поверх
            if (inPoly([x, y], polys[i].pts)) return hitOf(polys[i].r, polys[i].u, polys[i].eq, polys[i].item);
        }
        return null;
    }

    function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }

    function showTip(hit, x, y) {
        var r = hit.r, st = hit.u >= 0 ? unitStats(r, hit.u) : null;
        var title = r.shelf ? ('Стеллаж ' + (hit.u + 1) + ' (вне шкафов)') : ('Шкаф ' + esc(r.name));
        var h = '<b class="dv-name">' + title + '</b><table>';
        h += '<tr><td>Высота</td><td>' + r.uHeight + 'U</td></tr>';
        if (st) {
            h += '<tr><td>Серверов</td><td><b>' + st.servers + '</b></td></tr>';
            h += '<tr><td>Корзин</td><td><b>' + st.chassis + '</b></td></tr>';
            h += '<tr><td>СХД</td><td><b>' + st.storages + '</b></td></tr>';
            if (st.tapes) h += '<tr><td>Ленточных библиотек</td><td><b>' + st.tapes + '</b></td></tr>';
            if (st.network) h += '<tr><td>Сетевого оборудования</td><td><b>' + st.network + '</b></td></tr>';
            var pk = r.packed && r.packed[hit.u], listed = [];
            if (pk) pk.forEach(function (lv) {
                (lv.items || []).forEach(function (pl) {
                    var it = pl.it, kindLab = { server: 'Сервер', chassis: 'Корзина', storage: 'СХД', 'switch': 'Сеть', tape: 'Лента' }[it.k] || it.k;
                    var title = it.nm || it.vd || kindLab;
                    if (it.k === 'chassis') {
                        var used = (it.blades || []).length, slots = it.slots || 0;
                        title += slots ? (' (' + used + '/' + slots + ' бл.)') : '';
                    }
                    listed.push('<tr><td colspan="2" style="padding-top:0.25rem">' + esc(kindLab) + ': <b>' + esc(title) + '</b></td></tr>');
                });
            });
            if (listed.length) h += listed.join('');
        } else {
            h += '<tr><td>Серверов</td><td><b>' + r.servers + '</b>' + (r.blades ? ' (+ блейдов ' + r.blades + ')' : '') + '</td></tr>';
            h += '<tr><td>Корзин</td><td><b>' + r.chassis + '</b></td></tr>';
            h += '<tr><td>СХД</td><td><b>' + r.storages + '</b></td></tr>';
            if (r.tapes) h += '<tr><td>Ленточных библиотек</td><td><b>' + r.tapes + '</b></td></tr>';
            if (r.network) h += '<tr><td>Сетевого оборудования</td><td><b>' + r.network + '</b></td></tr>';
            h += '<tr><td>Мощность</td><td>' + (r.powerAvgW > 0 ? '<b>' + kw(r.powerAvgW) + '</b> (ср. 24ч)' : '<b>нет данных</b>') + '</td></tr>';
            if (r.powerPeakW > 0) h += '<tr><td>Пик 24ч</td><td>' + kw(r.powerPeakW) + '</td></tr>';
            if (r.powerRatedW > 0) h += '<tr><td>Паспорт</td><td>' + kw(r.powerRatedW) + '</td></tr>';
        }
        h += '</table><div class="dv-hint">Детали ' + (r.shelf ? 'стеллажа' : 'шкафа') + ' - справа</div>';
        tip.innerHTML = h;
        tip.hidden = false;
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var px = x + 16, py = y + 16;
        if (px + tw > G.w - 4) px = x - tw - 16;
        if (py + th > G.h - 4) py = G.h - th - 4;
        tip.style.left = Math.max(4, px) + 'px';
        tip.style.top = Math.max(4, py) + 'px';
    }

    // deep-link: ?rack=&hw= (сервер) или ?rack=&kind=&eq= (любой тип оборудования)
    var qsFocus = new URLSearchParams(location.search);
    var focusRackId = parseInt(qsFocus.get('rack') || '0', 10) || 0;
    var focusEqId = parseInt(qsFocus.get('eq') || qsFocus.get('hw') || '0', 10) || 0;
    var focusKind = (qsFocus.get('kind') || (qsFocus.get('hw') ? 'server' : '') || '').toLowerCase();

    var selTimer = null;
    function selectHit(hit) {
        if (!hit) return;
        if (selTimer) { clearTimeout(selTimer); selTimer = null; }
        if (IT) { selected = hit; render(); return; }                // IT: справа список, шкаф во фрейме не открываем
        if (sameHit(selected, hit) && document.getElementById('dvFrame').getAttribute('src') !== 'about:blank') return;
        selected = hit;
        var r = hit.r, st = hit.u >= 0 ? unitStats(r, hit.u) : null;
        var title = r.shelf ? ('Стеллаж ' + (hit.u + 1)) : esc(r.name);
        var pinEq = focusEqId > 0 && focusRackId === r.id ? focusEqId : 0;
        var pinKind = pinEq ? (focusKind || 'server') : '';
        // при подсветке конкретного устройства на стеллаже - все секции (иначе может быть в другой секции)
        var shelfQ = (r.shelf && !pinEq) ? ('shelfUnit=' + (hit.u + 1)) : '';
        var pinQ = '';
        if (pinEq) {
            if (pinKind === 'server') pinQ = 'hw=' + pinEq;
            else pinQ = 'kind=' + encodeURIComponent(pinKind) + '&eq=' + pinEq;
        }
        var url = '/sprsrv/rackcabinetview/' + r.id + (shelfQ || pinQ ? '?' : '') + shelfQ + (pinQ ? ((shelfQ ? '&' : '') + pinQ) : '');
        var emb = '/sprsrv/rackcabinetview/' + r.id + '?embed=1' + (shelfQ ? '&' + shelfQ : '') + (pinQ ? '&' + pinQ : '');
        var sel = document.getElementById('dvSel');
        var t = '<b>' + title + '</b><span class="dv-sep">&middot;</span>' + r.uHeight + 'U';
        if (st) {
            t += '<span class="dv-sep">&middot;</span>серверов ' + st.servers
                + '<span class="dv-sep">&middot;</span>корзин ' + st.chassis
                + (st.tapes ? '<span class="dv-sep">&middot;</span>лент. библиотек ' + st.tapes : '')
                + (st.network ? '<span class="dv-sep">&middot;</span>сеть ' + st.network : '')
                + '<span class="dv-sep">&middot;</span>СХД ' + st.storages;
        } else {
            t += '<span class="dv-sep">&middot;</span>серверов ' + r.servers + (r.blades ? '+' + r.blades + ' бл.' : '')
                + '<span class="dv-sep">&middot;</span>корзин ' + r.chassis + (r.tapes ? '<span class="dv-sep">&middot;</span>лент. библиотек ' + r.tapes : '') + (r.network ? '<span class="dv-sep">&middot;</span>сеть ' + r.network : '')
                + '<span class="dv-sep">&middot;</span>СХД ' + r.storages
                + '<span class="dv-sep">&middot;</span>' + (r.powerAvgW > 0 ? '<b>' + kw(r.powerAvgW) + '</b>' + (r.powerPeakW > 0 ? ' (пик ' + kw(r.powerPeakW) + ')' : '') : 'мощность н/д');
        }
        t += '<span class="dv-sep">&middot;</span><a href="' + url + '" target="_blank" rel="noopener">открыть</a>';
        sel.innerHTML = t;
        document.getElementById('dvFrame').src = emb;
        render();
    }

    // ---- масштаб колесом мыши с индикатором в процентах ----
    var zoomInd = document.getElementById('dvZoomInd'), zoomIndTimer = null;
    function showZoomInd() {
        if (!zoomInd) return;
        zoomInd.textContent = 'Масштаб: ' + Math.round(cam.zoom * 100) + '%';
        zoomInd.classList.add('dv-zi-on');
        clearTimeout(zoomIndTimer);
        zoomIndTimer = setTimeout(function () { zoomInd.classList.remove('dv-zi-on'); }, 1000);
    }
    function zoomAt(f, px, py) {
        var nz = Math.max(0.3, Math.min(6, cam.zoom * f));
        f = nz / cam.zoom;
        // точка под курсором остаётся на месте: смещение экранных координат относительно центра окна масштабируется
        cam.panX = px - G.w / 2 - (px - G.w / 2 - cam.panX) * f;
        cam.panY = py - G.h / 2 - (py - G.h / 2 - cam.panY) * f;
        cam.zoom = nz;
        setCamera(); tip.hidden = true; render(); showZoomInd();
    }
    canvas.addEventListener('wheel', function (e) {
        e.preventDefault();
        var rect = canvas.getBoundingClientRect();
        var dy = e.deltaMode === 1 ? e.deltaY * 16 : e.deltaY;
        zoomAt(Math.exp(-dy * 0.0015), e.clientX - rect.left, e.clientY - rect.top);
    }, { passive: false });

    // ---- изменение ракурса: тянем за угол основания; корректируем поворот и наклон так, чтобы угол следовал за курсором ----
    function cornerScreen(i, yaw, pitch) {
        var sYaw = cam.yaw, sPitch = cam.pitch;
        cam.yaw = yaw; cam.pitch = pitch; setCamera();
        var p = cornerPoints()[i];
        cam.yaw = sYaw; cam.pitch = sPitch; setCamera();
        return p;
    }
    var PITCH_MIN = 12 * Math.PI / 180, PITCH_MAX = 80 * Math.PI / 180;
    function rotateByCornerDrag(i, dx, dy) {
        var eps = 0.01, p0 = cornerScreen(i, cam.yaw, cam.pitch);
        var py = cornerScreen(i, cam.yaw + eps, cam.pitch), pp = cornerScreen(i, cam.yaw, cam.pitch + eps);
        var j11 = (py[0] - p0[0]) / eps, j21 = (py[1] - p0[1]) / eps, j12 = (pp[0] - p0[0]) / eps, j22 = (pp[1] - p0[1]) / eps;
        var det = j11 * j22 - j12 * j21, dyaw, dpitch;
        if (Math.abs(det) > 1e-3) { dyaw = (dx * j22 - dy * j12) / det; dpitch = (j11 * dy - j21 * dx) / det; }
        else { dyaw = -dx * 0.008; dpitch = dy * 0.004; }
        var lim = 0.2;
        dyaw = Math.max(-lim, Math.min(lim, dyaw)); dpitch = Math.max(-lim, Math.min(lim, dpitch));
        cam.yaw += dyaw;
        cam.pitch = Math.max(PITCH_MIN, Math.min(PITCH_MAX, cam.pitch + dpitch));
        if (cam.yaw > Math.PI * 2) cam.yaw -= Math.PI * 2; else if (cam.yaw < 0) cam.yaw += Math.PI * 2;
        setCamera();
    }
    function resetView() { focusedGroup = null; cam.yaw = YAW0; cam.pitch = PITCH0; cam.zoom = 1; cam.panX = 0; cam.panY = 0; setCamera(); render(); showZoomInd(); }

    // ---- мышь: поворот за угол, прокрутка за пустое место, наведение/выбор шкафа ----
    var dragMode = null, dragCorner = -1, dragStart = null, lastPt = null, dragMoved = false;
    canvas.addEventListener('mousedown', function (e) {
        if (e.button !== 0) return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        tip.hidden = true;
        var ci = cornerAt(x, y);
        dragStart = { x: x, y: y }; lastPt = { x: x, y: y }; dragMoved = false;
        if (ci >= 0) { dragMode = 'rotate'; dragCorner = ci; canvas.style.cursor = 'move'; }
        else if (!pick(x, y)) { dragMode = 'pan'; canvas.style.cursor = 'grabbing'; }
        else dragMode = 'click';
        e.preventDefault();
    });
    window.addEventListener('mousemove', function (e) {
        if (!dragMode || dragMode === 'click') return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        if (!dragMoved && Math.abs(x - dragStart.x) + Math.abs(y - dragStart.y) > 3) dragMoved = true;
        if (!dragMoved) return;
        var dx = x - lastPt.x, dy = y - lastPt.y;
        lastPt = { x: x, y: y };
        if (dragMode === 'rotate') rotateByCornerDrag(dragCorner, dx, dy);
        else { cam.panX += dx; cam.panY += dy; setCamera(); }
        render();
    });
    window.addEventListener('mouseup', function (e) {
        if (!dragMode) return;
        var mode = dragMode, moved = dragMoved;
        dragMode = null; dragCorner = -1;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        canvas.style.cursor = cornerAt(x, y) >= 0 ? 'move' : 'default';
        if (mode === 'click' && !moved) {
            var h = pick(x, y);
            if (IT) { itClick(h, x, y); }
            else if (h) selectHit(h);
        }
        else if (IT && mode === 'pan' && !moved) {
            var sg = siteLabelAt(x, y);
            if (sg) focusGroup(sg); else itClick(null, x, y);
        }
        else render();
    });
    canvas.addEventListener('mousemove', function (e) {
        if (dragMode && dragMode !== 'click') return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        var ci = cornerAt(x, y);
        var h = ci >= 0 ? null : pick(x, y);
        var changed = false;
        if (ci !== hoverCorner) { hoverCorner = ci; changed = true; }
        if (IT) {                                                    // IT: наведение на оборудование - расширенная карточка
            if (!sameHit(h, hovered)) { hovered = h; changed = true; }
            itHover(h, x, y);
        } else if (!sameHit(h, hovered)) {
            hovered = h; changed = true;
            // объект под курсором становится выбранным (с небольшой задержкой, чтобы
            // при быстром движении мыши не загружать фрейм для каждой секции)
            if (selTimer) { clearTimeout(selTimer); selTimer = null; }
            if (h && !sameHit(h, selected)) selTimer = setTimeout(function () { selTimer = null; if (sameHit(hovered, h)) selectHit(h); }, 150);
        }
        if (changed) render();
        canvas.style.cursor = ci >= 0 ? 'move' : ((h || (IT && siteLabelAt(x, y))) ? 'pointer' : 'grab');
        canvas.title = ci >= 0 ? 'Потяните за угол, чтобы изменить ракурс' : '';
    });
    canvas.addEventListener('mouseleave', function () { hovered = null; hoverCorner = -1; if (selTimer) { clearTimeout(selTimer); selTimer = null; } tip.hidden = true; if (IT) itHideSoon(); if (!dragMode) render(); });
    canvas.addEventListener('dblclick', function (e) {
        var rect = canvas.getBoundingClientRect();
        if (cornerAt(e.clientX - rect.left, e.clientY - rect.top) >= 0 || !pick(e.clientX - rect.left, e.clientY - rect.top)) resetView();
    });
    document.getElementById('dvResetView').addEventListener('click', resetView);

    // масштаб шкафа в правой части меняется колесом мыши внутри фрейма; индикатор масштаба показывает сам фрейм (всплывающий)

    var dvFrameEl = document.getElementById('dvFrame');
    if (dvFrameEl) dvFrameEl.addEventListener('load', pushThemeToFrame);
    var glowRaf = 0, hasPowerGlow = maxPower > 0;
    function resize() { layout(); render(); }
    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    function glowFrame(now) {
        glowRaf = 0;
        glowT = (now || performance.now()) * 0.001;
        render();
        // столбы мощности и/или мерцание светодиодов на полках
        glowRaf = requestAnimationFrame(glowFrame);
    }
    layout();
    requestAnimationFrame(glowFrame);

    // ---- режим IT: карточка оборудования при наведении, список справа ----
    var itCard = document.getElementById('itCard'), itListBody = document.getElementById('itListBody');
    var itHideT = 0, itShowT = 0, itPos = { x: 0, y: 0 }, itCardKey = '', itCardOver = false, itPinned = false, pinnedEq = '';
    function itKv(it) { return it.rows || []; }
    function itCardHtml(h) {
        var it = h.item || {}, r = h.r;
        var kind = it.kind || it.k || '', name = it.name || it.nm || '';
        var vend = ((it.vendor || '') + ' ' + (it.model || '')).trim() || it.vd || '';
        var pw = it.power != null ? it.power : it.p;
        var where = [];
        if (r.shelf) where.push('Стеллаж ' + (h.u + 1) + ' (вне шкафов)'); else where.push('Шкаф ' + esc(r.name));
        if (!r.shelf && it.startU > 0) where.push('U' + it.startU + (it.sizeU > 1 ? '–' + (it.startU + it.sizeU - 1) : '') + (it.sizeGuess ? ' (условно)' : ''));
        var sg = groupById(r.siteId);
        if (sg) where.unshift('<b>' + esc(sg.name) + '</b>');
        var role = it.role || it.r || '';
        var s = '<b class="dv-name">' + esc(name) + '</b><div style="color:#9fb0c2;font-size:0.76rem">' + esc(ROLE_LABEL[role] || KIND_LABEL[kind] || kind) + (vend ? ' &middot; ' + esc(vend) : '') + '</div>';
        s += '<div style="margin-top:0.2rem">' + where.join(' &middot; ') + '</div><table>';
        s += '<tr><td>Питание</td><td>' + (pw === 1 ? 'включен' : (pw === 2 ? 'выключен / спячка' : 'нет данных')) + '</td></tr>';
        itKv(it).forEach(function (kv) { if (kv && kv.v) s += '<tr><td>' + esc(kv.k) + '</td><td>' + esc(kv.v) + '</td></tr>'; });
        s += '</table>';
        if ((it.links || []).length) {
            s += '<div class="it-lk">';
            it.links.forEach(function (l) { s += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + '</a>'; });
            s += '</div>';
        }
        return s;
    }
    function showItCard(h, x, y) {
        if (!itCard || !h || !h.item) return;
        itCardKey = h.eq;
        itCard.innerHTML = itCardHtml(h);
        itCard.hidden = false;
        var tw = itCard.offsetWidth, th = itCard.offsetHeight, px = x + 14, py = y + 8;
        if (px + tw > G.w - 4) px = x - tw - 14;
        if (py + th > G.h - 4) py = G.h - th - 4;
        itCard.style.left = Math.max(4, px) + 'px';
        itCard.style.top = Math.max(4, py) + 'px';
    }
    function itHideCard() { if (itCard) itCard.hidden = true; itCardKey = ''; itPinned = false; }
    function itHideSoon() {
        clearTimeout(itShowT); clearTimeout(itHideT);
        itHideT = setTimeout(function () { if (!itPinned && !itCardOver) itHideCard(); }, 350);
    }
    function itHover(h, x, y) {
        itPos = { x: x, y: y };
        var key = h && h.eq ? h.eq : '';
        if (!key) { itHideSoon(); return; }
        clearTimeout(itHideT);
        if (key === itCardKey && itCard && !itCard.hidden) return;
        clearTimeout(itShowT);
        itShowT = setTimeout(function () {
            if (hovered && hovered.eq === key) { itPinned = false; showItCard(hovered, itPos.x, itPos.y); markList(key, false); }
        }, itCard && itCard.hidden ? 60 : 220);
    }
    function itClick(h, x, y) {
        if (h && h.eq) {
            clearTimeout(itShowT); clearTimeout(itHideT);
            pinnedEq = selEq = h.eq;
            showItCard(h, x, y); itPinned = true;
            markList(h.eq, true);
        } else {
            pinnedEq = selEq = ''; itHideCard(); markList('', false);
        }
        render();
    }
    if (itCard) {
        itCard.addEventListener('mouseenter', function () { itCardOver = true; clearTimeout(itHideT); });
        itCard.addEventListener('mouseleave', function () { itCardOver = false; itHideSoon(); });
    }
    function markList(key, scroll) {
        if (!itListBody) return;
        var first = null;
        Array.prototype.forEach.call(itListBody.querySelectorAll('.it-li'), function (el) {
            var on = !!key && el.getAttribute('data-eq') === key;
            el.classList.toggle('it-on', on);
            if (on && !first) first = el;
        });
        if (first && scroll) first.scrollIntoView({ block: 'nearest' });
    }
    function groupById(id) { for (var i = 0; i < groups.length; i++) if (groups[i].id === id) return groups[i]; return null; }
    // чипы площадок над сценой: клик - камера на площадку
    (function () {
        var box = document.getElementById('itSiteChips');
        if (!box) return;
        if (!groups.length) return;
        var lab = document.createElement('span'); lab.className = 'text-muted small'; lab.textContent = 'Площадки:'; box.appendChild(lab);
        groups.forEach(function (g) {
            var b = document.createElement('button');
            b.type = 'button'; b.className = 'btn btn-sm btn-outline-secondary rounded-pill';
            b.textContent = g.name + ' (' + (g.equip || 0) + ')';
            b.addEventListener('click', function () { focusGroup(g); });
            box.appendChild(b);
        });
        if (groups.length > 1) {
            var all = document.createElement('button');
            all.type = 'button'; all.className = 'btn btn-sm btn-outline-primary rounded-pill'; all.textContent = 'Все площадки';
            all.addEventListener('click', focusAll);
            box.appendChild(all);
        }
    })();
    // список справа: площадка -> тип (серверы, хосты по кластерам, СХД, ВМ); без размещения - в конце
    function buildItList() {
        if (!itListBody) return;
        var eq = IT.equipment || [];
        if (!eq.length) { itListBody.innerHTML = '<div class="p-3 text-muted small">Для ИТ-системы не найдено серверов, ВМ и СХД</div>'; return; }
        var kinds = ['server', 'host', 'storage', 'vm'], titles = { server: 'Серверы', host: 'Хосты ESXi', storage: 'СХД', vm: 'Виртуальные машины' };
        var html = '';
        function itemHtml(e) {
            var idx = eq.indexOf(e);
                var loc = e.rackName ? ((e.siteName ? esc(e.siteName) + ' / ' : '') + esc(e.rackName))
                    : (e.kind === 'vm' ? (e.subtitle || 'хост не определён') : (e.kind === 'host' ? 'шкаф хоста не указан' : 'размещение не найдено'));
            var col = ROLE_COLOR[e.role] || KIND_COLOR[e.kind] || '#999';
            var s = '<div class="it-li" data-i="' + idx + '" data-site="' + (e.siteId || 0) + '" data-eq="' + esc(e.eqKey || '') + '">'
                + '<div class="it-nm"><i style="display:inline-block;width:0.6rem;height:0.6rem;border-radius:2px;margin-right:0.4rem;background:' + col + '"></i>' + esc(e.name) + '</div>'
                + '<div class="it-sub">' + esc(e.subtitle || '') + '</div><div class="it-sub">' + loc + '</div>'
                + '<div class="it-det">';
            var rows = (e.rows || []).filter(function (kv) { return kv && kv.v; });
            if (rows.length) { s += '<dl>'; rows.forEach(function (kv) { s += '<dt>' + esc(kv.k) + '</dt><dd>' + esc(kv.v) + '</dd>'; }); s += '</dl>'; }
            s += '<div class="it-lk">';
            (e.links || []).forEach(function (l) { s += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + '</a>'; });
            return s + '</div></div></div>';
        }
        function section(title, siteId, list) {
            if (!list.length) return;
            html += '<div class="it-grp" data-site="' + siteId + '">' + esc(title) + ' (' + list.length + ')</div>';
            kinds.forEach(function (kd) {
                var grp = list.filter(function (e) { return e.kind === kd; });
                if (!grp.length) return;
                html += '<div class="it-sub2">' + titles[kd] + ' - ' + grp.length + '</div>';
                if (kd === 'host') {
                    grp.sort(function (a, b) { return String(a.cluster || '').localeCompare(b.cluster || '') || String(a.name).localeCompare(b.name); });
                    var cur = null;
                    grp.forEach(function (e) {
                        if (e.cluster !== cur) { cur = e.cluster; html += '<div class="it-sub2 it-cl">Кластер: ' + esc(cur || 'не указан') + '</div>'; }
                        html += itemHtml(e);
                    });
                } else grp.forEach(function (e) { html += itemHtml(e); });
            });
        }
        groups.forEach(function (g) { section('Площадка ' + g.name, g.id, eq.filter(function (e) { return e.siteId === g.id; })); });
        var orphan = eq.filter(function (e) { return !(e.siteId > 0) || !groupById(e.siteId); });
        var onHost = orphan.filter(function (e) { return e.kind === 'vm' && e.eqKey; });
        var rest = orphan.filter(function (e) { return !(e.kind === 'vm' && e.eqKey); });
        section('ВМ на хосте без шкафа', 0, onHost);
        section('Без размещения в шкафах', 0, rest);
        itListBody.innerHTML = html;
        Array.prototype.forEach.call(itListBody.querySelectorAll('.it-grp'), function (el) {
            el.style.cursor = 'pointer';
            el.addEventListener('click', function () { var g = groupById(parseInt(el.getAttribute('data-site'), 10)); if (g) focusGroup(g); });
        });
        Array.prototype.forEach.call(itListBody.querySelectorAll('.it-li'), function (el) {
            var key = el.getAttribute('data-eq') || '';
            el.addEventListener('mouseenter', function () { if (key) { selEq = key; render(); } });
            el.addEventListener('mouseleave', function () { if (selEq !== pinnedEq) { selEq = pinnedEq; render(); } });
            el.addEventListener('click', function (ev) {
                if (ev.target.tagName === 'A') return;
                el.classList.toggle('it-open');
                pinnedEq = selEq = key; markList(key, false);
                var g = groupById(parseInt(el.getAttribute('data-site'), 10));
                if (g && key && focusedGroup !== g) focusGroup(g); else render();
            });
        });
    }

    // deep-link ?rack= — выделить шкаф/стеллаж; иначе первый шкаф с оборудованием / секция стеллажа
    var focusCab = focusRackId > 0 ? allRacks.filter(function (r) { return r.id === focusRackId; })[0] : null;
    if (IT) {
        buildItList();
        var itFocus = qsFocus.get('focus') || '';
        if (itFocus) {
            pinnedEq = selEq = itFocus; markList(itFocus, true);
            Array.prototype.forEach.call(itListBody ? itListBody.querySelectorAll('.it-li.it-on') : [], function (el) { el.classList.add('it-open'); });
            render();
        }
    } else if (focusCab) {
        selectHit(focusCab.shelf ? hitOf(focusCab, 0) : hitOf(focusCab, -1));
    } else {
        var firstRack = racks.filter(function (r) { return r.servers + r.chassis + r.storages > 0; })[0] || racks[0];
        var firstShelf = shelves[0];
        if (firstRack) selectHit(hitOf(firstRack, -1));
        else if (firstShelf && firstShelf.units > 0) selectHit(hitOf(firstShelf, 0));
        else if (firstShelf) selectHit(hitOf(firstShelf, 0));
    }
})();
</script>
