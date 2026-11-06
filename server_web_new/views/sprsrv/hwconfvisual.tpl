<style>
    .hcv-layout { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
    .hcv-side { flex: 0 0 360px; max-width: 100%; display: flex; flex-direction: column; gap: 0.75rem; }
    .hcv-main { flex: 1 1 480px; min-width: 0; }
    .hcv-card {
        background: var(--bs-tertiary-bg); border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem; padding: 0.85rem 1rem;
    }
    .hcv-card h6 { color: var(--bs-emphasis-color); font-weight: 600; margin-bottom: 0.5rem; }
    .hcv-legend { list-style: none; padding: 0; margin: 0 0 0.5rem 0; }
    .hcv-legend li { margin-bottom: 0.25rem; }
    .hcv-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 0.35rem; vertical-align: middle; }
    .hcv-dot-on { background: #35e06a; box-shadow: 0 0 6px #35e06a; }
    .hcv-dot-stby { background: #ffab1a; box-shadow: 0 0 6px #ffab1a; }
    .hcv-dot-unk { background: #5b616b; }
    .hcv-stage { position: relative; display: inline-block; line-height: 0; max-width: 100%; }
    .hcv-stage canvas { display: block; max-width: 100%; height: auto; }
    .hcv-detail-title { font-weight: 700; font-size: 1rem; color: var(--bs-emphasis-color); word-break: break-word; }
    .hcv-detail-sub { color: var(--bs-secondary-color); margin-bottom: 0.45rem; font-size: 0.9rem; }
    .hcv-detail table { width: 100%; margin: 0.35rem 0; font-size: 0.875rem; }
    .hcv-detail td { padding: 2px 0; vertical-align: top; }
    .hcv-detail td:first-child { color: var(--bs-secondary-color); white-space: nowrap; padding-right: 0.65rem; }
    .hcv-detail .hcv-pw { font-weight: 600; }
    .hcv-detail .hcv-links { display: flex; flex-direction: column; gap: 0.2rem; margin-top: 0.5rem; border-top: 1px solid var(--bs-border-color); padding-top: 0.5rem; }
    .hcv-empty-hint { color: var(--bs-secondary-color); font-size: 0.9rem; }
</style>

<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Отображение сервера по модели</h2>
        {{if .HW}}
        <div class="text-muted">
            {{if .ServerName}}{{.ServerName}} &middot; {{end}}{{.HW.Vendor}} {{.HW.Model}}
            {{if .HW.SerialNum}} &middot; S/N {{.HW.SerialNum}}{{end}}
            &middot; дисков: {{.DisksCount}}
        </div>
        {{end}}
    </div>
    <div class="d-flex gap-2">
        {{if .HW}}
        <a href="/sprsrv/hwconfedit-tab/{{.HW.Id}}" class="btn btn-outline-secondary rounded-pill px-3" target="_blank">Конфигурация</a>
        {{end}}
        <a href="/sprsrv/hwconflist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

<div class="hcv-layout" id="hcvLayout">
    <aside class="hcv-side">
        <div class="hcv-card">
            <h6>Выделенный элемент</h6>
            <div id="hcvDetail" class="hcv-detail">
                <div class="hcv-empty-hint">Наведите или кликните на диск в корзине — справа появятся его параметры. Клик по корпусу показывает данные сервера.</div>
            </div>
        </div>
        <div class="hcv-card">
            <h6>Состояние питания</h6>
            <ul class="hcv-legend small">
                <li><span class="hcv-dot hcv-dot-on"></span> Включено</li>
                <li><span class="hcv-dot hcv-dot-stby"></span> Выключено</li>
                <li><span class="hcv-dot hcv-dot-unk"></span> Не определено</li>
            </ul>
            <div class="text-muted small">Отрисовка лицевой панели по вендору/модели, как в просмотре шкафа ЦОД. Диски — из компонент конфигурации (Type=2).</div>
        </div>
    </aside>
    <div class="hcv-main" id="hcvMain">
        <div class="hcv-stage" id="hcvStage">
            <canvas id="hcvBase"></canvas>
            <canvas id="hcvOver" style="position:absolute;left:0;top:0;pointer-events:none;"></canvas>
        </div>
    </div>
</div>

<script>
(function () {
    'use strict';

    var DATA = {{.VisualJSON}};
    var base = document.getElementById('hcvBase');
    var over = document.getElementById('hcvOver');
    var detail = document.getElementById('hcvDetail');
    if (!DATA || !DATA.server || !base || !over) return;

    var srv = DATA.server;
    var disks = DATA.disks || [];
    var bctx = base.getContext('2d');
    var octx = over.getContext('2d');
    var TAU = Math.PI * 2;
    var C_ON = '#35e06a', C_STBY = '#ffab1a', C_UID = '#3da5ff', C_ERR = '#ff453a';
    var PAD = 28, RATIO = 10.86;
    var UH = 44;
    var sizeU = Math.max(1, srv.sizeU || 2);
    var regions = [];
    var hovered = null, selected = null;
    var ledPulseT = 0;

    var THEMES = {
        hpe: { bezel: ['#383b41', '#1a1c20'], ear: ['#2c2f34', '#16181b'], cage: '#0e0f11', logo: 'hpe', cols1: 4, rows1: 2, bayLed: '#3de08a' },
        dell: { bezel: ['#2e3034', '#16171a'], ear: ['#cfd3d9', '#8a9099'], cage: '#0c0d0f', logo: 'dell', cols1: 5, rows1: 2, bayLed: '#3da5ff' },
        lenovo: { bezel: ['#222428', '#0f1012'], ear: ['#2b2d31', '#141518'], cage: '#08080a', logo: 'lenovo', cols1: 5, rows1: 2, bayLed: '#ff6a3d', accent: '#e2231a' },
        depo: { bezel: ['#2a2d33', '#121418'], ear: ['#3a3e46', '#1c1e22'], cage: '#0b0c0e', logo: 'depo', cols1: 4, rows1: 2, bayLed: '#6b67e9', accent: '#6b67e9' },
        generic: { bezel: ['#50545b', '#2d3035'], ear: ['#a3a9b1', '#6c7179'], cage: '#15161a', logo: '', cols1: 4, rows1: 2, bayLed: '#3de08a' }
    };

    function styleOf(it) {
        var s = ((it.vendor || '') + ' ' + (it.model || '') + ' ' + (it.name || '')).toLowerCase();
        if (/hpe|hp |hewlett|proliant|\bdl\d|\bml\d|\bbl\d/.test(s)) return THEMES.hpe;
        if (/dell|poweredge|\br\d{3}/.test(s)) return THEMES.dell;
        if (/lenovo|thinksystem|\bsr\d{3}/.test(s)) return THEMES.lenovo;
        if (/\bdepo\b|\bstorm\s*[345]\d{3}/.test(s)) return THEMES.depo;
        return THEMES.generic;
    }

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
    function hash(n) { n = (Math.imul(n + 1, 2654435761) >>> 0); return (n % 1000) / 1000; }
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
    function earSlots(c, x, y, ew, n) {
        for (var k = 0; k < n; k++) {
            var cy = y + (k + 0.5) * UH;
            c.fillStyle = 'rgba(0,0,0,0.7)';
            c.beginPath(); c.ellipse(x + ew / 2, cy, ew * 0.2, UH * 0.13, 0, 0, TAU); c.fill();
            c.strokeStyle = 'rgba(255,255,255,0.12)'; c.lineWidth = 0.6; c.stroke();
        }
    }
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
        } else if (kind === 'depo') {
            var r2 = h * 0.42, cx = x + r2, cy = y + h / 2;
            var segs = ['#85cfff', '#6b67e9', '#7ac943', '#f5c518'];
            for (var i = 0; i < 4; i++) {
                c.beginPath(); c.moveTo(cx, cy);
                c.arc(cx, cy, r2, -Math.PI / 2 + i * Math.PI / 2, -Math.PI / 2 + (i + 1) * Math.PI / 2);
                c.closePath(); c.fillStyle = segs[i]; c.fill();
            }
            c.beginPath(); c.arc(cx, cy, r2 * 0.38, 0, TAU); c.fillStyle = '#121418'; c.fill();
            txt(c, 'DEPO', x + r2 * 2.15, y + h / 2, fs * 0.95, '#f2f4f8', '800', 'left');
        }
    }

    function driveFlat(c, x, y, w, h, on, idx, th, occupied) {
        fillRR(c, x, y, w, h, 1.5, vg(c, y, h, [[0, occupied ? '#4a4e55' : '#2a2c31'], [1, occupied ? '#202327' : '#17181b']]));
        strokeRR(c, x + 0.5, y + 0.5, w - 1, h - 1, 1.5, 'rgba(0,0,0,0.8)', 1);
        if (!occupied) {
            c.fillStyle = 'rgba(0,0,0,0.45)';
            for (var i = 1; i <= 2; i++) c.fillRect(x + w * 0.15, y + h * (0.25 + i * 0.2), w * 0.7, Math.max(1, h * 0.06));
            return;
        }
        fillRR(c, x + w * 0.08, y + h * 0.16, w * 0.84, h * 0.2, 1, '#14161a');
        c.fillStyle = 'rgba(0,0,0,0.55)';
        for (var j = 0; j < 5; j++) c.fillRect(x + w * 0.1 + j * (w * 0.8 / 5), y + h * 0.5, Math.max(1, w * 0.06), h * 0.28);
        led(c, x + w * 0.14, y + h * 0.86, Math.max(1.1, h * 0.07), on ? (hash(idx) > 0.45 ? '#ffffff' : '#2fd36a') : '#000', on);
        led(c, x + w * 0.3, y + h * 0.86, Math.max(1.1, h * 0.07), th.bayLed || C_UID, false);
    }
    function driveVert(c, x, y, w, h, on, idx, th, occupied) {
        fillRR(c, x, y, w, h, 1, hg(c, x, w, [[0, occupied ? '#2a2d32' : '#1a1b1e'], [0.5, occupied ? '#4c5057' : '#2a2c30'], [1, '#222428']]));
        strokeRR(c, x + 0.5, y + 0.5, w - 1, h - 1, 1, 'rgba(0,0,0,0.85)', 1);
        if (!occupied) return;
        fillRR(c, x + w * 0.18, y + h * 0.7, w * 0.64, h * 0.2, 1, '#9096a0');
        c.fillStyle = 'rgba(0,0,0,0.5)';
        for (var i = 0; i < 5; i++) c.fillRect(x + w * 0.2, y + h * (0.22 + i * 0.09), w * 0.6, Math.max(0.8, h * 0.03));
        led(c, x + w / 2, y + h * 0.1, Math.max(1, Math.min(w * 0.18, h * 0.04)), on ? (hash(idx) > 0.4 ? (th.bayLed || '#2fd36a') : '#2fd36a') : '#000', on);
    }

    function bayConfig(n, th, diskCount) {
        var cfg;
        if (n <= 1) cfg = { flat: true, cols: th.cols1, rows: th.rows1 };
        else if (n === 2) cfg = { flat: false, cols: 24, rows: 1 };
        else cfg = { flat: false, cols: 12, rows: 2 };
        var capacity = cfg.cols * cfg.rows;
        if (diskCount > capacity) {
            // расширяем сетку, чтобы все диски из конфигурации были видны
            if (cfg.flat) {
                cfg.cols = Math.max(cfg.cols, Math.ceil(diskCount / cfg.rows));
            } else if (n === 2) {
                cfg.cols = Math.max(cfg.cols, diskCount);
            } else {
                cfg.cols = Math.max(cfg.cols, Math.ceil(diskCount / cfg.rows));
            }
        }
        return cfg;
    }

    function drawBladeFace(c, it, x, y, w, h, regs) {
        var syn = /synergy|sy12|12000|bl\d/i.test((it.vendor || '') + ' ' + (it.model || '') + ' ' + (it.name || ''));
        var p = it.power, on = (p === 1);
        fillRR(c, x, y + 0.5, w, h - 1, 3, vg(c, y, h, [[0, syn ? '#2d3138' : '#34373c'], [1, '#17191c']]));
        strokeRR(c, x + 0.5, y + 0.5, w - 1, h - 1, 3, 'rgba(0,0,0,0.85)', 1);
        if (syn) { c.fillStyle = '#00b5e2'; c.fillRect(x + 2, y + 2, w - 4, Math.max(2, h * 0.02)); }
        fillRR(c, x + w * 0.08, y + h * 0.88, w * 0.84, h * 0.07, 2, '#9096a0');
        // два дисковых отсека блейда
        var dw = w * 0.72, dh = h * 0.14, dx = x + w * 0.14;
        for (var i = 0; i < 2; i++) {
            var dy = y + h * (0.12 + i * 0.22);
            var d = disks[i] || null;
            fillRR(c, dx, dy, dw, dh, 2, '#0e0f11');
            led(c, dx + dw * 0.12, dy + dh * 0.5, Math.max(2, dw * 0.025), '#2fd36a', on && !!d);
            if (d) regs.push({ x: dx, y: dy, w: dw, h: dh, disk: d });
        }
        var r = Math.max(8, w * 0.08);
        powerButton(c, x + w / 2, y + h * 0.62, r, p);
        led(c, x + w * 0.22, y + h * 0.62, Math.max(2, w * 0.02), C_UID, false);
        txt(c, it.model || it.name || 'Blade', x + w / 2, y + h * 0.78, Math.max(10, UH * 0.28), 'rgba(235,238,242,0.55)', '600', 'center');
        regs.push({ x: x, y: y, w: w, h: h, body: true });
    }

    function drawServer(c, it, x, y, w, h, th, regs) {
        var n = Math.max(1, it.sizeU || sizeU), p = it.power, on = (p === 1);
        fillRR(c, x, y + 0.5, w, h - 1, 2, vg(c, y, h, [[0, th.bezel[0]], [1, th.bezel[1]]]));
        c.fillStyle = 'rgba(255,255,255,0.10)'; c.fillRect(x + 2, y + 1, w - 4, 1);
        if (th.accent) { c.fillStyle = th.accent; c.fillRect(x + w * 0.045, y + 1.5, w * 0.91, Math.max(1, UH * 0.03)); }
        var ew = w * 0.042;
        var earFill = vg(c, y, h, [[0, th.ear[0]], [1, th.ear[1]]]);
        fillRR(c, x, y + 0.5, ew, h - 1, 2, earFill);
        fillRR(c, x + w - ew, y + 0.5, ew, h - 1, 2, earFill);
        earSlots(c, x, y, ew, n); earSlots(c, x + w - ew, y, ew, n);

        var cx = x + w * 0.09;
        var r = Math.max(3.2, UH * 0.15);
        var pcy = y + h * (n === 1 ? 0.36 : 0.28);
        powerButton(c, cx, pcy, r, p);
        var ly = y + h * (n === 1 ? 0.78 : 0.56);
        led(c, cx - r * 1.1, ly, Math.max(1.4, UH * 0.05), C_UID, false);
        led(c, cx + r * 0.2, ly, Math.max(1.4, UH * 0.05), C_ON, on);
        led(c, cx + r * 1.5, ly, Math.max(1.4, UH * 0.05), C_ERR, false);
        if (n >= 2 && th.logo) drawLogo(c, th.logo, x + w * 0.052, y + h * 0.74, Math.min(UH * 0.5, h * 0.24));

        var cgx = x + w * 0.145, cgw = w * 0.655, cgy = y + h * 0.1, cgh = h * 0.8;
        fillRR(c, cgx, cgy, cgw, cgh, 2, th.cage);
        var cfg = bayConfig(n, th, disks.length);
        var cellW = cgw / cfg.cols, cellH = cgh / cfg.rows, idx = it.id * 7;
        var bay = 0;
        for (var rr_ = 0; rr_ < cfg.rows; rr_++) {
            for (var cc = 0; cc < cfg.cols; cc++) {
                var gap = cfg.flat ? 2 : 1;
                var bx = cgx + cc * cellW + gap, by = cgy + rr_ * cellH + gap;
                var bw = cellW - gap * 2, bh = cellH - gap * 2;
                var d = disks[bay] || null;
                if (cfg.flat) driveFlat(c, bx, by, bw, bh, on, idx++, th, !!d);
                else driveVert(c, bx, by, bw, bh, on, idx++, th, !!d);
                if (d) regs.push({ x: bx, y: by, w: bw, h: bh, disk: d });
                bay++;
            }
        }

        var vx = x + w * 0.815, vw = w * 0.14;
        if (n >= 2) {
            fillRR(c, vx, y + h * 0.1, vw, h * 0.2, 1.5, '#0b0c0e');
            c.fillStyle = '#2a2d32'; c.fillRect(vx + vw * 0.1, y + h * 0.18, vw * 0.8, Math.max(1, h * 0.03));
            ventDots(c, vx, y + h * 0.36, vw, h * 0.54, on);
        } else {
            if (th.logo) drawLogo(c, th.logo, vx + vw * 0.08, y + h * 0.1, Math.min(UH * 0.26, h * 0.3));
            ventDots(c, vx, y + h * 0.44, vw, h * 0.3, on);
        }
        fillRR(c, vx + vw * 0.55, y + h * 0.9 - UH * 0.09, vw * 0.18, UH * 0.09, 1, '#0b0c0e');
        fillRR(c, vx + vw * 0.8, y + h * 0.9 - UH * 0.09, vw * 0.18, UH * 0.09, 1, '#0b0c0e');
        if (UH >= 30) txt(c, (it.model || '').replace(/^(HPE?|Dell|Lenovo|DEPO)\s+/i, ''), x + w * 0.955, y + h - UH * 0.13, Math.max(7, UH * 0.2), 'rgba(230,233,238,0.55)', '500', 'right');

        regs.push({ x: x, y: y, w: w, h: h, body: true });
    }

    function layout() {
        var avail = (document.getElementById('hcvMain') || base.parentElement).clientWidth || 720;
        var maxW = Math.max(420, Math.min(900, avail - PAD * 2));
        UH = Math.max(32, Math.min(56, Math.floor(maxW / RATIO)));
        var w, h;
        if (srv.formFactor === 2) {
            // блейд крупно: вертикальная панель
            w = Math.round(UH * 3.2);
            h = Math.round(UH * 8);
        } else {
            w = Math.round(UH * RATIO);
            h = sizeU * UH;
        }
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
        fillRR(bctx, 0, 0, g.W, g.H, 10, '#0e1014');
        if (srv.formFactor === 2) {
            drawBladeFace(bctx, srv, g.x, g.y, g.w, g.h, regions);
        } else {
            drawServer(bctx, srv, g.x, g.y, g.w, g.h, styleOf(srv), regions);
        }
        regions.sort(function (a, b) {
            return (a.body ? 1 : 0) - (b.body ? 1 : 0);
        });
        drawOver();
    }

    function drawOver() {
        octx.clearRect(0, 0, g.W, g.H);
        var r = selected || hovered;
        if (!r || r.body || !r.disk) return;
        octx.fillStyle = 'rgba(77,171,247,0.18)';
        rr(octx, r.x, r.y, r.w, r.h, 2); octx.fill();
        octx.save();
        octx.strokeStyle = '#4dabf7'; octx.lineWidth = 2; octx.shadowColor = '#4dabf7'; octx.shadowBlur = 10;
        rr(octx, r.x + 0.5, r.y + 0.5, r.w - 1, r.h - 1, 2); octx.stroke();
        octx.restore();
    }

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (m) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m];
        });
    }
    function powerText(p) {
        if (p === 1) return '<span class="hcv-pw" style="color:' + C_ON + '">&#9679; Включен</span>';
        if (p === 2) return '<span class="hcv-pw" style="color:' + C_STBY + '">&#9679; Выключен</span>';
        return '<span class="hcv-pw" style="color:#8a919c">&#9679; Не определено</span>';
    }

    function renderDetail(r) {
        if (!r) {
            detail.innerHTML = '<div class="hcv-empty-hint">Наведите или кликните на диск в корзине — справа появятся его параметры. Клик по корпусу показывает данные сервера.</div>';
            return;
        }
        if (r.disk) {
            var d = r.disk;
            var dh = '<div class="hcv-detail-title">' + esc(d.name) + '</div>';
            dh += '<div class="hcv-detail-sub">Диск' + (d.subTypeName ? ' &middot; ' + esc(d.subTypeName) : '') +
                (d.capacityGb > 0 ? ' &middot; ' + esc(d.capacityGb.toFixed(1)) + ' ГБ' : '') + '</div>';
            dh += '<table>';
            (d.rows || []).forEach(function (kv) {
                dh += '<tr><td>' + esc(kv.k) + '</td><td>' + esc(kv.v) + '</td></tr>';
            });
            dh += '</table>';
            detail.innerHTML = dh;
            return;
        }
        var h = '<div class="hcv-detail-title">' + esc(srv.name) + '</div>';
        h += '<div class="hcv-detail-sub">' + (srv.formFactor === 2 ? 'Блейд-сервер' : 'Сервер') +
            (((srv.vendor || '') + ' ' + (srv.model || '')).trim() ? ' &middot; ' + esc(((srv.vendor || '') + ' ' + (srv.model || '')).trim()) : '') + '</div>';
        h += '<table><tr><td>Питание</td><td>' + powerText(srv.power) + '</td></tr>';
        h += '<tr><td>Высота</td><td>' + esc(srv.sizeU) + 'U</td></tr>';
        (srv.rows || []).forEach(function (kv) {
            h += '<tr><td>' + esc(kv.k) + '</td><td>' + esc(kv.v) + '</td></tr>';
        });
        h += '</table>';
        if (srv.links && srv.links.length) {
            h += '<div class="hcv-links">';
            srv.links.forEach(function (l) {
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
        base.style.cursor = (r && r.disk) ? 'pointer' : (r ? 'pointer' : 'default');
        if (r === hovered) return;
        hovered = r;
        drawOver();
        // при наведении на диск сразу показываем параметры (если нет закреплённого клика по другому диску)
        if (!selected || selected.body) {
            if (r && r.disk) renderDetail(r);
            else if (!selected) renderDetail({ body: true });
        }
    });
    base.addEventListener('mouseleave', function () {
        hovered = null;
        drawOver();
        if (!selected || selected.body) renderDetail(selected || { body: true });
    });
    base.addEventListener('click', function (e) {
        var rc = base.getBoundingClientRect();
        var scaleX = g.W / rc.width, scaleY = g.H / rc.height;
        var r = hit((e.clientX - rc.left) * scaleX, (e.clientY - rc.top) * scaleY);
        selected = r || { body: true };
        renderDetail(selected);
        drawOver();
    });

    window.addEventListener('resize', function () {
        g = layout();
        paint();
        if (selected) renderDetail(selected);
    });

    paint();
    selected = { body: true };
    renderDetail(selected);

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
