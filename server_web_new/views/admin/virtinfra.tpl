<div class="hero-text mb-3">
    <h2>Виртуальная инфраструктура</h2>
    <p class="text-muted small mb-0">
        Иерархия vCenter → Datacenter → Cluster → Host. Наведите курсор на элемент, чтобы увидеть его структуру
        и параметры (для кластера — количество хостов/ВМ, для хоста — CPU/память).
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}

<div class="infra-canvas-shell border rounded mb-3">
    <div class="infra-canvas-toolbar d-flex flex-wrap align-items-center gap-3 px-3 py-2 border-bottom">
        <button type="button" class="btn btn-sm btn-outline-secondary" id="virtInfraBack" disabled>&larr; Все vCenter</button>
        <div class="d-flex flex-wrap align-items-center gap-2 infra-legend">
            <span class="small text-muted me-1">Легенда:</span>
            <span class="infra-legend-item"><img src="/static/infra2/icons/esxi-host.svg" alt=""> Хост ESXi</span>
        </div>
        <div class="ms-auto small text-muted" id="virtInfraStatus">Загрузка…</div>
    </div>

    <div class="infra-canvas-viewport position-relative">
        <canvas id="virtinfra-canvas" width="1280" height="720"></canvas>
        <div id="virtinfra-tooltip" class="infra-canvas-tooltip d-none" role="tooltip">
            <div class="infra-tooltip-title fw-semibold"></div>
            <div class="infra-tooltip-body small"></div>
        </div>
    </div>
</div>

<div id="virtinfra-data" class="d-none" data-payload-b64="{{.VirtInfraJSON}}"></div>

