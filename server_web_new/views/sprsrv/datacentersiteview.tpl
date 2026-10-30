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
        </div>
        <div class="dv-stage" id="dvStage">
            <canvas id="dvCanvas"></canvas>
            <div id="dvTip" class="dv-tip" hidden></div>
            <div id="dvZoomInd" class="dv-zoom-ind">Масштаб: 100%</div>
            <button type="button" id="dvResetView" class="dv-reset-view" title="Сбросить масштаб и ракурс (двойной клик по пустому месту - то же)">&#8634; Вид</button>
            <div class="dv-view-hint">Колесо - масштаб (и справа у шкафа) &middot; тяните за угол основания - ракурс &middot; тяните пустое место - сдвиг</div>
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
    function glowRGB(frac, dark) {
        var stops = [[253, 224, 120], [251, 191, 36], [245, 158, 11]];     // один и тот же светлый янтарный ряд в обеих темах
        var t = Math.max(0, Math.min(1, frac)) * 2, i = Math.min(1, Math.floor(t)), f = t - i;
        var a = stops[i], b = stops[i + 1];
        return Math.round(a[0] + (b[0] - a[0]) * f) + ',' + Math.round(a[1] + (b[1] - a[1]) * f) + ',' + Math.round(a[2] + (b[2] - a[2]) * f);
    }
    // столб + светящееся "навершие" на крыше шкафа (cx, cy - центр крыши в экранных координатах)
    function drawGlow(cx, cy, bw, bh, frac, dark, mode) {
        var rgb = glowRGB(frac, dark), a0 = (dark ? 0.32 : 0.5) + (dark ? 0.4 : 0.35) * frac;
        ctx.save();
        ctx.globalCompositeOperation = mode;
        ctx.fillStyle = lin(0, cy, 0, cy - bh, [[0, 'rgba(' + rgb + ',' + a0.toFixed(2) + ')'], [1, 'rgba(' + rgb + ',0)']]);
        ctx.fillRect(cx - bw / 2, cy - bh, bw, bh);
        ctx.fillStyle = 'rgba(' + rgb + ',' + Math.min(0.95, a0 + 0.3).toFixed(2) + ')';
        ctx.beginPath(); ctx.ellipse(cx, cy, bw * 0.52, bw * 0.24, 0, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }
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
        list.sort(function (a, b) { return (b.su - a.su) || ((rank[a.k] || 9) - (rank[b.k] || 9)) || String(a.vd).localeCompare(String(b.vd)); });
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

    // отдельные рисуемые объекты: шкафы и секции стеллажа (у секции - свой центр для сортировки по глубине)
    var items3d = [];
    racks.forEach(function (r) { items3d.push({ r: r, shelfUnit: -1, ca: r.gi + 0.5, cb: r.gj + 0.5 }); });
    shelves.forEach(function (r) { for (var u = 0; u < r.units; u++) items3d.push({ r: r, shelfUnit: u, ca: r.gi + u + 0.5, cb: r.gj + 0.5 }); });

    var FW = 0.60, FD = 0.80;          // размеры шкафа в клетках (ширина 600, глубина 1000 при клетке 1.2 м)
    // выбранный/наведённый объект: { r, u } где u = -1 для шкафа, 0..N-1 для секции стеллажа
    var selected = null, hovered = null, hoverCorner = -1;
    var polys = [];                    // силуэты для hit-test: {r, u, pts}
    var G = { w: 0, h: 0, dpr: 1, s: 1, baseS: 1 };
    function sameHit(a, b) { return !!a && !!b && a.r === b.r && a.u === b.u; }
    function hitOf(r, u) { return { r: r, u: u == null ? -1 : u }; }
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
        var fc = floorCorners().map(function (c) { return P(c[0], c[1], 0); });
        poly(fc, lin(0, Math.min.apply(null, fc.map(function (p) { return p[1]; })), 0, Math.max.apply(null, fc.map(function (p) { return p[1]; })), [[0, PAL.floorTop], [1, PAL.floorBot]]), PAL.floorEdge, 1.5);
        ctx.strokeStyle = PAL.grid; ctx.lineWidth = 1;
        for (var i = 0; i <= cols; i++) { var a = P(i, 0, 0), b = P(i, rows, 0); ctx.beginPath(); ctx.moveTo(a[0], a[1]); ctx.lineTo(b[0], b[1]); ctx.stroke(); }
        for (var j = 0; j <= rows; j++) { var c = P(0, j, 0), d = P(cols, j, 0); ctx.beginPath(); ctx.moveTo(c[0], c[1]); ctx.lineTo(d[0], d[1]); ctx.stroke(); }
    }

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

        // --- передняя перфорированная дверь (локальные координаты 120 x 300) ---
        if (hasDoor) {
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

        // световой столб над шкафом пропорционально мощности
        if (maxPower > 0 && r.powerAvgW > 0) {
            var frac = r.powerAvgW / maxPower;
            var cx = (top[0][0] + top[2][0]) / 2, cy = (top[0][1] + top[2][1]) / 2;
            var bh = H * CV.zk * (0.35 + 0.9 * frac), bw = (a1 - a0) * s * 0.75;
            drawGlow(cx, cy, bw, bh, frac, PAL === PALS.dark, PAL.glowMode);
        }

        polys.push({ r: r, u: -1, pts: boxHull(a0, a1, b0, b1, 0, H) });
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
        ctx.fillStyle = on ? (round ? '#3aa0ff' : '#35e06a') : '#8a919c';
        ctx.beginPath(); ctx.arc(lc[0], lc[1], Math.max(1.5, rad * G.s * 0.14), 0, TAU); ctx.fill();
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
            ctx.fillStyle = 'rgba(0,0,0,0.55)';
            var rows3 = it.k === 'storage' ? 4 : ((it.k === 'chassis' || it.k === 'tape') ? 3 : 1);
            for (var q = 0; q < rows3; q++) {
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
                    drawShelfItem(a0, a1, b0, b1, zb + pl.off * unitH, pl.it.su * unitH * 0.86, pl.it, lit);   // 0.86 - видимый зазор между устройствами
                });
            }
        }
        posts.forEach(function (p) { if (p.d > medD) line(p.a, p.b, 0, H, pc, post * 1.15); });
        labels.push({ a0: a0, a1: a1, b0: b0, b1: b1, H: H, sel: isSel, num: u + 1 });
        var hl = boxHull(a0, a1, b0, b1, 0, H);
        if (isSel || isHov) boxWire(P, a0, a1, b0, b1, 0, H, CV.aPos ? a0 : a1, CV.bPos ? b0 : b1, isSel ? PAL.sel : PAL.hov, isSel ? 2.6 : 2);
        polys.push({ r: r, u: u, pts: hl });
    }

    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        drawFloor();
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
            if (inPoly([x, y], polys[i].pts)) return hitOf(polys[i].r, polys[i].u);
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

    // deep-link из /sprsrv/location-uri/:id → ?rack=&hw=
    var qsFocus = new URLSearchParams(location.search);
    var focusRackId = parseInt(qsFocus.get('rack') || '0', 10) || 0;
    var focusHwId = parseInt(qsFocus.get('hw') || '0', 10) || 0;

    var selTimer = null;
    function selectHit(hit) {
        if (!hit) return;
        if (selTimer) { clearTimeout(selTimer); selTimer = null; }
        if (sameHit(selected, hit) && document.getElementById('dvFrame').getAttribute('src') !== 'about:blank') return;
        selected = hit;
        var r = hit.r, st = hit.u >= 0 ? unitStats(r, hit.u) : null;
        var title = r.shelf ? ('Стеллаж ' + (hit.u + 1)) : esc(r.name);
        var pinHw = focusHwId > 0 && focusRackId === r.id ? focusHwId : 0;
        // при подсветке конкретного сервера на стеллаже - все секции (иначе устройство может быть в другой секции)
        var shelfQ = (r.shelf && !pinHw) ? ('shelfUnit=' + (hit.u + 1)) : '';
        var url = '/sprsrv/rackcabinetview/' + r.id + (shelfQ || pinHw ? '?' : '') + shelfQ + (pinHw ? ((shelfQ ? '&' : '') + 'hw=' + pinHw) : '');
        var emb = '/sprsrv/rackcabinetview/' + r.id + '?embed=1' + (shelfQ ? '&' + shelfQ : '') + (pinHw ? '&hw=' + pinHw : '');
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
    function resetView() { cam.yaw = YAW0; cam.pitch = PITCH0; cam.zoom = 1; cam.panX = 0; cam.panY = 0; setCamera(); render(); showZoomInd(); }

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
        if (mode === 'click' && !moved) { var h = pick(x, y); if (h) selectHit(h); }
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
        if (!sameHit(h, hovered)) {
            hovered = h; changed = true;
            // объект под курсором становится выбранным (с небольшой задержкой, чтобы
            // при быстром движении мыши не загружать фрейм для каждой секции)
            if (selTimer) { clearTimeout(selTimer); selTimer = null; }
            if (h && !sameHit(h, selected)) selTimer = setTimeout(function () { selTimer = null; if (sameHit(hovered, h)) selectHit(h); }, 150);
        }
        if (changed) render();
        canvas.style.cursor = ci >= 0 ? 'move' : (h ? 'pointer' : 'grab');
        canvas.title = ci >= 0 ? 'Потяните за угол, чтобы изменить ракурс' : '';
    });
    canvas.addEventListener('mouseleave', function () { hovered = null; hoverCorner = -1; if (selTimer) { clearTimeout(selTimer); selTimer = null; } tip.hidden = true; if (!dragMode) render(); });
    canvas.addEventListener('dblclick', function (e) {
        var rect = canvas.getBoundingClientRect();
        if (cornerAt(e.clientX - rect.left, e.clientY - rect.top) >= 0 || !pick(e.clientX - rect.left, e.clientY - rect.top)) resetView();
    });
    document.getElementById('dvResetView').addEventListener('click', resetView);

    // масштаб шкафа в правой части меняется колесом мыши внутри фрейма; индикатор масштаба показывает сам фрейм (всплывающий)

    document.getElementById('dvFrame').addEventListener('load', pushThemeToFrame);
    function resize() { layout(); render(); }
    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    layout(); render();

    // deep-link ?rack= — выделить шкаф/стеллаж; иначе первый шкаф с оборудованием / секция стеллажа
    var focusCab = focusRackId > 0 ? allRacks.filter(function (r) { return r.id === focusRackId; })[0] : null;
    if (focusCab) {
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
