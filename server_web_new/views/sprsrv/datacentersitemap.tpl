<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Карта площадок/ЦОД</h2>
        <div class="text-muted" id="smSummary">ЦОД: {{.SiteCount}}</div>
    </div>
    <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/datacentersitelist/0">К справочнику площадок</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<div class="sm-wrap">
    <div class="sm-bar d-flex align-items-center justify-content-between gap-2">
        <div class="sm-bar-text">Только ЦОД: города &rarr; здания &rarr; этажи. Колесо - масштаб, перетаскивание - прокрутка, клик по шкафу/стеллажу/ЦОД - детали и выбор (ЦОД выдвигается из стопки), правая кнопка - переход к оборудованию площадки.</div>
        <div class="btn-group flex-shrink-0" role="group" aria-label="Масштаб карты">
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomOut" title="Уменьшить">&minus;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomFit" title="Вписать всю карту в окно" style="min-width:3.6rem">100%</button>
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomIn" title="Увеличить">+</button>
        </div>
    </div>
    <div class="sm-stage" id="smStage">
        <canvas id="smCanvas"></canvas>
        <div id="smTip" class="sm-tip" hidden></div>
        <div id="smPanel" class="sm-tip sm-panel" hidden></div>
        <div id="smCtx" class="sm-ctx" hidden></div>
    </div>
</div>

