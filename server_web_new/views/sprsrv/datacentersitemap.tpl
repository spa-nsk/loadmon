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
        <div class="sm-bar-text">Только ЦОД: города &rarr; здания &rarr; этажи. Потоки: площадка &harr; здание &harr; город &harr; города. Колесо - масштаб, перетаскивание или полоса справа - прокрутка, клик по шкафу/стеллажу/ЦОД - детали и выбор, правая кнопка - переход к оборудованию площадки.</div>
        <div class="btn-group flex-shrink-0" role="group" aria-label="Масштаб карты">
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomOut" title="Уменьшить">&minus;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomFit" title="Вписать всю карту в окно" style="min-width:3.6rem">100%</button>
            <button type="button" class="btn btn-outline-secondary btn-sm py-0" id="smZoomIn" title="Увеличить">+</button>
        </div>
    </div>
    <div class="sm-filters d-flex flex-wrap align-items-center gap-2">
        <label class="d-flex align-items-center gap-1 mb-0">Город
            <select id="smCitySel" class="form-select form-select-sm" style="width:auto;max-width:15rem;" title="Перейти к городу"></select></label>
        <label class="d-flex align-items-center gap-1 mb-0">Площадка/ЦОД
            <select id="smSiteSel" class="form-select form-select-sm" style="width:auto;max-width:34rem;" title="Перейти к площадке/ЦОД"></select></label>
        <button type="button" id="smAll" class="btn btn-outline-secondary btn-sm py-0" title="Вписать всю карту в окно">Вся карта</button>
        <span class="text-muted small ms-auto">Выбор запоминается</span>
    </div>
    <div class="sm-stage" id="smStage">
        <canvas id="smCanvas"></canvas>
        <canvas id="smVScroll" title="Положение по высоте карты"></canvas>
        <div id="smTip" class="sm-tip" hidden></div>
        <div id="smPanel" class="sm-tip sm-panel" hidden></div>
        <div id="smCtx" class="sm-ctx" hidden></div>
    </div>
</div>

