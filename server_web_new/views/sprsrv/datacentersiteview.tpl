<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Оборудование ЦОД {{.SiteName}}</h2>
        <div class="text-muted" id="dvSummary">Шкафов: {{.RackCount}}</div>
    </div>
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacentersitelist">К справочнику площадок</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<div class="dv-layout">
    <div class="dv-left">
        <div class="dv-sel" id="dvSel">Наведите курсор на шкаф</div>
        <div class="dv-stage" id="dvStage">
            <canvas id="dvCanvas"></canvas>
            <div id="dvTip" class="dv-tip" hidden></div>
        </div>
    </div>
    <div class="dv-right">
        <iframe id="dvFrame" class="dv-frame" title="Шкаф" src="about:blank"></iframe>
    </div>
</div>

<style>
    .dv-layout { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr); gap: 0.75rem; height: calc(100vh - 170px); min-height: 520px; }
    @media (max-width: 1100px) { .dv-layout { grid-template-columns: 1fr; height: auto; } .dv-left { height: 60vh; } .dv-frame { height: 80vh; } }
    .dv-left { display: flex; flex-direction: column; gap: 0.4rem; }
    .dv-left, .dv-right { min-height: 0; }
    .dv-sel { flex: 0 0 auto; padding: 0.35rem 0.7rem; border-radius: 0.5rem; border: 1px solid var(--bs-border-color); background: var(--bs-tertiary-bg); font-size: 0.9rem; }
    .dv-sel .dv-sep { color: var(--bs-secondary-color); margin: 0 0.35rem; }
    .dv-stage { position: relative; width: 100%; flex: 1 1 auto; min-height: 0; border-radius: 0.6rem; overflow: hidden; border: 1px solid var(--bs-border-color); background: #17144f; }
    #dvCanvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; }
    .dv-right { display: flex; flex-direction: column; border: 1px solid var(--bs-border-color); border-radius: 0.6rem; background: var(--bs-body-bg); overflow: hidden; }
    .dv-right-head { padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--bs-border-color); }
    .dv-frame { flex: 1 1 auto; width: 100%; border: 0; min-height: 0; background: var(--bs-body-bg); }
    .dv-tip { position: absolute; z-index: 5; pointer-events: none; min-width: 190px; padding: 0.55rem 0.7rem; border-radius: 0.5rem; font-size: 0.82rem; line-height: 1.35;
              background: rgba(14, 13, 48, 0.94); color: #e8e8ff; border: 1px solid rgba(120, 130, 255, 0.45); box-shadow: 0 6px 22px rgba(0, 0, 0, 0.45); }
    .dv-tip b.dv-name { font-size: 0.95rem; color: #fff; }
    .dv-tip table { border-collapse: collapse; margin-top: 0.25rem; }
    .dv-tip td { padding: 1px 0; vertical-align: top; }
    .dv-tip td:first-child { color: #a9aef0; padding-right: 0.7rem; white-space: nowrap; }
    .dv-tip .dv-hint { margin-top: 0.35rem; color: #8f94d8; font-size: 0.75rem; }
</style>

<script>
(function () {
    'use strict';

    var DATA = {{.DCJSON}};
    var canvas = document.getElementById('dvCanvas');
    if (!DATA || !canvas || !DATA.racks || !DATA.racks.length) return;

    var stage = document.getElementById('dvStage');
    var tip = document.getElementById('dvTip');
    var ctx = canvas.getContext('2d');
    var racks = DATA.racks;
    var TAU = Math.PI * 2;

    // ---- сводка ----
    var kw = function (w) { return w >= 1000 ? (w / 1000).toFixed(1) + ' кВт' : Math.round(w) + ' Вт'; };
    (function () {
        var s = 'Шкафов: <b>' + racks.length + '</b>';
        if (DATA.powerAvgW > 0) s += ' &middot; мощность (средняя 24ч): <b>' + kw(DATA.powerAvgW) + '</b>';
        if (DATA.powerPeakW > 0) s += ' &middot; пик: <b>' + kw(DATA.powerPeakW) + '</b>';
        document.getElementById('dvSummary').innerHTML = s;
    })();

    // ---- раскладка шкафов по сетке ----
    var n = racks.length;
    var cols = Math.max(1, Math.ceil(Math.sqrt(n * 1.3)));
    var rows = Math.ceil(n / cols);
    var maxPower = 0;
    racks.forEach(function (r, k) {
        r.gi = k % cols; r.gj = Math.floor(k / cols);
        if (r.powerAvgW > maxPower) maxPower = r.powerAvgW;
    });
    var order = racks.slice().sort(function (a, b) { return (a.gi + a.gj) - (b.gi + b.gj) || a.gi - b.gi; });

    var FW = 0.60, FD = 0.80;          // размеры шкафа в клетках (ширина 600, глубина 1000 при клетке 1.2 м)
    var selected = null, hovered = null;
    var polys = [];                    // силуэты для hit-test: {r, pts}
    var G = { w: 0, h: 0, dpr: 1, s: 1, ox: 0, oy: 0 };

    // изометрия: a - вдоль "вправо-вниз", b - вдоль "влево-вниз"
    function P(a, b, z) { return [(a - b) * G.s, (a + b) * G.s / 2 - z]; }
    function hz(r) { return G.s * 1.55 * Math.max(0.5, Math.min(1.4, r.uHeight / 42)); }

    function layout() {
        var rect = stage.getBoundingClientRect();
        G.dpr = window.devicePixelRatio || 1;
        G.w = Math.max(200, rect.width); G.h = Math.max(200, rect.height);
        canvas.width = Math.round(G.w * G.dpr); canvas.height = Math.round(G.h * G.dpr);
        // масштаб по габаритам сцены
        var spanX = (cols + rows + 0.6), spanY = (cols + rows) / 2 + 1.55 * 1.4 + 0.9;
        G.s = Math.min(G.w * 0.94 / spanX, G.h * 0.9 / spanY);
        G.ox = G.w / 2 + (rows - cols) * G.s / 2;
        G.oy = G.h - 0.5 * G.s - ((cols + rows) / 2) * G.s - 0.2 * G.s + ((cols + rows) / 2 + 0.2) * G.s * 0; // пересчитывается ниже
        // поставим сцену по центру по вертикали
        var topY = -(1.55 * 1.4) * G.s - 0.6 * G.s;
        var botY = ((cols + rows) / 2) * G.s + 0.3 * G.s;
        G.oy = (G.h - (botY - topY)) / 2 - topY;
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

    function drawFloor() {
        // фон
        ctx.fillStyle = lin(0, 0, 0, G.h, [[0, '#1b1763'], [1, '#100e3a']]);
        ctx.fillRect(0, 0, G.w, G.h);
        // плита пола
        var m = 0.35;
        var f = [P(-m, -m, 0), P(cols + m, -m, 0), P(cols + m, rows + m, 0), P(-m, rows + m, 0)];
        // P(a,b): a - по "cols"? у нас gi вдоль a, gj вдоль b
        f = [P(-m, -m, 0), P(cols + m, -m, 0), P(cols + m, rows + m, 0), P(-m, rows + m, 0)];
        ctx.save();
        ctx.translate(G.ox, G.oy);
        poly(f, lin(0, f[0][1], 0, f[2][1], [[0, '#2b2585'], [1, '#1d1a63']]), 'rgba(110,200,255,0.55)', 1.5);
        // сетка плиток
        ctx.strokeStyle = 'rgba(140,150,255,0.18)'; ctx.lineWidth = 1;
        for (var i = 0; i <= cols; i++) { var a = P(i, 0, 0), b = P(i, rows, 0); ctx.beginPath(); ctx.moveTo(a[0], a[1]); ctx.lineTo(b[0], b[1]); ctx.stroke(); }
        for (var j = 0; j <= rows; j++) { var c = P(0, j, 0), d = P(cols, j, 0); ctx.beginPath(); ctx.moveTo(c[0], c[1]); ctx.lineTo(d[0], d[1]); ctx.stroke(); }
        ctx.restore();
    }

    // локальная система грани: origin (screen), ux/uy - вектор на Wl, vx/vy - вектор на Hl
    function faceTransform(o, ux, uy, vx, vy, Wl, Hl) {
        ctx.transform(ux / Wl, uy / Wl, vx / Hl, vy / Hl, o[0], o[1]);
    }

    function drawRack(r) {
        var a0 = r.gi + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = r.gj + (1 - FD) / 2, b1 = b0 + FD;
        var H = hz(r), s = G.s;
        var isSel = selected === r, isHov = hovered === r;

        // тень на полу
        poly([P(a0 - 0.05, b0 + 0.05, 0), P(a1 + 0.12, b0 + 0.05, 0), P(a1 + 0.12, b1 + 0.1, 0), P(a0 - 0.05, b1 + 0.1, 0)], 'rgba(0,0,0,0.28)');

        var left = [P(a0, b1, H), P(a1, b1, H), P(a1, b1, 0), P(a0, b1, 0)];    // передняя дверь
        var right = [P(a1, b1, H), P(a1, b0, H), P(a1, b0, 0), P(a1, b1, 0)];   // боковая панель
        var top = [P(a0, b0, H), P(a1, b0, H), P(a1, b1, H), P(a0, b1, H)];

        var lit = isHov || isSel;
        poly(left, lin(0, left[0][1], 0, left[3][1], [[0, lit ? '#3a3e48' : '#2c2f37'], [1, lit ? '#262930' : '#1b1d22']]));
        poly(right, lin(0, right[0][1], 0, right[3][1], [[0, lit ? '#2a2d34' : '#202228'], [1, '#121317']]));
        poly(top, lit ? '#4a4e5a' : '#3a3d47');

        // --- передняя перфорированная дверь (локальные координаты 120 x 300) ---
        ctx.save();
        faceTransform(left[0], (a1 - a0) * s, (a1 - a0) * s / 2, 0, H, 120, 300);
        // рамка двери
        ctx.strokeStyle = 'rgba(255,255,255,0.14)'; ctx.lineWidth = 1.4;
        ctx.strokeRect(5, 5, 110, 290);
        // перфорация
        ctx.fillStyle = 'rgba(0,0,0,0.55)';
        for (var yy = 40; yy < 282; yy += 9) {
            for (var xx = 14; xx < 94; xx += 8) {
                ctx.beginPath(); ctx.arc(xx + ((((yy - 40) / 9) | 0) % 2 ? 4 : 0), yy, 2.1, 0, TAU); ctx.fill();
            }
        }
        // ручка с замком
        ctx.fillStyle = lin(98, 0, 108, 0, [[0, '#b8bcc6'], [1, '#6f7480']]);
        ctx.fillRect(100, 120, 7, 60);
        ctx.fillStyle = '#111'; ctx.beginPath(); ctx.arc(103.5, 150, 2.2, 0, TAU); ctx.fill();
        // планка с логотипом
        ctx.fillStyle = 'rgba(0,0,0,0.6)'; ctx.fillRect(12, 12, 96, 22);
        ctx.fillStyle = '#3dcd58'; ctx.font = '700 15px "Segoe UI", Arial, sans-serif'; ctx.textBaseline = 'middle'; ctx.textAlign = 'left';
        ctx.fillText('APC', 18, 24);
        ctx.fillStyle = 'rgba(230,232,240,0.6)'; ctx.font = '600 8px "Segoe UI", Arial, sans-serif';
        ctx.fillText('NetShelter', 58, 24);
        ctx.restore();

        // --- боковая панель: вертикальные рёбра ---
        ctx.save();
        faceTransform(right[0], (b1 - b0) * s, -(b1 - b0) * s / 2, 0, H, 160, 300);
        ctx.strokeStyle = 'rgba(255,255,255,0.08)'; ctx.lineWidth = 1.2;
        ctx.strokeRect(8, 8, 144, 284);
        ctx.fillStyle = 'rgba(255,255,255,0.05)';
        ctx.fillRect(75, 8, 10, 284);
        ctx.restore();

        // --- верх: вентиляторная панель и кабельные вводы ---
        ctx.save();
        faceTransform(P(a0, b0, H), (a1 - a0) * s, (a1 - a0) * s / 2, -(b1 - b0) * s, (b1 - b0) * s / 2, 100, 130);
        ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillRect(12, 14, 76, 28);
        ctx.fillStyle = 'rgba(255,255,255,0.12)';
        for (var k = 0; k < 6; k++) ctx.fillRect(16 + k * 12, 18, 7, 20);
        ctx.fillStyle = 'rgba(0,0,0,0.45)'; ctx.fillRect(12, 70, 34, 40); ctx.fillRect(54, 70, 34, 40);
        ctx.restore();

        // кромки
        var edge = isSel ? '#58e6ff' : (isHov ? '#ffffff' : 'rgba(255,255,255,0.16)');
        var elw = isSel ? 2.4 : (isHov ? 2 : 1);
        poly(left, null, edge, elw); poly(right, null, edge, elw); poly(top, null, edge, elw);

        // световой столб над шкафом пропорционально мощности
        if (maxPower > 0 && r.powerAvgW > 0) {
            var frac = r.powerAvgW / maxPower;
            var cx = (top[0][0] + top[2][0]) / 2, cy = (top[0][1] + top[2][1]) / 2;
            var bh = H * (0.35 + 0.9 * frac), bw = (a1 - a0) * s * 0.75;
            ctx.save();
            ctx.globalCompositeOperation = 'lighter';
            ctx.fillStyle = lin(0, cy, 0, cy - bh, [[0, 'rgba(60,220,230,' + (0.25 + 0.4 * frac).toFixed(2) + ')'], [1, 'rgba(60,220,230,0)']]);
            ctx.fillRect(cx - bw / 2, cy - bh, bw, bh);
            ctx.restore();
        }

        // силуэт для hit-test (экранные координаты без translate)
        var sil = [P(a0, b1, 0), P(a1, b1, 0), P(a1, b0, 0), P(a1, b0, H), P(a0, b0, H), P(a0, b1, H)];
        polys.push({ r: r, pts: sil.map(function (p) { return [p[0] + G.ox, p[1] + G.oy]; }) });

        // подпись под шкафом
        if (s > 24) {
            var bp = P((a0 + a1) / 2, b1 + 0.02, 0);
            var fs = Math.max(9, Math.min(13, s * 0.17));
            ctx.font = '600 ' + fs + 'px "Segoe UI", Arial, sans-serif';
            var tw = ctx.measureText(r.name).width + 10;
            ctx.fillStyle = isSel ? 'rgba(40,150,190,0.9)' : 'rgba(10,9,40,0.72)';
            ctx.fillRect(bp[0] - tw / 2, bp[1] + 2, tw, fs + 5);
            ctx.fillStyle = '#e8eaff'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillText(r.name, bp[0], bp[1] + 4 + (fs + 3) / 2);
        }
    }

    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        drawFloor();
        polys = [];
        ctx.save();
        ctx.translate(G.ox, G.oy);
        order.forEach(drawRack);
        ctx.restore();
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
            if (inPoly([x, y], polys[i].pts)) return polys[i].r;
        }
        return null;
    }

    function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }

    function showTip(r, x, y) {
        var h = '<b class="dv-name">Шкаф ' + esc(r.name) + '</b><table>';
        h += '<tr><td>Высота</td><td>' + r.uHeight + 'U</td></tr>';
        h += '<tr><td>Серверов</td><td><b>' + r.servers + '</b>' + (r.blades ? ' (+ блейдов ' + r.blades + ')' : '') + '</td></tr>';
        h += '<tr><td>Корзин</td><td><b>' + r.chassis + '</b></td></tr>';
        h += '<tr><td>СХД</td><td><b>' + r.storages + '</b></td></tr>';
        h += '<tr><td>Мощность</td><td>' + (r.powerAvgW > 0 ? '<b>' + kw(r.powerAvgW) + '</b> (ср. 24ч)' : '<b>нет данных</b>') + '</td></tr>';
        if (r.powerPeakW > 0) h += '<tr><td>Пик 24ч</td><td>' + kw(r.powerPeakW) + '</td></tr>';
        if (r.powerRatedW > 0) h += '<tr><td>Паспорт</td><td>' + kw(r.powerRatedW) + '</td></tr>';
        h += '</table><div class="dv-hint">Детали шкафа - справа</div>';
        tip.innerHTML = h;
        tip.hidden = false;
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var px = x + 16, py = y + 16;
        if (px + tw > G.w - 4) px = x - tw - 16;
        if (py + th > G.h - 4) py = G.h - th - 4;
        tip.style.left = Math.max(4, px) + 'px';
        tip.style.top = Math.max(4, py) + 'px';
    }

    var selTimer = null;
    function selectRack(r) {
        if (selTimer) { clearTimeout(selTimer); selTimer = null; }
        if (selected === r && document.getElementById('dvFrame').getAttribute('src') !== 'about:blank') return;
        selected = r;
        var sel = document.getElementById('dvSel');
        var t = '<b>' + esc(r.name) + '</b><span class="dv-sep">&middot;</span>' + r.uHeight + 'U'
            + '<span class="dv-sep">&middot;</span>серверов ' + r.servers + (r.blades ? '+' + r.blades + ' бл.' : '')
            + '<span class="dv-sep">&middot;</span>корзин ' + r.chassis
            + '<span class="dv-sep">&middot;</span>СХД ' + r.storages
            + '<span class="dv-sep">&middot;</span>' + (r.powerAvgW > 0 ? '<b>' + kw(r.powerAvgW) + '</b>' + (r.powerPeakW > 0 ? ' (пик ' + kw(r.powerPeakW) + ')' : '') : 'мощность н/д')
            + '<span class="dv-sep">&middot;</span><a href="/sprsrv/rackcabinetview/' + r.id + '" target="_blank" rel="noopener">открыть</a>';
        sel.innerHTML = t;
        document.getElementById('dvFrame').src = '/sprsrv/rackcabinetview/' + r.id + '?embed=1';
        render();
    }

    canvas.addEventListener('mousemove', function (e) {
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        var r = pick(x, y);
        if (r !== hovered) {
            hovered = r;
            // шкаф под курсором становится выбранным (с небольшой задержкой, чтобы
            // при быстром движении мыши не загружать фрейм для каждого шкафа)
            if (selTimer) { clearTimeout(selTimer); selTimer = null; }
            if (r && r !== selected) selTimer = setTimeout(function () { selTimer = null; if (hovered === r) selectRack(r); }, 150);
            render();
        }
        canvas.style.cursor = r ? 'pointer' : 'default';
    });
    canvas.addEventListener('mouseleave', function () { hovered = null; if (selTimer) { clearTimeout(selTimer); selTimer = null; } tip.hidden = true; render(); });
    canvas.addEventListener('click', function (e) {
        var rect = canvas.getBoundingClientRect();
        var r = pick(e.clientX - rect.left, e.clientY - rect.top);
        if (r) selectRack(r);
    });

    function resize() { layout(); render(); }
    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    layout(); render();

    // по умолчанию - первый шкаф с оборудованием
    var first = racks.filter(function (r) { return r.servers + r.chassis + r.storages > 0; })[0] || racks[0];
    selectRack(first);
})();
</script>