<style>
    .sm-wrap { display: flex; flex-direction: column; gap: 0.4rem; height: calc(100vh - 170px); min-height: 520px; }
    .sm-bar { flex: 0 0 auto; padding: 0.35rem 0.7rem; border-radius: 0.5rem; border: 1px solid var(--bs-border-color); background: var(--bs-tertiary-bg); font-size: 0.85rem; }
    .sm-bar-text { min-width: 0; color: var(--bs-secondary-color); }
    .sm-stage { position: relative; width: 100%; flex: 1 1 auto; min-height: 0; border-radius: 0.6rem; overflow: hidden; border: 1px solid var(--bs-border-color); background: #17144f; }
    #smCanvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; cursor: grab; touch-action: none; }
    #smCanvas.sm-drag { cursor: grabbing; }
    .sm-tip { position: absolute; z-index: 5; pointer-events: none; min-width: 210px; max-width: 320px; padding: 0.55rem 0.7rem; border-radius: 0.5rem; font-size: 0.82rem; line-height: 1.35;
              background: rgba(14, 13, 48, 0.95); color: #e8e8ff; border: 1px solid rgba(120, 130, 255, 0.45); box-shadow: 0 6px 22px rgba(0, 0, 0, 0.45); }
    .sm-panel { z-index: 6; pointer-events: auto; top: 10px; right: 10px; left: auto; border-color: rgba(88, 230, 255, 0.7); }
    .sm-tip b.sm-name { font-size: 0.95rem; color: #fff; }
    .sm-tip .sm-sub { color: #a9aef0; font-size: 0.78rem; margin-bottom: 0.2rem; }
    .sm-tip table { border-collapse: collapse; margin-top: 0.25rem; width: 100%; }
    .sm-tip td { padding: 1px 0; vertical-align: top; }
    .sm-tip td:first-child { color: #a9aef0; padding-right: 0.7rem; white-space: nowrap; }
    .sm-tip .sm-hint { margin-top: 0.35rem; color: #8f94d8; font-size: 0.75rem; }
    .sm-tip a { color: #7df0ff; }
    .sm-ctx { position: absolute; z-index: 9; min-width: 250px; padding: 0.3rem; border-radius: 0.5rem; background: rgba(14, 13, 48, 0.97); border: 1px solid rgba(120, 130, 255, 0.6); box-shadow: 0 8px 26px rgba(0, 0, 0, 0.55); }
    .sm-ctx .sm-ctx-title { padding: 0.25rem 0.6rem; font-size: 0.75rem; color: #a9aef0; border-bottom: 1px solid rgba(120, 130, 255, 0.3); margin-bottom: 0.2rem; }
    .sm-ctx a { display: block; padding: 0.35rem 0.6rem; border-radius: 0.35rem; font-size: 0.85rem; color: #e8e8ff; text-decoration: none; }
    .sm-ctx a:hover { background: rgba(88, 230, 255, 0.18); color: #fff; }
    .sm-close { position: absolute; top: 2px; right: 8px; background: none; border: 0; color: #a9aef0; font-size: 1.2rem; line-height: 1; cursor: pointer; }
</style>

<script>
(function () {
    'use strict';

    var DATA = {{.MapJSON}};
    var canvas = document.getElementById('smCanvas');
    var stage = document.getElementById('smStage');
    var tip = document.getElementById('smTip');
    var panel = document.getElementById('smPanel');
    var pctBtn = document.getElementById('smZoomFit');
    if (!canvas || !DATA || !DATA.length) return;

    var ctx = canvas.getContext('2d');
    var TAU = Math.PI * 2;
    var U = 60;                                    // пикселей мира на клетку (клетка = 1.2 м)
    var FONT = '"Segoe UI", Arial, sans-serif';
    var SLAB = 0.1;                                // тонкая основа ЦОД
    var FW = 0.60, FD = 0.80;                      // размер шкафа в клетках
    var cam = { k: 1, tx: 0, ty: 0 };
    var G = { w: 0, h: 0, dpr: 1 };
    var hovSite = null, hovRack = null, selSite = null, selRack = null, hovCity = null;
    var hits = [];                                 // [{site, rack, pts}] в координатах мира, порядок = порядок рисования
    var overlay = [];                              // выноски/подписи - рисуются последним проходом поверх всего
    var allSites = [];

    var kw = function (w) { return w >= 1000 ? (w / 1000).toFixed(1) + ' кВт' : Math.round(w) + ' Вт'; };
    function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    function floorTitle(f) { return f === 0 ? 'Подвал (0)' : 'Этаж ' + f; }
    function hz(r) { return 1.55 * Math.max(0.5, Math.min(1.4, r.uHeight / 42)); }

    // ---- раскладка: города -> здания -> этажи -> ЦОД ----
    function shelf(items, limit, gap) {
        var rows = [], cur = { items: [], w: 0, h: 0 };
        items.forEach(function (it) {
            if (cur.items.length && cur.w + gap + it.w > limit) { rows.push(cur); cur = { items: [], w: 0, h: 0 }; }
            it.x = cur.w + (cur.items.length ? gap : 0);
            cur.w = it.x + it.w; cur.h = Math.max(cur.h, it.h);
            cur.items.push(it);
        });
        if (cur.items.length) rows.push(cur);
        var y = 0, W = 0;
        rows.forEach(function (r) { r.y = y; y += r.h + gap; W = Math.max(W, r.w); r.items.forEach(function (it) { it.y = r.y; }); });
        return { w: W, h: Math.max(0, y - gap) };
    }

    // ---- оборудование стеллажа: раскладка по секциям/полкам (те же правила, что в просмотре оборудования площадки) ----
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

    var PAD_L = 3;                                 // запас слева под подписи этажей и выдвижение ЦОД
    var siteCnt = 0, rackCnt = 0, bldCnt = 0;
    var shelfCnt = 0;
    var cities = DATA.map(function (c) {
        var bl = c.buildings.map(function (b) {
            var A = 0, B = 0, half = 0;
            b.floors.forEach(function (f) {
                var x = 0, bf = 0;
                f.sites.forEach(function (s) {
                    s.normal = s.racks.filter(function (r) { return !r.shelf; });
                    s.shelves = s.racks.filter(function (r) { return r.shelf; });
                    var n = s.normal.length, su = 0;
                    s.shelves.forEach(function (r) { packShelf(r); su += r.units; });
                    s.cols = n ? Math.max(1, Math.ceil(Math.sqrt(n * 1.3))) : 0;     // нет шкафов - стеллаж по левому краю
                    s.rows = n ? Math.ceil(n / s.cols) : 1;
                    s.shelfUnits = su;
                    s.shelfStart = s.cols ? s.cols + 0.6 : 0.15;                       // стеллаж - справа от группы шкафов
                    var gridW = s.cols ? s.cols + (su ? 0.6 + su : 0) : (su ? su + 0.3 : 2);
                    var shelfOff = 0;
                    s.shelves.forEach(function (r) { r.gi = s.shelfStart + shelfOff; r.gj = Math.floor((s.rows - 1) / 2); shelfOff += r.units; });
                    s.maxPower = 0; s.normal.forEach(function (r) { if (r.powerAvgW > s.maxPower) s.maxPower = r.powerAvgW; });
                    s.sA = gridW + 0.6; s.sB = s.rows + 0.6;
                    s.aOff = x; x += s.sA + 0.8; bf = Math.max(bf, s.sB);
                    s.e = 0; s.tgt = 0; s.floor = f.floor; allSites.push(s);
                    siteCnt++; rackCnt += n; shelfCnt += s.shelves.length;
                });
                f.A = Math.max(0.5, x - 0.8); f.B = bf;
                A = Math.max(A, f.A); B = Math.max(B, f.B);
                half = Math.max(half, (f.A + f.B) / 2);
            });
            var n = b.floors.length, FH = half + 2.5;
            bldCnt++;
            return { b: b, A: A, B: B, n: n, FH: FH,
                     w: (A + B + PAD_L) * U, h: (2.3 + (n - 1) * FH + (A + B) / 2 + 1.8) * U };
        });
        // один город - одна горизонтальная строка: здания (адреса) слева направо, общая линия основания
        var x = 0, rowH = 0;
        bl.forEach(function (b) { b.x = x; x += b.w + 2.4 * U; rowH = Math.max(rowH, b.h); });
        bl.forEach(function (b) { b.y = rowH - b.h; });
        return { c: c, blds: bl, w: Math.max(x - 2.4 * U, 6 * U), h: rowH + 0.6 * U };
    });
    var WORLD = { w: 0, h: 0 };
    (function () {
        var y = 0;
        cities.forEach(function (ct) { ct.x = 0; ct.y = y; y += ct.h + 1.2 * U; WORLD.w = Math.max(WORLD.w, ct.w); });
        WORLD.h = Math.max(1, y - 1.2 * U);
    })();
    var LW = 180;                                  // ширина левого списка городов (в пикселях экрана)
    var cityBoxes = [];
    document.getElementById('smSummary').innerHTML = 'Городов: <b>' + cities.length + '</b> &middot; зданий: <b>' + bldCnt + '</b> &middot; ЦОД: <b>' + siteCnt + '</b> &middot; шкафов: <b>' + rackCnt + '</b>' + (shelfCnt ? ' &middot; стеллажей: <b>' + shelfCnt + '</b>' : '');

    // изометрия: a - вправо-вниз, b - влево-вниз, z - вверх (в клетках)
    function P(o, a, b, z) { return [o.x + (a - b) * U, o.y + (a + b) * U / 2 - z * U]; }

    // ---- примитивы ----
    function poly(pts, fill, stroke, lw) {
        ctx.beginPath(); ctx.moveTo(pts[0][0], pts[0][1]);
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
    function lw(v) { return Math.max(v * 0.6, v / Math.sqrt(cam.k)); }
    function faceTransform(o, ux, uy, vx, vy, Wl, Hl) { ctx.transform(ux / Wl, uy / Wl, vx / Hl, vy / Hl, o[0], o[1]); }

    // ---- шкаф (в стиле левого вида просмотра оборудования ЦОД) ----
    function drawRack(so, site, r, gi, gj) {
        var a0 = 0.3 + gi + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = 0.3 + gj + (1 - FD) / 2, b1 = b0 + FD;
        var z0 = SLAB, H = z0 + hz(r), s = U;
        var isSel = selRack === r, isHov = hovRack === r, lit = isSel || isHov;

        poly([P(so, a0 - 0.05, b0 + 0.05, z0), P(so, a1 + 0.1, b0 + 0.05, z0), P(so, a1 + 0.1, b1 + 0.08, z0), P(so, a0 - 0.05, b1 + 0.08, z0)], 'rgba(0,0,0,0.28)');
        var left = [P(so, a0, b1, H), P(so, a1, b1, H), P(so, a1, b1, z0), P(so, a0, b1, z0)];
        var right = [P(so, a1, b1, H), P(so, a1, b0, H), P(so, a1, b0, z0), P(so, a1, b1, z0)];
        var top = [P(so, a0, b0, H), P(so, a1, b0, H), P(so, a1, b1, H), P(so, a0, b1, H)];
        poly(left, lin(0, left[0][1], 0, left[3][1], [[0, lit ? '#3a3e48' : '#2c2f37'], [1, lit ? '#262930' : '#1b1d22']]));
        poly(right, lin(0, right[0][1], 0, right[3][1], [[0, lit ? '#2a2d34' : '#202228'], [1, '#121317']]));
        poly(top, lit ? '#4a4e5a' : '#3a3d47');

        // передняя дверь
        ctx.save();
        faceTransform(left[0], (a1 - a0) * s, (a1 - a0) * s / 2, 0, (H - z0) * s, 120, 300);
        ctx.strokeStyle = 'rgba(255,255,255,0.14)'; ctx.lineWidth = 1.4;
        ctx.strokeRect(5, 5, 110, 290);
        if (cam.k > 1.3) {
            ctx.fillStyle = 'rgba(0,0,0,0.55)';
            for (var yy = 40; yy < 282; yy += 9) for (var xx = 14; xx < 94; xx += 8) {
                ctx.beginPath(); ctx.arc(xx + ((((yy - 40) / 9) | 0) % 2 ? 4 : 0), yy, 2.1, 0, TAU); ctx.fill();
            }
        } else {
            ctx.fillStyle = 'rgba(0,0,0,0.4)';
            for (var q = 0; q < 6; q++) ctx.fillRect(14, 50 + q * 38, 80, 20);
        }
        ctx.fillStyle = lin(98, 0, 108, 0, [[0, '#b8bcc6'], [1, '#6f7480']]);
        ctx.fillRect(100, 120, 7, 60);
        ctx.restore();

        // боковая панель
        ctx.save();
        faceTransform(right[0], (b1 - b0) * s, -(b1 - b0) * s / 2, 0, (H - z0) * s, 160, 300);
        ctx.strokeStyle = 'rgba(255,255,255,0.08)'; ctx.lineWidth = 1.2;
        ctx.strokeRect(8, 8, 144, 284);
        ctx.restore();

        // маркировка на крышке
        if (cam.k > 0.22) {
            ctx.save();
            faceTransform(P(so, a0, b0, H), (a1 - a0) * s, (a1 - a0) * s / 2, -(b1 - b0) * s, (b1 - b0) * s / 2, 100, 130);
            var fsz = 34, nm = r.name;
            ctx.font = '800 ' + fsz + 'px ' + FONT;
            var tw = ctx.measureText(nm).width;
            if (tw > 92) { fsz = Math.max(12, Math.floor(fsz * 92 / tw)); ctx.font = '800 ' + fsz + 'px ' + FONT; }
            while (nm.length > 2 && ctx.measureText(nm).width > 92) nm = nm.slice(0, -2) + '…';
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.fillStyle = 'rgba(0,0,0,0.6)'; ctx.fillText(nm, 51.5, 66.5);
            ctx.fillStyle = isSel ? '#7df0ff' : (isHov ? '#aef4ff' : '#d9dcea');
            ctx.fillText(nm, 50, 65);
            ctx.restore();
        }
        var edge = isSel ? '#58e6ff' : (isHov ? '#8fefff' : 'rgba(255,255,255,0.16)');
        var elw = lw(isSel ? 2.6 : (isHov ? 2.2 : 1));
        if (isSel || isHov) { var tn = isSel ? SEL_TINT : HOV_TINT; poly(left, tn); poly(right, tn); poly(top, tn); }
        poly(left, null, edge, elw); poly(right, null, edge, elw); poly(top, null, edge, elw);

        // световой столб над шкафом пропорционально мощности (относительно самого мощного шкафа площадки)
        if (site.maxPower > 0 && r.powerAvgW > 0) {
            var frac = r.powerAvgW / site.maxPower;
            var pcx = (top[0][0] + top[2][0]) / 2, pcy = (top[0][1] + top[2][1]) / 2;
            var pbh = (H - z0) * U * (0.35 + 0.9 * frac), pbw = (a1 - a0) * U * 0.75;
            ctx.save();
            ctx.globalCompositeOperation = 'lighter';
            ctx.fillStyle = lin(0, pcy, 0, pcy - pbh, [[0, 'rgba(60,220,230,' + (0.25 + 0.4 * frac).toFixed(2) + ')'], [1, 'rgba(60,220,230,0)']]);
            ctx.fillRect(pcx - pbw / 2, pcy - pbh, pbw, pbh);
            ctx.restore();
        }

        hits.push({ site: site, rack: r, pts: [P(so, a0, b1, z0), P(so, a1, b1, z0), P(so, a1, b0, z0), P(so, a1, b0, H), P(so, a0, b0, H), P(so, a0, b1, H)] });
    }

    // ---- стеллаж в стиле просмотра оборудования площадки: рамка из стоек и полок, оборудование непрозрачными блоками ----
    var SEL_TINT = 'rgba(88,230,255,0.34)', HOV_TINT = 'rgba(88,230,255,0.18)';   // выбран / под курсором - голубая подкраска
    function drawShelfAP(so, a0, a1, b0, b1, z, h, it, lit) {
        var cx = (a0 + a1) / 2, cb = (b0 + b1) / 2, rad = Math.min(a1 - a0, b1 - b0) * 0.34, hh = Math.min(h * 0.45, 0.05);
        var round = /unifi|uap|ubiquiti/i.test(it.vd), on = it.p === 1, pts = [], q;
        for (q = 0; q < 20; q++) { var an = q / 20 * TAU; pts.push([cx + Math.cos(an) * rad * (round ? 1 : 1.25), cb + Math.sin(an) * rad * (round ? 1 : 0.95)]); }
        var top = pts.map(function (p) { return P(so, p[0], p[1], z + hh); });
        var bot = pts.map(function (p) { return P(so, p[0], p[1], z); });
        poly(bot.concat(top.slice().reverse()), '#aab1bb');
        for (q = 0; q < 20; q++) { var nn = (q + 1) % 20; poly([top[q], top[nn], bot[nn], bot[q]], q > 4 && q < 15 ? '#c4cad2' : '#9aa1ab'); }
        poly(top, lit ? '#ffffff' : '#eef0f3', 'rgba(0,0,0,0.3)', 0.7);
        var lc = P(so, cx, cb, z + hh);
        ctx.fillStyle = on ? (round ? '#3aa0ff' : '#35e06a') : '#8a919c';
        ctx.beginPath(); ctx.arc(lc[0], lc[1], Math.max(1.2, rad * U * 0.14), 0, TAU); ctx.fill();
    }
    function drawShelfItem(so, a0, a1, b0, b1, z, h, it, lit) {
        if (isAP(it)) { drawShelfAP(so, a0, a1, b0, b1, z, h, it, lit); return; }
        var acc = shelfStyle(it.vd), on = it.p === 1;
        var ia0 = a0 + 0.03, ia1 = a1 - 0.03, ib0 = b0 + 0.05, ib1 = b1 - 0.02, zt = z + h;
        var front = [P(so, ia0, ib1, zt), P(so, ia1, ib1, zt), P(so, ia1, ib1, z), P(so, ia0, ib1, z)];
        var side = [P(so, ia1, ib1, zt), P(so, ia1, ib0, zt), P(so, ia1, ib0, z), P(so, ia1, ib1, z)];
        var top = [P(so, ia0, ib0, zt), P(so, ia1, ib0, zt), P(so, ia1, ib1, zt), P(so, ia0, ib1, zt)];
        poly(side, '#1e2026');
        poly(front, lin(0, front[0][1], 0, front[2][1], [[0, lit ? '#454a56' : '#353841'], [1, '#22252b']]), 'rgba(0,0,0,0.6)', 0.7);
        poly(top, lit ? '#6a7080' : '#575c69', 'rgba(255,255,255,0.25)', 0.7);
        if (cam.k > 0.7) {
            ctx.save();
            faceTransform(front[0], (ia1 - ia0) * U, (ia1 - ia0) * U / 2, 0, h * U, 100, 40);
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
        } else {
            poly(front, null, null); ctx.fillStyle = acc; ctx.fillRect(front[0][0], front[0][1], front[1][0] - front[0][0], Math.max(1, U * 0.03));
        }
        poly(front, null, lit ? '#8fefff' : 'rgba(255,255,255,0.22)', 0.8);
    }
    function drawShelfUnit(so, site, r, u, labels) {
        var isSel = selRack === r, isHov = hovRack === r, lit = isSel || isHov;
        var H = 1.55, z0 = SLAB, post = Math.max(1.2, U * 0.035);
        var zs = H * 0.2, tb = H * 0.022;
        var a0 = 0.3 + r.gi + u + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = 0.3 + r.gj + (1 - FD) / 2, b1 = b0 + FD;
        poly([P(so, a0 - 0.05, b0 + 0.05, z0), P(so, a1 + 0.1, b0 + 0.05, z0), P(so, a1 + 0.1, b1 + 0.08, z0), P(so, a0 - 0.05, b1 + 0.08, z0)], 'rgba(0,0,0,0.28)');
        var line = function (x, y, za, zb, col, w) {
            var p0 = P(so, x, y, za), p1 = P(so, x, y, zb);
            ctx.strokeStyle = col; ctx.lineWidth = w; ctx.lineCap = 'round';
            ctx.beginPath(); ctx.moveTo(p0[0], p0[1]); ctx.lineTo(p1[0], p1[1]); ctx.stroke();
        };
        var pc = isSel ? '#58e6ff' : (isHov ? '#8fefff' : '#5a7898');
        line(a0, b0, z0, z0 + H, pc, post); line(a1, b0, z0, z0 + H, pc, post);          // задние стойки
        var d0 = P(so, a0, b0, z0), d1 = P(so, a1, b0, z0 + H * 0.96);                    // диагональная связь
        ctx.strokeStyle = 'rgba(70,92,118,0.6)'; ctx.lineWidth = Math.max(1, post * 0.6);
        ctx.beginPath(); ctx.moveTo(d0[0], d0[1]); ctx.lineTo(d1[0], d1[1]); ctx.stroke();
        for (var k = 0; k < 5; k++) {                                                      // полки
            var z = z0 + H * 0.04 + zs * k;
            poly([P(so, a0, b0, z + tb), P(so, a1, b0, z + tb), P(so, a1, b1, z + tb), P(so, a0, b1, z + tb)], 'rgba(110,130,150,0.16)', lit ? '#8fefff' : 'rgba(110,200,255,0.5)', 0.9);
            var frontf = [P(so, a0, b1, z + tb), P(so, a1, b1, z + tb), P(so, a1, b1, z), P(so, a0, b1, z)];
            poly(frontf, lin(0, frontf[0][1], 0, frontf[2][1], [[0, '#5a7898'], [1, '#2a3b4e']]));
        }
        var pk = r.packed && r.packed[u];
        if (pk) {
            for (var k2 = 0; k2 < 5; k2++) {
                var zb = z0 + H * 0.04 + zs * k2 + tb, unitH = (zs - tb - H * 0.012) / SH_CLEAR;
                pk[k2].items.forEach(function (pl) { drawShelfItem(so, a0, a1, b0, b1, zb + pl.off * unitH, pl.it.su * unitH, pl.it, lit); });
            }
        }
        line(a0, b1, z0, z0 + H, pc, post * 1.15); line(a1, b1, z0, z0 + H, pc, post * 1.15);   // передние стойки
        labels.push({ a0: a0, a1: a1, b0: b0, b1: b1, z: z0 + H, sel: isSel });
        var hl = [P(so, a0, b1, z0), P(so, a1, b1, z0), P(so, a1, b0, z0), P(so, a1, b0, z0 + H), P(so, a0, b0, z0 + H), P(so, a0, b1, z0 + H)];
        if (isSel) poly(hl, SEL_TINT, '#58e6ff', lw(2.6)); else if (isHov) poly(hl, HOV_TINT, '#8fefff', lw(2.2));
        // световой столб над стеллажом по общей мощности
        hits.push({ site: site, rack: r, pts: hl });
    }
    function drawShelfLabel(so, L) {
        if (cam.k < 0.22) return;
        ctx.save();
        faceTransform(P(so, L.a0, L.b0, L.z), (L.a1 - L.a0) * U, (L.a1 - L.a0) * U / 2, -(L.b1 - L.b0) * U, (L.b1 - L.b0) * U / 2, 100, 130);
        ctx.font = '800 17px ' + FONT; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillStyle = 'rgba(0,0,0,0.6)'; ctx.fillText('СТЕЛЛАЖ', 51, 66);
        ctx.fillStyle = L.sel ? '#7df0ff' : '#e6e9f2'; ctx.fillText('СТЕЛЛАЖ', 50, 65);
        ctx.restore();
    }

    // ---- один ЦОД: тонкая основа + шкафы ----
    function drawSite(o, site, zf) {
        var t = site.e * site.e * (3 - 2 * site.e) * 1.5;              // выдвижение (плавно)
        var p0 = P(o, site.aOff - t, t, zf);
        var so = { x: p0[0], y: p0[1] };
        var A = site.sA, B = site.sB;
        var isSel = selSite === site, isHov = hovSite === site;

        if (site.e > 0.01) {                                             // контур "пустого места" в стопке
            var q0 = P(o, site.aOff, 0, zf);
            var oo = { x: q0[0], y: q0[1] };
            ctx.save(); ctx.setLineDash([6, 5]);
            poly([P(oo, 0, 0, SLAB), P(oo, A, 0, SLAB), P(oo, A, B, SLAB), P(oo, 0, B, SLAB)], null, 'rgba(150,160,255,0.45)', lw(1.2));
            ctx.restore();
            poly([P(so, 0.1, B + 0.1, 0), P(so, A + 0.15, B + 0.1, 0), P(so, A + 0.15, 0.1, 0), P(so, 0.1, 0.1, 0)], 'rgba(0,0,0,' + (0.25 * site.e).toFixed(2) + ')');
        }
        var top = [P(so, 0, 0, SLAB), P(so, A, 0, SLAB), P(so, A, B, SLAB), P(so, 0, B, SLAB)];
        var fL = [P(so, 0, B, SLAB), P(so, A, B, SLAB), P(so, A, B, 0), P(so, 0, B, 0)];
        var fR = [P(so, A, B, SLAB), P(so, A, 0, SLAB), P(so, A, 0, 0), P(so, A, B, 0)];
        poly(fL, '#1a1766'); poly(fR, '#14124d');
        poly(top, lin(0, top[0][1], 0, top[2][1], [[0, '#2b2585'], [1, '#1d1a63']]));
        var edge = isSel ? '#58e6ff' : (isHov ? '#8fefff' : 'rgba(110,200,255,0.6)');
        var elw = lw(isSel ? 2.4 : (isHov ? 2 : 1.4));
        poly(top, null, edge, elw); poly(fL, null, edge, elw); poly(fR, null, edge, elw);
        if (cam.k > 0.5) {
            ctx.strokeStyle = 'rgba(140,150,255,0.16)'; ctx.lineWidth = lw(1);
            for (var i = 1; i < A; i++) { var p = P(so, i, 0, SLAB), q = P(so, i, B, SLAB); ctx.beginPath(); ctx.moveTo(p[0], p[1]); ctx.lineTo(q[0], q[1]); ctx.stroke(); }
            for (var j = 1; j < B; j++) { var p2 = P(so, 0, j, SLAB), q2 = P(so, A, j, SLAB); ctx.beginPath(); ctx.moveTo(p2[0], p2[1]); ctx.lineTo(q2[0], q2[1]); ctx.stroke(); }
        }
        hits.push({ site: site, rack: null, pts: top });

        // шкафы: сзади вперёд
        var rs = site.normal.map(function (r, i) { return { r: r, gi: i % site.cols, gj: Math.floor(i / site.cols) }; });
        rs.sort(function (x, y) { return (x.gi + x.gj) - (y.gi + y.gj) || x.gi - y.gi; });
        site.shelves.forEach(function (r) { for (var u = 0; u < r.units; u++) rs.push({ r: r, gi: r.gi + u, gj: r.gj, shelfUnit: u }); });
        rs.sort(function (x, y) { return (x.gi + x.gj) - (y.gi + y.gj) || x.gi - y.gi; });
        var shelfLabels = [];
        rs.forEach(function (it) { if (it.shelfUnit !== undefined) drawShelfUnit(so, site, it.r, it.shelfUnit, shelfLabels); else drawRack(so, site, it.r, it.gi, it.gj); });
        shelfLabels.forEach(function (L) { drawShelfLabel(so, L); });

        // выноска с названием в плоскости этажа - перед передней кромкой основы (поверх всего)
        overlay.push({ site: site, so: so, A: A, B: B });
    }

    function drawCallout(c) {
        var site = c.site, so = c.so, isSel = selSite === site, isHov = hovSite === site;
        var pa0 = 0.1, pa1 = c.A - 0.1, pb0 = c.B + 0.12, pb1 = c.B + 0.78;
        var plate = [P(so, pa0, pb0, SLAB), P(so, pa1, pb0, SLAB), P(so, pa1, pb1, SLAB), P(so, pa0, pb1, SLAB)];
        poly(plate, 'rgba(14,13,48,0.94)', isSel ? '#58e6ff' : (isHov ? '#8fefff' : 'rgba(120,130,255,0.75)'), lw(1.3));
        if (cam.k < 0.2) return;
        var Wl = 300, Hl = Math.max(30, Math.round(300 * (pb1 - pb0) / (pa1 - pa0)));
        ctx.save();
        faceTransform(P(so, pa0, pb0, SLAB), (pa1 - pa0) * U, (pa1 - pa0) * U / 2, -(pb1 - pb0) * U, (pb1 - pb0) * U / 2, Wl, Hl);
        var txt = site.room, fsz = Hl * 0.62;
        ctx.font = '800 ' + fsz + 'px ' + FONT;
        var tw = ctx.measureText(txt).width;
        if (tw > Wl - 14) { fsz = Math.max(10, Math.floor(fsz * (Wl - 14) / tw)); ctx.font = '800 ' + fsz + 'px ' + FONT; }
        while (txt.length > 2 && ctx.measureText(txt).width > Wl - 14) txt = txt.slice(0, -2) + '…';
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillStyle = isSel ? '#7df0ff' : '#ffffff';
        ctx.fillText(txt, Wl / 2, Hl / 2 + 1);
        ctx.restore();
    }

    function drawBuilding(bx, by, bl) {
        var o = { x: bx + (bl.B + PAD_L) * U, y: by + (2.3 + (bl.n - 1) * bl.FH) * U };
        // пунктир по задней грани, соединяющий этажи
        if (bl.n > 1) {
            var d0 = P(o, 0, 0, SLAB), d1 = P(o, 0, 0, (bl.n - 1) * bl.FH + SLAB);
            ctx.save(); ctx.setLineDash([7, 6]); ctx.strokeStyle = 'rgba(150,160,255,0.65)'; ctx.lineWidth = lw(1.4);
            ctx.beginPath(); ctx.moveTo(d0[0], d0[1]); ctx.lineTo(d1[0], d1[1]); ctx.stroke(); ctx.restore();
        }
        var late = [];
        bl.b.floors.forEach(function (fl, k) {
            var zf = k * bl.FH;
            fl.sites.forEach(function (s) {
                if (s.e > 0.01 || s === selSite) { late.push({ s: s, zf: zf }); return; }
                drawSite(o, s, zf);
            });
            var lp = P(o, 0, fl.B, zf + SLAB);
            overlay.push({ label: floorTitle(fl.floor), x: lp[0] - 12, y: lp[1] - 4 });
        });
        late.forEach(function (it) { drawSite(o, it.s, it.zf); });      // выдвинутые - поверх стопки
        var ax = o.x + (bl.A - bl.B) * U / 2, ay = o.y + (bl.A + bl.B) * U / 2 + 1.35 * U;
        overlay.push({ addr: bl.b.address, x: ax, y: ay, maxW: Math.max(bl.w, 6 * U) });
    }

    function drawOverlayText(c) {
        ctx.save();
        if (c.label !== undefined) {
            ctx.font = '600 15px ' + FONT; ctx.textAlign = 'right'; ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillText(c.label, c.x + 1, c.y + 1);
            ctx.fillStyle = '#a9aef0'; ctx.fillText(c.label, c.x, c.y);
        } else {
            var fs = 20;
            ctx.font = '700 ' + fs + 'px ' + FONT;
            var tw = ctx.measureText(c.addr).width;
            if (tw > c.maxW) { fs = Math.max(11, Math.floor(fs * c.maxW / tw)); ctx.font = '700 ' + fs + 'px ' + FONT; }
            var t = c.addr; while (t.length > 3 && ctx.measureText(t).width > c.maxW) t = t.slice(0, -2) + '…';
            ctx.textAlign = 'center'; ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillText(t, c.x + 1, c.y + 1);
            ctx.fillStyle = '#e8e8ff'; ctx.fillText(t, c.x, c.y);
        }
        ctx.restore();
    }

    // ---- левый список городов: колонка в экранных координатах, плашка на высоту строки города ----
    function drawCityList() {
        cityBoxes = [];
        ctx.save();
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        ctx.fillStyle = 'rgba(14,13,48,0.92)';
        ctx.fillRect(0, 0, LW, G.h);
        ctx.strokeStyle = 'rgba(120,130,255,0.5)'; ctx.lineWidth = 1;
        ctx.beginPath(); ctx.moveTo(LW + 0.5, 0); ctx.lineTo(LW + 0.5, G.h); ctx.stroke();
        cities.forEach(function (ct) {
            var top = cam.ty + (ct.y - 0.3 * U) * cam.k, bot = cam.ty + (ct.y + ct.h + 0.6 * U) * cam.k;
            if (bot < 0 || top > G.h) return;
            var t0 = Math.max(top, 0), b0 = Math.min(bot, G.h);
            cityBoxes.push({ ct: ct, y0: t0, y1: b0 });
            var isCur = hovCity === ct;
            ctx.fillStyle = isCur ? 'rgba(70,80,200,0.55)' : 'rgba(40,40,120,0.45)';
            ctx.fillRect(6, t0 + 3, LW - 12, Math.max(10, b0 - t0 - 6));
            ctx.strokeStyle = isCur ? '#58e6ff' : 'rgba(110,200,255,0.45)';
            ctx.strokeRect(6.5, t0 + 3.5, LW - 13, Math.max(10, b0 - t0 - 6) - 1);
            var cy = (t0 + b0) / 2;
            var sc = 0; ct.blds.forEach(function (b) { b.b.floors.forEach(function (f) { sc += f.sites.length; }); });
            ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
            var fs = 17, nm = ct.c.name;
            ctx.font = '800 ' + fs + 'px ' + FONT;
            while (nm.length > 3 && ctx.measureText(nm).width > LW - 28) nm = nm.slice(0, -2) + '…';
            ctx.fillStyle = '#ffffff'; ctx.fillText(nm, 14, cy - 8);
            ctx.font = '600 12px ' + FONT; ctx.fillStyle = '#a9aef0';
            ctx.fillText('зданий ' + ct.blds.length + ' · ЦОД ' + sc, 14, cy + 11);
        });
        ctx.restore();
    }
    function cityAt(mx, my) {
        if (mx > LW) return null;
        for (var i = 0; i < cityBoxes.length; i++) if (my >= cityBoxes[i].y0 && my <= cityBoxes[i].y1) return cityBoxes[i].ct;
        return null;
    }
    function fitCity(ct) {                          // показать строку города целиком
        var m = 30, availW = G.w - LW - 2 * m, availH = G.h - 2 * m;
        var k = Math.min(availW / ct.w, availH / (ct.h + 0.6 * U));
        cam.k = Math.max(0.05, Math.min(1.5, k));
        cam.tx = LW + m; cam.ty = m - (ct.y - 0.3 * U) * cam.k;
        if ((ct.h + 0.6 * U) * cam.k < availH) cam.ty += (availH - (ct.h + 0.6 * U) * cam.k) / 2;
        redraw();
    }

    // ---- сцена ----
    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        ctx.fillStyle = lin(0, 0, 0, G.h, [[0, '#1b1763'], [1, '#100e3a']]);
        ctx.fillRect(0, 0, G.w, G.h);
        ctx.save();
        ctx.translate(cam.tx, cam.ty); ctx.scale(cam.k, cam.k);
        var vx0 = -cam.tx / cam.k, vy0 = -cam.ty / cam.k, vx1 = vx0 + G.w / cam.k, vy1 = vy0 + G.h / cam.k;
        hits = []; overlay = [];
        // подложка строк городов (чередование) - на всю видимую ширину
        cities.forEach(function (ct, i) {
            if (ct.y > vy1 || ct.y + ct.h + 1.2 * U < vy0) return;
            ctx.fillStyle = i % 2 ? 'rgba(255,255,255,0.025)' : 'rgba(0,0,0,0.12)';
            ctx.fillRect(vx0, ct.y - 0.3 * U, vx1 - vx0, ct.h + 0.9 * U);
        });
        cities.forEach(function (ct) {
            if (ct.x > vx1 || ct.x + ct.w < vx0 || ct.y > vy1 || ct.y + ct.h < vy0) return;
            ct.blds.forEach(function (bl) {
                var bx = ct.x + bl.x, by = ct.y + bl.y;
                if (bx > vx1 || bx + bl.w < vx0 || by > vy1 || by + bl.h < vy0) return;
                drawBuilding(bx, by, bl);
            });
        });
        overlay.forEach(function (c) { if (c.site) drawCallout(c); else drawOverlayText(c); });
        ctx.restore();
        drawCityList();
        pctBtn.textContent = Math.round(cam.k * 100) + '%';
    }
    var raf = 0;
    function redraw() { if (!raf) raf = requestAnimationFrame(frame); }
    function frame() {
        raf = 0;
        var busy = false;
        allSites.forEach(function (s) {
            if (s.e !== s.tgt) {
                var d = s.tgt - s.e;
                s.e = Math.abs(d) < 0.02 ? s.tgt : s.e + d * 0.22;
                busy = true;
            }
        });
        render();
        if (busy) redraw();
    }

    // ---- камера ----
    function fit() {
        var m = 40, availW = G.w - LW - 2 * m;
        var k = Math.min(availW / WORLD.w, (G.h - 2 * m) / WORLD.h);
        cam.k = Math.max(0.05, Math.min(1.2, k));
        cam.tx = LW + m + Math.max(0, (availW - WORLD.w * cam.k) / 2);
        cam.ty = m;
        if (WORLD.h * cam.k + 2 * m < G.h) cam.ty = (G.h - WORLD.h * cam.k) / 2;
    }
    var fitted = false;
    function resize() {
        var rect = stage.getBoundingClientRect();
        G.dpr = window.devicePixelRatio || 1;
        G.w = Math.max(200, rect.width); G.h = Math.max(200, rect.height);
        canvas.width = Math.round(G.w * G.dpr); canvas.height = Math.round(G.h * G.dpr);
        if (!fitted) { fit(); fitted = true; }
        render();
    }
    function zoomAt(f, px, py) {
        var nk = Math.max(0.05, Math.min(4, cam.k * f));
        f = nk / cam.k;
        cam.tx = px - (px - cam.tx) * f; cam.ty = py - (py - cam.ty) * f; cam.k = nk;
        redraw();
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
    function pick(mx, my) {
        var wx = (mx - cam.tx) / cam.k, wy = (my - cam.ty) / cam.k;
        for (var i = hits.length - 1; i >= 0; i--) if (inPoly([wx, wy], hits[i].pts)) return hits[i];
        return null;
    }

    // ---- подсказка/панель ----
    function siteInfo(s) {
        var h = '<b class="sm-name">' + esc(s.name) + '</b>';
        h += '<div class="sm-sub">' + floorTitle(s.floor) + (s.room && s.room !== s.name ? ' &middot; комната ' + esc(s.room) : '') + ' &middot; шкафов ' + s.normal.length + (s.shelves.length ? ' + стеллаж' : '') + '</div><table>';
        h += '<tr><td>Серверов</td><td><b>' + s.servers + '</b>' + (s.blades ? ' (+ блейдов ' + s.blades + ')' : '') + '</td></tr>';
        h += '<tr><td>Блейд-корзин</td><td><b>' + s.chassis + '</b></td></tr>';
        h += '<tr><td>СХД</td><td><b>' + s.storages + '</b></td></tr>';
        h += '<tr><td>Сетевого оборудования</td><td><b>' + s.switches + '</b></td></tr>';
        h += '<tr><td>Ленточных библиотек</td><td><b>' + s.tapes + '</b></td></tr>';
        h += '<tr><td>Мощность</td><td>' + (s.powerAvgW > 0 ? '<b>' + kw(s.powerAvgW) + '</b> (ср. 24ч)' : '<b>нет данных</b>') + '</td></tr>';
        if (s.powerPeakW > 0) h += '<tr><td>Пик 24ч</td><td>' + kw(s.powerPeakW) + '</td></tr>';
        if (s.powerRatedW > 0) h += '<tr><td>Паспорт</td><td>' + kw(s.powerRatedW) + '</td></tr>';
        return h + '</table>';
    }
    function rackInfo(r) {
        var h = '<b class="sm-name">' + (r.shelf ? 'Стеллаж (оборудование вне шкафов)' : 'Шкаф ' + esc(r.name)) + '</b><div class="sm-sub">' + (r.shelf ? 'секций ' + (r.units || 1) + ' &middot; 5 полок' : r.uHeight + 'U') + '</div><table>';
        h += '<tr><td>Серверов</td><td><b>' + r.servers + '</b>' + (r.blades ? ' (+ блейдов ' + r.blades + ')' : '') + '</td></tr>';
        h += '<tr><td>Корзин</td><td><b>' + r.chassis + '</b></td></tr>';
        h += '<tr><td>СХД</td><td><b>' + r.storages + '</b></td></tr>';
        if (r.tapes) h += '<tr><td>Ленточных библиотек</td><td><b>' + r.tapes + '</b></td></tr>';
        if (r.network) h += '<tr><td>Сетевого оборудования</td><td><b>' + r.network + '</b></td></tr>';
        h += '<tr><td>Мощность</td><td>' + (r.powerAvgW > 0 ? '<b>' + kw(r.powerAvgW) + '</b> (ср. 24ч)' : '<b>нет данных</b>') + '</td></tr>';
        if (r.powerPeakW > 0) h += '<tr><td>Пик 24ч</td><td>' + kw(r.powerPeakW) + '</td></tr>';
        if (r.powerRatedW > 0) h += '<tr><td>Паспорт</td><td>' + kw(r.powerRatedW) + '</td></tr>';
        return h + '</table>';
    }
    function showTip(hit, x, y) {
        tip.innerHTML = (hit.rack ? rackInfo(hit.rack) + '<div class="sm-sub" style="margin-top:.3rem">ЦОД: ' + esc(hit.site.name) + '</div>' : siteInfo(hit.site)) + '<div class="sm-hint">Клик - выделить</div>';
        tip.hidden = false;
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var px = x + 16, py = y + 16;
        if (px + tw > G.w - 4) px = x - tw - 16;
        if (py + th > G.h - 4) py = G.h - th - 4;
        tip.style.left = Math.max(4, px) + 'px'; tip.style.top = Math.max(4, py) + 'px';
    }
    function select(site, rack) {
        if (selSite && selSite !== site) selSite.tgt = 0;
        selSite = site; selRack = rack;
        if (site) {
            site.tgt = 1;
            var h = '<button type="button" class="sm-close" title="Закрыть" id="smPanelClose">&times;</button>' + siteInfo(site);
            h += '<div class="sm-hint"><a href="/sprsrv/datacentersiteview/' + site.id + '" target="_blank" rel="noopener">Открыть оборудование ЦОД</a></div>';
            if (rack) {
                h += '<hr style="border-color:rgba(120,130,255,.4);margin:.5rem 0">' + rackInfo(rack);
                h += '<div class="sm-hint"><a href="/sprsrv/rackcabinetview/' + rack.id + '" target="_blank" rel="noopener">' + (rack.shelf ? 'Открыть стеллаж' : 'Открыть шкаф') + '</a></div>';
            }
            panel.innerHTML = h; panel.hidden = false;
            document.getElementById('smPanelClose').addEventListener('click', function () { select(null, null); });
        } else panel.hidden = true;
        redraw();
    }

    // ---- мышь: перетаскивание, клик, колесо ----
    var drag = null;
    canvas.addEventListener('mousedown', function (e) {
        if (e.button !== 0) return;
        drag = { x: e.clientX, y: e.clientY, tx: cam.tx, ty: cam.ty, moved: false };
        tip.hidden = true;
    });
    window.addEventListener('mousemove', function (e) {
        if (!drag) return;
        var dx = e.clientX - drag.x, dy = e.clientY - drag.y;
        if (!drag.moved && Math.abs(dx) + Math.abs(dy) > 7) { drag.moved = true; canvas.classList.add('sm-drag'); }
        if (drag.moved) { cam.tx = drag.tx + dx; cam.ty = drag.ty + dy; redraw(); }
    });
    window.addEventListener('mouseup', function (e) {
        if (!drag) return;
        var d = drag; drag = null; canvas.classList.remove('sm-drag');
        if (d.moved) return;
        var rect = canvas.getBoundingClientRect();
        if (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom) return;
        var cx = e.clientX - rect.left, cy = e.clientY - rect.top;
        var cc = cityAt(cx, cy);
        if (cc) { fitCity(cc); return; }
        var h = pick(cx, cy);
        if (!h) select(null, null);
        else if (h.rack) select(h.site, h.rack === selRack ? null : h.rack);
        else select(h.site === selSite && !selRack ? null : h.site, null);
    });
    canvas.addEventListener('mousemove', function (e) {
        if (drag && drag.moved) return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        var hc = cityAt(x, y);
        if (hc !== hovCity) { hovCity = hc; canvas.style.cursor = hc ? 'pointer' : ''; redraw(); }
        var h = hc ? null : pick(x, y);
        var ns = h ? h.site : null, nr = h ? h.rack : null;
        if (ns !== hovSite || nr !== hovRack) { hovSite = ns; hovRack = nr; redraw(); }
        if (h) showTip(h, x, y); else tip.hidden = true;
    });
    canvas.addEventListener('mouseleave', function () { hovCity = null; hovSite = null; hovRack = null; tip.hidden = true; redraw(); });
    canvas.addEventListener('wheel', function (e) {
        e.preventDefault();
        var rect = canvas.getBoundingClientRect();
        var dy = e.deltaMode === 1 ? e.deltaY * 16 : e.deltaY;
        zoomAt(Math.pow(1.0018, -dy), e.clientX - rect.left, e.clientY - rect.top);
        tip.hidden = true;
    }, { passive: false });

    document.getElementById('smZoomIn').addEventListener('click', function () { zoomAt(1.25, G.w / 2, G.h / 2); });
    document.getElementById('smZoomOut').addEventListener('click', function () { zoomAt(0.8, G.w / 2, G.h / 2); });
    pctBtn.addEventListener('click', function () { fit(); redraw(); });


    // ---- контекстное меню (правая кнопка): переход к оборудованию площадки/ЦОД ----
    var ctxMenu = document.getElementById('smCtx');
    function hideCtx() { if (ctxMenu) { ctxMenu.hidden = true; ctxMenu.innerHTML = ''; } }
    function showCtx(hit, x, y) {
        var s = hit.site, h = '<div class="sm-ctx-title">' + esc(s.name) + ' &middot; ' + floorTitle(s.floor) + '</div>';
        h += '<a href="/sprsrv/datacentersiteview/' + s.id + '" target="_blank" rel="noopener">Просмотр оборудования площадки/ЦОД</a>';
        if (hit.rack) h += '<a href="/sprsrv/rackcabinetview/' + hit.rack.id + '" target="_blank" rel="noopener">' + (hit.rack.shelf ? 'Открыть стеллаж' : 'Открыть шкаф ' + esc(hit.rack.name)) + '</a>';
        h += '<a href="/sprsrv/datacentersiteedit/' + s.id + '" target="_blank" rel="noopener">Редактировать площадку/ЦОД</a>';
        ctxMenu.innerHTML = h; ctxMenu.hidden = false;
        var tw = ctxMenu.offsetWidth, th = ctxMenu.offsetHeight;
        ctxMenu.style.left = Math.max(4, Math.min(x, G.w - tw - 4)) + 'px';
        ctxMenu.style.top = Math.max(4, Math.min(y, G.h - th - 4)) + 'px';
    }
    canvas.addEventListener('contextmenu', function (e) {
        e.preventDefault();
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        if (cityAt(x, y)) { hideCtx(); return; }
        var h = pick(x, y);
        if (!h) { hideCtx(); return; }
        tip.hidden = true;
        select(h.site, h.rack || null);
        showCtx(h, x, y);
    });
    document.addEventListener('click', function (e) { if (ctxMenu && !ctxMenu.contains(e.target)) hideCtx(); });
    document.addEventListener('keydown', function (e) { if (e.key === 'Escape') hideCtx(); });
    canvas.addEventListener('mousedown', hideCtx);
    canvas.addEventListener('wheel', hideCtx, { passive: true });

    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    resize();
})();
</script>