<script>
(function () {
    'use strict';

    // См. комментарий в hwinfra.tpl: atob() даёт бинарную строку байт, а не
    // UTF-8 текст - без TextDecoder JSON.parse ломает кириллицу.
    function b64ToUtf8(b64) {
        var binary = atob(b64);
        var bytes = new Uint8Array(binary.length);
        for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
        return new TextDecoder('utf-8').decode(bytes);
    }
    var raw = document.getElementById('virtinfra-data').getAttribute('data-payload-b64') || '';
    var DATA = { vcenters: [], stats: {} };
    try { DATA = JSON.parse(b64ToUtf8(raw)); } catch (e) { console.error('virtinfra: bad payload', e); }
    DATA.vcenters = DATA.vcenters || [];

    var canvas = document.getElementById('virtinfra-canvas');
    var ctx = canvas.getContext('2d', { alpha: false });
    var tooltip = document.getElementById('virtinfra-tooltip');
    var statusEl = document.getElementById('virtInfraStatus');
    var backBtn = document.getElementById('virtInfraBack');
    var viewport = document.querySelector('.infra-canvas-viewport');

    var ICON_HOST = new Image();
    ICON_HOST.src = '/static/infra2/icons/esxi-host.svg';
    ICON_HOST.onload = function () { draw(); };

    var dpr = window.devicePixelRatio || 1;
    var cssW = 1280, cssH = 720;
    var hitRegions = [];

    // Стек навигации: 'root' -> 'vcenter' -> 'datacenter' (кластеры с хостами показаны сразу на уровне datacenter).
    var stack = [{ view: 'root' }];

    function resize() {
        if (!viewport) return;
        var w = Math.max(640, viewport.clientWidth - 4);
        var h = Math.max(420, Math.floor(w * 9 / 16));
        cssW = w; cssH = h;
        dpr = window.devicePixelRatio || 1;
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        draw();
    }
    window.addEventListener('resize', resize);

    function cssVarColors() {
        var dark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
        return {
            bg: dark ? '#1a1d20' : '#ffffff',
            card: dark ? '#22262a' : '#f8f9fa',
            border: dark ? '#3a3f44' : '#dee2e6',
            text: dark ? '#e9ecef' : '#212529',
            muted: dark ? '#adb5bd' : '#6c757d',
            accent: dark ? '#4dabf7' : '#0d6efd',
            ok: '#2fb344',
            warn: '#f0ad4e',
            crit: '#dc3545'
        };
    }

    function roundRect(x, y, w, h, r) {
        ctx.beginPath();
        ctx.moveTo(x + r, y);
        ctx.arcTo(x + w, y, x + w, y + h, r);
        ctx.arcTo(x + w, y + h, x, y + h, r);
        ctx.arcTo(x, y + h, x, y, r);
        ctx.arcTo(x, y, x + w, y, r);
        ctx.closePath();
    }
    function trunc(s, n) {
        s = String(s || '');
        return s.length > n ? s.slice(0, n - 1) + '…' : s;
    }
    function typeWorkLabel(tw) {
        return tw === 1 ? 'продуктовая среда' : tw === 2 ? 'внутренний тест' : tw === 3 ? 'внешний тест' : 'не указано';
    }

    function drawCardsGrid(items, cardW, cardH, renderCard) {
        var C = cssVarColors();
        ctx.fillStyle = C.bg;
        ctx.fillRect(0, 0, cssW, cssH);
        hitRegions = [];
        if (!items.length) {
            ctx.fillStyle = C.muted;
            ctx.font = '14px system-ui, sans-serif';
            ctx.fillText('Нет данных.', 16, 30);
            return;
        }
        var pad = 16, gap = 16;
        var perRow = Math.max(1, Math.floor((cssW - pad) / (cardW + gap)));
        var x = pad, y = pad, col = 0, rowH = cardH;
        items.forEach(function (item, idx) {
            renderCard(item, x, y, cardW, cardH, C);
            col++;
            if (col >= perRow) { col = 0; x = pad; y += rowH + gap; } else { x += cardW + gap; }
        });
    }

    function draw() {
        var top = stack[stack.length - 1];
        if (top.view === 'root') {
            drawCardsGrid(DATA.vcenters, 260, 130, function (vc, x, y, w, h, C) {
                var dcCount = (vc.datacenters || []).length;
                var clusterCount = 0, hostCount = 0, vmCount = 0;
                (vc.datacenters || []).forEach(function (dc) {
                    clusterCount += (dc.clusters || []).length;
                    (dc.clusters || []).forEach(function (cl) { hostCount += cl.hostCount || 0; vmCount += cl.vmCount || 0; });
                });
                roundRect(x, y, w, h, 10);
                ctx.fillStyle = C.card; ctx.fill();
                ctx.strokeStyle = C.border; ctx.stroke();
                ctx.fillStyle = C.text;
                ctx.font = '600 15px system-ui, sans-serif';
                ctx.fillText('vCenter: ' + trunc(vc.name, 22), x + 12, y + 24);
                ctx.font = '13px system-ui, sans-serif';
                [
                    'Datacenter: ' + dcCount,
                    'Кластеров: ' + clusterCount,
                    'Хостов: ' + hostCount + ', ВМ: ' + vmCount
                ].forEach(function (line, i) { ctx.fillText(line, x + 12, y + 48 + i * 20); });
                ctx.fillStyle = C.accent;
                ctx.font = '12px system-ui, sans-serif';
                ctx.fillText('Открыть →', x + 12, y + h - 12);
                hitRegions.push({
                    x: x, y: y, w: w, h: h,
                    onClick: function () { stack.push({ view: 'vcenter', vc: vc }); backBtn.disabled = false; draw(); },
                    tooltip: { title: vc.name, lines: ['Datacenter: ' + dcCount, 'Кластеров: ' + clusterCount, 'Хостов: ' + hostCount, 'ВМ: ' + vmCount] }
                });
            });
            statusEl.textContent = DATA.vcenters.length + ' vCenter';
        } else if (top.view === 'vcenter') {
            drawCardsGrid(top.vc.datacenters || [], 260, 120, function (dc, x, y, w, h, C) {
                var clusterCount = (dc.clusters || []).length, hostCount = 0, vmCount = 0;
                (dc.clusters || []).forEach(function (cl) { hostCount += cl.hostCount || 0; vmCount += cl.vmCount || 0; });
                roundRect(x, y, w, h, 10);
                ctx.fillStyle = C.card; ctx.fill();
                ctx.strokeStyle = C.border; ctx.stroke();
                ctx.fillStyle = C.text;
                ctx.font = '600 15px system-ui, sans-serif';
                ctx.fillText('Datacenter: ' + trunc(dc.name, 20), x + 12, y + 24);
                ctx.font = '13px system-ui, sans-serif';
                [
                    'Кластеров: ' + clusterCount,
                    'Хостов: ' + hostCount + ', ВМ: ' + vmCount
                ].forEach(function (line, i) { ctx.fillText(line, x + 12, y + 48 + i * 20); });
                ctx.fillStyle = C.accent;
                ctx.font = '12px system-ui, sans-serif';
                ctx.fillText('Открыть →', x + 12, y + h - 12);
                hitRegions.push({
                    x: x, y: y, w: w, h: h,
                    onClick: function () { stack.push({ view: 'datacenter', vc: top.vc, dc: dc }); draw(); },
                    tooltip: { title: dc.name, lines: ['Кластеров: ' + clusterCount, 'Хостов: ' + hostCount, 'ВМ: ' + vmCount] }
                });
            });
            statusEl.textContent = top.vc.name;
        } else if (top.view === 'datacenter') {
            drawClustersWithHosts(top.dc.clusters || []);
            statusEl.textContent = top.vc.name + ' / ' + top.dc.name;
        }
    }

    function drawClustersWithHosts(clusters) {
        var C = cssVarColors();
        ctx.fillStyle = C.bg;
        ctx.fillRect(0, 0, cssW, cssH);
        hitRegions = [];
        if (!clusters.length) {
            ctx.fillStyle = C.muted;
            ctx.font = '14px system-ui, sans-serif';
            ctx.fillText('Нет кластеров.', 16, 30);
            return;
        }
        var pad = 16, y = pad;
        clusters.forEach(function (cl) {
            var hosts = cl.hosts || [];
            var rowH = 34;
            var hostsPerRow = Math.max(1, Math.floor((cssW - pad - 160) / 150));
            var hostRows = Math.max(1, Math.ceil(hosts.length / hostsPerRow));
            var blockH = rowH + hostRows * 50 + 12;

            ctx.fillStyle = C.text;
            ctx.font = '600 13px system-ui, sans-serif';
            var twColor = cl.typeWork === 1 ? C.ok : cl.typeWork === 2 ? C.warn : C.muted;
            ctx.fillText('⬤', pad, y + 16);
            ctx.fillStyle = twColor;
            ctx.fillText('⬤', pad, y + 16);
            ctx.fillStyle = C.text;
            ctx.fillText('Кластер: ' + cl.name + '  (хостов: ' + (cl.hostCount || 0) + ', ВМ: ' + (cl.vmCount || 0) + ')', pad + 16, y + 16);

            hitRegions.push({
                x: pad, y: y, w: cssW - pad * 2, h: rowH,
                tooltip: {
                    title: cl.name,
                    lines: ['Тип: ' + typeWorkLabel(cl.typeWork), 'Хостов: ' + (cl.hostCount || 0), 'ВМ: ' + (cl.vmCount || 0)]
                }
            });

            var hx = pad + 16, hy = y + rowH, col = 0;
            hosts.forEach(function (h) {
                var hw = 145, hh = 44;
                roundRect(hx, hy, hw, hh, 6);
                ctx.fillStyle = C.card; ctx.fill();
                ctx.strokeStyle = h.powerOn === false ? C.muted : C.ok;
                ctx.lineWidth = 1.3;
                ctx.stroke();
                if (ICON_HOST.complete) ctx.drawImage(ICON_HOST, hx + 4, hy + 4, 22, 16);
                ctx.fillStyle = C.text;
                ctx.font = '10px system-ui, sans-serif';
                ctx.fillText(trunc(h.name, 16), hx + 30, hy + 14);
                ctx.fillStyle = C.muted;
                ctx.font = '9px system-ui, sans-serif';
                var cpuMem = (h.cpuCount ? h.cpuCount + ' CPU' : '') + (h.memSizeGb ? ', ' + Math.round(h.memSizeGb) + ' ГБ' : '');
                ctx.fillText(trunc(cpuMem || '—', 20), hx + 4, hy + 32);

                hitRegions.push({
                    x: hx, y: hy, w: hw, h: hh,
                    tooltip: {
                        title: h.name + ' (host)',
                        lines: [
                            h.serialNum ? ('S/N: ' + h.serialNum) : null,
                            h.cpuCount ? ('CPU: ' + h.cpuCount + ' (логических: ' + (h.cpuLogical || 0) + ')') : null,
                            h.memSizeGb ? ('Память: ' + Math.round(h.memSizeGb) + ' ГБ') : null,
                            'Питание: ' + (h.powerOn ? 'включён' : 'выключен'),
                            'Кластер: ' + cl.name
                        ].filter(Boolean)
                    }
                });

                col++;
                if (col >= hostsPerRow) { col = 0; hx = pad + 16; hy += hh + 6; } else { hx += hw + 6; }
            });

            y += blockH + 10;
        });
    }

    function regionAt(px, py) {
        for (var i = hitRegions.length - 1; i >= 0; i--) {
            var r = hitRegions[i];
            if (px >= r.x && px <= r.x + r.w && py >= r.y && py <= r.y + r.h) return r;
        }
        return null;
    }
    function hideTooltip() { tooltip.classList.add('d-none'); }
    function showTooltip(r, clientX, clientY) {
        if (!r.tooltip) { hideTooltip(); return; }
        var titleEl = tooltip.querySelector('.infra-tooltip-title');
        var bodyEl = tooltip.querySelector('.infra-tooltip-body');
        titleEl.textContent = r.tooltip.title || '';
        bodyEl.innerHTML = '';
        (r.tooltip.lines || []).forEach(function (line) {
            var div = document.createElement('div');
            div.textContent = line;
            bodyEl.appendChild(div);
        });
        var rect = viewport.getBoundingClientRect();
        tooltip.style.left = (clientX - rect.left + 14) + 'px';
        tooltip.style.top = (clientY - rect.top + 14) + 'px';
        tooltip.classList.remove('d-none');
    }

    canvas.addEventListener('mousemove', function (ev) {
        var rect = canvas.getBoundingClientRect();
        var px = ev.clientX - rect.left, py = ev.clientY - rect.top;
        var r = regionAt(px, py);
        canvas.style.cursor = r && r.onClick ? 'pointer' : 'default';
        if (r) showTooltip(r, ev.clientX, ev.clientY); else hideTooltip();
    });
    canvas.addEventListener('mouseleave', hideTooltip);
    canvas.addEventListener('click', function (ev) {
        var rect = canvas.getBoundingClientRect();
        var px = ev.clientX - rect.left, py = ev.clientY - rect.top;
        var r = regionAt(px, py);
        if (r && r.onClick) { hideTooltip(); r.onClick(); }
    });

    backBtn.addEventListener('click', function () {
        if (stack.length > 1) stack.pop();
        backBtn.disabled = stack.length <= 1;
        hideTooltip();
        draw();
    });

    resize();
})();
</script>
