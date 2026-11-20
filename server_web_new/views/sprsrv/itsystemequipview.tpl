<style>
    .it-system-dropdown {
        z-index: 1000;
        max-height: 280px;
        overflow-y: auto;
        box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, .25);
    }
    .it-system-dropdown .list-group-item { cursor: pointer; }
    .it-system-dropdown .list-group-item small {
        display: block;
        color: var(--bs-secondary-color);
    }
    .itse-layout {
        display: grid;
        grid-template-columns: minmax(0, 1.35fr) minmax(280px, 0.75fr);
        gap: 0.75rem;
        height: calc(100vh - 280px);
        min-height: 520px;
    }
    @media (max-width: 1100px) {
        .itse-layout { grid-template-columns: 1fr; height: auto; }
        .itse-viz { height: 55vh; }
        .itse-list { max-height: 50vh; }
    }
    .itse-viz {
        position: relative;
        min-height: 0;
        border-radius: 0.6rem;
        overflow: hidden;
        border: 1px solid var(--bs-border-color);
        background: linear-gradient(#171d25, #0e1318);
    }
    [data-bs-theme="light"] .itse-viz {
        background: linear-gradient(#e8eef5, #d5dee8);
    }
    .itse-viz canvas { display: block; width: 100%; height: 100%; }
    .itse-list {
        min-height: 0;
        overflow: auto;
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        background: var(--bs-body-bg);
    }
    .itse-list-item {
        padding: 0.55rem 0.75rem;
        border-bottom: 1px solid var(--bs-border-color);
        cursor: pointer;
    }
    .itse-list-item:hover, .itse-list-item.active {
        background: var(--bs-tertiary-bg);
    }
    .itse-list-item .itse-kind {
        font-size: 0.7rem;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: var(--bs-secondary-color);
    }
    .itse-card {
        position: absolute;
        z-index: 20;
        min-width: 260px;
        max-width: 360px;
        padding: 0.75rem 0.9rem;
        border-radius: 0.55rem;
        border: 1px solid var(--bs-border-color);
        background: var(--bs-body-bg);
        box-shadow: 0 0.75rem 1.5rem rgba(0,0,0,.28);
        pointer-events: auto;
    }
    .itse-card h6 { margin: 0 0 0.35rem; font-size: 0.95rem; }
    .itse-card dl { margin: 0; font-size: 0.8rem; }
    .itse-card dt { color: var(--bs-secondary-color); font-weight: 500; }
    .itse-card dd { margin: 0 0 0.25rem; word-break: break-all; }
    .itse-card .itse-links { margin-top: 0.45rem; display: flex; flex-direction: column; gap: 0.2rem; }
    .itse-card .itse-links a { font-size: 0.8rem; }
    .itse-empty {
        display: flex; align-items: center; justify-content: center;
        min-height: 280px; color: var(--bs-secondary-color);
        border: 1px dashed var(--bs-border-color); border-radius: 0.6rem;
        background: var(--bs-tertiary-bg);
    }
</style>

<div class="hero-text mb-3">
    <h2>Оборудование ИТ-системы</h2>
    <p class="text-muted small mb-0">
        Площадки и шкафы с оборудованием выбранной ИТ-системы (серверы и СХД по LUN ВМ / по имени аппаратного сервера).
        {{if .HasSystem}}Система: <strong>{{.SystemName}}</strong> ({{.ITSystemKey}}).{{end}}
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form id="itseFilter" method="GET" class="hw-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-12">
            <label class="form-label">ИТ система</label>
            <div class="row g-2">
                <div class="col-md-4">
                    <input type="text" name="ITSystemKey" id="ITSystemKey" class="form-control"
                        value="{{.ITSystemKey}}" placeholder="CMDB-" autocomplete="off">
                    <div id="ITSystemKeyHint" class="form-text"></div>
                </div>
                <div class="col-md-6 position-relative">
                    <input type="text" id="ITSystemSearch" class="form-control"
                        placeholder="Поиск по справочнику: код, название, статус..." autocomplete="off">
                    <div id="ITSystemSearchResults" class="list-group position-absolute w-100 d-none it-system-dropdown"></div>
                </div>
                <div class="col-md-2 d-flex gap-2">
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
        </div>
    </div>
    {{if .HasSystem}}
    <div class="row mt-2">
        <div class="col-12 text-muted small" id="itseSummary">
            Площадок: {{.Stats.Sites}} · Шкафов: {{.Stats.Racks}} ·
            Серверов: {{.Stats.Servers}} · ВМ: {{.Stats.VMs}} · СХД: {{.Stats.Storages}}
        </div>
    </div>
    {{end}}
</form>

{{if not .HasSystem}}
<div class="itse-empty">Выберите ИТ-систему и нажмите «Показать»</div>
{{else}}
<div class="itse-layout" id="itseLayout">
    <div class="itse-viz" id="itseViz">
        <canvas id="itseCanvas"></canvas>
        <div id="itseCard" class="itse-card d-none"></div>
    </div>
    <div class="itse-list" id="itseList">
        <div class="p-2 border-bottom fw-semibold small">Оборудование ИТ-системы</div>
        <div id="itseListBody"></div>
    </div>
</div>
{{end}}

<script>
(function () {
    // Тот же initITSystemPicker, что в view2.tpl / updatebase.tpl.
    function initITSystemPicker() {
        var keyInput = document.getElementById('ITSystemKey');
        var hint = document.getElementById('ITSystemKeyHint');
        var searchInput = document.getElementById('ITSystemSearch');
        var results = document.getElementById('ITSystemSearchResults');
        var dataList = document.getElementById('ITSystemDataList');
        if (!keyInput || !searchInput || !results || !dataList) return;

        var items = Array.prototype.map.call(dataList.querySelectorAll('.it-system-item'), function (el) {
            return {
                key: el.getAttribute('data-key') || '',
                label: el.getAttribute('data-label') || '',
                full: el.getAttribute('data-full') || '',
                status: el.getAttribute('data-status') || '',
                type: el.getAttribute('data-type') || '',
                biz: el.getAttribute('data-biz') || '',
                cfo: el.getAttribute('data-cfo') || ''
            };
        });

        function findByKey(key) {
            if (!key) return null;
            var lower = key.toLowerCase();
            for (var i = 0; i < items.length; i++) {
                if (items[i].key.toLowerCase() === lower) return items[i];
            }
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
            if (match) {
                hint.textContent = describe(match) + (match.status ? ' (' + match.status + ')' : '');
                hint.classList.remove('text-danger');
            } else if (keyInput.value.trim() !== '') {
                hint.textContent = 'Код не найден в справочнике';
                hint.classList.add('text-danger');
            } else {
                hint.textContent = '';
                hint.classList.remove('text-danger');
            }
        }
        function hideResults() {
            results.classList.add('d-none');
            results.innerHTML = '';
        }
        function escapeHtml(s) {
            var div = document.createElement('div');
            div.textContent = s;
            return div.innerHTML;
        }
        function renderResults(query) {
            var q = query.trim().toLowerCase();
            if (q === '') { hideResults(); return; }
            var matches = items.filter(function (item) {
                return item.key.toLowerCase().indexOf(q) !== -1 ||
                    item.label.toLowerCase().indexOf(q) !== -1 ||
                    item.full.toLowerCase().indexOf(q) !== -1 ||
                    item.status.toLowerCase().indexOf(q) !== -1 ||
                    item.type.toLowerCase().indexOf(q) !== -1 ||
                    item.biz.toLowerCase().indexOf(q) !== -1 ||
                    item.cfo.toLowerCase().indexOf(q) !== -1;
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
                row.innerHTML = '<div>' + escapeHtml(item.key) + (item.label ? ' — ' + escapeHtml(item.label) : '') + '</div>' +
                    (item.full ? '<small>' + escapeHtml(item.full) + '</small>' : '');
                row.addEventListener('click', function () {
                    keyInput.value = item.key;
                    searchInput.value = describe(item);
                    updateHint();
                    hideResults();
                });
                results.appendChild(row);
            });
            results.classList.remove('d-none');
        }
        searchInput.addEventListener('input', function () { renderResults(searchInput.value); });
        searchInput.addEventListener('focus', function () {
            if (searchInput.value.trim() !== '') renderResults(searchInput.value);
        });
        document.addEventListener('click', function (evt) {
            if (evt.target !== searchInput && !results.contains(evt.target)) hideResults();
        });
        keyInput.addEventListener('input', updateHint);
        keyInput.addEventListener('blur', updateHint);
        var initial = findByKey(keyInput.value.trim());
        if (initial) searchInput.value = describe(initial);
        updateHint();
    }

    document.addEventListener('DOMContentLoaded', initITSystemPicker);

    var DATA = {{.ITJSON}};
    if (!DATA || !DATA.sites) return;

    var canvas = document.getElementById('itseCanvas');
    var viz = document.getElementById('itseViz');
    var cardEl = document.getElementById('itseCard');
    var listBody = document.getElementById('itseListBody');
    if (!canvas || !viz) return;

    var ctx = canvas.getContext('2d');
    var dpr = window.devicePixelRatio || 1;
    var hitRegions = [];
    var hovered = null;
    var selectedKey = null;

    var KIND_COLOR = {
        server: '#38bdf8',
        storage: '#a78bfa',
        chassis: '#34d399',
        vm: '#fbbf24'
    };
    var KIND_LABEL = {
        server: 'Сервер',
        storage: 'СХД',
        chassis: 'Корзина',
        vm: 'ВМ'
    };

    function itemKey(it) { return (it.kind || '') + ':' + (it.id || 0); }

    function buildList() {
        if (!listBody) return;
        listBody.innerHTML = '';
        (DATA.equipment || []).forEach(function (eq) {
            var div = document.createElement('div');
            div.className = 'itse-list-item';
            div.dataset.key = itemKey(eq);
            div.innerHTML =
                '<div class="itse-kind">' + escapeHtml(KIND_LABEL[eq.kind] || eq.kind) + '</div>' +
                '<div class="fw-semibold">' + escapeHtml(eq.name || '') + '</div>' +
                '<div class="small text-muted">' + escapeHtml(eq.subtitle || '') +
                (eq.siteName ? ' · ' + escapeHtml(eq.siteName) : '') +
                (eq.rackName ? ' / ' + escapeHtml(eq.rackName) : '') + '</div>';
            div.addEventListener('mouseenter', function () {
                showCardFromEquip(eq, null);
                selectedKey = itemKey(eq);
                highlightList();
                render();
            });
            div.addEventListener('mouseleave', function () {
                hideCardSoon();
            });
            div.addEventListener('click', function () {
                selectedKey = itemKey(eq);
                highlightList();
                showCardFromEquip(eq, null);
                render();
            });
            listBody.appendChild(div);
        });
        if (!(DATA.equipment || []).length) {
            listBody.innerHTML = '<div class="p-3 text-muted small">Нет записей</div>';
        }
    }

    function highlightList() {
        if (!listBody) return;
        Array.prototype.forEach.call(listBody.querySelectorAll('.itse-list-item'), function (el) {
            el.classList.toggle('active', el.dataset.key === selectedKey);
        });
    }

    function escapeHtml(s) {
        var d = document.createElement('div');
        d.textContent = s == null ? '' : String(s);
        return d.innerHTML;
    }

    function showCardFromEquip(eq, anchor) {
        if (!cardEl || !eq) return;
        var rows = eq.rows || [];
        var links = eq.links || [];
        var html = '<h6>' + escapeHtml(eq.name || '') + '</h6>';
        html += '<div class="small text-muted mb-1">' + escapeHtml(eq.subtitle || KIND_LABEL[eq.kind] || eq.kind) + '</div>';
        if (eq.siteName || eq.rackName) {
            html += '<div class="small mb-2">' + escapeHtml([eq.siteName, eq.rackName].filter(Boolean).join(' / ')) + '</div>';
        }
        if (rows.length) {
            html += '<dl>';
            rows.forEach(function (r) {
                if (!r || !r.v) return;
                html += '<dt>' + escapeHtml(r.k) + '</dt><dd>' + escapeHtml(r.v) + '</dd>';
            });
            html += '</dl>';
        }
        if (links.length) {
            html += '<div class="itse-links">';
            links.forEach(function (l) {
                html += '<a href="' + escapeHtml(l.url) + '" target="_blank" rel="noopener">' + escapeHtml(l.title) + '</a>';
            });
            html += '</div>';
        }
        cardEl.innerHTML = html;
        cardEl.classList.remove('d-none');
        placeCard(anchor);
        clearTimeout(cardEl._hideT);
    }

    function showCardFromItem(it, rack, site, anchor) {
        showCardFromEquip({
            kind: it.kind, id: it.id, name: it.name,
            subtitle: (KIND_LABEL[it.kind] || it.kind) + (it.model ? ' · ' + it.model : ''),
            siteName: site ? site.name : '', rackName: rack ? rack.name : '',
            rows: it.rows || [], links: it.links || []
        }, anchor);
    }

    function placeCard(anchor) {
        if (!cardEl || cardEl.classList.contains('d-none')) return;
        var vw = viz.clientWidth, vh = viz.clientHeight;
        var cw = cardEl.offsetWidth || 280, ch = cardEl.offsetHeight || 160;
        var x = 16, y = 16;
        if (anchor) {
            x = Math.min(anchor.x + 14, vw - cw - 8);
            y = Math.min(Math.max(8, anchor.y - 20), vh - ch - 8);
            if (x < 8) x = 8;
        }
        cardEl.style.left = x + 'px';
        cardEl.style.top = y + 'px';
    }

    function hideCardSoon() {
        clearTimeout(cardEl._hideT);
        cardEl._hideT = setTimeout(function () {
            if (!cardEl.matches(':hover')) cardEl.classList.add('d-none');
        }, 180);
    }

    cardEl.addEventListener('mouseenter', function () { clearTimeout(cardEl._hideT); });
    cardEl.addEventListener('mouseleave', hideCardSoon);

    function resize() {
        var w = viz.clientWidth, h = viz.clientHeight;
        canvas.width = Math.max(1, Math.floor(w * dpr));
        canvas.height = Math.max(1, Math.floor(h * dpr));
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        render();
    }

    function themeDark() {
        return document.documentElement.getAttribute('data-bs-theme') !== 'light';
    }

    function render() {
        var w = viz.clientWidth, h = viz.clientHeight;
        hitRegions = [];
        ctx.clearRect(0, 0, w, h);
        var dark = themeDark();
        var text = dark ? '#e2e8f0' : '#0f172a';
        var muted = dark ? '#94a3b8' : '#64748b';
        var rackBg = dark ? '#1e293b' : '#f1f5f9';
        var rackBorder = dark ? '#334155' : '#94a3b8';
        var emptyU = dark ? '#0f172a' : '#e2e8f0';

        var sites = DATA.sites || [];
        if (!sites.length) {
            ctx.fillStyle = muted;
            ctx.font = '14px system-ui,sans-serif';
            ctx.fillText('Нет площадок с размещённым оборудованием этой ИТ-системы', 24, 40);
            return;
        }

        var pad = 16;
        var y = pad;
        var rackW = 72;
        var rackGap = 18;
        var labelH = 22;
        var siteGap = 28;

        sites.forEach(function (site) {
            ctx.fillStyle = text;
            ctx.font = '600 14px system-ui,sans-serif';
            ctx.fillText(site.name || ('Площадка ' + site.id), pad, y + 14);
            y += labelH + 6;

            var racks = site.racks || [];
            var x = pad;
            var maxH = 0;
            racks.forEach(function (rack) {
                var uH = Math.max(1, rack.uHeight || 42);
                var unitPx = Math.min(8, Math.max(3, (h * 0.45) / uH));
                var rh = uH * unitPx + 18;
                if (rh > maxH) maxH = rh;

                if (x + rackW > w - pad) {
                    x = pad;
                    y += maxH + rackGap;
                    maxH = rh;
                }

                // cabinet body
                ctx.fillStyle = rackBg;
                ctx.strokeStyle = rackBorder;
                ctx.lineWidth = 1.5;
                roundRect(ctx, x, y, rackW, uH * unitPx + 8, 4);
                ctx.fill();
                ctx.stroke();

                // empty units
                for (var u = 0; u < uH; u++) {
                    var uy = y + 4 + (uH - 1 - u) * unitPx;
                    ctx.fillStyle = emptyU;
                    ctx.fillRect(x + 4, uy + 0.5, rackW - 8, unitPx - 1);
                }

                (rack.items || []).forEach(function (it) {
                    var start = it.startU || 0;
                    var size = it.sizeU || 1;
                    if (start <= 0) {
                        // без позиции — полоска сверху
                        start = uH;
                        size = Math.min(2, uH);
                    }
                    size = Math.max(1, size);
                    var topU = start + size - 1;
                    if (topU > uH) topU = uH;
                    if (start < 1) start = 1;
                    var iy = y + 4 + (uH - topU) * unitPx;
                    var ih = size * unitPx - 1;
                    var col = KIND_COLOR[it.kind] || '#f472b6';
                    var lit = selectedKey === itemKey(it) || (hovered && hovered.key === itemKey(it));
                    ctx.fillStyle = lit ? col : shade(col, dark ? 0.75 : 0.85);
                    ctx.fillRect(x + 5, iy, rackW - 10, ih);
                    ctx.strokeStyle = lit ? '#fff' : col;
                    ctx.lineWidth = lit ? 2 : 1;
                    ctx.strokeRect(x + 5, iy, rackW - 10, ih);

                    hitRegions.push({
                        x: x + 4, y: iy, w: rackW - 8, h: ih,
                        key: itemKey(it), item: it, rack: rack, site: site
                    });
                });

                ctx.fillStyle = muted;
                ctx.font = '11px system-ui,sans-serif';
                var label = rack.name || '';
                if (label.length > 12) label = label.slice(0, 11) + '…';
                ctx.fillText(label, x, y + uH * unitPx + 20);

                x += rackW + rackGap;
            });
            y += maxH + siteGap;
        });
    }

    function shade(hex, f) {
        var m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        if (!m) return hex;
        var r = Math.round(parseInt(m[1], 16) * f);
        var g = Math.round(parseInt(m[2], 16) * f);
        var b = Math.round(parseInt(m[3], 16) * f);
        return 'rgb(' + r + ',' + g + ',' + b + ')';
    }

    function roundRect(c, x, y, w, h, r) {
        c.beginPath();
        c.moveTo(x + r, y);
        c.arcTo(x + w, y, x + w, y + h, r);
        c.arcTo(x + w, y + h, x, y + h, r);
        c.arcTo(x, y + h, x, y, r);
        c.arcTo(x, y, x + w, y, r);
        c.closePath();
    }

    function hitTest(mx, my) {
        for (var i = hitRegions.length - 1; i >= 0; i--) {
            var r = hitRegions[i];
            if (mx >= r.x && mx <= r.x + r.w && my >= r.y && my <= r.y + r.h) return r;
        }
        return null;
    }

    canvas.addEventListener('mousemove', function (ev) {
        var rect = canvas.getBoundingClientRect();
        var mx = ev.clientX - rect.left, my = ev.clientY - rect.top;
        var h = hitTest(mx, my);
        if (h) {
            hovered = h;
            selectedKey = h.key;
            highlightList();
            showCardFromItem(h.item, h.rack, h.site, { x: h.x + h.w, y: h.y });
            render();
        } else if (hovered) {
            hovered = null;
            hideCardSoon();
            render();
        }
    });
    canvas.addEventListener('mouseleave', function () {
        hovered = null;
        hideCardSoon();
        render();
    });

    buildList();
    resize();
    window.addEventListener('resize', resize);
    if (window.MutationObserver) {
        new MutationObserver(render).observe(document.documentElement, { attributes: true, attributeFilter: ['data-bs-theme'] });
    }
})();
</script>