<style>
    .sm-wrap { display: flex; flex-direction: column; gap: 0.4rem; height: calc(100vh - 170px); min-height: 520px; }
    .sm-bar { flex: 0 0 auto; padding: 0.35rem 0.7rem; border-radius: 0.5rem; border: 1px solid var(--bs-border-color); background: var(--bs-tertiary-bg); font-size: 0.85rem; }
    .sm-filters { flex: 0 0 auto; font-size: 0.85rem; }
    #smVScroll { position: absolute; right: 3px; top: 6px; width: 14px; z-index: 4; cursor: pointer; }
    .sm-bar-text { min-width: 0; color: var(--bs-secondary-color); }
    .sm-stage { position: relative; width: 100%; flex: 1 1 auto; min-height: 0; border-radius: 0.6rem; overflow: hidden; border: 1px solid var(--bs-border-color); background: linear-gradient(#171d25, #0e1318); }
    [data-bs-theme="light"] .sm-stage { background: linear-gradient(#f3f6fa, #dbe3ec); }
    #smCanvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; cursor: grab; touch-action: none; }
    #smCanvas.sm-drag { cursor: grabbing; }
    /* всплывающие подсказки, панель и меню - палитра тёмной и светлой тем */
    .sm-tip { position: absolute; z-index: 5; pointer-events: none; min-width: 210px; max-width: 320px; padding: 0.55rem 0.7rem; border-radius: 0.5rem; font-size: 0.82rem; line-height: 1.35;
              background: rgba(17, 23, 30, 0.96); color: #e6edf3; border: 1px solid rgba(110, 190, 190, 0.5); box-shadow: 0 6px 22px rgba(0, 0, 0, 0.45); }
    [data-bs-theme="light"] .sm-tip { background: rgba(255, 255, 255, 0.97); color: #1d2b3a; border-color: rgba(13, 148, 136, 0.45); box-shadow: 0 6px 22px rgba(30, 50, 80, 0.22); }
    .sm-panel { z-index: 6; pointer-events: auto; top: 10px; right: 28px; left: auto; border-color: rgba(45, 212, 191, 0.7); }
    [data-bs-theme="light"] .sm-panel { border-color: rgba(13, 148, 136, 0.7); }
    .sm-tip b.sm-name { font-size: 0.95rem; color: #f8fafc; }
    [data-bs-theme="light"] .sm-tip b.sm-name { color: #0f1b2a; }
    .sm-tip .sm-sub { color: #9fb0c2; font-size: 0.78rem; margin-bottom: 0.2rem; }
    [data-bs-theme="light"] .sm-tip .sm-sub { color: #52667a; }
    .sm-tip table { border-collapse: collapse; margin-top: 0.25rem; width: 100%; }
    .sm-tip td { padding: 1px 0; vertical-align: top; }
    .sm-tip td:first-child { color: #9fb0c2; padding-right: 0.7rem; white-space: nowrap; }
    [data-bs-theme="light"] .sm-tip td:first-child { color: #52667a; }
    .sm-tip .sm-hint { margin-top: 0.35rem; color: #8196aa; font-size: 0.75rem; }
    [data-bs-theme="light"] .sm-tip .sm-hint { color: #64788c; }
    .sm-tip a { color: #5eead4; }
    [data-bs-theme="light"] .sm-tip a { color: #0f766e; }
    .sm-tip hr { border-color: rgba(140, 165, 190, 0.4) !important; }
    .sm-ctx { position: absolute; z-index: 9; min-width: 250px; padding: 0.3rem; border-radius: 0.5rem; background: rgba(17, 23, 30, 0.97); border: 1px solid rgba(110, 190, 190, 0.55); box-shadow: 0 8px 26px rgba(0, 0, 0, 0.55); }
    .sm-ctx .sm-ctx-title { padding: 0.25rem 0.6rem; font-size: 0.75rem; color: #9fb0c2; border-bottom: 1px solid rgba(140, 165, 190, 0.3); margin-bottom: 0.2rem; }
    .sm-ctx a { display: block; padding: 0.35rem 0.6rem; border-radius: 0.35rem; font-size: 0.85rem; color: #e6edf3; text-decoration: none; }
    .sm-ctx a:hover { background: rgba(45, 212, 191, 0.18); color: #fff; }
    [data-bs-theme="light"] .sm-ctx { background: rgba(255, 255, 255, 0.98); border-color: rgba(13, 148, 136, 0.45); box-shadow: 0 8px 26px rgba(30, 50, 80, 0.25); }
    [data-bs-theme="light"] .sm-ctx .sm-ctx-title { color: #52667a; border-bottom-color: rgba(60, 90, 120, 0.25); }
    [data-bs-theme="light"] .sm-ctx a { color: #1d2b3a; }
    [data-bs-theme="light"] .sm-ctx a:hover { background: rgba(13, 148, 136, 0.12); color: #0f1b2a; }
    .sm-close { position: absolute; top: 2px; right: 8px; background: none; border: 0; color: #9fb0c2; font-size: 1.2rem; line-height: 1; cursor: pointer; }
    [data-bs-theme="light"] .sm-close { color: #52667a; }
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
    var hovSite = null, hovRack = null, hovShelfUnit = -1;
    var selSite = null, selRack = null, selShelfUnit = -1, hovCity = null;
    var hits = [];                                 // [{site, rack, shelfUnit, pts}] в координатах мира, порядок = порядок рисования
    var overlay = [];                              // выноски/подписи - рисуются последним проходом поверх всего
    var allSites = [];
    // статистика одной секции стеллажа (как в просмотре оборудования площадки)
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
    function sameShelfUnit(rack, u, otherRack, otherU) {
        if (rack !== otherRack) return false;
        if (!rack || !rack.shelf || (rack.units || 1) <= 1) return true;
        return u === otherU;
    }

    // ---- цветовые схемы (светлая/тёмная тема сайта, атрибут data-bs-theme): приглушённая графитово-бирюзовая палитра ----
    // Изображения оборудования (шкафы APC, устройства вендоров, стойки стеллажа) от темы не зависят.
    var PALS = {
        dark: {
            bgTop: '#171d25', bgBot: '#0e1318', bandEven: 'rgba(0,0,0,0.16)', bandOdd: 'rgba(255,255,255,0.025)',
            slabTop: ['#2a3a4c', '#202d3c'], slabL: '#18222d', slabR: '#121a23', slabEdge: 'rgba(110,190,190,0.55)', slabGrid: 'rgba(150,175,195,0.14)',
            dash: 'rgba(160,185,205,0.6)', plateFill: 'rgba(15,21,28,0.94)', plateStroke: 'rgba(140,165,190,0.6)', plateText: '#f1f5f9',
            floorText: '#9fb0c2', addrText: '#e8eef5', textShadow: 'rgba(0,0,0,0.55)', deckEdge: 'rgba(120,190,200,0.5)', labelText: '#e6e9f2', labelShadow: 'rgba(0,0,0,0.6)',
            listBg: 'rgba(12,17,23,0.95)', listLine: 'rgba(120,160,185,0.4)', cityBg: 'rgba(38,52,68,0.55)', cityBgHov: 'rgba(45,212,191,0.2)', cityEdge: 'rgba(120,160,185,0.4)',
            title: '#f1f5f9', sub: '#9fb0c2', sel: '#2dd4bf', hov: '#99f6e4', selTxt: '#5eead4',
            sbTrack: 'rgba(12,17,23,0.6)', sbTick: 'rgba(150,175,195,0.35)', sbThumb: 'rgba(45,212,191,0.5)', sbThumbDim: 'rgba(45,212,191,0.22)',
            glowMode: 'lighter',
            // потоки данных между ЦОД разных городов (янтарь, как в образце, но мягче)
            flowGlow: '251,146,60', flowCore: '253,224,171', flowPulse: '255,248,230', flowHub: '251,191,36'
        },
        light: {
            bgTop: '#f3f6fa', bgBot: '#dbe3ec', bandEven: 'rgba(30,50,80,0.07)', bandOdd: 'rgba(255,255,255,0.4)',
            slabTop: ['#d3deea', '#bccbdc'], slabL: '#8ea2b8', slabR: '#7b90a8', slabEdge: 'rgba(40,82,110,0.8)', slabGrid: 'rgba(40,70,100,0.22)',
            dash: 'rgba(60,90,120,0.55)', plateFill: 'rgba(255,255,255,0.95)', plateStroke: 'rgba(60,90,120,0.55)', plateText: '#1d2b3a',
            floorText: '#52667a', addrText: '#1d2b3a', textShadow: 'rgba(255,255,255,0.7)', deckEdge: 'rgba(52,96,120,0.5)', labelText: '#17324a', labelShadow: 'rgba(255,255,255,0.9)',
            listBg: 'rgba(245,248,251,0.96)', listLine: 'rgba(60,90,120,0.3)', cityBg: 'rgba(255,255,255,0.85)', cityBgHov: 'rgba(13,148,136,0.14)', cityEdge: 'rgba(60,90,120,0.32)',
            title: '#1d2b3a', sub: '#52667a', sel: '#0d9488', hov: '#14b8a6', selTxt: '#0f766e',
            sbTrack: 'rgba(30,50,80,0.12)', sbTick: 'rgba(60,90,120,0.4)', sbThumb: 'rgba(13,148,136,0.5)', sbThumbDim: 'rgba(13,148,136,0.22)',
            glowMode: 'source-over',
            flowGlow: '234,88,12', flowCore: '251,146,60', flowPulse: '255,237,213', flowHub: '245,158,11'
        }
    };
    var PAL = PALS.dark;

    // оттенки янтаря для световых столбов: от светло-янтарного (малая мощность) к насыщенно-оранжевому (максимум)
    function glowRGB(frac, dark) {
        var stops = [[253, 224, 120], [251, 191, 36], [245, 158, 11]];     // один и тот же светлый янтарный ряд в обеих темах
        var t = Math.max(0, Math.min(1, frac)) * 2, i = Math.min(1, Math.floor(t)), f = t - i;
        var a = stops[i], b = stops[i + 1];
        return Math.round(a[0] + (b[0] - a[0]) * f) + ',' + Math.round(a[1] + (b[1] - a[1]) * f) + ',' + Math.round(a[2] + (b[2] - a[2]) * f);
    }
    // анимированный световой столб с языками пламени; высота/ширина по-прежнему от мощности (frac)
    function drawGlow(cx, cy, bw, bh, frac, dark, mode, seed) {
        var t = flowT, ph0 = (seed || 0) * 0.618;
        var rgb = glowRGB(frac, dark);
        var rgbHot = '255,244,200', rgbCool = '249,115,22';
        var aBase = (dark ? 0.28 : 0.42) + (dark ? 0.35 : 0.3) * frac;
        var flick = 0.92 + 0.08 * Math.sin(t * 3.1 + ph0) + 0.04 * Math.sin(t * 7.4 + ph0 * 2);
        var H = bh * flick, W = bw * (0.92 + 0.08 * Math.sin(t * 2.2 + ph0));
        ctx.save();
        ctx.globalCompositeOperation = mode;
        // мягкий столб-основание (конус вверх)
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
        // языки пламени: число и амплитуда растут с мощностью
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
        // яркое ядро
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
        // навершие на крыше
        ctx.fillStyle = 'rgba(' + rgb + ',' + Math.min(0.95, aBase + 0.28).toFixed(2) + ')';
        ctx.beginPath(); ctx.ellipse(cx, cy, W * 0.52, W * 0.24, 0, 0, Math.PI * 2); ctx.fill();
        ctx.restore();
    }
    function fadeSafe(v) { return Math.max(0, Math.min(1, v)); }
    function setPal() {
        PAL = document.documentElement.getAttribute('data-bs-theme') === 'light' ? PALS.light : PALS.dark;
        stage.style.background = 'linear-gradient(' + PAL.bgTop + ',' + PAL.bgBot + ')';
    }
    setPal();
    if (window.MutationObserver) new MutationObserver(function () { setPal(); redraw(); }).observe(document.documentElement, { attributes: true, attributeFilter: ['data-bs-theme'] });

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

    var FLOOR_GAP = 1.0;                           // зазор между этажами (в клетках) сверх высоты диагонали плиты верхнего этажа
    var PAD_L = 3;                                 // запас слева под подписи этажей и выдвижение ЦОД
    var siteCnt = 0, rackCnt = 0, bldCnt = 0;
    var shelfCnt = 0;
    var hasPowerGlow = false;                      // есть шкафы с мощностью → анимация пламени
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
                    if (s.maxPower > 0) hasPowerGlow = true;
                    s.sA = gridW + 0.6; s.sB = s.rows + 0.6;
                    s.aOff = x; x += s.sA + 0.8; bf = Math.max(bf, s.sB);
                    s.e = 0; s.tgt = 0; s.floor = f.floor; allSites.push(s);
                    siteCnt++; rackCnt += n; shelfCnt += s.shelves.length;
                });
                f.A = Math.max(0.5, x - 0.8); f.B = bf;
                A = Math.max(A, f.A); B = Math.max(B, f.B);
                half = Math.max(half, (f.A + f.B) / 2);
            });
            // высота этажей: зазор считается по размеру каждого вышележащего этажа (а не по самому большому в здании),
            // шкафы нижнего этажа допускается слегка перекрывать плитой верхнего - так стопка заметно компактнее
            var n = b.floors.length, zs = [0];
            for (var fi = 1; fi < n; fi++) zs.push(zs[fi - 1] + (b.floors[fi].A + b.floors[fi].B) / 2 + FLOOR_GAP);
            var ztop = zs[n - 1];
            bldCnt++;
            return { b: b, A: A, B: B, n: n, zs: zs, ztop: ztop,
                     w: (A + B + PAD_L) * U, h: (2.3 + ztop + (A + B) / 2 + 1.8) * U };
        });
        // один город - одна горизонтальная строка: здания (адреса) слева направо, общая линия основания
        var x = 0, rowH = 0;
        bl.forEach(function (b) { b.x = x; x += b.w + 2.4 * U; rowH = Math.max(rowH, b.h); });
        bl.forEach(function (b) { b.y = rowH - b.h; });
        return { c: c, blds: bl, w: Math.max(x - 2.4 * U, 6 * U), h: rowH + 0.6 * U };
    });
    var sitesById = {};
    cities.forEach(function (ct) { ct.blds.forEach(function (bl) { bl.b.floors.forEach(function (fl, k) { fl.sites.forEach(function (s) { s.ct = ct; s.bl = bl; s.k = k; sitesById[s.id] = s; }); }); }); });
    var WORLD = { w: 0, h: 0 };
    (function () {
        var y = 0;
        cities.forEach(function (ct) { ct.x = 0; ct.y = y; y += ct.h + 1.2 * U; WORLD.w = Math.max(WORLD.w, ct.w); });
        WORLD.h = Math.max(1, y - 1.2 * U);
    })();
    var LW = 180;                                  // ширина левого списка городов (в пикселях экрана)
    var cityBoxes = [];
    // иерархия потоков: площадка/ЦОД → узел здания → узел города → между городами
    var FLOW_NODES = [];
    var FLOW_LINKS = [];
    var flowStats = { siteBld: 0, bldCity: 0, cityCity: 0 };
    (function buildFlowGraph() {
        var seed = 1;
        function rnd() { seed = (seed * 1664525 + 1013904223) >>> 0; return seed / 4294967296; }
        function addLink(a, b, kind) {
            var lift = kind === 'city-city' ? 0.42 : (kind === 'bld-city' ? 0.26 : 0.14);
            var speed = kind === 'city-city' ? (0.05 + rnd() * 0.06) : (kind === 'bld-city' ? (0.07 + rnd() * 0.08) : (0.1 + rnd() * 0.1));
            FLOW_LINKS.push({
                a: a, b: b, kind: kind,
                phase: rnd(), speed: speed,
                dir: rnd() < 0.5 ? 1 : -1,
                pulses: kind === 'city-city' ? 2 + ((rnd() * 2) | 0) : (1 + ((rnd() * 2) | 0)),
                bow: (rnd() - 0.5) * (kind === 'city-city' ? 0.18 : 0.12),
                lift: lift
            });
            if (kind === 'site-bld') flowStats.siteBld++;
            else if (kind === 'bld-city') flowStats.bldCity++;
            else flowStats.cityCity++;
        }
        var cityIdx = [];
        cities.forEach(function (ct) {
            var cNode = FLOW_NODES.length;
            FLOW_NODES.push({ kind: 'city', ct: ct });
            cityIdx.push(cNode);
            ct.blds.forEach(function (bl) {
                var bNode = FLOW_NODES.length;
                FLOW_NODES.push({ kind: 'bld', ct: ct, bl: bl });
                addLink(bNode, cNode, 'bld-city');
                bl.b.floors.forEach(function (fl) {
                    fl.sites.forEach(function (s) {
                        var sNode = FLOW_NODES.length;
                        FLOW_NODES.push({ kind: 'site', site: s });
                        addLink(sNode, bNode, 'site-bld');
                    });
                });
            });
        });
        for (var i = 0; i < cityIdx.length; i++) {
            for (var j = i + 1; j < cityIdx.length; j++) addLink(cityIdx[i], cityIdx[j], 'city-city');
        }
    })();
    var flowT = 0;
    // мерцание светодиодов включённого оборудования на полках (спячка/выкл — статично)
    function ledBright(seed) {
        var t = flowT, s = (seed || 0) * 12.9898;
        var a = 0.62 + 0.38 * (0.5 + 0.5 * Math.sin(t * 2.15 + s));
        var b = 0.78 + 0.22 * (0.5 + 0.5 * Math.sin(t * 5.4 + s * 1.73));
        var dip = 1 - 0.12 * Math.max(0, Math.sin(t * 15.5 + s * 2.9));
        return Math.min(1, Math.max(0.48, a * b * dip));
    }
    document.getElementById('smSummary').innerHTML = 'Городов: <b>' + cities.length + '</b> &middot; зданий: <b>' + bldCnt + '</b> &middot; ЦОД: <b>' + siteCnt + '</b> &middot; шкафов: <b>' + rackCnt + '</b>' + (shelfCnt ? ' &middot; стеллажей: <b>' + shelfCnt + '</b>' : '') + (FLOW_LINKS.length ? ' &middot; потоков: <b>' + FLOW_LINKS.length + '</b> (площадка–здание ' + flowStats.siteBld + ', здание–город ' + flowStats.bldCity + ', город–город ' + flowStats.cityCity + ')' : '');

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
        var isSel = selRack === r, isHov = hovRack === r, lit = false;   // корпус не меняет цвет, подсвечивается только контур

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
            ctx.fillStyle = '#d9dcea';
            ctx.fillText(nm, 50, 65);
            ctx.restore();
        }
        var edge = isSel ? PAL.sel : (isHov ? PAL.hov : 'rgba(255,255,255,0.16)');
        var elw = lw(isSel ? 3 : (isHov ? 2.4 : 1));
        poly(left, null, edge, elw); poly(right, null, edge, elw); poly(top, null, edge, elw);

        // световой столб-пламя над шкафом пропорционально мощности (относительно самого мощного шкафа площадки)
        if (site.maxPower > 0 && r.powerAvgW > 0) {
            var frac = r.powerAvgW / site.maxPower;
            var pcx = (top[0][0] + top[2][0]) / 2, pcy = (top[0][1] + top[2][1]) / 2;
            var pbh = (H - z0) * U * (0.35 + 0.9 * frac), pbw = (a1 - a0) * U * 0.75;
            drawGlow(pcx, pcy, pbw, pbh, frac, PAL === PALS.dark, PAL.glowMode, r.id || 0);
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
        poly(top, '#eef0f3', 'rgba(0,0,0,0.3)', 0.7);
        var lc = P(so, cx, cb, z + hh);
        ctx.save();
        if (on) {
            var abr = ledBright(cx * 17 + cb * 9 + (it.vd || '').length);
            ctx.globalAlpha = 0.55 + 0.45 * abr;
            ctx.shadowColor = round ? '#3aa0ff' : '#35e06a';
            ctx.shadowBlur = Math.max(2, rad * U * 0.2 * abr);
        }
        ctx.fillStyle = on ? (round ? '#3aa0ff' : '#35e06a') : '#8a919c';
        ctx.beginPath(); ctx.arc(lc[0], lc[1], Math.max(1.2, rad * U * 0.14), 0, TAU); ctx.fill();
        ctx.restore();
    }
    function drawShelfItem(so, a0, a1, b0, b1, z, h, it, lit) {
        if (isAP(it)) { drawShelfAP(so, a0, a1, b0, b1, z, h, it, lit); return; }
        var acc = shelfStyle(it.vd), on = it.p === 1;
        var ia0 = a0 + 0.03, ia1 = a1 - 0.03, ib0 = b0 + 0.05, ib1 = b1 - 0.02, zt = z + h;
        var front = [P(so, ia0, ib1, zt), P(so, ia1, ib1, zt), P(so, ia1, ib1, z), P(so, ia0, ib1, z)];
        var side = [P(so, ia1, ib1, zt), P(so, ia1, ib0, zt), P(so, ia1, ib0, z), P(so, ia1, ib1, z)];
        var top = [P(so, ia0, ib0, zt), P(so, ia1, ib0, zt), P(so, ia1, ib1, zt), P(so, ia0, ib1, zt)];
        poly(side, '#1e2026');
        poly(front, lin(0, front[0][1], 0, front[2][1], [[0, '#353841'], [1, '#22252b']]), 'rgba(0,0,0,0.6)', 0.7);
        poly(top, '#575c69', 'rgba(255,255,255,0.25)', 0.7);
        ctx.save(); ctx.strokeStyle = acc; ctx.lineWidth = Math.max(1.2, U * 0.03); ctx.lineCap = 'butt';   // цветная полоса вендора по передней кромке
        ctx.beginPath(); ctx.moveTo(front[0][0], front[0][1]); ctx.lineTo(front[1][0], front[1][1]); ctx.stroke(); ctx.restore();
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
            // p=1 включено — мерцает; p=2 спячка/выкл — статичный янтарь
            if (on) {
                var sbr = ledBright(a0 * 13 + b0 * 7 + (it.vd || '').length * 0.3);
                ctx.globalAlpha = 0.55 + 0.45 * sbr;
                ctx.shadowColor = '#35e06a'; ctx.shadowBlur = 3 + 4 * sbr;
            }
            ctx.fillStyle = on ? '#35e06a' : (it.p === 2 ? '#ffab1a' : '#8a919c');
            ctx.beginPath(); ctx.arc(88, 24, 4, 0, TAU); ctx.fill();
            ctx.restore();
        } else {
            poly(front, null, null); ctx.fillStyle = acc; ctx.fillRect(front[0][0], front[0][1], front[1][0] - front[0][0], Math.max(1, U * 0.03));
        }
        poly(front, null, 'rgba(255,255,255,0.22)', 0.8);
    }
    // замкнутый каркас выделения: видимые рёбра - сплошные, скрытые (за стеллажом) - тонкие полупрозрачные; ракурс на карте фиксирован
    function boxWire(so, a0, a1, b0, b1, z0, z1, color, w) {
        function seg(p, q, hidden) {
            ctx.save(); ctx.strokeStyle = color; ctx.lineWidth = hidden ? Math.max(1, w * 0.5) : w; ctx.globalAlpha = hidden ? 0.45 : 1; ctx.lineCap = 'round';
            ctx.beginPath(); ctx.moveTo(p[0], p[1]); ctx.lineTo(q[0], q[1]); ctx.stroke(); ctx.restore();
        }
        var A = [a0, a1], B = [b0, b1], i, j;
        for (i = 0; i < 2; i++) for (j = 0; j < 2; j++) seg(P(so, A[i], B[j], z0), P(so, A[i], B[j], z1), i === 0 && j === 0);
        [z0, z1].forEach(function (z) {
            for (j = 0; j < 2; j++) seg(P(so, a0, B[j], z), P(so, a1, B[j], z), z === z0 && j === 0);
            for (i = 0; i < 2; i++) seg(P(so, A[i], b0, z), P(so, A[i], b1, z), z === z0 && i === 0);
        });
    }
    function drawShelfUnit(so, site, r, u, labels) {
        // при нескольких секциях выделяем каждую отдельно (как в просмотре оборудования площадки)
        var isSel = sameShelfUnit(r, u, selRack, selShelfUnit);
        var isHov = sameShelfUnit(r, u, hovRack, hovShelfUnit);
        var lit = false;
        var H = 1.55, z0 = SLAB, post = Math.max(1.6, U * 0.05);
        var zs = H * 0.2, tb = H * 0.022;
        var a0 = 0.3 + r.gi + u + (1 - FW) / 2, a1 = a0 + FW;
        var b0 = 0.3 + r.gj + (1 - FD) / 2, b1 = b0 + FD;
        poly([P(so, a0 - 0.05, b0 + 0.05, z0), P(so, a1 + 0.1, b0 + 0.05, z0), P(so, a1 + 0.1, b1 + 0.08, z0), P(so, a0 - 0.05, b1 + 0.08, z0)], 'rgba(0,0,0,0.28)');
        var line = function (x, y, za, zb, col, w) {
            var p0 = P(so, x, y, za), p1 = P(so, x, y, zb);
            ctx.strokeStyle = col; ctx.lineWidth = w; ctx.lineCap = 'round';
            ctx.beginPath(); ctx.moveTo(p0[0], p0[1]); ctx.lineTo(p1[0], p1[1]); ctx.stroke();
        };
        var pc = isSel ? PAL.sel : (isHov ? PAL.hov : '#5a7898');
        line(a0, b0, z0, z0 + H, pc, post); line(a1, b0, z0, z0 + H, pc, post);          // задние стойки
        var d0 = P(so, a0, b0, z0), d1 = P(so, a1, b0, z0 + H * 0.96);                    // диагональная связь
        ctx.strokeStyle = 'rgba(70,92,118,0.6)'; ctx.lineWidth = Math.max(1, post * 0.6);
        ctx.beginPath(); ctx.moveTo(d0[0], d0[1]); ctx.lineTo(d1[0], d1[1]); ctx.stroke();
        // полки снизу вверх: настил, балка, затем оборудование этой полки; вышележащий настил рисуется позже и частично прикрывает оборудование ниже
        var pk = r.packed && r.packed[u];
        for (var k = 0; k < 5; k++) {
            var z = z0 + H * 0.04 + zs * k;
            poly([P(so, a0, b0, z + tb), P(so, a1, b0, z + tb), P(so, a1, b1, z + tb), P(so, a0, b1, z + tb)], 'rgba(125,145,165,0.5)', PAL.deckEdge, 0.9);
            var frontf = [P(so, a0, b1, z + tb), P(so, a1, b1, z + tb), P(so, a1, b1, z), P(so, a0, b1, z)];
            poly(frontf, lin(0, frontf[0][1], 0, frontf[2][1], [[0, '#5a7898'], [1, '#2a3b4e']]));
            if (pk) {
                var zb = z + tb, unitH = (zs - tb - H * 0.012) / SH_CLEAR;
                pk[k].items.forEach(function (pl) { drawShelfItem(so, a0, a1, b0, b1, zb + pl.off * unitH, pl.it.su * unitH * 0.86, pl.it, lit); });
            }
        }
        line(a0, b1, z0, z0 + H, pc, post * 1.15); line(a1, b1, z0, z0 + H, pc, post * 1.15);   // передние стойки
        // номер секции = порядок наполнения (как в просмотре оборудования площадки)
        labels.push({ a0: a0, a1: a1, b0: b0, b1: b1, z: z0 + H, sel: isSel, num: u + 1, multi: (r.units || 1) > 1 });
        var hl = [P(so, a0, b1, z0), P(so, a1, b1, z0), P(so, a1, b0, z0), P(so, a1, b0, z0 + H), P(so, a0, b0, z0 + H), P(so, a0, b1, z0 + H)];
        if (isSel) boxWire(so, a0, a1, b0, b1, z0, z0 + H, PAL.sel, lw(3)); else if (isHov) boxWire(so, a0, a1, b0, b1, z0, z0 + H, PAL.hov, lw(2.4));
        hits.push({ site: site, rack: r, shelfUnit: u, pts: hl });
    }
    function drawShelfLabel(so, L) {
        if (cam.k < 0.22) return;
        // номер добавляем, когда секций больше одной (иначе просто «СТЕЛЛАЖ»)
        var txt = L.multi ? ('СТЕЛЛАЖ ' + L.num) : 'СТЕЛЛАЖ';
        ctx.save();
        faceTransform(P(so, L.a0, L.b0, L.z), (L.a1 - L.a0) * U, (L.a1 - L.a0) * U / 2, -(L.b1 - L.b0) * U, (L.b1 - L.b0) * U / 2, 100, 130);
        ctx.font = '800 ' + (L.multi ? '17' : '19') + 'px ' + FONT; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.fillStyle = PAL.labelShadow; ctx.fillText(txt, 51, 66);
        ctx.fillStyle = L.sel ? PAL.selTxt : PAL.labelText; ctx.fillText(txt, 50, 65);
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
            poly([P(oo, 0, 0, SLAB), P(oo, A, 0, SLAB), P(oo, A, B, SLAB), P(oo, 0, B, SLAB)], null, PAL.dash, lw(1.2));
            ctx.restore();
            poly([P(so, 0.1, B + 0.1, 0), P(so, A + 0.15, B + 0.1, 0), P(so, A + 0.15, 0.1, 0), P(so, 0.1, 0.1, 0)], 'rgba(0,0,0,' + (0.25 * site.e).toFixed(2) + ')');
        }
        var top = [P(so, 0, 0, SLAB), P(so, A, 0, SLAB), P(so, A, B, SLAB), P(so, 0, B, SLAB)];
        var fL = [P(so, 0, B, SLAB), P(so, A, B, SLAB), P(so, A, B, 0), P(so, 0, B, 0)];
        var fR = [P(so, A, B, SLAB), P(so, A, 0, SLAB), P(so, A, 0, 0), P(so, A, B, 0)];
        poly(fL, PAL.slabL); poly(fR, PAL.slabR);
        poly(top, lin(0, top[0][1], 0, top[2][1], [[0, PAL.slabTop[0]], [1, PAL.slabTop[1]]]));
        var edge = isSel ? PAL.sel : (isHov ? PAL.hov : PAL.slabEdge);
        var elw = lw(isSel ? 2.4 : (isHov ? 2 : 1.4));
        poly(top, null, edge, elw); poly(fL, null, edge, elw); poly(fR, null, edge, elw);
        if (cam.k > 0.5) {
            ctx.strokeStyle = PAL.slabGrid; ctx.lineWidth = lw(1);
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
        poly(plate, PAL.plateFill, isSel ? PAL.sel : (isHov ? PAL.hov : PAL.plateStroke), lw(1.3));
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
        ctx.fillStyle = isSel ? PAL.selTxt : PAL.plateText;
        ctx.fillText(txt, Wl / 2, Hl / 2 + 1);
        ctx.restore();
    }

    function drawBuilding(bx, by, bl) {
        var o = { x: bx + (bl.B + PAD_L) * U, y: by + (2.3 + bl.ztop) * U };
        // пунктир по задней грани, соединяющий этажи
        if (bl.n > 1) {
            var d0 = P(o, 0, 0, SLAB), d1 = P(o, 0, 0, bl.ztop + SLAB);
            ctx.save(); ctx.setLineDash([7, 6]); ctx.strokeStyle = PAL.dash; ctx.lineWidth = lw(1.4);
            ctx.beginPath(); ctx.moveTo(d0[0], d0[1]); ctx.lineTo(d1[0], d1[1]); ctx.stroke(); ctx.restore();
        }
        var late = [];
        bl.b.floors.forEach(function (fl, k) {
            var zf = bl.zs[k];
            fl.sites.forEach(function (s) {
                if (s.e > 0.01) { late.push({ s: s, zf: zf }); return; }
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
            ctx.fillStyle = PAL.textShadow; ctx.fillText(c.label, c.x + 1, c.y + 1);
            ctx.fillStyle = PAL.floorText; ctx.fillText(c.label, c.x, c.y);
        } else {
            var fs = 20;
            ctx.font = '700 ' + fs + 'px ' + FONT;
            var tw = ctx.measureText(c.addr).width;
            if (tw > c.maxW) { fs = Math.max(11, Math.floor(fs * c.maxW / tw)); ctx.font = '700 ' + fs + 'px ' + FONT; }
            var t = c.addr; while (t.length > 3 && ctx.measureText(t).width > c.maxW) t = t.slice(0, -2) + '…';
            ctx.textAlign = 'center'; ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = PAL.textShadow; ctx.fillText(t, c.x + 1, c.y + 1);
            ctx.fillStyle = PAL.addrText; ctx.fillText(t, c.x, c.y);
        }
        ctx.restore();
    }

    // ---- левый список городов: колонка в экранных координатах, плашка на высоту строки города ----
    function drawCityList() {
        cityBoxes = [];
        ctx.save();
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        ctx.fillStyle = PAL.listBg;
        ctx.fillRect(0, 0, LW, G.h);
        ctx.strokeStyle = PAL.listLine; ctx.lineWidth = 1;
        ctx.beginPath(); ctx.moveTo(LW + 0.5, 0); ctx.lineTo(LW + 0.5, G.h); ctx.stroke();
        cities.forEach(function (ct) {
            var top = cam.ty + (ct.y - 0.3 * U) * cam.k, bot = cam.ty + (ct.y + ct.h + 0.6 * U) * cam.k;
            if (bot < 0 || top > G.h) return;
            var t0 = Math.max(top, 0), b0 = Math.min(bot, G.h);
            cityBoxes.push({ ct: ct, y0: t0, y1: b0 });
            var isCur = hovCity === ct;
            ctx.fillStyle = isCur ? PAL.cityBgHov : PAL.cityBg;
            ctx.fillRect(6, t0 + 3, LW - 12, Math.max(10, b0 - t0 - 6));
            ctx.strokeStyle = isCur ? PAL.sel : PAL.cityEdge;
            ctx.strokeRect(6.5, t0 + 3.5, LW - 13, Math.max(10, b0 - t0 - 6) - 1);
            var cy = (t0 + b0) / 2, ph = b0 - t0 - 6;
            var sc = 0; ct.blds.forEach(function (b) { b.b.floors.forEach(function (f) { sc += f.sites.length; }); });
            ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
            var nm = ct.c.name;
            if (ph >= 36) {                                           // достаточно места: название и сводка
                ctx.font = '800 17px ' + FONT;
                while (nm.length > 3 && ctx.measureText(nm).width > LW - 28) nm = nm.slice(0, -2) + '…';
                ctx.fillStyle = PAL.title; ctx.fillText(nm, 14, cy - 8);
                ctx.font = '600 12px ' + FONT; ctx.fillStyle = PAL.sub;
                ctx.fillText('зданий ' + ct.blds.length + ' · ЦОД ' + sc, 14, cy + 11);
            } else if (ph >= 11) {                                    // плашка низкая: только название по высоте плашки, без наложения
                var fs = Math.min(15, ph - 3);
                ctx.font = '700 ' + fs + 'px ' + FONT;
                while (nm.length > 3 && ctx.measureText(nm).width > LW - 28) nm = nm.slice(0, -2) + '…';
                ctx.fillStyle = PAL.title; ctx.fillText(nm, 14, cy);
            }
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
        cam.tx = LW + m; cam.ty = m - (ct.y - 0.3 * U) * cam.k;                 // по левому краю, строка города - вверху
        redraw();
    }
    function fitBuilding(ct, bl) {                  // показать здание целиком (слева, по высоте окна)
        var m = 30, availW = G.w - LW - 2 * m, availH = G.h - 2 * m;
        var k = Math.min(availW / bl.w, availH / bl.h);
        cam.k = Math.max(0.05, Math.min(1.6, k));
        cam.tx = LW + m - (ct.x + bl.x) * cam.k;
        cam.ty = m - (ct.y + bl.y) * cam.k;
        if (bl.h * cam.k < availH) cam.ty += (availH - bl.h * cam.k) / 2;
        redraw();
    }
    function focusSite(site) { fitBuilding(site.ct, site.bl); select(site, null); }

    // ---- фильтры вверху: быстрый переход к городу / площадке; последний выбор запоминается ----
    var citySel = document.getElementById('smCitySel'), siteSel = document.getElementById('smSiteSel');
    var LS_CITY = 'datacentersitemap.city', LS_SITE = 'datacentersitemap.site';
    function lsGet(k) { try { return localStorage.getItem(k) || ''; } catch (e) { return ''; } }
    function lsSet(k, v) { try { if (v) localStorage.setItem(k, v); else localStorage.removeItem(k); } catch (e) { } }
    function cityByName(nm) { for (var i = 0; i < cities.length; i++) if (cities[i].c.name === nm) return cities[i]; return null; }
    function fillCities() {
        var h = '<option value="">Все города (' + cities.length + ')</option>';
        cities.forEach(function (ct) { h += '<option value="' + esc(ct.c.name) + '">' + esc(ct.c.name) + '</option>'; });
        citySel.innerHTML = h;
    }
    function fillSites(cityName) {
        var h = '<option value="">Все площадки/ЦОД</option>';
        cities.forEach(function (ct) {
            if (cityName && ct.c.name !== cityName) return;
            var g = '';
            ct.blds.forEach(function (bl) { bl.b.floors.forEach(function (fl) { fl.sites.forEach(function (s) {
                g += '<option value="' + s.id + '">' + esc(bl.b.address + ' · ' + floorTitle(fl.floor) + ' · ' + s.room) + '</option>';
            }); }); });
            h += cityName ? g : '<optgroup label="' + esc(ct.c.name) + '">' + g + '</optgroup>';
        });
        siteSel.innerHTML = h;
    }
    citySel.addEventListener('change', function () {
        var nm = citySel.value;
        lsSet(LS_CITY, nm); lsSet(LS_SITE, '');
        fillSites(nm);
        select(null, null);
        var ct = nm ? cityByName(nm) : null;
        if (ct) fitCity(ct); else { fit(); redraw(); }
    });
    siteSel.addEventListener('change', function () {
        var s = sitesById[siteSel.value];
        lsSet(LS_SITE, siteSel.value);
        if (s) { citySel.value = s.ct.c.name; lsSet(LS_CITY, s.ct.c.name); focusSite(s); }
    });
    document.getElementById('smAll').addEventListener('click', function () { citySel.value = ''; siteSel.value = ''; lsSet(LS_CITY, ''); lsSet(LS_SITE, ''); fillSites(''); fit(); redraw(); });
    function initView() {                           // при открытии: восстановить последний выбор, иначе - первый город
        fillCities(); fillSites('');
        var sid = lsGet(LS_SITE), cnm = lsGet(LS_CITY);
        var s = sid ? sitesById[sid] : null;
        if (s) { citySel.value = s.ct.c.name; fillSites(s.ct.c.name); siteSel.value = String(s.id); focusSite(s); return; }
        var ct = cnm ? cityByName(cnm) : null;
        if (ct) { citySel.value = ct.c.name; fillSites(ct.c.name); fitCity(ct); return; }
        fitCity(cities[0]);
    }

    // ---- вертикальная полоса прокрутки: положение окна просмотра на всей карте (риски - границы городов) ----
    var sb = document.getElementById('smVScroll'), sbCtx = sb ? sb.getContext('2d') : null;
    var SB_W = 14, SB_PAD = 20, sbDrag = null;
    function sbGeom() {
        var trackH = Math.max(40, G.h - 12), contentH = WORLD.h * cam.k + 2 * SB_PAD, range = Math.max(0, contentH - G.h);
        var off = Math.min(Math.max(SB_PAD - cam.ty, 0), range);
        var thumbH = Math.max(24, trackH * Math.min(1, G.h / contentH));
        return { trackH: trackH, contentH: contentH, range: range, off: off, thumbH: thumbH, thumbY: range > 0 ? off / range * (trackH - thumbH) : 0 };
    }
    function drawScroll() {
        if (!sb) return;
        var g = sbGeom();
        sbCtx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        sbCtx.clearRect(0, 0, SB_W, g.trackH);
        sbCtx.fillStyle = PAL.sbTrack; sbCtx.fillRect(4, 0, 8, g.trackH);
        sbCtx.strokeStyle = PAL.sbTick; sbCtx.lineWidth = 1;
        cities.forEach(function (ct) {                                         // границы городов
            var y = Math.round((ct.y * cam.k + SB_PAD) / g.contentH * g.trackH) + 0.5;
            sbCtx.beginPath(); sbCtx.moveTo(2, y); sbCtx.lineTo(SB_W - 2, y); sbCtx.stroke();
        });
        sbCtx.fillStyle = g.range > 0 ? PAL.sbThumb : PAL.sbThumbDim;
        sbCtx.fillRect(2, g.thumbY, SB_W - 4, g.thumbH);
        sbCtx.strokeStyle = PAL.sel; sbCtx.strokeRect(2.5, g.thumbY + 0.5, SB_W - 5, g.thumbH - 1);
        // подсказка: какие города сейчас видны
        var vy0 = -cam.ty / cam.k, vy1 = (G.h - cam.ty) / cam.k, a = -1, b = -1;
        cities.forEach(function (ct, i) { if (ct.y + ct.h >= vy0 && ct.y <= vy1) { if (a < 0) a = i; b = i; } });
        sb.title = 'Города ' + (a + 1) + '–' + (b + 1) + ' из ' + cities.length + ' (масштаб ' + Math.round(cam.k * 100) + '%)';
    }
    function sbSetOff(off) { var g = sbGeom(); cam.ty = SB_PAD - Math.min(Math.max(off, 0), g.range); redraw(); }
    if (sb) {
        sb.addEventListener('mousedown', function (e) {
            var rect = sb.getBoundingClientRect(), y = e.clientY - rect.top, g = sbGeom();
            if (y >= g.thumbY && y <= g.thumbY + g.thumbH) sbDrag = { y: e.clientY, off: g.off };
            else { sbSetOff((y - g.thumbH / 2) / Math.max(1, g.trackH - g.thumbH) * g.range); sbDrag = { y: e.clientY, off: sbGeom().off }; }
            e.preventDefault(); e.stopPropagation();
        });
        window.addEventListener('mousemove', function (e) {
            if (!sbDrag) return;
            var g = sbGeom();
            sbSetOff(sbDrag.off + (e.clientY - sbDrag.y) / Math.max(1, g.trackH - g.thumbH) * g.range);
        });
        window.addEventListener('mouseup', function () { sbDrag = null; });
    }

    // ---- потоки: площадка ↔ здание ↔ город ↔ города ----
    function buildingOrigin(ct, bl) {
        var bx = ct.x + bl.x, by = ct.y + bl.y;
        return { x: bx + (bl.B + PAD_L) * U, y: by + (2.3 + bl.ztop) * U };
    }
    // центр плиты площадки/ЦОД (с учётом выдвижения)
    function siteAnchor(site) {
        var bl = site.bl, o = buildingOrigin(site.ct, bl);
        var zf = bl.zs[site.k];
        var t = site.e * site.e * (3 - 2 * site.e) * 1.5;
        var p0 = P(o, site.aOff - t, t, zf);
        var so = { x: p0[0], y: p0[1] };
        return P(so, site.sA * 0.5, site.sB * 0.5, SLAB);
    }
    function flowNodePos(n) {
        if (n.kind === 'site') return siteAnchor(n.site);
        if (n.kind === 'bld') {
            var o = buildingOrigin(n.ct, n.bl);
            // узел здания - над стопкой этажей, у переднего края
            return P(o, n.bl.A * 0.5, n.bl.B * 0.62, n.bl.ztop + SLAB + 2.1);
        }
        // узел города - сверху по центру строки города
        return [n.ct.x + Math.max(n.ct.w, 4 * U) * 0.5, n.ct.y + 0.4 * U];
    }
    function flowCtrl(a, b, bow, liftK) {
        var mx = (a[0] + b[0]) * 0.5, my = (a[1] + b[1]) * 0.5;
        var dx = b[0] - a[0], dy = b[1] - a[1];
        var dist = Math.sqrt(dx * dx + dy * dy) || 1;
        var lift = Math.min(dist * liftK, 11 * U) + dist * 0.03;
        var nx = -dy / dist, ny = dx / dist;
        return [mx + nx * dist * bow, my - lift + ny * dist * bow * 0.3];
    }
    function flowPoint(a, c, b, t) {
        var u = 1 - t;
        return [u * u * a[0] + 2 * u * t * c[0] + t * t * b[0], u * u * a[1] + 2 * u * t * c[1] + t * t * b[1]];
    }
    function flowVisible(a, c, b, vx0, vy0, vx1, vy1, m) {
        function ok(p) { return p[0] >= vx0 - m && p[0] <= vx1 + m && p[1] >= vy0 - m && p[1] <= vy1 + m; }
        return ok(a) || ok(b) || ok(c) || ok(flowPoint(a, c, b, 0.33)) || ok(flowPoint(a, c, b, 0.66));
    }
    function strokeFlow(a, c, b, steps) {
        ctx.beginPath();
        for (var s = 0; s <= steps; s++) {
            var p = flowPoint(a, c, b, s / steps);
            if (s === 0) ctx.moveTo(p[0], p[1]); else ctx.lineTo(p[0], p[1]);
        }
        ctx.stroke();
    }
    // визуальный вес уровня: внутри здания тоньше, между городами - ярче и шире
    var FLOW_STYLE = {
        'site-bld':  { glow: 5.5,  fiber: 1.5, core: 0.7,  aGlow: 0.08, aFiber: 0.28, aCore: 0.42, hub: 0, pulse: 5.5 },
        'bld-city':  { glow: 8.5,  fiber: 2.0, core: 0.95, aGlow: 0.1,  aFiber: 0.34, aCore: 0.5,  hub: 0, pulse: 7 },
        'city-city': { glow: 13,   fiber: 2.6, core: 1.25, aGlow: 0.12, aFiber: 0.4,  aCore: 0.58, hub: 0, pulse: 9 }
    };
    var HUB_STYLE = {
        site: { r: 4.2, ring: 0 },
        bld:  { r: 7.2, ring: 1.1 },
        city: { r: 11.5, ring: 1.6 }
    };
    function drawFlowHub(p, kind, dark, gRGB, cRGB, hRGB) {
        var hs = HUB_STYLE[kind] || HUB_STYLE.site;
        var r = lw(hs.r);
        var rg = ctx.createRadialGradient(p[0], p[1], 0, p[0], p[1], r * 2.4);
        var a0 = kind === 'city' ? (dark ? 0.62 : 0.5) : (kind === 'bld' ? (dark ? 0.5 : 0.4) : (dark ? 0.42 : 0.35));
        rg.addColorStop(0, 'rgba(' + hRGB + ',' + a0.toFixed(2) + ')');
        rg.addColorStop(0.4, 'rgba(' + gRGB + ',' + (a0 * 0.45).toFixed(2) + ')');
        rg.addColorStop(1, 'rgba(' + gRGB + ',0)');
        ctx.fillStyle = rg;
        ctx.beginPath(); ctx.arc(p[0], p[1], r * 2.4, 0, TAU); ctx.fill();
        if (hs.ring > 0) {
            ctx.strokeStyle = 'rgba(' + cRGB + ',' + (dark ? 0.55 : 0.5) + ')';
            ctx.lineWidth = lw(hs.ring);
            ctx.beginPath(); ctx.arc(p[0], p[1], r * 0.95, 0, TAU); ctx.stroke();
        }
        ctx.fillStyle = 'rgba(' + cRGB + ',' + (dark ? 0.9 : 0.8) + ')';
        ctx.beginPath(); ctx.arc(p[0], p[1], Math.max(1.1, r * (kind === 'city' ? 0.32 : 0.26)), 0, TAU); ctx.fill();
    }
    function drawFlows(vx0, vy0, vx1, vy1) {
        if (!FLOW_LINKS.length) return;
        var pos = new Array(FLOW_NODES.length);
        for (var i = 0; i < FLOW_NODES.length; i++) pos[i] = flowNodePos(FLOW_NODES[i]);
        var dark = PAL === PALS.dark;
        var steps = FLOW_LINKS.length > 160 ? 18 : 26;
        var margin = 7 * U;
        var gRGB = PAL.flowGlow, cRGB = PAL.flowCore, pRGB = PAL.flowPulse, hRGB = PAL.flowHub;
        var order = ['site-bld', 'bld-city', 'city-city'];
        var byKind = { 'site-bld': [], 'bld-city': [], 'city-city': [] };
        FLOW_LINKS.forEach(function (L) { byKind[L.kind].push(L); });

        ctx.save();
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';

        // трассы снизу вверх по уровню (городские поверх)
        order.forEach(function (kind) {
            var st = FLOW_STYLE[kind];
            byKind[kind].forEach(function (L) {
                var a = pos[L.a], b = pos[L.b], c = flowCtrl(a, b, L.bow, L.lift);
                if (!flowVisible(a, c, b, vx0, vy0, vx1, vy1, margin)) return;
                ctx.strokeStyle = 'rgba(' + gRGB + ',' + st.aGlow.toFixed(3) + ')';
                ctx.lineWidth = lw(st.glow);
                strokeFlow(a, c, b, steps);
                ctx.strokeStyle = 'rgba(' + gRGB + ',' + ((dark ? 1 : 0.9) * st.aFiber).toFixed(3) + ')';
                ctx.lineWidth = lw(st.fiber);
                strokeFlow(a, c, b, steps);
                ctx.strokeStyle = 'rgba(' + cRGB + ',' + ((dark ? 1 : 0.95) * st.aCore).toFixed(3) + ')';
                ctx.lineWidth = lw(st.core);
                strokeFlow(a, c, b, steps);
            });
        });

        // узлы: площадки, здания, города
        FLOW_NODES.forEach(function (n, ni) {
            var p = pos[ni];
            if (p[0] < vx0 - 2 * U || p[0] > vx1 + 2 * U || p[1] < vy0 - 2 * U || p[1] > vy1 + 2 * U) return;
            drawFlowHub(p, n.kind, dark, gRGB, cRGB, hRGB);
        });

        // импульсы
        if (dark) ctx.globalCompositeOperation = 'lighter';
        order.forEach(function (kind) {
            var st = FLOW_STYLE[kind];
            byKind[kind].forEach(function (L) {
                var a = pos[L.a], b = pos[L.b], c = flowCtrl(a, b, L.bow, L.lift);
                if (!flowVisible(a, c, b, vx0, vy0, vx1, vy1, margin)) return;
                for (var p = 0; p < L.pulses; p++) {
                    var tt = (flowT * L.speed * L.dir + L.phase + p / L.pulses) % 1;
                    if (tt < 0) tt += 1;
                    var fade = Math.min(1, Math.min(tt, 1 - tt) * 6);
                    if (fade < 0.04) continue;
                    var pt = flowPoint(a, c, b, tt);
                    var t2 = Math.max(0, Math.min(1, tt + 0.025 * L.dir));
                    if (Math.abs(t2 - tt) < 1e-4) t2 = Math.max(0, Math.min(1, tt - 0.025 * L.dir));
                    var ahead = flowPoint(a, c, b, t2);
                    var vx = (ahead[0] - pt[0]) * L.dir, vy = (ahead[1] - pt[1]) * L.dir;
                    var len = Math.sqrt(vx * vx + vy * vy) || 1;
                    vx /= len; vy /= len;
                    var pr = lw(st.pulse * (0.85 + 0.35 * fade));
                    var rg = ctx.createRadialGradient(pt[0], pt[1], 0, pt[0], pt[1], pr);
                    rg.addColorStop(0, 'rgba(' + pRGB + ',' + (0.92 * fade).toFixed(3) + ')');
                    rg.addColorStop(0.28, 'rgba(' + cRGB + ',' + (0.65 * fade).toFixed(3) + ')');
                    rg.addColorStop(0.65, 'rgba(' + gRGB + ',' + (0.28 * fade).toFixed(3) + ')');
                    rg.addColorStop(1, 'rgba(' + gRGB + ',0)');
                    ctx.fillStyle = rg;
                    ctx.beginPath(); ctx.arc(pt[0], pt[1], pr, 0, TAU); ctx.fill();
                    ctx.strokeStyle = 'rgba(' + cRGB + ',' + (0.45 * fade).toFixed(3) + ')';
                    ctx.lineWidth = lw(kind === 'city-city' ? 2 : 1.5);
                    ctx.beginPath();
                    ctx.moveTo(pt[0] - vx * pr * 1.15, pt[1] - vy * pr * 1.15);
                    ctx.lineTo(pt[0] + vx * pr * 0.35, pt[1] + vy * pr * 0.35);
                    ctx.stroke();
                }
            });
        });
        ctx.restore();
    }

    // ---- сцена ----
    function render() {
        ctx.setTransform(G.dpr, 0, 0, G.dpr, 0, 0);
        ctx.fillStyle = lin(0, 0, 0, G.h, [[0, PAL.bgTop], [1, PAL.bgBot]]);
        ctx.fillRect(0, 0, G.w, G.h);
        ctx.save();
        ctx.translate(cam.tx, cam.ty); ctx.scale(cam.k, cam.k);
        var vx0 = -cam.tx / cam.k, vy0 = -cam.ty / cam.k, vx1 = vx0 + G.w / cam.k, vy1 = vy0 + G.h / cam.k;
        hits = []; overlay = [];
        // подложка строк городов (чередование) - на всю видимую ширину
        cities.forEach(function (ct, i) {
            if (ct.y > vy1 || ct.y + ct.h + 1.2 * U < vy0) return;
            ctx.fillStyle = i % 2 ? PAL.bandOdd : PAL.bandEven;
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
        drawFlows(vx0, vy0, vx1, vy1);              // дуги поверх площадок, под UI списка городов
        ctx.restore();
        drawCityList();
        pctBtn.textContent = Math.round(cam.k * 100) + '%';
        drawScroll();
    }
    var raf = 0;
    function redraw() { if (!raf) raf = requestAnimationFrame(frame); }
    function frame(now) {
        raf = 0;
        flowT = (now || performance.now()) * 0.001;
        var busy = false;
        allSites.forEach(function (s) {
            if (s.e !== s.tgt) {
                var d = s.tgt - s.e;
                s.e = Math.abs(d) < 0.02 ? s.tgt : s.e + d * 0.22;
                busy = true;
            }
        });
        render();
        // потоки, пламя над шкафами и мерцание светодиодов включённого оборудования
        redraw();
    }

    // ---- камера ----
    function fit() {
        var m = 20, availW = G.w - LW - 2 * m;
        var k = Math.min(availW / WORLD.w, (G.h - 2 * m) / WORLD.h);
        cam.k = Math.max(0.02, Math.min(1.5, k));
        cam.tx = LW + m;                               // изображение выравнивается по левому и верхнему краю
        cam.ty = m;
    }
    var fitted = false;
    function resize() {
        var rect = stage.getBoundingClientRect();
        G.dpr = window.devicePixelRatio || 1;
        G.w = Math.max(200, rect.width); G.h = Math.max(200, rect.height);
        canvas.width = Math.round(G.w * G.dpr); canvas.height = Math.round(G.h * G.dpr);
        if (sb) { sb.width = Math.round(SB_W * G.dpr); sb.height = Math.round(Math.max(40, G.h - 12) * G.dpr); sb.style.width = SB_W + 'px'; sb.style.height = Math.max(40, G.h - 12) + 'px'; }
        if (!fitted) { fitted = true; initView(); }
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
    function rackInfo(r, shelfUnit) {
        var multi = r.shelf && (r.units || 1) > 1 && shelfUnit != null && shelfUnit >= 0;
        var st = multi ? unitStats(r, shelfUnit) : null;
        var title = r.shelf
            ? (multi ? ('Стеллаж ' + (shelfUnit + 1) + ' (вне шкафов)') : 'Стеллаж (оборудование вне шкафов)')
            : ('Шкаф ' + esc(r.name));
        var sub = r.shelf
            ? (multi ? ('секция ' + (shelfUnit + 1) + ' из ' + r.units + ' &middot; 5 полок') : ('секций ' + (r.units || 1) + ' &middot; 5 полок'))
            : (r.uHeight + 'U');
        var h = '<b class="sm-name">' + title + '</b><div class="sm-sub">' + sub + '</div><table>';
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
        return h + '</table>';
    }
    function rackViewUrl(r, shelfUnit) {
        var url = '/sprsrv/rackcabinetview/' + r.id;
        if (r.shelf) url += '?shelfUnit=' + ((shelfUnit != null && shelfUnit >= 0 ? shelfUnit : 0) + 1);
        return url;
    }
    function showTip(hit, x, y) {
        tip.innerHTML = (hit.rack
            ? rackInfo(hit.rack, hit.shelfUnit) + '<div class="sm-sub" style="margin-top:.3rem">ЦОД: ' + esc(hit.site.name) + '</div>'
            : siteInfo(hit.site)) + '<div class="sm-hint">Клик - выделить</div>';
        tip.hidden = false;
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var px = x + 16, py = y + 16;
        if (px + tw > G.w - 4) px = x - tw - 16;
        if (py + th > G.h - 4) py = G.h - th - 4;
        tip.style.left = Math.max(4, px) + 'px'; tip.style.top = Math.max(4, py) + 'px';
    }
    function select(site, rack, shelfUnit) {
        if (selSite && selSite !== site) selSite.tgt = 0;
        selSite = site;
        selRack = rack || null;
        selShelfUnit = (rack && rack.shelf) ? (shelfUnit == null || shelfUnit < 0 ? 0 : shelfUnit) : -1;
        if (site) {
            var h = '<button type="button" class="sm-close" title="Закрыть" id="smPanelClose">&times;</button>' + siteInfo(site);
            h += '<div class="sm-hint"><a href="/sprsrv/datacentersiteview/' + site.id + '" target="_blank" rel="noopener">Открыть оборудование ЦОД</a></div>';
            if (rack) {
                h += '<hr style="margin:.5rem 0">' + rackInfo(rack, selShelfUnit);
                h += '<div class="sm-hint"><a href="' + rackViewUrl(rack, selShelfUnit) + '" target="_blank" rel="noopener">'
                    + (rack.shelf ? ((rack.units || 1) > 1 ? 'Открыть стеллаж ' + (selShelfUnit + 1) : 'Открыть стеллаж') : 'Открыть шкаф') + '</a></div>';
            }
            panel.innerHTML = h; panel.hidden = false;
            document.getElementById('smPanelClose').addEventListener('click', function () { select(null, null, -1); });
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
        if (!h) select(null, null, -1);
        else if (h.rack) {
            var same = sameShelfUnit(h.rack, h.shelfUnit, selRack, selShelfUnit);
            select(h.site, same ? null : h.rack, same ? -1 : h.shelfUnit);
        } else select(h.site === selSite && !selRack ? null : h.site, null, -1);
    });
    canvas.addEventListener('mousemove', function (e) {
        if (drag && drag.moved) return;
        var rect = canvas.getBoundingClientRect();
        var x = e.clientX - rect.left, y = e.clientY - rect.top;
        var hc = cityAt(x, y);
        if (hc !== hovCity) { hovCity = hc; canvas.style.cursor = hc ? 'pointer' : ''; redraw(); }
        var h = hc ? null : pick(x, y);
        var ns = h ? h.site : null, nr = h ? h.rack : null;
        var nu = (h && h.rack && h.rack.shelf) ? (h.shelfUnit == null ? 0 : h.shelfUnit) : -1;
        if (ns !== hovSite || nr !== hovRack || nu !== hovShelfUnit) { hovSite = ns; hovRack = nr; hovShelfUnit = nu; redraw(); }
        if (h) showTip(h, x, y); else tip.hidden = true;
    });
    canvas.addEventListener('mouseleave', function () { hovCity = null; hovSite = null; hovRack = null; hovShelfUnit = -1; tip.hidden = true; redraw(); });
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
        if (hit.rack) {
            var su = hit.rack.shelf ? (hit.shelfUnit == null ? 0 : hit.shelfUnit) : -1;
            var label = hit.rack.shelf
                ? ((hit.rack.units || 1) > 1 ? 'Открыть стеллаж ' + (su + 1) : 'Открыть стеллаж')
                : ('Открыть шкаф ' + esc(hit.rack.name));
            h += '<a href="' + rackViewUrl(hit.rack, su) + '" target="_blank" rel="noopener">' + label + '</a>';
        }
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
        select(h.site, h.rack || null, h.rack && h.rack.shelf ? h.shelfUnit : -1);
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
