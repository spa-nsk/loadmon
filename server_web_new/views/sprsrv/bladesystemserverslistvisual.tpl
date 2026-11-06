<style>
    .bcv-layout { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
    .bcv-side { flex: 0 0 340px; max-width: 100%; display: flex; flex-direction: column; gap: 0.75rem; }
    .bcv-main { flex: 1 1 420px; min-width: 0; }
    .bcv-card {
        background: var(--bs-tertiary-bg); border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem; padding: 0.85rem 1rem;
    }
    .bcv-card h6 { color: var(--bs-emphasis-color); font-weight: 600; margin-bottom: 0.5rem; }
    .bcv-legend { list-style: none; padding: 0; margin: 0 0 0.5rem 0; }
    .bcv-legend li { margin-bottom: 0.25rem; }
    .bcv-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 0.35rem; vertical-align: middle; }
    .bcv-dot-on { background: #35e06a; box-shadow: 0 0 6px #35e06a; }
    .bcv-dot-stby { background: #ffab1a; box-shadow: 0 0 6px #ffab1a; }
    .bcv-dot-unk { background: #5b616b; }
    .bcv-stage { position: relative; display: inline-block; line-height: 0; max-width: 100%; }
    .bcv-stage canvas { display: block; max-width: 100%; height: auto; }
    .bcv-detail-title { font-weight: 700; font-size: 1rem; color: var(--bs-emphasis-color); word-break: break-word; }
    .bcv-detail-sub { color: var(--bs-secondary-color); margin-bottom: 0.45rem; font-size: 0.9rem; }
    .bcv-detail table { width: 100%; margin: 0.35rem 0; font-size: 0.875rem; }
    .bcv-detail td { padding: 2px 0; vertical-align: top; }
    .bcv-detail td:first-child { color: var(--bs-secondary-color); white-space: nowrap; padding-right: 0.65rem; }
    .bcv-detail .bcv-pw { font-weight: 600; }
    .bcv-detail .bcv-links { display: flex; flex-direction: column; gap: 0.2rem; margin-top: 0.5rem; border-top: 1px solid var(--bs-border-color); padding-top: 0.5rem; }
    .bcv-empty-hint { color: var(--bs-secondary-color); font-size: 0.9rem; }
</style>

<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Отображение блэйд корзины с серверами</h2>
        {{if .BS}}
        <div class="text-muted">
            {{.BS.Name}} &middot; {{.BS.Vendor}} {{.BS.Model}} &middot; слотов {{.BS.Size}}, занято {{.ServersCount}}
        </div>
        {{end}}
    </div>
    <div class="d-flex gap-2">
        {{if .BS}}
        <a href="/sprsrv/bladesystemserverslist/{{.BS.Id}}" class="btn btn-outline-secondary rounded-pill px-3">Список серверов</a>
        <a href="/sprsrv/bladesystemsedit-tab/{{.BS.Id}}" class="btn btn-outline-secondary rounded-pill px-3" target="_blank">Корзина</a>
        {{end}}
        <a href="/sprsrv/bladesystemslist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку корзин</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

<div class="bcv-layout" id="bcvLayout">
    <aside class="bcv-side">
        <div class="bcv-card">
            <h6>Выделенный блейд</h6>
            <div id="bcvDetail" class="bcv-detail">
                <div class="bcv-empty-hint">Кликните по слоту блейда на рисунке корзины — справа появятся его параметры. Клик по корпусу корзины показывает данные корзины.</div>
            </div>
        </div>
        <div class="bcv-card">
            <h6>Состояние питания</h6>
            <ul class="bcv-legend small">
                <li><span class="bcv-dot bcv-dot-on"></span> Включено</li>
                <li><span class="bcv-dot bcv-dot-stby"></span> Выключено</li>
                <li><span class="bcv-dot bcv-dot-unk"></span> Не определено</li>
            </ul>
            <div class="text-muted small">Отрисовка как в просмотре оборудования шкафа ЦОД.</div>
        </div>
    </aside>
    <div class="bcv-main" id="bcvMain">
        <div class="bcv-stage" id="bcvStage">
            <canvas id="bcvBase"></canvas>
            <canvas id="bcvOver" style="position:absolute;left:0;top:0;pointer-events:none;"></canvas>
        </div>
    </div>
</div>

<script>
(function () {
    'use strict';

    var DATA = {{.ChassisJSON}};
    var base = document.getElementById('bcvBase');
    var over = document.getElementById('bcvOver');
    var detail = document.getElementById('bcvDetail');
    if (!DATA || !DATA.chassis || !base || !over) return;

    var it = DATA.chassis;
    var bctx = base.getContext('2d');
    var octx = over.getContext('2d');
    var TAU = Math.PI * 2;
    var C_ON = '#35e06a', C_STBY = '#ffab1a', C_UID = '#3da5ff', C_ERR = '#ff453a';
    var PAD = 24;
    var UH = 36; // высота 1U в пикселях — корзина крупнее, чем в шкафу
    var sizeU = Math.max(6, it.sizeU || 10);
    var regions = [];
    var hovered = null, selected = null;
    var ledPulseT = 0;

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
    function fillRR(c, x, y, w, h, r, fill) { rr(c, x, y, w, h, r); c.fillStyle = fill; c.fill(); }
    function strokeRR(c, x, y, w, h, r, stroke, lw) { rr(c, x, y, w, h, r); c.strokeStyle = stroke; c.lineWidth = lw || 1; c.stroke(); }
    function ledIsLiveColor(color) {
        return !!(color && color !== C_STBY && color !== C_ERR && color !== C_UID && color !== '#000' && color !== '#1e2126');
    }
    function ledBright(seed) {
        var t = ledPulseT, s = (seed || 0) * 12.9898;
        var a = 0.62 + 0.38 * (0.5 + 0.5 * Math.sin(t * 2.15 + s));
        var b = 0.78 + 0.22 * (0.5 + 0.5 * Math.sin(t * 5.4 + s * 1.73));
        var dip = 1 - 0.12 * Math.max(0, Math.sin(t * 15.5 + s * 2.9));
        return Math.min(1, Math.max(0.48, a * b * dip));
    }
    function led(c, cx, cy, r, color, on) {
        if (on) {
            var live = ledIsLiveColor(color), br = live ? ledBright(cx * 0.13 + cy * 0.71) : 1;
            c.save();
            if (live) c.globalAlpha = br;
            c.shadowColor = color; c.shadowBlur = r * (2.2 + (live ? br * 1.1 : 0.4));
            c.fillStyle = color; c.beginPath(); c.arc(cx, cy, r, 0, TAU); c.fill();
            c.restore();
            c.save();
            if (live) c.globalAlpha = 0.4 + 0.45 * br;
            c.fillStyle = 'rgba(255,255,255,0.65)';
            c.beginPath(); c.arc(cx - r * 0.28, cy - r * 0.3, r * 0.34, 0, TAU); c.fill();
            c.restore();
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
    function pwrColor(p) { return p === 1 ? C_ON : (p === 2 ? C_STBY : null); }
    function powerButton(c, cx, cy, r, p) {
        var col = pwrColor(p);
        c.lineWidth = Math.max(1.4, r * 0.34);
        if (col) {
            var live = p === 1, br = live ? ledBright(cx * 0.19 + cy * 0.53) : 1;
            c.save();
            if (live) c.globalAlpha = 0.55 + 0.45 * br;
            c.shadowColor = col; c.shadowBlur = r * (live ? (1.4 + br * 1.2) : 1.8);
            c.strokeStyle = col; c.beginPath(); c.arc(cx, cy, r * 1.22, 0, TAU); c.stroke();
            c.restore();
        } else { c.strokeStyle = '#2b2f35'; c.beginPath(); c.arc(cx, cy, r * 1.22, 0, TAU); c.stroke(); }
        var gr = c.createRadialGradient(cx - r * 0.3, cy - r * 0.35, r * 0.1, cx, cy, r);
        gr.addColorStop(0, '#f1f3f5'); gr.addColorStop(1, '#868c95');
        c.fillStyle = gr; c.beginPath(); c.arc(cx, cy, r, 0, TAU); c.fill();
        c.strokeStyle = '#2a2d32'; c.lineWidth = Math.max(1, r * 0.22); c.lineCap = 'round';
        c.beginPath(); c.arc(cx, cy + r * 0.05, r * 0.5, -Math.PI * 0.35, Math.PI * 1.35, false); c.stroke();
        c.beginPath(); c.moveTo(cx, cy - r * 0.62); c.lineTo(cx, cy - r * 0.05); c.stroke();
        c.lineCap = 'butt';
    }
    function earSlots(c, x, y, ew, n) {
        for (var k = 0; k < n; k++) {
            var cy = y + (k + 0.5) * UH;
            c.fillStyle = 'rgba(0,0,0,0.7)';
            c.beginPath(); c.ellipse(x + ew / 2, cy, ew * 0.2, UH * 0.13, 0, 0, TAU); c.fill();
            c.strokeStyle = 'rgba(255,255,255,0.12)'; c.lineWidth = 0.6; c.stroke();
        }
    }

    // ---------- блейд / корзина (как rackcabinetview.tpl) ----------
    function drawBlade(c, bx, by, bw, bh, b, syn, slotNum) {
        if (!b) {
            fillRR(c, bx, by, bw, bh, 1.5, vg(c, by, bh, [[0, '#2a2c31'], [1, '#17181b']]));
            c.fillStyle = 'rgba(0,0,0,0.55)';
            for (var i = 1; i <= 3; i++) c.fillRect(bx + bw * i / 4 - 0.5, by + bh * 0.1, 1.2, bh * 0.8);
            txt(c, String(slotNum), bx + bw / 2, by + bh - UH * 0.16, Math.max(7, UH * 0.2), 'rgba(255,255,255,0.22)', '500', 'center');
            return;
        }
        var p = b.power;
        fillRR(c, bx, by, bw, bh, 1.5, vg(c, by, bh, [[0, syn ? '#2d3138' : '#34373c'], [1, '#17191c']]));
        strokeRR(c, bx + 0.5, by + 0.5, bw - 1, bh - 1, 1.5, 'rgba(0,0,0,0.85)', 1);
        if (syn) { c.fillStyle = '#00b5e2'; c.fillRect(bx + 1, by + 1, bw - 2, Math.max(1, bh * 0.03)); }
        fillRR(c, bx + bw * 0.08, by + bh * 0.84, bw * 0.84, bh * 0.1, 1, '#9096a0');
        fillRR(c, bx + bw * 0.14, by + bh * 0.14, bw * 0.72, bh * 0.17, 1, '#0e0f11');
        fillRR(c, bx + bw * 0.14, by + bh * 0.36, bw * 0.72, bh * 0.17, 1, '#0e0f11');
        led(c, bx + bw * 0.24, by + bh * 0.225, Math.max(0.9, bw * 0.03), '#2fd36a', p === 1);
        led(c, bx + bw * 0.24, by + bh * 0.445, Math.max(0.9, bw * 0.03), '#2fd36a', p === 1);
        var r = Math.max(2.2, bw * 0.13);
        powerButton(c, bx + bw / 2, by + bh * 0.68, r, p);
        led(c, bx + bw * 0.2, by + bh * 0.68, Math.max(1, bw * 0.035), C_UID, false);
        txt(c, String(slotNum), bx + bw / 2, by + bh * 0.08, Math.max(8, UH * 0.22), 'rgba(235,238,242,0.55)', '600', 'center');
    }

    function drawChassis(c, chassis, x, y, w, h, regs) {
        var syn = /synergy|sy12|12000/i.test((chassis.vendor || '') + ' ' + (chassis.model || '') + ' ' + (chassis.name || ''));
        var p = chassis.power, on = (p === 1);
        var slots = chassis.slots > 0 ? chassis.slots : (syn ? 12 : 16);
        var cols = Math.ceil(slots / 2), rows = 2;
        var bladeList = chassis.blades || [];
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, syn ? '#2a2d34' : '#3a3d42'], [1, '#17181b']]));
        c.fillStyle = 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        if (syn) { c.fillStyle = '#00b5e2'; c.fillRect(x + 3, y + 2, w - 6, Math.max(1, UH * 0.04)); }
        var ew = w * 0.03;
        var earFill = vg(c, y, h, [[0, '#4a4d53'], [1, '#202226']]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill);
        fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        earSlots(c, x, y, ew, Math.max(1, Math.round(h / UH)));
        earSlots(c, x + w - ew, y, ew, Math.max(1, Math.round(h / UH)));

        var ax = x + ew + w * 0.006, aw = w - 2 * ew - w * 0.012;
        var ay = y + h * 0.045, ah = h * 0.66;
        var sw = aw / cols, sh = ah / rows;
        var bySlot = {};
        for (var i = 0; i < bladeList.length; i++) bySlot[bladeList[i].slot] = bladeList[i];
        for (var s = 1; s <= slots; s++) {
            var col = (s - 1) % cols, row = Math.floor((s - 1) / cols);
            var bx = ax + col * sw + 1, by = ay + row * sh + 1, bw = sw - 2, bh = sh - 2;
            var b = bySlot[s];
            drawBlade(c, bx, by, bw, bh, b, syn, s);
            if (b) regs.push({ x: bx, y: by, w: bw, h: bh, blade: b, chassis: chassis });
            else regs.push({ x: bx, y: by, w: bw, h: bh, blade: null, slot: s, chassis: chassis, empty: true });
        }

        var py = y + h * 0.745, ph = h * 0.21;
        var psuW = aw * 0.125, gapx = aw * 0.012;
        var dispW = aw * 0.16;
        var total = psuW * 6 + dispW + gapx * 6;
        var px = ax + (aw - total) / 2;
        var title = chassis.name || chassis.model || chassis.vendor || '';
        for (var k = 0; k < 7; k++) {
            if (k === 3) {
                fillRR(c, px, py, dispW, ph, 2, '#0b0c0e');
                fillRR(c, px + dispW * 0.1, py + ph * 0.18, dispW * 0.8, ph * 0.5, 1.5, on ? '#0c3d57' : '#10151a');
                if (title) txt(c, title, px + dispW / 2, py + ph * 0.43, Math.max(6, Math.min(UH * 0.22, dispW * 0.24)), on ? '#8fd8ff' : 'rgba(200,210,220,0.75)', '600', 'center');
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
        var sub = chassis.model || chassis.vendor || '';
        if (sub) txt(c, sub, x + w * 0.5, y + h * 0.985 - UH * 0.04, Math.max(6, UH * 0.2), 'rgba(235,238,242,0.5)', '500', 'center');
        if (bladeList.length === 0) {
            txt(c, 'слотов: ' + slots + ' (пусто)', x + w * 0.5, ay + ah * 0.5, Math.max(8, UH * 0.24), 'rgba(235,238,242,0.35)', '500', 'center');
        }
        // область клика по корпусу корзины (после блейдов в hit-тесте — ниже приоритет)
        regs.push({ x: x, y: y, w: w, h: h, blade: null, chassis: chassis, body: true });
    }

    function layout() {
        var avail = (document.getElementById('bcvMain') || base.parentElement).clientWidth || 640;
        // ширина 19" панели ≈ UH * 10.86 (как в rackcabinetview); подгоняем UH под доступную ширину
        var maxW = Math.max(360, Math.min(720, avail - PAD * 2));
        UH = Math.max(28, Math.min(48, Math.floor(maxW / 10.86)));
        var w = Math.round(UH * 10.86);
        var h = sizeU * UH;
        var dpr = window.devicePixelRatio || 1;
        var W = w + PAD * 2, H = h + PAD * 2;
        base.width = Math.round(W * dpr); base.height = Math.round(H * dpr);
        over.width = base.width; over.height = base.height;
        base.style.width = W + 'px'; base.style.height = H + 'px';
        over.style.width = W + 'px'; over.style.height = H + 'px';
        bctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        octx.setTransform(dpr, 0, 0, dpr, 0, 0);
        return { W: W, H: H, x: PAD, y: PAD, w: w, h: h, dpr: dpr };
    }

    var g = layout();

    function paint() {
        regions = [];
        bctx.clearRect(0, 0, g.W, g.H);
        bctx.fillStyle = '#0e1014';
        fillRR(bctx, 0, 0, g.W, g.H, 8, '#0e1014');
        drawChassis(bctx, it, g.x, g.y, g.w, g.h, regions);
        // блейды выше body в hit-тесте
        regions.sort(function (a, b) {
            var pa = a.body ? 2 : (a.empty ? 1 : 0);
            var pb = b.body ? 2 : (b.empty ? 1 : 0);
            return pa - pb;
        });
        drawOver();
    }

    function drawOver() {
        octx.clearRect(0, 0, g.W, g.H);
        var r = selected || hovered;
        if (!r || r.body) return;
        octx.fillStyle = 'rgba(77,171,247,0.14)';
        rr(octx, r.x, r.y, r.w, r.h, 3); octx.fill();
        octx.save();
        octx.strokeStyle = '#4dabf7'; octx.lineWidth = 2; octx.shadowColor = '#4dabf7'; octx.shadowBlur = 10;
        rr(octx, r.x + 1, r.y + 1, r.w - 2, r.h - 2, 3); octx.stroke();
        octx.restore();
    }

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (m) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m];
        });
    }
    function powerText(p) {
        if (p === 1) return '<span class="bcv-pw" style="color:' + C_ON + '">&#9679; Включен</span>';
        if (p === 2) return '<span class="bcv-pw" style="color:' + C_STBY + '">&#9679; Выключен</span>';
        return '<span class="bcv-pw" style="color:#8a919c">&#9679; Не определено</span>';
    }

    function renderDetail(r) {
        if (!r) {
            detail.innerHTML = '<div class="bcv-empty-hint">Кликните по слоту блейда на рисунке корзины — справа появятся его параметры. Клик по корпусу корзины показывает данные корзины.</div>';
            return;
        }
        var b = r.blade, ch = r.chassis || it;
        if (r.empty) {
            detail.innerHTML = '<div class="bcv-detail-title">Слот ' + esc(r.slot) + '</div>' +
                '<div class="bcv-detail-sub">Пустой слот корзины «' + esc(ch.name) + '»</div>';
            return;
        }
        var name = b ? b.name : ch.name;
        var sub = b ? (b.model || 'Блейд-сервер') : ((ch.vendor || '') + ' ' + (ch.model || '')).trim();
        var power = b ? b.power : ch.power;
        var rows = b ? b.rows : ch.rows;
        var links = b ? b.links : ch.links;
        var kind = b ? 'Блейд-сервер' : 'Блейд-корзина';
        var pos = b ? ('Корзина «' + esc(ch.name) + '», слот ' + b.slot) : ('высота ' + (ch.sizeU || sizeU) + 'U');
        var h = '<div class="bcv-detail-title">' + esc(name) + '</div>';
        h += '<div class="bcv-detail-sub">' + esc(kind) + (sub ? ' &middot; ' + esc(sub) : '') + '</div>';
        h += '<table><tr><td>Питание</td><td>' + powerText(power) + '</td></tr>';
        h += '<tr><td>Положение</td><td>' + pos + '</td></tr>';
        (rows || []).forEach(function (kv) {
            h += '<tr><td>' + esc(kv.k) + '</td><td>' + esc(kv.v) + '</td></tr>';
        });
        h += '</table>';
        if (links && links.length) {
            h += '<div class="bcv-links">';
            links.forEach(function (l) {
                h += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + ' &rarr;</a>';
            });
            h += '</div>';
        }
        detail.innerHTML = h;
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
        var scaleX = g.W / rc.width, scaleY = g.H / rc.height;
        var r = hit((e.clientX - rc.left) * scaleX, (e.clientY - rc.top) * scaleY);
        base.style.cursor = r ? 'pointer' : 'default';
        if (r === hovered) return;
        hovered = r;
        drawOver();
    });
    base.addEventListener('mouseleave', function () {
        hovered = null;
        drawOver();
    });
    base.addEventListener('click', function (e) {
        var rc = base.getBoundingClientRect();
        var scaleX = g.W / rc.width, scaleY = g.H / rc.height;
        var r = hit((e.clientX - rc.left) * scaleX, (e.clientY - rc.top) * scaleY);
        selected = r;
        renderDetail(r);
        drawOver();
    });

    window.addEventListener('resize', function () {
        g = layout();
        paint();
        if (selected) renderDetail(selected);
    });

    // первичная отрисовка; по умолчанию — параметры корзины
    paint();
    selected = { chassis: it, blade: null, body: true };
    renderDetail(selected);

    // лёгкое мерцание светодиодов включённого оборудования (~18 fps), как в rackcabinetview
    var ledLast = 0;
    function ledTick(now) {
        requestAnimationFrame(ledTick);
        if (now - ledLast < 55) return;
        ledLast = now;
        ledPulseT = now * 0.001;
        paint();
    }
    requestAnimationFrame(ledTick);
})();
</script>
