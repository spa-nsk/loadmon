{{if .Embed}}<style>
    .sidebar-panel { display: none !important; }
    /* показ во фрейме ЦОД: только отрисованный шкаф; без позиции - снизу */
    .hero-text { display: none !important; }
    /* блок "без позиции" - справа от шкафа, если помещается без прокрутки, иначе под ним */
    .rv-layout { flex-direction: row; flex-wrap: wrap; gap: 0.75rem; align-items: flex-start; }
    .rv-main { order: 1; flex: 0 0 auto; width: auto; padding-bottom: 0; }
    .rv-side { order: 2; flex: 1 1 240px; min-width: 0; max-width: 100%; }
    .rv-side > .rv-card:not(#rvUnplacedCard) { display: none; }
    /* фон берёт родительская страница ЦОД (общий градиент темы) */
    html, body, .content-panel { background: transparent !important; }
</style>{{end}}
<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Шкаф {{.CabinetName}}</h2>
        <div class="text-muted">
            {{if .SiteName}}<b>{{.SiteName}}</b> &middot; {{end}}высота {{.UHeight}}U &middot; оборудования: {{.ItemCount}}
        </div>
    </div>
    <div class="d-flex align-items-center gap-2">
        <div class="btn-group" role="group" aria-label="Масштаб">
            <button type="button" class="btn btn-outline-secondary btn-sm" id="rvZoomOut" title="Уменьшить">&minus;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" id="rvZoomReset" title="Масштаб по умолчанию">100%</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" id="rvZoomIn" title="Увеличить">+</button>
        </div>
        <a class="btn btn-outline-primary btn-sm rounded-pill px-3" href="/sprsrv/rackcabinetedit/{{.Id}}" target="_blank">Редактировать шкаф</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<div class="rv-layout">
    <aside class="rv-side">
        <div class="rv-card">
            <h6>Сводка</h6>
            <div id="rvSummary" class="small"></div>
        </div>
        <div class="rv-card">
            <h6>Состояние питания</h6>
            <ul class="rv-legend small">
                <li><span class="rv-dot rv-dot-on"></span> Включено (светодиод питания горит зелёным)</li>
                <li><span class="rv-dot rv-dot-stby"></span> Выключено (дежурный режим, янтарный)</li>
                <li><span class="rv-dot rv-dot-unk"></span> Состояние не определено</li>
            </ul>
            <div class="text-muted small">Наведите курсор на оборудование - появятся данные и ссылки. Клик закрепляет подсказку.</div>
        </div>
        <div class="rv-card" id="rvUnplacedCard" hidden>
            <h6>Без позиции в шкафу<span id="rvUnplacedCount" class="text-muted fw-normal"></span></h6>
            <div id="rvUnplaced" class="small"></div>
        </div>
    </aside>

    <div class="rv-main">
        <div class="rv-stage" id="rvStage">
            <canvas id="rvBase"></canvas>
            <canvas id="rvOver"></canvas>
            <div id="rvTip" class="rv-tip" hidden></div>
        </div>
    </div>
</div>

<style>
    .rv-layout { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
    .rv-side { flex: 0 0 300px; max-width: 100%; display: flex; flex-direction: column; gap: 0.75rem; }
    .rv-main { flex: 1 1 auto; min-width: 0; padding-bottom: 1rem; }
    .rv-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 0.8rem 1rem;
    }
    .rv-card h6 { color: var(--bs-emphasis-color); font-weight: 600; margin-bottom: 0.5rem; }
    .rv-legend { list-style: none; padding: 0; margin: 0 0 0.5rem 0; }
    .rv-legend li { margin-bottom: 0.25rem; }
    .rv-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 0.35rem; vertical-align: middle; }
    .rv-dot-on { background: #35e06a; box-shadow: 0 0 6px #35e06a; }
    .rv-dot-stby { background: #ffab1a; box-shadow: 0 0 6px #ffab1a; }
    .rv-dot-unk { background: #5b616b; }
    .rv-stage { position: relative; display: inline-block; line-height: 0; }
    .rv-stage canvas { display: block; }
    #rvOver { position: absolute; left: 0; top: 0; pointer-events: none; }
    #rvBase { cursor: default; }
    .rv-tip {
        position: absolute;
        z-index: 20;
        width: 340px;
        max-width: 92vw;
        line-height: 1.35;
        font-size: 0.85rem;
        color: var(--bs-body-color);
        background: var(--bs-body-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        box-shadow: 0 8px 28px rgba(0, 0, 0, 0.45);
        padding: 0.7rem 0.85rem;
    }
    .rv-tip.rv-pinned { border-color: #4dabf7; }
    .rv-tip-title { font-weight: 700; font-size: 0.95rem; color: var(--bs-emphasis-color); padding-right: 1.4rem; word-break: break-word; }
    .rv-tip-sub { color: var(--bs-secondary-color); margin-bottom: 0.35rem; }
    .rv-tip-close { position: absolute; right: 0.45rem; top: 0.25rem; border: 0; background: transparent; color: var(--bs-secondary-color); font-size: 1.1rem; line-height: 1; cursor: pointer; }
    .rv-tip table { width: 100%; margin: 0.35rem 0; }
    .rv-tip td { padding: 1px 0; vertical-align: top; }
    .rv-tip td:first-child { color: var(--bs-secondary-color); white-space: nowrap; padding-right: 0.6rem; }
    .rv-tip .rv-pw { font-weight: 600; }
    .rv-tip .rv-links { display: flex; flex-direction: column; gap: 0.15rem; margin-top: 0.4rem; border-top: 1px solid var(--bs-border-color); padding-top: 0.4rem; }
    .rv-tip .rv-links a { text-decoration: none; }
    .rv-tip .rv-links a:hover { text-decoration: underline; }
    .rv-unpl-item { padding: 0.35rem 0; border-bottom: 1px solid var(--bs-border-color); }
    .rv-unpl-item:last-child { border-bottom: 0; }
    .rv-unpl-item a { display: block; margin: 0.1rem 0 0 0; white-space: normal; overflow-wrap: anywhere; }
</style>

<script>
(function () {
    'use strict';

    var DATA = {{.RackJSON}};
    var base = document.getElementById('rvBase');
    if (!DATA || !base) return;

    var over = document.getElementById('rvOver');
    var stage = document.getElementById('rvStage');
    var tip = document.getElementById('rvTip');
    var bctx = base.getContext('2d');
    var octx = over.getContext('2d');
    var TAU = Math.PI * 2;

    var C_ON = '#35e06a', C_STBY = '#ffab1a', C_UID = '#3da5ff', C_ERR = '#ff453a';
    var RATIO = 10.86;          // ширина 19" панели / высота 1U (482.6 / 44.45)
    var GUT = 40, TOP = 38, BOT = 24;
    var ZKEY = 'rackview.uh';
    var UH = 40;
    var EMBED = {{if .Embed}}true{{else}}false{{end}};
    var fitUH = 40, userZoomed = false;
    if (EMBED) TOP = 16; // во фрейме ЦОД заголовок шкафа не рисуется - верхнюю панель сжимаем

    // ---------- стеллаж (площадка без шкафов: склады, рабочие комнаты) ----------
    // Секция стеллажа - габариты шкафа 42U (высота 42U, ширина 19" устройства), 5 полок. На полке
    // устройства стоят друг на друге снизу вверх; тяжёлое (СХД, корзины) - на нижних полках; не
    // поместившееся - следующая секция справа. Раскладка в юнитах (U).
    var SH = null;
    var SH_FULL = 42, SH_LEV = 5, SH_UP = 1.1, SH_BEAM = 0.8, SH_FEET = 0.5, SH_PAD = 0.7, SH_GAP = 1.6, SH_GAPU = 0.12;
    function shelfSize(it) {
        if (it.kind === 'switch' && netClass(it).fam === 'ap') return Math.max(2, it.sizeU || 0);   // точка доступа: не меньше 2U
        if (it.sizeU > 0) return it.sizeU;
        if (it.kind === 'chassis') return 10;
        if (it.kind === 'tape') return 6;
        if (it.kind === 'switch') return netClass(it).fam === 'ap' ? 2 : 1;
        return 2; // сервер/СХД без указанной высоты - стилизованный сервер 2U
    }
    function prepShelf() {
        var clear = (SH_FULL - SH_FEET - SH_BEAM * (SH_LEV + 1)) / SH_LEV;   // свободная высота между полками
        var list = DATA.items.slice();
        list.forEach(function (it) { it.su = Math.min(shelfSize(it), clear); it.suFull = shelfSize(it); });
        var rank = { storage: 0, tape: 1, chassis: 1, server: 2, 'switch': 3 };
        list.sort(function (a, b) { return (b.su - a.su) || ((rank[a.kind] || 9) - (rank[b.kind] || 9)) || String(a.name).localeCompare(String(b.name)); });
        var units = [];
        function newUnit() {
            var lv = [], y = SH_FEET;
            for (var k = 0; k < SH_LEV; k++) { lv.push({ deck: y + SH_BEAM, items: [], used: 0 }); y += SH_BEAM + clear; }
            var u = { levels: lv }; units.push(u); return u;
        }
        newUnit();
        list.forEach(function (it) {
            for (var ui = 0; ; ui++) {
                if (ui >= units.length) newUnit();
                var lv = units[ui].levels;
                for (var k = 0; k < lv.length; k++) {
                    if (lv[k].used + it.su <= clear + 0.001) { lv[k].items.push({ it: it, off: lv[k].used }); lv[k].used += it.su + SH_GAPU; return; }
                }
            }
        });
        var innerW = RATIO + 2 * SH_PAD, unitW = innerW + 2 * SH_UP;
        SH = { units: units, innerW: innerW, unitW: unitW, clear: clear, height: SH_FULL, width: units.length * unitW + (units.length - 1) * SH_GAP };
        DATA.uHeight = SH_FULL + 2;
    }
    if (DATA.shelf) prepShelf();
    // подбор масштаба: шкаф по высоте равен области просмотра (во фрейме ЦОД)
    function calcFit() {
        var avail = window.innerHeight - 20 - TOP - BOT;
        var f = Math.floor(avail / Math.max(1, DATA.uHeight));
        if (SH) f = Math.min(f, Math.floor((window.innerWidth - 2 * GUT - 24) / SH.width));   // стеллаж вписывается и по ширине окна
        return Math.max(SH ? 6 : 12, Math.min(90, f));
    }
    if (EMBED) {
        fitUH = calcFit(); UH = fitUH;
    } else {
        try {
            var zs = parseInt(localStorage.getItem(ZKEY), 10);
            if (zs >= 20 && zs <= 90) UH = zs;
        } catch (e) { }
    }

    var g = { W: 0, H: 0, devW: 0, dpr: 1 };
    var regions = [];
    var hovered = null, shown = null, pinned = false;
    var showTimer = null, hideTimer = null;

    // ---------- примитивы ----------
    function rr(c, x, y, w, h, r) {
        r = Math.max(0, Math.min(r, w / 2, h / 2));
        c.beginPath();
        c.moveTo(x + r, y);
        c.lineTo(x + w - r, y); c.quadraticCurveTo(x + w, y, x + w, y + r);
        c.lineTo(x + w, y + h - r); c.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
        c.lineTo(x + r, y + h); c.quadraticCurveTo(x, y + h, x, y + h - r);
        c.lineTo(x, y + r); c.quadraticCurveTo(x, y, x + r, y);
        c.closePath();
    }
    function vg(c, y, h, stops) {
        var gr = c.createLinearGradient(0, y, 0, y + h);
        for (var i = 0; i < stops.length; i++) gr.addColorStop(stops[i][0], stops[i][1]);
        return gr;
    }
    function hg(c, x, w, stops) {
        var gr = c.createLinearGradient(x, 0, x + w, 0);
        for (var i = 0; i < stops.length; i++) gr.addColorStop(stops[i][0], stops[i][1]);
        return gr;
    }
    function fillRR(c, x, y, w, h, r, fill) { rr(c, x, y, w, h, r); c.fillStyle = fill; c.fill(); }
    function strokeRR(c, x, y, w, h, r, stroke, lw) { rr(c, x, y, w, h, r); c.strokeStyle = stroke; c.lineWidth = lw || 1; c.stroke(); }
    function led(c, cx, cy, r, color, on) {
        if (on) {
            c.save();
            c.shadowColor = color; c.shadowBlur = r * 2.6;
            c.fillStyle = color; c.beginPath(); c.arc(cx, cy, r, 0, TAU); c.fill();
            c.restore();
            c.fillStyle = 'rgba(255,255,255,0.65)';
            c.beginPath(); c.arc(cx - r * 0.28, cy - r * 0.3, r * 0.34, 0, TAU); c.fill();
        } else {
            c.fillStyle = '#1e2126'; c.beginPath(); c.arc(cx, cy, r, 0, TAU); c.fill();
            c.strokeStyle = 'rgba(0,0,0,0.65)'; c.lineWidth = Math.max(0.5, r * 0.25); c.stroke();
        }
    }
    function txt(c, s, x, y, size, color, weight, align) {
        c.font = (weight || '600') + ' ' + size + 'px "Segoe UI", Arial, sans-serif';
        c.fillStyle = color; c.textAlign = align || 'left'; c.textBaseline = 'middle';
        c.fillText(s, x, y);
    }
    function hash(n) { n = (Math.imul(n + 1, 2654435761) >>> 0); return (n % 1000) / 1000; }
    function pwrColor(p) { return p === 1 ? C_ON : (p === 2 ? C_STBY : null); }

    function powerButton(c, cx, cy, r, p) {
        var col = pwrColor(p);
        // кольцо индикации
        c.lineWidth = Math.max(1.4, r * 0.34);
        if (col) { c.save(); c.shadowColor = col; c.shadowBlur = r * 1.8; c.strokeStyle = col; c.beginPath(); c.arc(cx, cy, r * 1.22, 0, TAU); c.stroke(); c.restore(); }
        else { c.strokeStyle = '#2b2f35'; c.beginPath(); c.arc(cx, cy, r * 1.22, 0, TAU); c.stroke(); }
        var gr = c.createRadialGradient(cx - r * 0.3, cy - r * 0.35, r * 0.1, cx, cy, r);
        gr.addColorStop(0, '#f1f3f5'); gr.addColorStop(1, '#868c95');
        c.fillStyle = gr; c.beginPath(); c.arc(cx, cy, r, 0, TAU); c.fill();
        // символ питания
        c.strokeStyle = '#2a2d32'; c.lineWidth = Math.max(1, r * 0.22); c.lineCap = 'round';
        c.beginPath(); c.arc(cx, cy + r * 0.05, r * 0.5, -Math.PI * 0.35, Math.PI * 1.35 + Math.PI * 0.0, false); c.stroke();
        c.beginPath(); c.moveTo(cx, cy - r * 0.62); c.lineTo(cx, cy - r * 0.05); c.stroke();
        c.lineCap = 'butt';
    }

    function ventDots(c, x, y, w, h, on) {
        var step = Math.max(4, UH * 0.13), r = Math.max(0.8, step * 0.2);
        c.fillStyle = on ? 'rgba(255,255,255,0.16)' : 'rgba(255,255,255,0.08)';
        for (var yy = y + step / 2; yy < y + h; yy += step) {
            var off = (Math.round((yy - y) / step) % 2) ? step / 2 : 0;
            for (var xx = x + step / 2 + off; xx < x + w; xx += step) {
                c.beginPath(); c.arc(xx, yy, r, 0, TAU); c.fill();
            }
        }
    }

    function earSlots(c, x, y, ew, n, rightSide) {
        for (var k = 0; k < n; k++) {
            var cy = y + (k + 0.5) * UH;
            c.fillStyle = 'rgba(0,0,0,0.7)';
            c.beginPath(); c.ellipse(x + ew / 2, cy, ew * 0.2, UH * 0.13, 0, 0, TAU); c.fill();
            c.strokeStyle = 'rgba(255,255,255,0.12)'; c.lineWidth = 0.6; c.stroke();
        }
    }

    // ---------- диски ----------
    function driveFlat(c, x, y, w, h, on, idx, th) {
        fillRR(c, x, y, w, h, 1.5, vg(c, y, h, [[0, '#4a4e55'], [1, '#202327']]));
        strokeRR(c, x + 0.5, y + 0.5, w - 1, h - 1, 1.5, 'rgba(0,0,0,0.8)', 1);
        // ручка-защёлка
        fillRR(c, x + w * 0.08, y + h * 0.16, w * 0.84, h * 0.2, 1, '#14161a');
        // прорези вентиляции
        c.fillStyle = 'rgba(0,0,0,0.55)';
        for (var i = 0; i < 5; i++) c.fillRect(x + w * 0.1 + i * (w * 0.8 / 5), y + h * 0.5, Math.max(1, w * 0.06), h * 0.28);
        // светодиоды
        led(c, x + w * 0.14, y + h * 0.86, Math.max(1.1, h * 0.07), on ? (hash(idx) > 0.45 ? '#ffffff' : '#2fd36a') : '#000', on);
        led(c, x + w * 0.3, y + h * 0.86, Math.max(1.1, h * 0.07), th.bayLed || C_UID, false);
    }
    function driveVert(c, x, y, w, h, on, idx, th) {
        fillRR(c, x, y, w, h, 1, hg(c, x, w, [[0, '#2a2d32'], [0.5, '#4c5057'], [1, '#222428']]));
        strokeRR(c, x + 0.5, y + 0.5, w - 1, h - 1, 1, 'rgba(0,0,0,0.85)', 1);
        // защёлка внизу
        fillRR(c, x + w * 0.18, y + h * 0.7, w * 0.64, h * 0.2, 1, '#9096a0');
        // решётка
        c.fillStyle = 'rgba(0,0,0,0.5)';
        for (var i = 0; i < 5; i++) c.fillRect(x + w * 0.2, y + h * (0.22 + i * 0.09), w * 0.6, Math.max(0.8, h * 0.03));
        // светодиод активности/статуса сверху
        led(c, x + w / 2, y + h * 0.1, Math.max(1, Math.min(w * 0.18, h * 0.04)), on ? (hash(idx) > 0.4 ? (th.bayLed || '#2fd36a') : '#2fd36a') : '#000', on);
    }

    // ---------- серверы ----------
    var THEMES = {
        hpe: { bezel: ['#383b41', '#1a1c20'], ear: ['#2c2f34', '#16181b'], cage: '#0e0f11', logo: 'hpe', cols1: 4, rows1: 2, bayLed: '#3de08a' },
        dell: { bezel: ['#2e3034', '#16171a'], ear: ['#cfd3d9', '#8a9099'], cage: '#0c0d0f', logo: 'dell', cols1: 5, rows1: 2, bayLed: '#3da5ff' },
        lenovo: { bezel: ['#222428', '#0f1012'], ear: ['#2b2d31', '#141518'], cage: '#08080a', logo: 'lenovo', cols1: 5, rows1: 2, bayLed: '#ff6a3d', accent: '#e2231a' },
        generic: { bezel: ['#50545b', '#2d3035'], ear: ['#a3a9b1', '#6c7179'], cage: '#15161a', logo: '', cols1: 4, rows1: 2, bayLed: '#3de08a' }
    };

    function drawLogo(c, kind, x, y, h) {
        var fs = Math.max(7, h * 0.4);
        if (kind === 'hpe') {
            var w = h * 1.7;
            c.strokeStyle = '#01a982'; c.lineWidth = Math.max(1.4, h * 0.14);
            c.strokeRect(x + c.lineWidth / 2, y + c.lineWidth / 2, w - c.lineWidth, h - c.lineWidth);
            txt(c, 'hpe', x + w / 2, y + h / 2 + 0.5, fs * 0.9, '#f2f5f4', '700', 'center');
        } else if (kind === 'dell') {
            var r = h * 0.5;
            c.strokeStyle = '#0076ce'; c.lineWidth = Math.max(1.4, h * 0.12);
            c.beginPath(); c.arc(x + r, y + r, r - c.lineWidth / 2, 0, TAU); c.stroke();
            txt(c, 'DELL', x + r, y + r + 0.5, fs * 0.62, '#e8f1fb', '800', 'center');
        } else if (kind === 'lenovo') {
            txt(c, 'Lenovo', x, y + h / 2, fs * 1.05, '#f4f4f4', '800', 'left');
        }
    }

    function bayConfig(n, th) {
        if (n <= 1) return { flat: true, cols: th.cols1, rows: th.rows1 };
        if (n === 2) return { flat: false, cols: 24, rows: 1 };
        return { flat: false, cols: 12, rows: 2 };
    }

    function drawServer(c, it, x, y, w, h, th) {
        var n = Math.max(1, it.sizeU), p = it.power, on = (p === 1);
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, th.bezel[0]], [1, th.bezel[1]]]));
        c.fillStyle = 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        if (th.accent) { c.fillStyle = th.accent; c.fillRect(x + w * 0.045, y + 1.5, w * 0.91, Math.max(1, UH * 0.03)); }
        var ew = w * 0.042;
        var earFill = vg(c, y, h, [[0, th.ear[0]], [1, th.ear[1]]]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill);
        fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        earSlots(c, x, y, ew, n); earSlots(c, x + w - ew, y, ew, n);

        // панель управления слева
        var cx = x + w * 0.09;
        var r = Math.max(3.2, UH * 0.15);
        var pcy = y + h * (n === 1 ? 0.36 : 0.28);
        powerButton(c, cx, pcy, r, p);
        var ly = y + h * (n === 1 ? 0.78 : 0.56);
        led(c, cx - r * 1.1, ly, Math.max(1.4, UH * 0.05), C_UID, false);
        led(c, cx + r * 0.2, ly, Math.max(1.4, UH * 0.05), C_ON, on);
        led(c, cx + r * 1.5, ly, Math.max(1.4, UH * 0.05), C_ERR, false);
        if (n >= 2 && th.logo) drawLogo(c, th.logo, x + w * 0.052, y + h * 0.74, Math.min(UH * 0.5, h * 0.24));

        // дисковая корзина
        var cgx = x + w * 0.145, cgw = w * 0.655, cgy = y + h * 0.1, cgh = h * 0.8;
        fillRR(c, cgx, cgy, cgw, cgh, 2, th.cage);
        var cfg = bayConfig(n, th);
        var cellW = cgw / cfg.cols, cellH = cgh / cfg.rows, idx = it.id * 7;
        for (var rr_ = 0; rr_ < cfg.rows; rr_++) {
            for (var cc = 0; cc < cfg.cols; cc++) {
                var gap = cfg.flat ? 2 : 1;
                var bx = cgx + cc * cellW + gap, by = cgy + rr_ * cellH + gap;
                var bw = cellW - gap * 2, bh = cellH - gap * 2;
                if (cfg.flat) driveFlat(c, bx, by, bw, bh, on, idx++, th); else driveVert(c, bx, by, bw, bh, on, idx++, th);
            }
        }

        // правая часть: медиа-отсек, порты, вентиляция
        var vx = x + w * 0.815, vw = w * 0.14;
        if (n >= 2) {
            fillRR(c, vx, y + h * 0.1, vw, h * 0.2, 1.5, '#0b0c0e');
            c.fillStyle = '#2a2d32'; c.fillRect(vx + vw * 0.1, y + h * 0.18, vw * 0.8, Math.max(1, h * 0.03));
            ventDots(c, vx, y + h * 0.36, vw, h * 0.54, on);
        } else {
            if (th.logo) drawLogo(c, th.logo, vx + vw * 0.08, y + h * 0.1, Math.min(UH * 0.26, h * 0.3));
            ventDots(c, vx, y + h * 0.44, vw, h * 0.3, on);
        }
        // USB-порты
        fillRR(c, vx + vw * 0.55, y + h * 0.9 - UH * 0.09, vw * 0.18, UH * 0.09, 1, '#0b0c0e');
        fillRR(c, vx + vw * 0.8, y + h * 0.9 - UH * 0.09, vw * 0.18, UH * 0.09, 1, '#0b0c0e');
        if (UH >= 30) txt(c, (it.model || '').replace(/^(HPE?|Dell|Lenovo)\s+/i, ''), x + w * 0.955, y + h - UH * 0.13, Math.max(7, UH * 0.2), 'rgba(230,233,238,0.55)', '500', 'right');
    }

    // ---------- блейд-корзины ----------
    function drawBlade(c, bx, by, bw, bh, b, syn, slotNum) {
        if (!b) {
            fillRR(c, bx, by, bw, bh, 1.5, vg(c, by, bh, [[0, '#2a2c31'], [1, '#17181b']]));
            c.fillStyle = 'rgba(0,0,0,0.55)';
            for (var i = 1; i <= 3; i++) c.fillRect(bx + bw * i / 4 - 0.5, by + bh * 0.1, 1.2, bh * 0.8);
            if (UH >= 30) txt(c, String(slotNum), bx + bw / 2, by + bh - UH * 0.16, Math.max(6, UH * 0.17), 'rgba(255,255,255,0.18)', '500', 'center');
            return;
        }
        var p = b.power;
        fillRR(c, bx, by, bw, bh, 1.5, vg(c, by, bh, [[0, syn ? '#2d3138' : '#34373c'], [1, '#17191c']]));
        strokeRR(c, bx + 0.5, by + 0.5, bw - 1, bh - 1, 1.5, 'rgba(0,0,0,0.85)', 1);
        if (syn) { c.fillStyle = '#00b5e2'; c.fillRect(bx + 1, by + 1, bw - 2, Math.max(1, bh * 0.03)); }
        // защёлка снизу
        fillRR(c, bx + bw * 0.08, by + bh * 0.84, bw * 0.84, bh * 0.1, 1, '#9096a0');
        // два диска
        fillRR(c, bx + bw * 0.14, by + bh * 0.14, bw * 0.72, bh * 0.17, 1, '#0e0f11');
        fillRR(c, bx + bw * 0.14, by + bh * 0.36, bw * 0.72, bh * 0.17, 1, '#0e0f11');
        led(c, bx + bw * 0.24, by + bh * 0.225, Math.max(0.9, bw * 0.03), '#2fd36a', p === 1);
        led(c, bx + bw * 0.24, by + bh * 0.445, Math.max(0.9, bw * 0.03), '#2fd36a', p === 1);
        // кнопка питания и UID
        var r = Math.max(2.2, bw * 0.13);
        powerButton(c, bx + bw / 2, by + bh * 0.68, r, p);
        led(c, bx + bw * 0.2, by + bh * 0.68, Math.max(1, bw * 0.035), C_UID, false);
    }

    function drawChassis(c, it, x, y, w, h, regs) {
        var syn = /synergy|sy12|12000/i.test((it.vendor + ' ' + it.model + ' ' + it.name));
        var p = it.power, on = (p === 1);
        var slots = it.slots > 0 ? it.slots : (syn ? 12 : 16);
        var cols = Math.ceil(slots / 2), rows = 2;
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, syn ? '#2a2d34' : '#3a3d42'], [1, '#17181b']]));
        c.fillStyle = 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        if (syn) { c.fillStyle = '#00b5e2'; c.fillRect(x + 3, y + 2, w - 6, Math.max(1, UH * 0.04)); }
        var ew = w * 0.03;
        var earFill = vg(c, y, h, [[0, '#4a4d53'], [1, '#202226']]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill);
        fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        earSlots(c, x, y, ew, Math.max(1, Math.round(h / UH))); earSlots(c, x + w - ew, y, ew, Math.max(1, Math.round(h / UH)));

        var ax = x + ew + w * 0.006, aw = w - 2 * ew - w * 0.012;
        var ay = y + h * 0.045, ah = h * 0.66;
        var sw = aw / cols, sh = ah / rows;
        var bySlot = {};
        for (var i = 0; i < it.blades.length; i++) bySlot[it.blades[i].slot] = it.blades[i];
        for (var s = 1; s <= slots; s++) {
            var col = (s - 1) % cols, row = Math.floor((s - 1) / cols);
            var bx = ax + col * sw + 1, by = ay + row * sh + 1, bw = sw - 2, bh = sh - 2;
            var b = bySlot[s];
            drawBlade(c, bx, by, bw, bh, b, syn, s);
            if (b) regs.push({ x: bx, y: by, w: bw, h: bh, it: it, blade: b });
        }

        // нижняя часть: блоки питания и дисплей Insight
        var py = y + h * 0.745, ph = h * 0.21;
        var psuW = aw * 0.125, gapx = aw * 0.012;
        var dispW = aw * 0.16;
        var total = psuW * 6 + dispW + gapx * 6;
        var px = ax + (aw - total) / 2;
        for (var k = 0; k < 7; k++) {
            if (k === 3) {
                fillRR(c, px, py, dispW, ph, 2, '#0b0c0e');
                fillRR(c, px + dispW * 0.1, py + ph * 0.18, dispW * 0.8, ph * 0.5, 1.5, on ? '#0c3d57' : '#10151a');
                if (on && UH >= 28) txt(c, it.name, px + dispW / 2, py + ph * 0.43, Math.max(6, UH * 0.19), '#8fd8ff', '600', 'center');
                led(c, px + dispW * 0.3, py + ph * 0.82, Math.max(1.2, UH * 0.04), '#2fd36a', on);
                led(c, px + dispW * 0.5, py + ph * 0.82, Math.max(1.2, UH * 0.04), C_UID, false);
                led(c, px + dispW * 0.7, py + ph * 0.82, Math.max(1.2, UH * 0.04), C_ERR, false);
                px += dispW + gapx;
                continue;
            }
            fillRR(c, px, py, psuW, ph, 2, vg(c, py, ph, [[0, '#2f3237'], [1, '#17191c']]));
            strokeRR(c, px + 0.5, py + 0.5, psuW - 1, ph - 1, 2, 'rgba(0,0,0,0.8)', 1);
            c.fillStyle = 'rgba(0,0,0,0.55)';
            for (var v = 0; v < 6; v++) c.fillRect(px + psuW * 0.12, py + ph * (0.14 + v * 0.1), psuW * 0.76, Math.max(1, ph * 0.04));
            led(c, px + psuW * 0.5, py + ph * 0.86, Math.max(1.2, UH * 0.045), '#2fd36a', on);
            px += psuW + gapx;
        }
        if (UH >= 26) txt(c, it.model, x + w * 0.5, y + h * 0.985 - UH * 0.04, Math.max(6, UH * 0.17), 'rgba(235,238,242,0.5)', '500', 'center');
    }

    // ---------- СХД ----------
    var STORAGE = {
        huawei: { chunk: 2, drives: 25, bezel: ['#3b3e44', '#18191d'], ear: ['#b2b6bd', '#70757d'], accent: '#c7000b', label: 'HUAWEI', sub: 'OceanStor', bayLed: '#3de08a' },
        unity: { chunk: 2, drives: 25, bezel: ['#34373c', '#16171a'], ear: ['#c8ccd2', '#868c95'], accent: '#0076ce', label: 'DELL EMC', sub: 'UNITY', bayLed: '#3da5ff' },
        vnx: { chunk: 3, drives: 15, bezel: ['#2c2e32', '#131416'], ear: ['#bfc3c9', '#7c828a'], accent: '#6b7280', label: 'EMC', sub: 'VNX', bayLed: '#3de08a' },
        lenovo: { chunk: 2, drives: 24, bezel: ['#23252a', '#101113'], ear: ['#2c2e33', '#151619'], accent: '#e2231a', label: 'Lenovo', sub: 'ThinkSystem DE', bayLed: '#ff6a3d' },
        generic: { chunk: 2, drives: 24, bezel: ['#4a4e55', '#2b2e33'], ear: ['#a3a9b1', '#6c7179'], accent: '#8892a0', label: '', sub: '', bayLed: '#3de08a' }
    };

    function drawStorageChunk(c, it, th, x, y, w, h, first, remainFrac) {
        var p = it.power, on = (p === 1);
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, th.bezel[0]], [1, th.bezel[1]]]));
        c.fillStyle = 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        c.fillStyle = th.accent; c.fillRect(x + w * 0.04, y + h - 2.5, w * 0.92, 1.5);
        var ew = w * 0.04;
        var earFill = vg(c, y, h, [[0, th.ear[0]], [1, th.ear[1]]]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill);
        fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        var nU = Math.max(1, Math.round(h / UH));
        earSlots(c, x, y, ew, nU); earSlots(c, x + w - ew, y, ew, nU);

        // левая панель: логотип + индикаторы
        var px = x + ew + w * 0.006, pw = w * 0.085;
        if (th.label) {
            txt(c, th.label, px + pw / 2, y + h * 0.2, Math.max(6, Math.min(UH * 0.2, pw * 0.2)), th.accent === '#c7000b' ? '#ff3b3b' : '#e9edf2', '800', 'center');
            if (th.sub) txt(c, th.sub, px + pw / 2, y + h * 0.34, Math.max(5.5, Math.min(UH * 0.15, pw * 0.15)), 'rgba(230,233,238,0.7)', '500', 'center');
        }
        var lr = Math.max(1.4, UH * 0.05);
        if (first) {
            powerButton(c, px + pw * 0.5, y + h * 0.6, Math.max(3, UH * 0.13), p === 1 ? 1 : 0);
            led(c, px + pw * 0.2, y + h * 0.86, lr, C_UID, false);
            led(c, px + pw * 0.5, y + h * 0.86, lr, C_ON, on);
            led(c, px + pw * 0.8, y + h * 0.86, lr, C_ERR, false);
        } else {
            led(c, px + pw * 0.5, y + h * 0.7, lr, C_ON, on);
        }

        // диски (вертикальные), для неполной последней полки - пропорционально
        var dx = px + pw + w * 0.01, dw = w - 2 * ew - pw - w * 0.03;
        var dy = y + h * 0.1, dh = h * 0.78;
        fillRR(c, dx, dy, dw, dh, 2, '#0b0c0e');
        var n = th.drives, sw = dw / n;
        var cnt = Math.max(1, Math.round(n * (remainFrac || 1)));
        for (var i = 0; i < n; i++) {
            if (i >= cnt) {
                c.fillStyle = '#17181b'; c.fillRect(dx + i * sw + 1, dy + 1, sw - 2, dh - 2);
                continue;
            }
            driveVert(c, dx + i * sw + 0.8, dy + 1, sw - 1.6, dh - 2, on, it.id * 11 + i, th);
        }
    }

    function drawStorage(c, it, x, y, w, h) {
        var s = (it.vendor + ' ' + it.model + ' ' + it.name).toLowerCase();
        var key = 'generic';
        if (/huawei|oceanstor|dorado/.test(s)) key = 'huawei';
        else if (/vnx/.test(s)) key = 'vnx';
        else if (/unity|dell|emc/.test(s)) key = 'unity';
        else if (/lenovo|thinksystem|\bde\d/.test(s)) key = 'lenovo';
        var th = STORAGE[key];
        var chunk = th.chunk, total = Math.max(1, it.sizeU);
        var done = 0, first = true;
        while (done < total) {
            var u = Math.min(chunk, total - done);
            // верхний блок в стойке рисуется первым: контроллерный блок - нижний (стартовый юнит)
            done += u;
            var cy = y + h - done * UH;      // от нижнего края вверх
            drawStorageChunk(c, it, th, x, cy, w, u * UH, first, u / chunk);
            first = false;
        }
    }


    // ---------- точки доступа Wi-Fi (Ubiquiti UniFi, Cisco Aironet/Catalyst 9100) ----------
    // Вид сверху на потолочное устройство, по центру отведённой высоты: UniFi AC Pro - круглая "шайба" с
    // кольцом светодиода; Cisco AIR-AP1142 - прямоугольный корпус с жалюзи; AIR-AP3802 - крупный корпус с
    // вентиляционными прорезями; AIR-AP1815 - малый квадрат; C9120AXI - квадрат со скошенными гранями и круглым индикатором.
    function drawAP(c, it, x, y, w, h, k) {
        var s = k.text, on = it.power === 1;
        var kind = /unifi|uap|ubiquiti/.test(s) ? 'unifi' : /1142/.test(s) ? 'a1142' : /3802|3702|3602/.test(s) ? 'a3802' : /1815|1810|1832|1852/.test(s) ? 'a1815' : /c9[1-9]\d{2}/.test(s) ? 'c9120' : 'round';
        var sz = Math.min(h * 0.88, w * 0.3), cx = x + w / 2, cy = y + h / 2;
        var asp = { unifi: 1, a1142: 1.5, a3802: 1.3, a1815: 1.0, c9120: 1.12, round: 1 }[kind];
        var bw = sz * asp, bh = sz;
        // тень и подставка-лоток
        c.fillStyle = 'rgba(0,0,0,0.35)'; c.beginPath(); c.ellipse(cx + 1, cy + bh * 0.46, bw * 0.55, bh * 0.1, 0, 0, TAU); c.fill();
        var body = c.createLinearGradient(cx - bw / 2, cy - bh / 2, cx + bw / 2, cy + bh / 2);
        body.addColorStop(0, '#f6f7f9'); body.addColorStop(1, '#bfc5ce');
        if (kind === 'unifi' || kind === 'round') {
            c.fillStyle = body; c.beginPath(); c.arc(cx, cy, bh / 2, 0, TAU); c.fill();
            c.strokeStyle = 'rgba(0,0,0,0.3)'; c.lineWidth = 1; c.stroke();
            c.strokeStyle = 'rgba(0,0,0,0.12)'; c.beginPath(); c.arc(cx, cy, bh * 0.42, 0, TAU); c.stroke();
            if (kind === 'unifi') {
                c.save();
                if (on) { c.shadowColor = '#3aa0ff'; c.shadowBlur = bh * 0.25; }
                c.strokeStyle = on ? '#3aa0ff' : '#9aa3ad'; c.lineWidth = Math.max(1.5, bh * 0.07);
                c.beginPath(); c.arc(cx, cy, bh * 0.17, 0, TAU); c.stroke();
                c.restore();
            } else led(c, cx, cy, Math.max(2, bh * 0.07), C_ON, on);
        } else {
            fillRR(c, cx - bw / 2, cy - bh / 2, bw, bh, bh * (kind === 'c9120' ? 0.3 : 0.14), body);
            strokeRR(c, cx - bw / 2 + 0.5, cy - bh / 2 + 0.5, bw - 1, bh - 1, bh * (kind === 'c9120' ? 0.3 : 0.14), 'rgba(0,0,0,0.3)', 1);
            c.strokeStyle = 'rgba(0,0,0,0.16)'; c.lineWidth = 1;
            if (kind === 'a1142') {                       // жалюзи по длинной стороне
                for (var i = 1; i < 6; i++) { var ly = cy - bh / 2 + bh * i / 6; c.beginPath(); c.moveTo(cx - bw * 0.42, ly); c.lineTo(cx + bw * 0.42, ly); c.stroke(); }
            } else if (kind === 'a3802') {                // вентиляционные прорези
                c.fillStyle = 'rgba(0,0,0,0.18)';
                for (var r = 0; r < 4; r++) for (var q = 0; q < 8; q++) c.fillRect(cx - bw * 0.38 + q * bw * 0.1, cy - bh * 0.32 + r * bh * 0.16, bw * 0.06, bh * 0.08);
            } else if (kind === 'c9120') {                // скошенная грань и круглый индикатор
                strokeRR(c, cx - bw * 0.42, cy - bh * 0.42, bw * 0.84, bh * 0.84, bh * 0.22, 'rgba(0,0,0,0.12)', 1);
            }
            var ledX = kind === 'c9120' ? cx : cx + bw * 0.34, ledY = kind === 'c9120' ? cy + bh * 0.28 : cy - bh * 0.3;
            led(c, ledX, ledY, Math.max(2, bh * (kind === 'c9120' ? 0.07 : 0.05)), kind === 'c9120' ? '#35c3ff' : C_ON, on);
            if (kind !== 'unifi' && bh >= 18) ciscoLogo(c, cx - bw * 0.36, cy + bh * 0.14, Math.max(5, bh * 0.2), false);
        }
    }

    // ---------- сетевое оборудование (Cisco и др.) ----------
    // Определение семейства по "вендор модель имя тип": Nexus, MDS, ASA/Firepower, маршрутизаторы
    // (ISR/ASR/Catalyst 8000), модульные шасси (Catalyst 4500/6500/9400/9600, Nexus 7000), Catalyst
    // (старые 2960/3560/3750/3850 - светлый корпус, Catalyst 9000 - тёмный), прочие производители.
    function netClass(it) {
        var s = ((it.vendor || '') + ' ' + (it.model || '') + ' ' + (it.name || '') + ' ' + (it.subtype || '')).toLowerCase();
        var cisco = /cisco|catalyst|nexus|\bws-c|\bn[3579]k|\bisr|\basr|\bmds|aironet|\bair-|\bc91\d{2}|\basa\b|firepower/.test(s);
        var fam = 'switch', era = 'old';
        if (/aironet|\bair-|\bc91\d{2}|unifi|\buap|ubiquiti|точка доступа|access point|wi-?fi|\bwlan/.test(s)) fam = 'ap';
        else if (/nexus|\bn[3579]k|\bn[3579]\d{3}/.test(s)) fam = 'nexus';
        else if (/\bmds/.test(s)) fam = 'mds';
        else if (/\basa\b|asa ?55|firepower|\bfpr|\bftd|межсетев|firewall/.test(s)) fam = 'asa';
        else if (/\bisr|\basr|router|маршрутиз|\bc8[0-9]{3}|\bcsr/.test(s)) fam = 'router';
        var modular = /4500|4506|4507|4510|6500|6504|6506|6509|6513|9400|9407|9410|9600|9606|\bn7k|nexus ?7|\basr ?9/.test(s);
        if (/9[2-6]\d{2}\b|\bc9\d{3}|catalyst 9|nexus|\bc8/.test(s)) era = 'new';
        var ports = 24, m = s.match(/-(\d{2})[a-z]/);
        if (m) ports = parseInt(m[1], 10);
        else if (/48|c9300|c9200|3850|2960x|3750x/.test(s) && !/24/.test(s)) ports = 48;
        if (ports !== 8 && ports !== 12 && ports !== 16 && ports !== 24 && ports !== 32 && ports !== 48) ports = 24;
        var vendor = /ubiquiti|unifi|\buap/.test(s) ? 'ubiquiti' : cisco ? 'cisco' : (/aruba|procurve|hpe|hp /.test(s) ? 'hpe' : /juniper/.test(s) ? 'juniper' : /huawei/.test(s) ? 'huawei' : /brocade|broadcom/.test(s) ? 'brocade' : /dell/.test(s) ? 'dell' : 'other');
        return { fam: fam, era: era, modular: modular || (it.sizeU >= 4 && fam === 'switch'), ports: ports, vendor: vendor, text: s };
    }
    var NET_ACCENT = { cisco: '#049fd9', hpe: '#ff8300', juniper: '#84b135', huawei: '#e0334d', brocade: '#d11f2f', dell: '#2a8be0', other: '#8b95a5', ubiquiti: '#006fff' };

    // логотип Cisco: пять штрихов-"мост" и надпись
    function ciscoLogo(c, x, y, h, dark) {
        var bw = Math.max(1, h * 0.09), hs = [0.45, 0.7, 1, 0.7, 0.45];
        c.fillStyle = dark ? '#e8ecf1' : '#1a6fa6';
        for (var i = 0; i < 5; i++) { var bh = h * hs[i]; c.fillRect(x + i * bw * 1.7, y + (h - bh) / 2, bw, bh); }
        if (h >= 8) txt(c, 'cisco', x + bw * 9, y + h / 2, Math.max(6, h * 0.62), dark ? '#e8ecf1' : '#1a6fa6', '800', 'left');
    }
    // ряд портов RJ45 / SFP
    function portRow(c, x, y, w, h, n, kind, light) {
        var cell = w / n, pw = Math.max(2, cell * 0.78), ph = h;
        for (var i = 0; i < n; i++) {
            var px = x + i * cell + (cell - pw) / 2;
            fillRR(c, px, y, pw, ph, 1, kind === 'sfp' ? '#0a0b0d' : '#0d0e10');
            c.fillStyle = kind === 'sfp' ? (light ? '#9aa1ab' : '#5b616b') : (light ? '#7b828c' : '#454a52');
            c.fillRect(px + pw * 0.12, y + ph * 0.18, pw * 0.76, ph * (kind === 'sfp' ? 0.4 : 0.34));
            if (kind !== 'sfp' && ph >= 6) { c.fillStyle = 'rgba(240,200,60,0.55)'; c.fillRect(px + pw * 0.12, y + ph * 0.68, pw * 0.76, Math.max(1, ph * 0.08)); }
        }
    }
    function drawNetwork(c, it, x, y, w, h) {
        var k = netClass(it), acc = NET_ACCENT[k.vendor], on = it.power === 1;
        if (k.fam === 'ap') { drawAP(c, it, x, y, w, h, k); return; }
        var light = (k.era === 'old' && k.fam === 'switch' && !k.modular && k.vendor === 'cisco');
        var top = light ? '#dfe3e8' : '#3a3e44', bot = light ? '#a2aab4' : '#17191c';
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, top], [1, bot]]));
        c.fillStyle = light ? 'rgba(255,255,255,0.55)' : 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        var ew = w * 0.034;
        var earFill = vg(c, y, h, light ? [[0, '#c3c9d1'], [1, '#868e99']] : [[0, '#4a4d53'], [1, '#202226']]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill); fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        earSlots(c, x, y, ew, Math.max(1, Math.round(h / UH))); earSlots(c, x + w - ew, y, ew, Math.max(1, Math.round(h / UH)));
        // акцентная полоса вендора
        c.fillStyle = acc; c.fillRect(x + ew + 1, y + 1.5, w - 2 * ew - 2, Math.max(1, UH * 0.035));
        var ix = x + ew + w * 0.01, iw = w - 2 * ew - w * 0.02;
        var lh = Math.min(h * 0.4, UH * 0.5);
        if (k.vendor === 'cisco') ciscoLogo(c, ix + iw * 0.005, y + h * 0.1, lh, !light);
        else if (UH >= 24) txt(c, k.vendor.toUpperCase(), ix + iw * 0.005, y + h * 0.28, Math.max(6, UH * 0.22), light ? '#31363d' : '#cfd5de', '800', 'left');
        var mn = (it.model || '').replace(/^(cisco|catalyst|hpe?|aruba)\s+/i, '');
        if (UH >= 30 && h <= UH * 2.2) txt(c, mn, ix + iw * 0.005, y + h * 0.82, Math.max(7, UH * 0.17), light ? 'rgba(30,35,42,0.75)' : 'rgba(230,233,238,0.55)', '600', 'left');
        // индикаторы системы
        var lr = Math.max(1.4, UH * 0.045), ly = y + h * (h > UH * 1.5 ? 0.55 : 0.58);
        led(c, ix + iw * 0.115, ly, lr, C_ON, on); led(c, ix + iw * 0.115 + lr * 3, ly, lr, C_STBY, it.power === 2); led(c, ix + iw * 0.115 + lr * 6, ly, lr, C_ERR, false);

        var px = ix + iw * 0.17, pw = iw * 0.63, ph = h * 0.8, py = y + h * 0.12;
        if (k.fam === 'asa') {
            // межсетевой экран: 8 медных портов, 2 SFP, USB, статусные LED
            portRow(c, px, py + ph * 0.18, pw * 0.58, ph * 0.5, 8, 'rj', false);
            portRow(c, px + pw * 0.62, py + ph * 0.18, pw * 0.14, ph * 0.5, 2, 'sfp', false);
            fillRR(c, px + pw * 0.82, py + ph * 0.22, pw * 0.07, ph * 0.4, 1, '#0a0b0d');
            fillRR(c, px + pw * 0.92, py + ph * 0.22, pw * 0.07, ph * 0.4, 1, '#0a0b0d');
            if (UH >= 24) txt(c, /firepower|fpr|ftd/.test(k.text) ? 'FIREPOWER' : 'ASA', px + pw * 0.58, py + ph * 0.92, Math.max(6, UH * 0.17), 'rgba(230,233,238,0.5)', '800', 'right');
        } else if (k.fam === 'router') {
            // маршрутизатор: модульные заглушки + WAN/LAN порты
            for (var s = 0; s < 3; s++) {
                var sx = px + s * pw * 0.24;
                fillRR(c, sx, py, pw * 0.22, ph, 1.5, vg(c, py, ph, [[0, '#2a2d33'], [1, '#17181b']]));
                strokeRR(c, sx + 0.5, py + 0.5, pw * 0.22 - 1, ph - 1, 1.5, 'rgba(255,255,255,0.12)', 1);
                c.fillStyle = '#8a919c'; c.fillRect(sx + pw * 0.02, py + ph * 0.1, 3, 3); c.fillRect(sx + pw * 0.22 - 5, py + ph * 0.1, 3, 3);
                c.fillRect(sx + pw * 0.02, py + ph * 0.82, 3, 3); c.fillRect(sx + pw * 0.22 - 5, py + ph * 0.82, 3, 3);
            }
            portRow(c, px + pw * 0.74, py + ph * 0.15, pw * 0.26, ph * 0.34, 4, 'rj', false);
            portRow(c, px + pw * 0.74, py + ph * 0.55, pw * 0.26, ph * 0.3, 2, 'sfp', false);
        } else if (k.modular) {
            // модульное шасси: супервизоры и линейные карты
            var cards = Math.max(2, Math.min(9, Math.round(h / UH / 1.6)));
            var chh = (h * 0.84) / cards;
            for (var q = 0; q < cards; q++) {
                var cy = y + h * 0.09 + q * chh;
                fillRR(c, px - iw * 0.02, cy + 0.5, pw + iw * 0.2, chh - 1, 1.5, vg(c, cy, chh, [[0, q < 2 ? '#3b3f46' : '#2b2e34'], [1, '#17181b']]));
                if (q < 2) { txt(c, 'SUP', px, cy + chh / 2, Math.max(6, UH * 0.15), 'rgba(230,233,238,0.5)', '700', 'left'); portRow(c, px + pw * 0.25, cy + chh * 0.25, pw * 0.4, chh * 0.5, 4, 'sfp', false); }
                else portRow(c, px, cy + chh * 0.2, pw * 0.9, chh * 0.6, 24, 'rj', false);
            }
        } else if (k.fam === 'nexus' || k.fam === 'mds') {
            // Nexus/MDS: два ряда SFP+/LC портов и блок QSFP
            var rows = k.fam === 'mds' ? 1 : 2, nn = k.fam === 'mds' ? 48 : 24;
            var rh = (ph * 0.8) / rows;
            for (var r = 0; r < rows; r++) portRow(c, px, py + ph * 0.1 + r * rh, pw * 0.8, rh * 0.85, nn, 'sfp', false);
            if (k.fam === 'nexus') {
                for (var qq = 0; qq < 3; qq++) { fillRR(c, px + pw * 0.83 + qq * pw * 0.056, py + ph * 0.1, pw * 0.05, ph * 0.8, 1, '#0a0b0d'); c.fillStyle = '#6f7580'; c.fillRect(px + pw * 0.835 + qq * pw * 0.056, py + ph * 0.18, pw * 0.04, ph * 0.3); }
            }
        } else {
            // коммутатор доступа: порты RJ45 в 1-2 ряда + SFP uplink
            var nrows = k.ports > 24 ? 2 : 1, ncols = Math.ceil(k.ports / nrows);
            var rh2 = (ph * 0.84) / nrows;
            for (var r2 = 0; r2 < nrows; r2++) portRow(c, px, py + ph * 0.08 + r2 * rh2, pw * 0.82, rh2 * 0.88, ncols, 'rj', light);
            portRow(c, px + pw * 0.86, py + ph * 0.08, pw * 0.14, ph * 0.38, 2, 'sfp', light);
            portRow(c, px + pw * 0.86, py + ph * 0.54, pw * 0.14, ph * 0.38, 2, 'sfp', light);
        }
        // консольный порт справа
        fillRR(c, x + w - ew - w * 0.05, y + h * 0.4, w * 0.035, Math.max(2, h * 0.2), 1, '#0a0b0d');
    }

    // ---------- ленточные библиотеки HPE StoreEver MSL ----------
    // MSL8096: ЖК-панель по центру, картриджные магазины слева и справа, снизу отсеки ленточных
    // накопителей. MSL6480: модули по 6U - левый край ЖК-панель и индикаторы, справа магазины,
    // в основном модуле снизу отсеки накопителей; в высоких (расширенных) библиотеках модули повторяются.
    function drawTape(c, it, x, y, w, h) {
        var s = ((it.model || '') + ' ' + (it.name || '')).toLowerCase();
        var is8096 = /8096/.test(s), on = it.power === 1, G_ = '#01a982';
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, '#2a2d33'], [0.5, '#1d1f23'], [1, '#121316']]));
        c.fillStyle = G_; c.fillRect(x + 2, y + 1, w - 4, Math.max(1.5, UH * 0.05));
        var ew = w * 0.03;
        var earFill = vg(c, y, h, [[0, '#4a4d53'], [1, '#202226']]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill); fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        var nU = Math.max(1, Math.round(h / UH));
        earSlots(c, x, y, ew, nU); earSlots(c, x + w - ew, y, ew, nU);
        var ix = x + ew + 2, iw = w - 2 * ew - 4;
        // логотип HPE (зелёная плашка) и название серии
        var lgh = Math.min(UH * 0.5, h * 0.12);
        fillRR(c, ix + iw * 0.01, y + UH * 0.18, lgh * 2.6, lgh, 2, G_);
        txt(c, 'HPE', ix + iw * 0.01 + lgh * 1.3, y + UH * 0.18 + lgh / 2, Math.max(6, lgh * 0.72), '#0b0c0e', '800', 'center');
        if (UH >= 24) txt(c, 'StoreEver ' + (/(msl\s*\d{4}|\d{4})/i.exec(it.model || '') ? 'MSL' + /(\d{4})/.exec(it.model)[1] : 'MSL'), ix + iw * 0.01 + lgh * 3, y + UH * 0.18 + lgh / 2, Math.max(7, UH * 0.2), 'rgba(230,233,238,0.7)', '700', 'left');

        var modH = Math.min(h, UH * 6), mods = Math.max(1, Math.round(h / modH));
        modH = h / mods;
        for (var mi = 0; mi < mods; mi++) {
            var my = y + mi * modH, first = (mi === 0);
            if (mi > 0) { c.fillStyle = 'rgba(0,0,0,0.6)'; c.fillRect(ix, my, iw, 1.5); c.fillStyle = 'rgba(255,255,255,0.08)'; c.fillRect(ix, my + 1.5, iw, 1); }
            var bodyY = my + (first ? UH * 0.8 : UH * 0.25), bodyH = modH - (first ? UH * 0.95 : UH * 0.4);
            var drvH = bodyH * 0.24, magH = bodyH - drvH - bodyH * 0.04;
            var lcdW = iw * 0.15, lcdX, magL, magR;
            if (is8096) { lcdX = ix + iw * 0.5 - lcdW / 2; magL = { x: ix + iw * 0.01, w: iw * 0.5 - lcdW / 2 - iw * 0.02 }; magR = { x: lcdX + lcdW + iw * 0.01, w: iw * 0.5 - lcdW / 2 - iw * 0.02 }; }
            else { lcdX = ix + iw * 0.01; magL = null; magR = { x: lcdX + lcdW + iw * 0.015, w: iw - lcdW - iw * 0.035 }; }
            // ЖК-панель управления
            if (first || is8096) {
                fillRR(c, lcdX, bodyY, lcdW, magH, 2, '#0d0e11');
                var sx = lcdX + lcdW * 0.08, sw = lcdW * 0.84, sh = Math.min(magH * 0.5, sw * 0.7);
                fillRR(c, sx, bodyY + magH * 0.08, sw, sh, 1.5, vg(c, bodyY, sh, [[0, '#1f6fb5'], [1, '#0b2c52']]));
                c.fillStyle = 'rgba(255,255,255,0.55)';
                for (var tl = 0; tl < 4; tl++) c.fillRect(sx + sw * 0.1, bodyY + magH * 0.08 + sh * (0.18 + tl * 0.2), sw * (tl % 2 ? 0.5 : 0.7), Math.max(1, sh * 0.05));
                var by0 = bodyY + magH * 0.08 + sh + magH * 0.06;
                for (var bt = 0; bt < 4; bt++) fillRR(c, sx + bt * sw * 0.26, by0, sw * 0.2, Math.max(2, magH * 0.06), 1, '#3a3e45');
                led(c, sx + sw * 0.12, by0 + magH * 0.14, Math.max(1.4, UH * 0.05), C_ON, on);
                led(c, sx + sw * 0.34, by0 + magH * 0.14, Math.max(1.4, UH * 0.05), C_STBY, it.power === 2);
                led(c, sx + sw * 0.56, by0 + magH * 0.14, Math.max(1.4, UH * 0.05), C_ERR, false);
            } else {
                magL = null; magR = { x: ix + iw * 0.01, w: iw * 0.98 };
            }
            // картриджные магазины
            function magazines(area) {
                if (!area) return;
                var cols = Math.max(2, Math.round(area.w / (UH * 1.9))), mw = area.w / cols;
                for (var cc = 0; cc < cols; cc++) {
                    var mx = area.x + cc * mw + 1;
                    fillRR(c, mx, bodyY, mw - 2, magH, 1.5, vg(c, bodyY, magH, [[0, '#2f3339'], [1, '#1a1c20']]));
                    strokeRR(c, mx + 0.5, bodyY + 0.5, mw - 3, magH - 1, 1.5, 'rgba(0,0,0,0.85)', 1);
                    var rowsN = Math.max(3, Math.round(magH / (UH * 0.42)));
                    c.fillStyle = 'rgba(0,0,0,0.5)';
                    for (var rr2 = 1; rr2 < rowsN; rr2++) c.fillRect(mx + 2, bodyY + magH * rr2 / rowsN, mw - 6, 1);
                    fillRR(c, mx + mw * 0.3, bodyY + magH * 0.42, mw * 0.4, Math.max(2, magH * 0.06), 1, '#8a919c');
                }
            }
            magazines(magL); magazines(magR);
            // отсеки ленточных накопителей (LTO) - нижняя полоса основного модуля
            if (first || is8096) {
                var dy = bodyY + magH + bodyH * 0.04, dn = is8096 ? 6 : 4, dwAll = iw * 0.98, dw = dwAll / dn;
                for (var di = 0; di < dn; di++) {
                    var dx = ix + iw * 0.01 + di * dw;
                    fillRR(c, dx + 1, dy, dw - 2, drvH, 1.5, vg(c, dy, drvH, [[0, '#3a3e45'], [1, '#1b1d21']]));
                    fillRR(c, dx + dw * 0.08, dy + drvH * 0.3, dw * 0.55, drvH * 0.4, 1, '#0a0b0d');
                    c.fillStyle = '#8a919c'; c.fillRect(dx + dw * 0.7, dy + drvH * 0.3, dw * 0.2, Math.max(1, drvH * 0.1));
                    led(c, dx + dw * 0.78, dy + drvH * 0.65, Math.max(1.2, UH * 0.04), C_ON, on && di < 3);
                }
            }
        }
        if (UH >= 30 && it.model) txt(c, it.model, x + w - ew - 6, y + h - UH * 0.16, Math.max(7, UH * 0.17), 'rgba(230,233,238,0.5)', '600', 'right');
    }

    // ---------- шкаф ----------
    function styleOf(it) {
        var s = (it.vendor + ' ' + it.model + ' ' + it.name).toLowerCase();
        if (/hpe|hp |hewlett|proliant|\bdl\d|\bml\d|\bbl\d/.test(s)) return THEMES.hpe;
        if (/dell|poweredge|\br\d{3}/.test(s)) return THEMES.dell;
        if (/lenovo|thinksystem|\bsr\d{3}/.test(s)) return THEMES.lenovo;
        return THEMES.generic;
    }

    function yOf(startU, size) { return TOP + (DATA.uHeight - (startU + size - 1)) * UH; }

    function layout() {
        g.dpr = window.devicePixelRatio || 1;
        g.devW = Math.round(UH * RATIO);
        g.W = GUT * 2 + (SH ? Math.round(SH.width * UH) : g.devW);
        g.H = TOP + DATA.uHeight * UH + BOT;
        [base, over].forEach(function (cv) {
            cv.width = Math.round(g.W * g.dpr); cv.height = Math.round(g.H * g.dpr);
            cv.style.width = g.W + 'px'; cv.style.height = g.H + 'px';
        });
    }

    function drawCabinet(c) {
        var W = g.W, H = g.H, dev = g.devW;
        // наружная рама
        fillRR(c, 0, 0, W, H, 8, hg(c, 0, W, [[0, '#14161a'], [0.5, '#23262c'], [1, '#14161a']]));
        strokeRR(c, 0.5, 0.5, W - 1, H - 1, 8, '#3a3e46', 1);
        // внутренний объём
        fillRR(c, GUT - 7, TOP - 6, dev + 14, DATA.uHeight * UH + 12, 3, '#08090b');
        // верхняя панель с названием
        fillRR(c, GUT - 7, 6, dev + 14, TOP - 14, 3, vg(c, 6, TOP - 14, [[0, '#2b2e35'], [1, '#1a1c21']]));
        if (!EMBED) txt(c, DATA.cabinetName + (DATA.site ? '  ·  ' + DATA.site : ''), W / 2, 6 + (TOP - 14) / 2 + 0.5, 13, '#cfd5de', '700', 'center');
        // рейки с отверстиями и номера юнитов
        var railW = UH * 0.55, fs = Math.max(8, Math.round(UH * 0.28));
        for (var u = 1; u <= DATA.uHeight; u++) {
            var y = yOf(u, 1);
            c.fillStyle = (u % 2) ? 'rgba(255,255,255,0.025)' : 'rgba(255,255,255,0.0)';
            c.fillRect(GUT, y, dev, UH);
            [GUT, GUT + dev - railW].forEach(function (rx) {
                c.fillStyle = vg(c, y, UH, [[0, '#7d838d'], [1, '#575c65']]);
                c.fillRect(rx, y, railW, UH);
                c.fillStyle = '#050607';
                for (var k = 0; k < 3; k++) {
                    var hy = y + UH * (k === 0 ? 0.17 : k === 1 ? 0.5 : 0.83);
                    c.fillRect(rx + railW * 0.5 - UH * 0.06, hy - UH * 0.06, UH * 0.12, UH * 0.12);
                }
            });
            c.fillStyle = 'rgba(0,0,0,0.5)'; c.fillRect(GUT, y, dev, 1);
            if (UH >= 24 || u % 5 === 0 || u === 1) {
                txt(c, String(u), GUT - 9, y + UH / 2 + 0.5, fs, '#8a919c', '500', 'right');
                txt(c, String(u), GUT + dev + 9, y + UH / 2 + 0.5, fs, '#8a919c', '500', 'left');
            }
        }
        // цоколь
        fillRR(c, GUT - 7, TOP + DATA.uHeight * UH + 6, dev + 14, BOT - 12, 2, '#101216');
    }

    function yU(u) { return TOP + (DATA.uHeight - u) * UH; } // y по высоте над полом (в U)

    // Стеллаж в виде спереди (по образцу складского металлического стеллажа: синие перфорированные
    // стойки, передние балки, настил); векторный вариант - static/infra2/icons/shelf-rack.svg.
    function drawShelf(c) {
        var W = g.W, H = g.H;
        fillRR(c, 0, 0, W, H, 8, hg(c, 0, W, [[0, '#14161a'], [0.5, '#1c1f25'], [1, '#14161a']]));
        strokeRR(c, 0.5, 0.5, W - 1, H - 1, 8, '#3a3e46', 1);
        if (!EMBED) txt(c, DATA.cabinetName + (DATA.site ? '  ·  ' + DATA.site : '') + '  ·  стеллаж', W / 2, 6 + (TOP - 14) / 2 + 0.5, 13, '#cfd5de', '700', 'center');
        c.fillStyle = vg(c, yU(SH_FEET), UH * SH_FEET + BOT, [[0, '#2a2f37'], [1, '#14161a']]);   // пол
        c.fillRect(6, yU(SH_FEET), W - 12, UH * SH_FEET + BOT - 6);
        SH.units.forEach(function (un, k) {
            var x0 = GUT + k * (SH.unitW + SH_GAP) * UH, uw = SH.unitW * UH, up = SH_UP * UH;
            var yTop = yU(SH_FULL), yBot = yU(SH_FEET);
            c.fillStyle = '#0b0e13'; c.fillRect(x0 + up, yTop + up * 0.2, uw - 2 * up, yBot - yTop - up * 0.2);   // задняя стенка
            c.strokeStyle = 'rgba(70,92,118,0.55)'; c.lineWidth = Math.max(1, UH * 0.1);                          // диагональные связи
            un.levels.forEach(function (lv) {
                c.beginPath(); c.moveTo(x0 + up, yU(lv.deck)); c.lineTo(x0 + uw - up, yU(lv.deck + SH.clear)); c.stroke();
            });
            [x0, x0 + uw - up].forEach(function (ux) {                                                              // стойки
                c.fillStyle = hg(c, ux, up, [[0, '#46607c'], [0.5, '#34495f'], [1, '#222f3f']]);
                c.fillRect(ux, yTop, up, yBot - yTop);
                c.fillStyle = 'rgba(255,255,255,0.18)'; c.fillRect(ux + 1, yTop, 1, yBot - yTop);
                for (var yy = yTop + UH * 0.8; yy < yBot - UH * 0.6; yy += UH * 1.0) fillRR(c, ux + up * 0.36, yy, up * 0.28, UH * 0.46, UH * 0.1, '#0a0d12');
                fillRR(c, ux - up * 0.1, yBot, up * 1.2, UH * SH_FEET, 2, '#161b22');
            });
            un.levels.forEach(function (lv) {
                var bx = x0 + up, bw = uw - 2 * up, yDeck = yU(lv.deck), bh = SH_BEAM * UH;
                lv.items.forEach(function (pl) {
                    var it = pl.it, x = bx + SH_PAD * UH, w = Math.round(RATIO * UH);
                    var h = it.su * UH, y = yU(lv.deck + pl.off + it.su);
                    var di = Object.assign({}, it, { sizeU: it.su });
                    c.save();
                    c.beginPath(); c.rect(x, y, w, h); c.clip();
                    if (it.kind === 'chassis') drawChassis(c, di, x, y, w, h, regions);
                    else if (it.kind === 'storage') drawStorage(c, di, x, y, w, h);
                    else if (it.kind === 'tape') drawTape(c, di, x, y, w, h);
                    else if (it.kind === 'switch') drawNetwork(c, di, x, y, w, h);
                    else drawServer(c, di, x, y, w, h, styleOf(it));
                    c.restore();
                    regions.push({ x: x, y: y, w: w, h: h, it: it, blade: null });
                });
                c.fillStyle = vg(c, yDeck - UH * 0.3, UH * 0.3, [[0, '#59636f'], [1, '#3b434d']]);                   // настил
                c.fillRect(bx, yDeck - UH * 0.3, bw, UH * 0.3);
                c.fillStyle = vg(c, yDeck, bh, [[0, '#4a6482'], [0.35, '#34495f'], [1, '#1d2a38']]);                  // передняя балка
                c.fillRect(bx, yDeck, bw, bh);
                c.fillStyle = 'rgba(255,255,255,0.22)'; c.fillRect(bx, yDeck, bw, 1);
                c.fillStyle = 'rgba(0,0,0,0.45)'; c.fillRect(bx, yDeck + bh - 1, bw, 1);
            });
            var yt2 = yU(SH_FULL - 0.2), bh2 = SH_BEAM * UH * 0.8;                                                // верхняя балка
            c.fillStyle = vg(c, yt2, bh2, [[0, '#4a6482'], [1, '#1d2a38']]);
            c.fillRect(x0 + up, yt2, uw - 2 * up, bh2);
        });
    }

    function drawAll() {
        regions = [];
        var c = bctx;
        c.setTransform(g.dpr, 0, 0, g.dpr, 0, 0);
        c.clearRect(0, 0, g.W, g.H);
        if (SH) {
            drawShelf(c);
            regions.sort(function (a, b) { return (a.blade ? 0 : 1) - (b.blade ? 0 : 1); });
            drawOver();
            return;
        }
        drawCabinet(c);
        DATA.items.forEach(function (it) {
            var x = GUT, w = g.devW, y = yOf(it.startU, it.sizeU), h = it.sizeU * UH;
            c.save();
            c.beginPath(); c.rect(x, y, w, h); c.clip();
            if (it.kind === 'chassis') drawChassis(c, it, x, y, w, h, regions);
            else if (it.kind === 'storage') drawStorage(c, it, x, y, w, h);
            else if (it.kind === 'tape') drawTape(c, it, x, y, w, h);
            else if (it.kind === 'switch') drawNetwork(c, it, x, y, w, h);
            else drawServer(c, it, x, y, w, h, styleOf(it));
            c.restore();
            regions.push({ x: x, y: y, w: w, h: h, it: it, blade: null });
        });
        // hit() берёт первое совпадение - области блейдов должны проверяться раньше общей области корзины
        regions.sort(function (a, b) { return (a.blade ? 0 : 1) - (b.blade ? 0 : 1); });
        drawOver();
    }

    // ---------- подсветка ----------
    function drawOver() {
        octx.setTransform(g.dpr, 0, 0, g.dpr, 0, 0);
        octx.clearRect(0, 0, g.W, g.H);
        var r = pinned ? shown : hovered;
        if (!r) return;
        octx.fillStyle = 'rgba(77,171,247,0.12)';
        rr(octx, r.x, r.y, r.w, r.h, 3); octx.fill();
        octx.save();
        octx.strokeStyle = '#4dabf7'; octx.lineWidth = 2; octx.shadowColor = '#4dabf7'; octx.shadowBlur = 10;
        rr(octx, r.x + 1, r.y + 1, r.w - 2, r.h - 2, 3); octx.stroke();
        octx.restore();
    }

    // ---------- подсказка ----------
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, function (m) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m]; }); }
    function powerText(p) {
        if (p === 1) return '<span class="rv-pw" style="color:' + C_ON + '">&#9679; Включен</span>';
        if (p === 2) return '<span class="rv-pw" style="color:' + C_STBY + '">&#9679; Выключен</span>';
        return '<span class="rv-pw" style="color:#8a919c">&#9679; Не определено</span>';
    }
    function kindText(it) { return { server: 'Сервер', chassis: 'Блейд-корзина', storage: 'СХД', 'switch': it.subtype || 'Сетевое оборудование', tape: 'Ленточная библиотека' }[it.kind] || ''; }

    function tipHtml(r) {
        var it = r.it, b = r.blade;
        var name = b ? b.name : it.name;
        var sub = b ? (b.model || 'Блейд-сервер') : ((it.vendor || '') + ' ' + (it.model || '')).trim();
        var power = b ? b.power : it.power;
        var rows = b ? b.rows : it.rows, links = b ? b.links : it.links;
        var pos;
        if (b) pos = 'Корзина «' + esc(it.name) + '», слот ' + b.slot;
        else if (it.startU > 0) pos = 'U' + it.startU + (it.sizeU > 1 ? '–U' + (it.startU + it.sizeU - 1) : '') + ' (' + it.sizeU + 'U' + (it.sizeGuess ? ', высота предположительно' : '') + ')';
        else pos = 'позиция не указана';
        var h = '<button type="button" class="rv-tip-close" title="Закрыть">&times;</button>';
        h += '<div class="rv-tip-title">' + esc(name) + '</div>';
        h += '<div class="rv-tip-sub">' + esc(kindText(b ? { kind: 'server' } : it)) + (sub ? ' &middot; ' + esc(sub) : '') + '</div>';
        h += '<table><tr><td>Питание</td><td>' + powerText(power) + '</td></tr>';
        h += '<tr><td>Положение</td><td>' + pos + '</td></tr>';
        (rows || []).forEach(function (kv) { h += '<tr><td>' + esc(kv.k) + '</td><td>' + esc(kv.v) + '</td></tr>'; });
        h += '</table>';
        if (links && links.length) {
            h += '<div class="rv-links">';
            links.forEach(function (l) { h += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + ' &rarr;</a>'; });
            h += '</div>';
        }
        return h;
    }

    function placeTip(r) {
        tip.hidden = false;
        var sr = stage.getBoundingClientRect();
        var tw = tip.offsetWidth, th = tip.offsetHeight;
        var left;
        if (window.innerWidth - sr.right >= tw + 12) left = g.W - 6;
        else if (sr.left >= tw + 12) left = -tw + 6;
        else left = Math.max(0, g.W - tw - 8);
        var yc = r.y + r.h / 2 - th / 2;
        var vp = sr.top + yc;
        vp = Math.max(8, Math.min(vp, window.innerHeight - th - 8));
        tip.style.left = left + 'px';
        tip.style.top = (vp - sr.top) + 'px';
    }

    function showTip(r) {
        shown = r;
        tip.innerHTML = tipHtml(r);
        tip.classList.toggle('rv-pinned', pinned);
        placeTip(r);
        drawOver();
    }
    function hideTip() {
        if (pinned) return;
        tip.hidden = true; shown = null;
        drawOver();
    }
    function unpin() {
        pinned = false; tip.classList.remove('rv-pinned');
        tip.hidden = true; shown = null; drawOver();
    }

    function hit(px, py) {
        for (var i = 0; i < regions.length; i++) {
            var r = regions[i];
            if (px >= r.x && px <= r.x + r.w && py >= r.y && py <= r.y + r.h) return r;
        }
        return null;
    }

    base.addEventListener('mousemove', function (e) {
        var rc = base.getBoundingClientRect();
        var r = hit(e.clientX - rc.left, e.clientY - rc.top);
        base.style.cursor = r ? 'pointer' : 'default';
        if (r === hovered) return;
        hovered = r;
        if (!pinned) drawOver();
        if (pinned) return;
        clearTimeout(showTimer); clearTimeout(hideTimer);
        if (r) {
            showTimer = setTimeout(function () { if (hovered === r) showTip(r); }, shown ? 160 : 90);
        } else {
            hideTimer = setTimeout(hideTip, 350);
        }
    });
    base.addEventListener('mouseleave', function () {
        hovered = null;
        if (pinned) return;
        drawOver();
        clearTimeout(showTimer); clearTimeout(hideTimer);
        hideTimer = setTimeout(hideTip, 350);
    });
    base.addEventListener('click', function (e) {
        var rc = base.getBoundingClientRect();
        var r = hit(e.clientX - rc.left, e.clientY - rc.top);
        clearTimeout(showTimer); clearTimeout(hideTimer);
        if (r) { pinned = true; showTip(r); } else { unpin(); }
    });
    tip.addEventListener('mouseenter', function () { clearTimeout(hideTimer); clearTimeout(showTimer); });
    tip.addEventListener('mouseleave', function () { if (!pinned) { clearTimeout(hideTimer); hideTimer = setTimeout(hideTip, 350); } });
    tip.addEventListener('click', function (e) { if (e.target && e.target.className === 'rv-tip-close') unpin(); });
    document.addEventListener('keydown', function (e) { if (e.key === 'Escape') unpin(); });
    document.addEventListener('click', function (e) { if (pinned && !stage.contains(e.target)) unpin(); });

    // ---------- боковая панель ----------
    function renderSide() {
        var items = DATA.items, on = 0, off = 0, unk = 0, usedU = 0, bl = 0, blOn = 0;
        function cnt(p) { if (p === 1) on++; else if (p === 2) off++; else unk++; }
        items.forEach(function (it) {
            usedU += it.sizeU;
            if (it.kind === 'chassis') { (it.blades || []).forEach(function (b) { bl++; if (b.power === 1) blOn++; }); }
            cnt(it.power);
        });
        if (SH) {
            var ss = '<div>Оборудование на стеллаже (площадка без шкафов): <b>' + items.length + '</b></div>';
            ss += '<div><span class="rv-dot rv-dot-on"></span>включено: <b>' + on + '</b> &nbsp; <span class="rv-dot rv-dot-stby"></span>выключено: <b>' + off + '</b> &nbsp; <span class="rv-dot rv-dot-unk"></span>н/д: <b>' + unk + '</b></div>';
            if (bl) ss += '<div>Блейдов в корзинах: <b>' + bl + '</b> (включено ' + blOn + ')</div>';
            ss += '<div class="text-muted">Высота устройства - из справочника, если не указана - сервер 2U.</div>';
            document.getElementById('rvSummary').innerHTML = ss;
            return;
        }
        var s = '<div>Занято юнитов: <b>' + usedU + '</b> из ' + DATA.uHeight + ' (' + Math.round(usedU * 100 / Math.max(1, DATA.uHeight)) + '%)</div>';
        s += '<div>Устройств на позициях: <b>' + items.length + '</b></div>';
        s += '<div><span class="rv-dot rv-dot-on"></span>включено: <b>' + on + '</b> &nbsp; <span class="rv-dot rv-dot-stby"></span>выключено: <b>' + off + '</b> &nbsp; <span class="rv-dot rv-dot-unk"></span>н/д: <b>' + unk + '</b></div>';
        s += '<div class="mt-1">Потребляемая мощность шкафа: ' + (DATA.powerAvgW > 0 ? '<b>' + Math.round(DATA.powerAvgW) + ' Вт</b> (средняя 24ч)' : '<b>нет данных</b>') + '</div><div class="text-muted">Корзины учтены целиком, блейды отдельно не суммируются.</div>';
        if (DATA.powerPeakW > 0) s += '<div>Пик за 24ч: <b>' + Math.round(DATA.powerPeakW) + ' Вт</b></div>';
        if (DATA.powerRatedW > 0) s += '<div>Паспортная максимальная: <b>' + Math.round(DATA.powerRatedW) + ' Вт</b></div>';
        s += '<div class="mt-1">Мощность шкафа (расчётный график): <a href="/sprsrv/powerrecent/rack/' + DATA.cabinetId + '" target="_blank" rel="noopener">за 3 суток</a> &middot; <a href="/sprsrv/powerview/rack/' + DATA.cabinetId + '" target="_blank" rel="noopener">за период</a></div>';
        if (bl) s += '<div>Блейдов в корзинах: <b>' + bl + '</b> (включено ' + blOn + ')</div>';
        document.getElementById('rvSummary').innerHTML = s;

        if (DATA.unplaced && DATA.unplaced.length) {
            document.getElementById('rvUnplacedCard').hidden = false;
            document.getElementById('rvUnplacedCount').textContent = ' (' + DATA.unplaced.length + ')';
            var h = '';
            DATA.unplaced.forEach(function (it) {
                h += '<div class="rv-unpl-item"><div><b>' + esc(it.name) + '</b></div><div class="text-muted">' + esc(kindText(it)) + ((it.vendor || it.model) ? ' &middot; ' + esc((it.vendor + ' ' + it.model).trim()) : '') + '</div>';
                h += '<div>' + powerText(it.power) + '</div><div>';
                (it.links || []).forEach(function (l) { h += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + '</a>'; });
                h += '</div></div>';
            });
            document.getElementById('rvUnplaced').innerHTML = h;
        }
    }

    // ---------- масштаб ----------
    function zoomLabel() {
        var pct = Math.round(UH * 100 / (EMBED ? fitUH : 40));
        document.getElementById('rvZoomReset').textContent = pct + '%';
        // во фрейме просмотра ЦОД индикатор масштаба показывает родительская страница
        if (window.parent !== window) { try { window.parent.postMessage({ type: 'rvZoomState', pct: pct }, location.origin); } catch (e) { } }
    }
    window.addEventListener('message', function (e) {
        if (e.origin !== location.origin) return;
        var m = e.data || {};
        if (m.type === 'dvTheme' && (m.theme === 'light' || m.theme === 'dark')) { document.documentElement.setAttribute('data-bs-theme', m.theme); return; }   // тема страницы ЦОД
        if (m.type !== 'rvZoom') return;
        if (m.cmd === 'in') { userZoomed = true; setZoom(UH + 4); } else if (m.cmd === 'out') { userZoomed = true; setZoom(UH - 4); } else if (m.cmd === 'reset') { userZoomed = false; if (EMBED) fitUH = calcFit(); setZoom(EMBED ? fitUH : 40); }
    });
    if (EMBED) {
        window.addEventListener('resize', function () {
            if (userZoomed) return;
            var f = calcFit();
            if (f !== UH) { fitUH = f; setZoom(f); }
        });
    }
    function setZoom(v) {
        UH = Math.max(SH ? 4 : (EMBED ? 8 : 20), Math.min(90, v));
        zoomLabel();
        if (!EMBED) { try { localStorage.setItem(ZKEY, String(UH)); } catch (e) { } }
        unpin(); hovered = null;
        layout(); drawAll();
    }
    document.getElementById('rvZoomIn').addEventListener('click', function () { setZoom(UH + 4); });
    document.getElementById('rvZoomOut').addEventListener('click', function () { setZoom(UH - 4); });
    document.getElementById('rvZoomReset').addEventListener('click', function () { setZoom(40); });

    renderSide();
    zoomLabel();
    layout();
    drawAll();

    // Нет оборудования на позициях, но есть оборудование без позиции: пустой шкаф
    // не рисуем - иначе список "без позиции" уходит под высокий пустой шкаф и не виден.
    if (!DATA.items.length && DATA.unplaced && DATA.unplaced.length) {
        document.querySelector('.rv-main').style.display = 'none';
        var side = document.querySelector('.rv-side');
        side.style.flex = '1 1 100%';
        var note = document.createElement('div');
        note.className = 'text-muted small mb-1';
        note.textContent = 'В шкафу нет оборудования с указанной позицией (юнитом) - показано оборудование без позиции.';
        document.getElementById('rvUnplacedCard').insertBefore(note, document.getElementById('rvUnplaced'));
    }
})();
</script>
