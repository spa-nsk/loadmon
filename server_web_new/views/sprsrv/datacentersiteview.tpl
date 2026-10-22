<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Оборудование ЦОД {{.SiteName}}</h2>
        <div class="text-muted" id="dvSummary">Шкафов: {{.RackCount}}</div>
    </div>
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacentersitelist/0">К справочнику площадок</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<div class="dv-layout">
    <div class="dv-left">
        <div class="dv-sel d-flex align-items-center justify-content-between gap-2">
            <div id="dvSel" class="dv-sel-text">Наведите курсор на шкаф</div>
            <div class="btn-group flex-shrink-0" role="group" aria-label="Масштаб шкафа">
                <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="dvZoomOut" title="Уменьшить шкаф">&minus;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="dvZoomReset" title="Масштаб по умолчанию" style="min-width:3.6rem">100%</button>
                <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="dvZoomIn" title="Увеличить шкаф">+</button>
            </div>
        </div>
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
    .dv-sel-text { min-width: 0; }
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
    var allRacks = DATA.racks;
    var racks = allRacks.filter(function (r) { return !r.shelf; });   // шкафы
    var shelves = allRacks.filter(function (r) { return r.shelf; });  // стеллаж площадки без шкафов ("вне шкафа")
    var TAU = Math.PI * 2;

    // ---- сводка ----
    var kw = function (w) { return w >= 1000 ? (w / 1000).toFixed(1) + ' кВт' : Math.round(w) + ' Вт'; };
    (function () {
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
        var list = (r.items || []).map(function (it) { return { k: it.k, vd: it.vd || '', p: it.p, u: it.u, su: Math.min(shelfSize(it), SH_CLEAR) }; });
        var rank = { storage: 0, tape: 1, chassis: 1, server: 2, 'switch': 3 };
        list.sort(function (a, b) { return (b.su - a.su) || ((rank[a.k] || 9) - (rank[b.k] || 9)); });
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
        if (/ubiquiti|unifi|\buap/.test(s)) return '#006fff';
        if (/cisco|catalyst|nexus|\bisr|\basr|\bmds|\basa\b|aironet|\bair-|\bc91\d{2}/.test(s)) return '#049fd9';
        return '#8b95a5';
    }

    // ---- раскладка шкафов по сетке ----
    var n = racks.length;
    var gridCols = n ? Math.max(1, Math.ceil(Math.sqrt(n * 1.3))) : 0;   // нет шкафов - стеллаж по левому краю
    var rows = n ? Math.ceil(n / gridCols) : 1;
    var maxPower = 0;
    racks.forEach(function (r, k) {
        r.gi = k % gridCols; r.gj = Math.floor(k / gridCols);
        if (r.powerAvgW > maxPower) maxPower = r.powerAvgW;
    });
    // стеллаж - справа от группы шкафов, габариты каждой секции как у шкафа 42U (на клетку по ширине)
    var shelfUnits = 0;
    shelves.forEach(function (r) {
        packShelf(r);
        r.gi = (gridCols ? gridCols + 0.6 : 0.15) + shelfUnits; r.gj = Math.floor((rows - 1) / 2);
        shelfUnits += r.units;
    });
    var cols = (gridCols ? gridCols + (shelfUnits ? 0.6 + shelfUnits : 0) : shelfUnits + 0.3);   // ширина сцены с учётом стеллажа
    var order = allRacks.slice().sort(function (a, b) { return (a.gi + a.gj) - (b.gi + b.gj) || a.gi - b.gi; });

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

        // --- верх: вентиляторная панель и надпись у лицевой кромки (в проекции) ---
        ctx.save();
        faceTransform(P(a0, b0, H), (a1 - a0) * s, (a1 - a0) * s / 2, -(b1 - b0) * s, (b1 - b0) * s / 2, 100, 130);
        ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillRect(12, 8, 76, 24);
        ctx.fillStyle = 'rgba(255,255,255,0.12)';
        for (var k = 0; k < 6; k++) ctx.fillRect(16 + k * 12, 11, 7, 18);
        ctx.fillStyle = 'rgba(0,0,0,0.45)'; ctx.fillRect(12, 40, 34, 28); ctx.fillRect(54, 40, 34, 28);
        // название шкафа - как будто написано на крышке перед лицевой панелью
        var fsz = 26;
        ctx.font = '800 ' + fsz + 'px "Segoe UI", Arial, sans-serif';
        var tw = ctx.measureText(r.name).width;
        if (tw > 88) { fsz = Math.max(9, Math.floor(fsz * 88 / tw)); ctx.font = '800 ' + fsz + 'px "Segoe UI", Arial, sans-serif'; }
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillStyle = 'rgba(0,0,0,0.55)'; ctx.fillText(r.name, 51.5, 102.5);
        ctx.fillStyle = isSel ? '#7df0ff' : (isHov ? '#ffffff' : '#d9dcea');
        ctx.fillText(r.name, 50, 101);
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
        for (q = 0; q < 20; q++) { var n = (q + 1) % 20; poly([top[q], top[n], bot[n], bot[q]], q > 4 && q < 15 ? '#c4cad2' : '#9aa1ab'); }
        poly(top, lit ? '#ffffff' : '#eef0f3', 'rgba(0,0,0,0.3)', 0.7);
        var lc = P(cx, cb, z + hh);
        ctx.fillStyle = on ? (round ? '#3aa0ff' : '#35e06a') : '#8a919c';
        ctx.beginPath(); ctx.arc(lc[0], lc[1], Math.max(1.5, rad * G.s * 0.14), 0, TAU); ctx.fill();
    }
    function drawShelfItem(a0, a1, b0, b1, z, h, it, lit) {
        if (isAP(it)) { drawShelfAP(a0, a1, b0, b1, z, h, it, lit); return; }
        var acc = shelfStyle(it.vd), on = it.p === 1;
        var ia0 = a0 + 0.03, ia1 = a1 - 0.03, ib0 = b0 + 0.05, ib1 = b1 - 0.02, zt = z + h;
        var front = [P(ia0, ib1, zt), P(ia1, ib1, zt), P(ia1, ib1, z), P(ia0, ib1, z)];
        var side = [P(ia1, ib1, zt), P(ia1, ib0, zt), P(ia1, ib0, z), P(ia1, ib1, z)];
        var top = [P(ia0, ib0, zt), P(ia1, ib0, zt), P(ia1, ib1, zt), P(ia0, ib1, zt)];
        poly(side, '#1e2026');
        poly(front, lin(0, front[0][1], 0, front[2][1], [[0, lit ? '#454a56' : '#353841'], [1, '#22252b']]), 'rgba(0,0,0,0.6)', 0.7);
        poly(top, lit ? '#6a7080' : '#575c69', 'rgba(255,255,255,0.25)', 0.7);
        // лицевая панель: цветная планка вендора, линии корзин/дисков, светодиод питания
        ctx.save();
        faceTransform(front[0], (ia1 - ia0) * G.s, (ia1 - ia0) * G.s / 2, 0, h, 100, 40);
        ctx.fillStyle = acc; ctx.fillRect(4, 3, 92, 4);
        ctx.fillStyle = 'rgba(0,0,0,0.55)';
        var rows = it.k === 'storage' ? 4 : ((it.k === 'chassis' || it.k === 'tape') ? 3 : 1);
        for (var q = 0; q < rows; q++) {
            if (it.k === 'chassis') ctx.fillRect(8 + q * 29, 11, 27, 24);
            else if (it.k === 'tape') { ctx.fillRect(8 + q * 22, 11, 20, 24); ctx.fillStyle = '#1f6fb5'; ctx.fillRect(74, 12, 18, 14); ctx.fillStyle = 'rgba(0,0,0,0.55)'; }
            else if (it.k === 'storage') ctx.fillRect(8, 11 + q * 7, 70, 5);
            else if (it.k === 'switch') { for (var pp = 0; pp < 16; pp++) ctx.fillRect(8 + pp * 5, 13 + (pp % 2) * 8, 4, 7); }
            else ctx.fillRect(8, 13, 62, 20);
        }
        if (it.k === 'server') { ctx.fillStyle = 'rgba(255,255,255,0.12)'; for (var d = 0; d < 6; d++) ctx.fillRect(12 + d * 10, 18, 6, 10); }
        ctx.fillStyle = on ? '#35e06a' : (it.p === 2 ? '#ffab1a' : '#8a919c');
        ctx.beginPath(); ctx.arc(88, 24, 4, 0, TAU); ctx.fill();
        ctx.restore();
        poly(front, null, lit ? '#ffffff' : 'rgba(255,255,255,0.22)', 0.8);
    }
    function drawShelf(r) {
        var isSel = selected === r, isHov = hovered === r, lit = isSel || isHov;
        var H = G.s * 1.55;                                      // высота как у шкафа 42U (hz)
        var post = Math.max(1.2, G.s * 0.035);
        var zs = H * 0.2, tb = H * 0.022;                         // шаг полок и толщина балки
        for (var u = 0; u < r.units; u++) {
            var a0 = r.gi + u + (1 - FW) / 2, a1 = a0 + FW;
            var b0 = r.gj + (1 - FD) / 2, b1 = b0 + FD;
            poly([P(a0 - 0.05, b0 + 0.05, 0), P(a1 + 0.12, b0 + 0.05, 0), P(a1 + 0.12, b1 + 0.1, 0), P(a0 - 0.05, b1 + 0.1, 0)], 'rgba(0,0,0,0.28)');
            var line = function (x, y, z0, z1, col, w) {
                var p0 = P(x, y, z0), p1 = P(x, y, z1);
                ctx.strokeStyle = col; ctx.lineWidth = w; ctx.lineCap = 'round';
                ctx.beginPath(); ctx.moveTo(p0[0], p0[1]); ctx.lineTo(p1[0], p1[1]); ctx.stroke();
            };
            var pc = lit ? '#7fa3c8' : '#5a7898';
            line(a0, b0, 0, H, pc, post); line(a1, b0, 0, H, pc, post);   // задние стойки
            // полки: слабая подложка, передняя балка
            for (var k = 0; k < 5; k++) {
                var z = H * 0.04 + zs * k;
                var topf = [P(a0, b0, z + tb), P(a1, b0, z + tb), P(a1, b1, z + tb), P(a0, b1, z + tb)];
                poly(topf, 'rgba(110,130,150,0.16)', lit ? '#ffffff' : 'rgba(110,200,255,0.5)', 0.9);
                var frontf = [P(a0, b1, z + tb), P(a1, b1, z + tb), P(a1, b1, z), P(a0, b1, z)];
                poly(frontf, lin(0, frontf[0][1], 0, frontf[2][1], [[0, '#5a7898'], [1, '#2a3b4e']]));
            }
            // оборудование: каждая секция - свои 5 полок
            var pk = r.packed && r.packed[u];
            if (pk) {
                for (var k2 = 0; k2 < 5; k2++) {
                    var zb = H * 0.04 + zs * k2 + tb, unitH = (zs - tb - H * 0.012) / SH_CLEAR;
                    pk[k2].items.forEach(function (pl) {
                        drawShelfItem(a0, a1, b0, b1, zb + pl.off * unitH, pl.it.su * unitH, pl.it, lit);
                    });
                }
            }
            line(a0, b1, 0, H, pc, post * 1.15); line(a1, b1, 0, H, pc, post * 1.15);   // передние стойки
            // название на верхней балке
            var zt = H * 0.04 + zs * 4 + tb;
            ctx.save();
            faceTransform(P(a0, b0, zt), (a1 - a0) * G.s, (a1 - a0) * G.s / 2, -(b1 - b0) * G.s, (b1 - b0) * G.s / 2, 100, 130);
            ctx.font = '800 18px "Segoe UI", Arial, sans-serif'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillText('СТЕЛЛАЖ', 51, 66);
            ctx.fillStyle = isSel ? '#7df0ff' : '#d9dcea'; ctx.fillText('СТЕЛЛАЖ', 50, 65);
            ctx.restore();
            var sil = [P(a0, b1, 0), P(a1, b1, 0), P(a1, b0, 0), P(a1, b0, H), P(a0, b0, H), P(a0, b1, H)];
            if (isSel || isHov) poly(sil, null, isSel ? '#58e6ff' : '#ffffff', isSel ? 2.2 : 1.8);
            polys.push({ r: r, pts: sil.map(function (p) { return [p[0] + G.ox, p[1] + G.oy]; }) });
        }
    }

    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        drawFloor();
        polys = [];
        ctx.save();
        ctx.translate(G.ox, G.oy);
        order.forEach(function (r) { if (r.shelf) drawShelf(r); else drawRack(r); });
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
        var h = '<b class="dv-name">' + (r.shelf ? 'Стеллаж (вне шкафов)' : 'Шкаф ' + esc(r.name)) + '</b><table>';
        h += '<tr><td>Высота</td><td>' + r.uHeight + 'U</td></tr>';
        h += '<tr><td>Серверов</td><td><b>' + r.servers + '</b>' + (r.blades ? ' (+ блейдов ' + r.blades + ')' : '') + '</td></tr>';
        h += '<tr><td>Корзин</td><td><b>' + r.chassis + '</b></td></tr>';
        h += '<tr><td>СХД</td><td><b>' + r.storages + '</b></td></tr>';
        if (r.tapes) h += '<tr><td>Ленточных библиотек</td><td><b>' + r.tapes + '</b></td></tr>';
        if (r.network) h += '<tr><td>Сетевого оборудования</td><td><b>' + r.network + '</b></td></tr>';
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
        var t = '<b>' + (r.shelf ? 'Стеллаж (вне шкафов)' : esc(r.name)) + '</b><span class="dv-sep">&middot;</span>' + r.uHeight + 'U'
            + '<span class="dv-sep">&middot;</span>серверов ' + r.servers + (r.blades ? '+' + r.blades + ' бл.' : '')
            + '<span class="dv-sep">&middot;</span>корзин ' + r.chassis + (r.tapes ? '<span class="dv-sep">&middot;</span>лент. библиотек ' + r.tapes : '') + (r.network ? '<span class="dv-sep">&middot;</span>сеть ' + r.network : '')
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

    // масштаб шкафа в правой части: кнопки управляют фреймом, индикатор берёт значение у него
    function zoomCmd(cmd) {
        var f = document.getElementById('dvFrame');
        if (f && f.contentWindow) f.contentWindow.postMessage({ type: 'rvZoom', cmd: cmd }, location.origin);
    }
    document.getElementById('dvZoomIn').addEventListener('click', function () { zoomCmd('in'); });
    document.getElementById('dvZoomOut').addEventListener('click', function () { zoomCmd('out'); });
    document.getElementById('dvZoomReset').addEventListener('click', function () { zoomCmd('reset'); });
    window.addEventListener('message', function (e) {
        if (e.origin !== location.origin) return;
        var m = e.data || {};
        if (m.type === 'rvZoomState') document.getElementById('dvZoomReset').textContent = m.pct + '%';
    });

    function resize() { layout(); render(); }
    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    layout(); render();

    // по умолчанию - первый шкаф с оборудованием
    var first = allRacks.filter(function (r) { return r.servers + r.chassis + r.storages > 0; })[0] || allRacks[0];
    selectRack(first);
})();
</script>
