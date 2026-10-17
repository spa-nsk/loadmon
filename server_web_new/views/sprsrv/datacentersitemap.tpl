<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Карта площадок/ЦОД</h2>
        <div class="text-muted" id="smSummary">Площадок: {{.SiteCount}}</div>
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
        <div class="sm-bar-text">Города &rarr; здания (адреса) &rarr; этажи &rarr; комнаты. Колесо мыши - масштаб, перетаскивание левой кнопкой - прокрутка, клик по площадке - детали.</div>
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
    var U = 70;                                   // пикселей мира на клетку
    var FONT = '"Segoe UI", Arial, sans-serif';
    var PLATE_T = 0.12, ROOM_H = 0.28;            // толщина плиты этажа и высота "коробки" комнаты
    var cam = { k: 1, tx: 0, ty: 0 };
    var G = { w: 0, h: 0, dpr: 1 };
    var hovered = null, selected = null;
    var hits = [];                                 // [{s, pts}] в координатах мира
    var callouts = [];                             // выноски - рисуются отдельным проходом поверх всего

    // ---- вспомогательное ----
    var kw = function (w) { return w >= 1000 ? (w / 1000).toFixed(1) + ' кВт' : Math.round(w) + ' Вт'; };
    function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    function floorTitle(f) { return f === 0 ? 'Подвал (0)' : 'Этаж ' + f; }

    // ---- раскладка: города -> здания -> этажи -> комнаты ----
    function shelf(items, limit, gap) {            // раскладка блоков {w,h} по строкам; возвращает размеры и позиции
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

    var siteCnt = 0, bldCnt = 0;
    var cities = DATA.map(function (c) {
        var bl = c.buildings.map(function (b) {
            var M = 1; b.floors.forEach(function (f) { M = Math.max(M, f.sites.length); siteCnt += f.sites.length; });
            var gc = Math.min(M, 4), gr = Math.ceil(M / gc);
            var A = gc * 1.15 + 0.15, B = gr * 1.15 + 0.15, n = b.floors.length;
            var FH = (A + B) / 2 + 1.0;
            bldCnt++;
            return { b: b, gc: gc, A: A, B: B, n: n, FH: FH,
                     w: (A + B) * U, h: (0.4 + (n - 1) * FH + (A + B) / 2 + 1.0) * U };
        });
        var sh = shelf(bl, 40 * U, 2.4 * U);
        return { c: c, blds: bl, w: Math.max(sh.w, 6 * U), h: sh.h + 1.8 * U, innerH: sh.h };
    });
    var cl = shelf(cities, Math.max(Math.max.apply(null, cities.map(function (c) { return c.w; })),
                                   cities.reduce(function (s, c) { return s + c.w; }, 0) / 2), 3.5 * U);
    var WORLD = { w: cl.w, h: cl.h };

    (function () {
        document.getElementById('smSummary').innerHTML = 'Городов: <b>' + cities.length + '</b> &middot; зданий: <b>' + bldCnt + '</b> &middot; площадок: <b>' + siteCnt + '</b>';
    })();

    // изометрия здания: a - вправо-вниз, b - влево-вниз, z - вверх (в клетках)
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

    // ---- рисование одной комнаты (площадки) ----
    function drawSite(o, s, a0, b0, zf) {
        var a1 = a0 + 1.0, b1 = b0 + 1.0;
        var zb = zf + PLATE_T, zt = zb + ROOM_H;
        var isSel = selected === s, isHov = hovered === s, lit = isSel || isHov;
        var dc = s.isDc;

        var left = [P(o, a0, b1, zt), P(o, a1, b1, zt), P(o, a1, b1, zb), P(o, a0, b1, zb)];
        var right = [P(o, a1, b1, zt), P(o, a1, b0, zt), P(o, a1, b0, zb), P(o, a1, b1, zb)];
        var top = [P(o, a0, b0, zt), P(o, a1, b0, zt), P(o, a1, b1, zt), P(o, a0, b1, zt)];

        poly(left, lin(0, left[0][1], 0, left[3][1], dc ? [[0, lit ? '#3b3695' : '#2f2b7c'], [1, '#221f63']] : [[0, lit ? '#47475f' : '#3a3a50'], [1, '#2a2a3c']]));
        poly(right, lin(0, right[0][1], 0, right[3][1], dc ? [[0, lit ? '#2c2985' : '#25226c'], [1, '#15133f']] : [[0, '#2f2f45'], [1, '#1c1c2a']]));
        poly(top, dc ? (lit ? '#4a45b0' : '#3a3592') : (lit ? '#58587a' : '#46465f'));

        // плитки на крыше
        if (cam.k > 0.3) {
            ctx.strokeStyle = 'rgba(140,150,255,0.22)'; ctx.lineWidth = lw(1);
            for (var i = 1; i < 5; i++) {
                var t = i / 5;
                var p0 = P(o, a0 + t, b0, zt), p1 = P(o, a0 + t, b1, zt), q0 = P(o, a0, b0 + t, zt), q1 = P(o, a1, b0 + t, zt);
                ctx.beginPath(); ctx.moveTo(p0[0], p0[1]); ctx.lineTo(p1[0], p1[1]); ctx.moveTo(q0[0], q0[1]); ctx.lineTo(q1[0], q1[1]); ctx.stroke();
            }
        }
        // мини-шкафы (в стиле левого вида): сетка 4x3 в задней части крыши
        if (cam.k > 0.45) {
            var nr = Math.min(s.racks, 12);
            for (var r = 0; r < nr; r++) {
                var ca = a0 + 0.12 + (r % 4) * 0.21, cb = b0 + 0.1 + Math.floor(r / 4) * 0.2;
                var ra0 = ca, ra1 = ca + 0.15, rb0 = cb, rb1 = cb + 0.14, rh = zt + 0.22;
                poly([P(o, ra0, rb1, rh), P(o, ra1, rb1, rh), P(o, ra1, rb1, zt), P(o, ra0, rb1, zt)], '#2c2f37');
                poly([P(o, ra1, rb1, rh), P(o, ra1, rb0, rh), P(o, ra1, rb0, zt), P(o, ra1, rb1, zt)], '#202228');
                poly([P(o, ra0, rb0, rh), P(o, ra1, rb0, rh), P(o, ra1, rb1, rh), P(o, ra0, rb1, rh)], '#4a4e5a', 'rgba(255,255,255,0.14)', lw(0.8));
            }
        }
        var edge = isSel ? '#58e6ff' : (isHov ? '#ffffff' : (dc ? 'rgba(110,200,255,0.55)' : 'rgba(190,190,220,0.4)'));
        var elw = lw(isSel ? 2.6 : (isHov ? 2.2 : 1.3));
        poly(left, null, edge, elw); poly(right, null, edge, elw); poly(top, null, edge, elw);

        hits.push({ s: s, pts: [P(o, a0, b1, zb), P(o, a1, b1, zb), P(o, a1, b0, zb), P(o, a1, b0, zt), P(o, a0, b0, zt), P(o, a0, b1, zt)] });
        callouts.push({ s: s, o: o, a0: a0, a1: a1, b0: b0, b1: b1, zt: zt });
    }

    // ---- выноска: плашка в плоскости этажа над комнатой + линия к крыше ----
    function drawCallout(c) {
        var s = c.s, o = c.o, isSel = selected === s, isHov = hovered === s;
        var pa0 = c.a0 + 0.04, pa1 = c.a1 - 0.04, pb0 = c.b0 + 0.58, pb1 = c.b1 - 0.02, zc = c.zt + 0.5;
        var Wl = 200, Hl = 64;
        var org = P(o, pa0, pb0, zc);
        var amid = (pa0 + pa1) / 2;
        var f0 = P(o, amid, pb1, zc), f1 = P(o, amid, pb1, c.zt);
        ctx.strokeStyle = isSel ? '#58e6ff' : 'rgba(150,160,255,0.8)'; ctx.lineWidth = lw(1.2);
        ctx.beginPath(); ctx.moveTo(f0[0], f0[1]); ctx.lineTo(f1[0], f1[1]); ctx.stroke();
        ctx.fillStyle = ctx.strokeStyle; ctx.beginPath(); ctx.arc(f1[0], f1[1], 2.2, 0, TAU); ctx.fill();

        var plate = [P(o, pa0, pb0, zc), P(o, pa1, pb0, zc), P(o, pa1, pb1, zc), P(o, pa0, pb1, zc)];
        poly(plate, 'rgba(14,13,48,0.94)', isSel ? '#58e6ff' : (isHov ? '#ffffff' : 'rgba(120,130,255,0.75)'), lw(1.3));
        if (cam.k < 0.3) return;
        ctx.save();
        faceTransform(org, (pa1 - pa0) * U, (pa1 - pa0) * U / 2, -(pb1 - pb0) * U, (pb1 - pb0) * U / 2, Wl, Hl);
        var txt = s.room, fsz = 40;
        ctx.font = '800 ' + fsz + 'px ' + FONT;
        var tw = ctx.measureText(txt).width;
        if (tw > 186) { fsz = Math.max(15, Math.floor(fsz * 186 / tw)); ctx.font = '800 ' + fsz + 'px ' + FONT; }
        while (txt.length > 2 && ctx.measureText(txt).width > 186) txt = txt.slice(0, -2) + '…';
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillStyle = isSel ? '#7df0ff' : '#ffffff';
        ctx.fillText(txt, Wl / 2, Hl / 2 + 1);
        ctx.restore();
    }

    function drawBuilding(bx, by, bl, vis) {
        var o = { x: bx + bl.B * U, y: by + (0.4 + (bl.n - 1) * bl.FH) * U };
        var A = bl.A, B = bl.B;
        bl.b.floors.forEach(function (fl, k) {
            var zf = k * bl.FH;
            // плита этажа
            var pl = [P(o, 0, 0, zf + PLATE_T), P(o, A, 0, zf + PLATE_T), P(o, A, B, zf + PLATE_T), P(o, 0, B, zf + PLATE_T)];
            var fL = [P(o, 0, B, zf + PLATE_T), P(o, A, B, zf + PLATE_T), P(o, A, B, zf), P(o, 0, B, zf)];
            var fR = [P(o, A, B, zf + PLATE_T), P(o, A, 0, zf + PLATE_T), P(o, A, 0, zf), P(o, A, B, zf)];
            poly(fL, '#1a1766'); poly(fR, '#14124d');
            poly(pl, lin(0, pl[0][1], 0, pl[2][1], [[0, '#2b2585'], [1, '#1d1a63']]), 'rgba(110,200,255,0.55)', lw(1.5));
            poly(fL, null, 'rgba(110,200,255,0.4)', lw(1)); poly(fR, null, 'rgba(110,200,255,0.4)', lw(1));
            if (cam.k > 0.3) {
                ctx.strokeStyle = 'rgba(140,150,255,0.15)'; ctx.lineWidth = lw(1);
                for (var i = 1; i < A; i++) { var p = P(o, i, 0, zf + PLATE_T), q = P(o, i, B, zf + PLATE_T); ctx.beginPath(); ctx.moveTo(p[0], p[1]); ctx.lineTo(q[0], q[1]); ctx.stroke(); }
                for (var j = 1; j < B; j++) { var p2 = P(o, 0, j, zf + PLATE_T), q2 = P(o, A, j, zf + PLATE_T); ctx.beginPath(); ctx.moveTo(p2[0], p2[1]); ctx.lineTo(q2[0], q2[1]); ctx.stroke(); }
            }
            // комнаты: сзади вперёд
            var rooms = fl.sites.map(function (s, i) { return { s: s, col: i % bl.gc, row: Math.floor(i / bl.gc) }; });
            rooms.sort(function (x, y) { return (x.col + x.row) - (y.col + y.row) || x.row - y.row; });
            rooms.forEach(function (r) { drawSite(o, r.s, 0.1 + r.col * 1.15, 0.1 + r.row * 1.15, zf + 0); });
            // подпись этажа слева от плиты (рисуется поверх, вместе с выносками)
            var lp = P(o, 0, B, zf + PLATE_T);
            callouts.push({ label: floorTitle(fl.floor), x: lp[0] - 10, y: lp[1] - 4 });
        });
        // адрес здания под ним
        var ax = o.x + (A - B) * U / 2, ay = o.y + (A + B) * U / 2 + 0.7 * U;
        callouts.push({ addr: bl.b.address, x: ax, y: ay, maxW: Math.max(bl.w, 6 * U) });
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

    // ---- сцена ----
    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        ctx.fillStyle = lin(0, 0, 0, G.h, [[0, '#1b1763'], [1, '#100e3a']]);
        ctx.fillRect(0, 0, G.w, G.h);
        ctx.save();
        ctx.translate(cam.tx, cam.ty); ctx.scale(cam.k, cam.k);
        var vx0 = -cam.tx / cam.k, vy0 = -cam.ty / cam.k, vx1 = vx0 + G.w / cam.k, vy1 = vy0 + G.h / cam.k;
        hits = []; callouts = [];
        var titles = [];
        cities.forEach(function (ct) {
            var cx = ct.x, cy = ct.y;
            if (cx > vx1 || cx + ct.w < vx0 || cy > vy1 || cy + ct.h < vy0) return;
            titles.push(ct);
            ct.blds.forEach(function (bl) {
                var bx = cx + bl.x, by = cy + 1.8 * U + bl.y;
                if (bx > vx1 || bx + bl.w < vx0 || by > vy1 || by + bl.h < vy0) return;
                drawBuilding(bx, by, bl);
            });
        });
        // заголовки городов
        titles.forEach(function (ct) {
            ctx.strokeStyle = 'rgba(110,200,255,0.4)'; ctx.lineWidth = lw(1.5);
            ctx.beginPath(); ctx.moveTo(ct.x, ct.y + 1.25 * U); ctx.lineTo(ct.x + ct.w, ct.y + 1.25 * U); ctx.stroke();
            ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
            ctx.font = '800 34px ' + FONT;
            ctx.fillStyle = '#ffffff'; ctx.fillText(ct.c.name, ct.x, ct.y + 1.0 * U);
            var nm = ctx.measureText(ct.c.name).width;
            var sc = 0; ct.blds.forEach(function (b) { b.b.floors.forEach(function (f) { sc += f.sites.length; }); });
            ctx.font = '600 16px ' + FONT; ctx.fillStyle = '#8f94d8';
            ctx.fillText('зданий: ' + ct.blds.length + ' · площадок: ' + sc, ct.x + nm + 16, ct.y + 1.0 * U);
        });
        // выноски, подписи этажей и адресов - поверх всей сцены
        callouts.forEach(function (c) { if (c.s) drawCallout(c); else drawOverlayText(c); });
        ctx.restore();
        pctBtn.textContent = Math.round(cam.k * 100) + '%';
    }
    var raf = 0;
    function redraw() { if (!raf) raf = requestAnimationFrame(function () { raf = 0; render(); }); }

    // ---- размеры/камера ----
    function fit() {
        var m = 40;
        var k = Math.min((G.w - 2 * m) / WORLD.w, (G.h - 2 * m) / WORLD.h);
        cam.k = Math.max(0.05, Math.min(1.2, k));
        cam.tx = (G.w - WORLD.w * cam.k) / 2;
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
        for (var i = hits.length - 1; i >= 0; i--) if (inPoly([wx, wy], hits[i].pts)) return hits[i].s;
        return null;
    }

    // ---- подсказка/панель ----
    function info(s, withLink) {
        var h = '<b class="sm-name">' + esc(s.name) + '</b>';
        h += '<div class="sm-sub">' + floorTitle(s.floor) + (s.room && s.room !== s.name ? ' &middot; комната ' + esc(s.room) : '') + ' &middot; ' + (s.isDc ? 'ЦОД' : 'площадка') + '</div><table>';
        h += '<tr><td>Шкафов</td><td><b>' + s.racks + '</b></td></tr>';
        h += '<tr><td>Серверов</td><td><b>' + s.servers + '</b>' + (s.blades ? ' (+ блейдов ' + s.blades + ')' : '') + '</td></tr>';
        h += '<tr><td>Блейд-корзин</td><td><b>' + s.chassis + '</b></td></tr>';
        h += '<tr><td>СХД</td><td><b>' + s.storages + '</b></td></tr>';
        h += '<tr><td>Коммутаторов</td><td><b>' + s.switches + '</b></td></tr>';
        h += '<tr><td>Мощность</td><td>' + (s.powerAvgW > 0 ? '<b>' + kw(s.powerAvgW) + '</b> (ср. 24ч)' : '<b>нет данных</b>') + '</td></tr>';
        if (s.powerPeakW > 0) h += '<tr><td>Пик 24ч</td><td>' + kw(s.powerPeakW) + '</td></tr>';
        if (s.powerRatedW > 0) h += '<tr><td>Паспорт</td><td>' + kw(s.powerRatedW) + '</td></tr>';
        h += '</table>';
        if (withLink) h += '<div class="sm-hint"><a href="/sprsrv/datacentersiteview/' + s.id + '" target="_blank" rel="noopener">Открыть оборудование ЦОД</a></div>';
        return h;
    }
    function showTip(s, x, y) {
        tip.innerHTML = info(s, false) + '<div class="sm-hint">Клик - закрепить</div>';
        tip.hidden = false;
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var px = x + 16, py = y + 16;
        if (px + tw > G.w - 4) px = x - tw - 16;
        if (py + th > G.h - 4) py = G.h - th - 4;
        tip.style.left = Math.max(4, px) + 'px'; tip.style.top = Math.max(4, py) + 'px';
    }
    function select(s) {
        selected = s;
        if (s) {
            panel.innerHTML = '<button type="button" class="sm-close" title="Закрыть" id="smPanelClose">&times;</button>' + info(s, true);
            panel.hidden = false;
            document.getElementById('smPanelClose').addEventListener('click', function () { select(null); });
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
        if (drag) {
            var dx = e.clientX - drag.x, dy = e.clientY - drag.y;
            if (!drag.moved && Math.abs(dx) + Math.abs(dy) > 4) { drag.moved = true; canvas.classList.add('sm-drag'); }
            if (drag.moved) { cam.tx = drag.tx + dx; cam.ty = drag.ty + dy; redraw(); }
        }
    });
    window.addEventListener('mouseup', function (e) {
        if (!drag) return;
        var d = drag; drag = null; canvas.classList.remove('sm-drag');
        if (!d.moved) {
            var rect = canvas.getBoundingClientRect();
            if (e.clientX >= rect.left && e.clientX <= rect.right && e.clientY >= rect.top && e.clientY <= rect.bottom) {
                var s = pick(e.clientX - rect.left, e.clientY - rect.top);
                select(s === selected ? null : s);
            }
        }
    });
    canvas.addEventListener('mousemove', function (e) {
        if (drag && drag.moved) return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        var s = pick(x, y);
        if (s !== hovered) { hovered = s; redraw(); }
        if (s && s !== selected) showTip(s, x, y); else tip.hidden = true;
    });
    canvas.addEventListener('mouseleave', function () { hovered = null; tip.hidden = true; redraw(); });
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

    if (window.ResizeObserver) new ResizeObserver(resize).observe(stage); else window.addEventListener('resize', resize);
    resize();
})();
</script>
