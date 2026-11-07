/*
 * bladechassis3d.js - 3D-визуализация блейд-корзины (HPE BladeSystem c7000 /
 * HPE Synergy 12000 / прочие) для /sprsrv/bladesystemserverslistvisual/:id.
 *
 * Зависит от three.js (static/js/three.min.js). Данные берутся из window.BCV_DATA
 * (см. controllers/sprbladesystems.go: bladeChassisVisualData).
 *
 * Управление:
 *   - колесо мыши                        - масштаб;
 *   - перетаскивание за угловой маркер   - вращение (Shift или правая кнопка - крен);
 *   - перетаскивание за фон              - вращение;
 *   - наведение                          - подсветка компонента + подсказка;
 *   - клик                               - закрепление и вывод параметров.
 *
 * Лицевая/тыльная стороны рисуются процедурно (canvas-текстуры) по виду
 * заводского исполнения: слоты полувысотных блейдов, БП и Insight Display
 * спереди; вентиляторы, interconnect, OA/Frame Link и входы питания сзади.
 */
(function () {
    'use strict';

    var DATA = window.BCV_DATA;
    var host = document.getElementById('bcv3dStage');
    if (!host) return;
    var detailEl = document.getElementById('bcvDetail');
    var listEl = document.getElementById('bcvVisible');
    var tipEl = document.getElementById('bcvTip');
    var zoomEl = document.getElementById('bcvZoom');
    var sideEl = document.getElementById('bcvSideLabel');

    function fail(msg) {
        host.innerHTML = '<div class="bcv-empty-hint p-4">' + msg + '</div>';
    }
    if (!DATA || !DATA.chassis) return fail('Нет данных корзины.');
    if (!window.THREE) return fail('Не загружена библиотека three.js.');
    var THREE = window.THREE;

    // ---------------------------------------------------------------- данные
    var CH = DATA.chassis;
    var COMPS = DATA.components || [];
    var BLADES = CH.blades || [];
    var vendorStr = ((CH.vendor || '') + ' ' + (CH.model || '') + ' ' + (CH.name || '')).toLowerCase();
    var SYN = /synergy|sy ?12|12000/.test(vendorStr);
    var HPE = SYN || /hpe|hp |hewlett|c7000|c3000|bladesystem|proliant/.test(vendorStr);
    var SK = SYN
        ? { body: '#262a31', body2: '#0f1216', accent: '#00b5e2', brand: 'HPE Synergy 12000 Frame', logo: true }
        : HPE
            ? { body: '#2a2d32', body2: '#101215', accent: '#01a982', brand: 'HPE BladeSystem c7000 Enclosure', logo: true }
            : { body: '#3a3d42', body2: '#15171a', accent: '#8d98a6', brand: ((CH.vendor || '') + ' ' + (CH.model || '')).trim() || 'Blade Enclosure', logo: false };
    var W = 445, H = 444.5, D = SYN ? 830 : 816; // мм
    var SLOTS = CH.slots > 0 ? CH.slots : (SYN ? 12 : 16);
    var COLS = Math.ceil(SLOTS / 2);

    var C_ON = '#35e06a', C_STBY = '#ffab1a', C_ERR = '#ff453a', C_UID = '#3da5ff', C_UNK = '#69707a';

    var KIND_TITLE = {
        chassis: 'Блейд-корзина', blade: 'Блейд-сервер', 'blade-empty': 'Пустой слот',
        psu: 'Блок питания', fan: 'Вентилятор', interconnect: 'Interconnect-модуль',
        manager: SYN ? 'Frame Link Module' : 'Onboard Administrator',
        display: 'Insight Display', inlet: 'Вход питания AC'
    };
    var GROUP_TITLE = {
        blade: 'Блейд-серверы', psu: 'Блоки питания', display: 'Дисплей',
        fan: 'Вентиляторы', interconnect: 'Interconnect-модули',
        manager: SYN ? 'Frame Link Module' : 'Onboard Administrator', inlet: 'Входы питания AC'
    };
    var SIDE_KINDS = { front: ['blade', 'psu', 'display'], rear: ['fan', 'interconnect', 'manager', 'inlet'] };

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (ch) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
        });
    }
    function pwColor(p) { return p === 1 ? C_ON : (p === 2 ? C_STBY : C_UNK); }
    function pwText(p) { return p === 1 ? 'OK / включено' : (p === 2 ? 'Выключено / неисправно / отсутствует' : 'Не определено'); }

    // ------------------------------------------------------- 2D-примитивы
    function mk(w, h) {
        var c = document.createElement('canvas');
        c.width = Math.max(2, Math.round(w)); c.height = Math.max(2, Math.round(h));
        return c;
    }
    function rr(x, X, Y, w, h, r) {
        r = Math.max(0, Math.min(r, w / 2, h / 2));
        x.beginPath();
        x.moveTo(X + r, Y);
        x.arcTo(X + w, Y, X + w, Y + h, r);
        x.arcTo(X + w, Y + h, X, Y + h, r);
        x.arcTo(X, Y + h, X, Y, r);
        x.arcTo(X, Y, X + w, Y, r);
        x.closePath();
    }
    function lg(x, x0, y0, x1, y1, stops) {
        var g = x.createLinearGradient(x0, y0, x1, y1);
        for (var i = 0; i < stops.length; i++) g.addColorStop(stops[i][0], stops[i][1]);
        return g;
    }
    function fillRR(x, X, Y, w, h, r, fill) { rr(x, X, Y, w, h, r); x.fillStyle = fill; x.fill(); }
    function strokeRR(x, X, Y, w, h, r, stroke, lw) { rr(x, X, Y, w, h, r); x.strokeStyle = stroke; x.lineWidth = lw || 1; x.stroke(); }
    function txt(x, s, X, Y, px, color, align, weight) {
        x.font = (weight || '600') + ' ' + Math.round(px) + 'px "Segoe UI", Arial, sans-serif';
        x.fillStyle = color; x.textAlign = align || 'center'; x.textBaseline = 'middle';
        x.fillText(s, X, Y);
    }
    function led(x, cx, cy, r, color, on) {
        x.save();
        if (on) { x.shadowColor = color; x.shadowBlur = r * 2.6; }
        x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
        x.fillStyle = on ? color : '#1d2024'; x.fill();
        x.restore();
        x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
        x.strokeStyle = 'rgba(0,0,0,.55)'; x.lineWidth = Math.max(1, r * 0.22); x.stroke();
    }
    function vents(x, X, Y, w, h, n, vertical, color) {
        x.fillStyle = color || '#07080a';
        for (var i = 0; i < n; i++) {
            if (vertical) {
                var vx = X + (i + 0.5) * (w / n);
                fillRR(x, vx - w / n * 0.2, Y, w / n * 0.4, h, w / n * 0.2, x.fillStyle);
            } else {
                var vy = Y + (i + 0.5) * (h / n);
                fillRR(x, X, vy - h / n * 0.22, w, h / n * 0.44, h / n * 0.2, x.fillStyle);
            }
        }
    }
    function noise(x, w, h, amount) {
        for (var i = 0; i < amount; i++) {
            x.fillStyle = 'rgba(255,255,255,' + (Math.random() * 0.035).toFixed(3) + ')';
            x.fillRect(Math.random() * w, Math.random() * h, 1 + Math.random() * 2, 1);
        }
    }
    function hpeLogo(x, X, Y, w, h) {
        // зелёный прямоугольник с «hpe»
        fillRR(x, X, Y, w, h, h * 0.18, SK.accent);
        txt(x, SYN ? 'HPE' : 'hpe', X + w / 2, Y + h * 0.54, h * 0.62, '#fff', 'center', '700');
    }

    // ---------------------------------------------------- текстуры компонентов
    function texBlade(b, slot, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        if (!b) {
            fillRR(x, 0, 0, w, h, w * 0.04, lg(x, 0, 0, w, 0, [[0, '#1b1d21'], [0.5, '#25282d'], [1, '#17191c']]));
            vents(x, w * 0.18, h * 0.12, w * 0.64, h * 0.76, 22, false, '#0a0b0d');
            strokeRR(x, 1, 1, w - 2, h - 2, w * 0.04, 'rgba(255,255,255,.07)', 1);
            txt(x, String(slot), w / 2, h * 0.045, Math.max(10, w * 0.17), 'rgba(255,255,255,.35)');
            return c;
        }
        var on = b.power === 1;
        fillRR(x, 0, 0, w, h, w * 0.05, lg(x, 0, 0, w, 0, [[0, '#2a2d32'], [0.5, '#3d4046'], [1, '#26292d']]));
        strokeRR(x, 1, 1, w - 2, h - 2, w * 0.05, 'rgba(255,255,255,.14)', 1);
        // цветная полоса модели
        x.fillStyle = SK.accent; x.fillRect(w * 0.08, h * 0.014, w * 0.84, Math.max(2, h * 0.007));
        txt(x, String(slot), w / 2, h * 0.05, Math.max(10, w * 0.17), 'rgba(255,255,255,.75)');
        // два отсека дисков SFF
        for (var i = 0; i < 2; i++) {
            var by = h * (0.085 + i * 0.1);
            fillRR(x, w * 0.1, by, w * 0.8, h * 0.083, 3, '#0b0c0e');
            fillRR(x, w * 0.14, by + h * 0.012, w * 0.72, h * 0.059, 2, lg(x, 0, by, 0, by + h * 0.07, [[0, '#4a4e55'], [1, '#2a2d32']]));
            x.fillStyle = '#0d0e10';
            for (var k = 0; k < 5; k++) x.fillRect(w * 0.2, by + h * (0.02 + k * 0.009), w * 0.5, Math.max(1, h * 0.003));
            led(x, w * 0.8, by + h * 0.041, Math.max(2, w * 0.03), C_ON, on && i === 0);
        }
        // вентиляционная решётка
        vents(x, w * 0.14, h * 0.3, w * 0.72, h * 0.3, 14, false, '#0a0b0d');
        // имя вертикально
        x.save();
        x.translate(w * 0.5, h * 0.45); x.rotate(-Math.PI / 2);
        var nm = String(b.name || '').split('.')[0];
        var fpx = Math.max(9, w * 0.14), maxLen = h * 0.3;
        x.font = '600 ' + Math.round(fpx) + 'px "Segoe UI", Arial, sans-serif';
        while (x.measureText(nm).width > maxLen && nm.length > 4) {
            if (fpx > w * 0.085) { fpx -= 1; x.font = '600 ' + Math.round(fpx) + 'px "Segoe UI", Arial, sans-serif'; }
            else nm = nm.slice(0, -2);
        }
        txt(x, nm, 0, 0, fpx, 'rgba(235,238,242,.9)', 'center', '600');
        x.restore();
        // логотип
        if (SK.logo) hpeLogo(x, w * 0.2, h * 0.63, w * 0.6, h * 0.045);
        // кнопка питания и светодиоды
        var py = h * 0.74;
        x.beginPath(); x.arc(w * 0.3, py, w * 0.11, 0, Math.PI * 2);
        x.fillStyle = lg(x, 0, py - w * 0.11, 0, py + w * 0.11, [[0, '#6b7078'], [1, '#2c2f34']]); x.fill();
        x.strokeStyle = 'rgba(0,0,0,.6)'; x.lineWidth = 1.5; x.stroke();
        led(x, w * 0.3, py, w * 0.045, on ? C_ON : C_STBY, true);
        led(x, w * 0.58, py, w * 0.04, C_UID, false);
        led(x, w * 0.75, py, w * 0.04, on ? C_ON : '#1d2024', on);
        // метка Model
        txt(x, (b.model || '').slice(0, 14), w / 2, h * 0.81, Math.max(8, w * 0.1), 'rgba(200,205,212,.65)', 'center', '500');
        // рычаг-фиксатор
        fillRR(x, w * 0.14, h * 0.88, w * 0.72, h * 0.07, 3, lg(x, 0, h * 0.88, 0, h * 0.95, [[0, '#d9dde3'], [1, '#8d939c']]));
        x.fillStyle = 'rgba(0,0,0,.4)'; x.fillRect(w * 0.3, h * 0.905, w * 0.4, Math.max(1, h * 0.005));
        x.fillRect(w * 0.3, h * 0.92, w * 0.4, Math.max(1, h * 0.005));
        return c;
    }

    function texPsu(st, bay, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        var comp = st.comp, absent = st.mode === 'absent';
        fillRR(x, 0, 0, w, h, w * 0.05, lg(x, 0, 0, w, 0, [[0, '#16181b'], [0.5, '#24272b'], [1, '#141619']]));
        strokeRR(x, 1, 1, w - 2, h - 2, w * 0.05, 'rgba(255,255,255,.1)', 1);
        if (absent) {
            vents(x, w * 0.12, h * 0.15, w * 0.76, h * 0.7, 14, false, '#07080a');
            txt(x, String(bay), w / 2, h * 0.07, w * 0.12, 'rgba(255,255,255,.3)');
            return c;
        }
        // решётка
        vents(x, w * 0.1, h * 0.1, w * 0.8, h * 0.52, 16, false, '#06070a');
        // ручка
        fillRR(x, w * 0.14, h * 0.7, w * 0.72, h * 0.12, 4, lg(x, 0, h * 0.7, 0, h * 0.82, [[0, '#cfd3d9'], [1, '#7f858e']]));
        // светодиоды
        var col = comp ? pwColor(comp.power) : C_UNK;
        led(x, w * 0.3, h * 0.89, w * 0.045, col, comp && comp.power === 1);
        led(x, w * 0.5, h * 0.89, w * 0.045, C_STBY, comp && comp.power === 2);
        txt(x, String(bay), w / 2, h * 0.055, w * 0.12, 'rgba(255,255,255,.55)');
        var lbl = comp && comp.model ? comp.model.replace(/^HPE?\s*/i, '').slice(0, 18) : '';
        if (lbl) txt(x, lbl, w / 2, h * 0.66, Math.max(8, w * 0.085), 'rgba(235,238,242,.75)', 'center', '500');
        return c;
    }

    function texDisplay(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, h * 0.08, lg(x, 0, 0, w, 0, [[0, '#17191c'], [0.5, '#262a2e'], [1, '#15171a']]));
        strokeRR(x, 1, 1, w - 2, h - 2, h * 0.08, 'rgba(255,255,255,.12)', 1);
        // экран
        fillRR(x, w * 0.08, h * 0.16, w * 0.84, h * 0.4, 4, '#02070c');
        fillRR(x, w * 0.1, h * 0.2, w * 0.8, h * 0.32, 2, lg(x, 0, h * 0.2, 0, h * 0.52, [[0, '#1f6fb2'], [1, '#0d3b66']]));
        txt(x, 'Insight Display', w / 2, h * 0.31, h * 0.085, '#d6ecff');
        txt(x, 'Enclosure OK', w / 2, h * 0.43, h * 0.075, '#9fd0ff', 'center', '500');
        // кнопки навигации
        var cx = w * 0.5, cy = h * 0.77, r = h * 0.1;
        x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
        x.fillStyle = lg(x, 0, cy - r, 0, cy + r, [[0, '#4a4e55'], [1, '#22252a']]); x.fill();
        x.strokeStyle = '#000'; x.lineWidth = 1.5; x.stroke();
        x.fillStyle = '#cfd3d9';
        [[0, -0.62], [0, 0.62], [-0.62, 0], [0.62, 0]].forEach(function (d, i) {
            x.beginPath();
            var ax = cx + d[0] * r, ay = cy + d[1] * r, s = r * 0.22;
            if (i < 2) { x.moveTo(ax, ay + (i ? s : -s)); x.lineTo(ax - s, ay + (i ? -s : s)); x.lineTo(ax + s, ay + (i ? -s : s)); }
            else { x.moveTo(ax + (i === 3 ? s : -s), ay); x.lineTo(ax + (i === 3 ? -s : s), ay - s); x.lineTo(ax + (i === 3 ? -s : s), ay + s); }
            x.closePath(); x.fill();
        });
        led(x, w * 0.14, h * 0.77, h * 0.035, C_ON, true);
        led(x, w * 0.86, h * 0.77, h * 0.035, C_UID, false);
        return c;
    }

    function texFan(st, bay, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, w * 0.04, lg(x, 0, 0, w, h, [[0, '#2a2d31'], [1, '#131518']]));
        strokeRR(x, 1, 1, w - 2, h - 2, w * 0.04, 'rgba(255,255,255,.1)', 1);
        var cx = w / 2, cy = h / 2, R = Math.min(w, h) * 0.43;
        var absent = st.mode === 'absent';
        x.beginPath(); x.arc(cx, cy, R, 0, Math.PI * 2);
        x.fillStyle = '#050607'; x.fill();
        if (!absent) {
            // лопасти
            x.save(); x.translate(cx, cy);
            for (var i = 0; i < 9; i++) {
                x.rotate(Math.PI * 2 / 9);
                x.beginPath(); x.moveTo(R * 0.2, 0);
                x.quadraticCurveTo(R * 0.7, -R * 0.35, R * 0.9, -R * 0.08);
                x.quadraticCurveTo(R * 0.65, R * 0.05, R * 0.2, R * 0.12);
                x.closePath();
                x.fillStyle = 'rgba(70,74,82,.9)'; x.fill();
            }
            x.restore();
            x.beginPath(); x.arc(cx, cy, R * 0.22, 0, Math.PI * 2);
            x.fillStyle = lg(x, 0, cy - R * 0.22, 0, cy + R * 0.22, [[0, '#585d66'], [1, '#202327']]); x.fill();
        }
        // защитная решётка (кольца + спицы)
        x.strokeStyle = 'rgba(155,160,168,.75)'; x.lineWidth = Math.max(1.2, R * 0.03);
        for (var r = 0.3; r < 1.01; r += 0.23) { x.beginPath(); x.arc(cx, cy, R * r, 0, Math.PI * 2); x.stroke(); }
        for (var s = 0; s < 8; s++) {
            var a = s * Math.PI / 4;
            x.beginPath(); x.moveTo(cx + Math.cos(a) * R * 0.22, cy + Math.sin(a) * R * 0.22);
            x.lineTo(cx + Math.cos(a) * R, cy + Math.sin(a) * R); x.stroke();
        }
        x.beginPath(); x.arc(cx, cy, R, 0, Math.PI * 2); x.lineWidth = Math.max(2, R * 0.07);
        x.strokeStyle = '#6c717a'; x.stroke();
        var col = st.comp ? pwColor(st.comp.power) : (absent ? '#1d2024' : C_UNK);
        led(x, w * 0.12, h * 0.12, Math.max(2.5, w * 0.035), col, !!(st.comp && st.comp.power));
        txt(x, String(bay), w * 0.88, h * 0.11, Math.max(9, w * 0.11), 'rgba(255,255,255,.55)');
        return c;
    }

    function portBlock(x, X, Y, w, h, n, color) {
        var pw = w / n;
        for (var i = 0; i < n; i++) {
            fillRR(x, X + i * pw + pw * 0.1, Y, pw * 0.8, h, 2, '#050607');
            fillRR(x, X + i * pw + pw * 0.2, Y + h * 0.15, pw * 0.6, h * 0.5, 1, color || '#8a6f2a');
            led(x, X + i * pw + pw * 0.5, Y + h * 1.18, Math.max(1.6, pw * 0.07), C_ON, true);
        }
    }

    function texInterconnect(st, bay, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        if (st.mode === 'absent') {
            fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, 0, [[0, '#1d1f23'], [1, '#2a2d32']]));
            vents(x, w * 0.1, h * 0.14, w * 0.8, h * 0.72, 12, false, '#08090b');
            txt(x, 'ICM ' + bay, w / 2, h * 0.07, h * 0.1, 'rgba(255,255,255,.3)');
            return c;
        }
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, 0, [[0, '#cfd3d8'], [0.5, '#e6e8eb'], [1, '#b8bdc4']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 4, 'rgba(0,0,0,.45)', 1.2);
        fillRR(x, w * 0.04, h * 0.1, w * 0.92, h * 0.12, 2, '#11151a');
        txt(x, 'Interconnect ' + bay, w * 0.08, h * 0.16, h * 0.085, '#d8dde3', 'left', '600');
        var cols = SYN ? 2 : 4;
        portBlock(x, w * 0.06, h * 0.32, w * 0.88, h * 0.12, SYN ? 8 : 8, '#8a6f2a');
        portBlock(x, w * 0.06, h * 0.58, w * 0.88, h * 0.12, SYN ? 8 : 4, '#7a7f87');
        var comp = st.comp;
        txt(x, comp && comp.model ? comp.model.slice(0, 26) : '', w / 2, h * 0.88, h * 0.075, '#2b2f35', 'center', '600');
        led(x, w * 0.9, h * 0.16, Math.max(2, h * 0.03), comp ? pwColor(comp.power) : C_UNK, !!(comp && comp.power));
        return c;
    }

    function texManager(st, n, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        if (st.mode === 'absent') {
            fillRR(x, 0, 0, w, h, 4, '#1d1f23');
            vents(x, w * 0.1, h * 0.2, w * 0.8, h * 0.6, 8, false, '#08090b');
            return c;
        }
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, 0, [[0, '#23262a'], [0.5, '#34373c'], [1, '#202327']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 4, 'rgba(255,255,255,.14)', 1.2);
        // ручка-рычаг
        fillRR(x, w * 0.03, h * 0.08, w * 0.05, h * 0.84, 3, lg(x, 0, 0, w * 0.1, 0, [[0, '#cfd3d9'], [1, '#868c95']]));
        txt(x, SYN ? 'Frame Link ' + n : 'OA ' + n, w * 0.52, h * 0.12, h * 0.1, '#e5e8ec');
        // LCD-панель
        fillRR(x, w * 0.14, h * 0.22, w * 0.34, h * 0.28, 2, '#06090d');
        fillRR(x, w * 0.16, h * 0.26, w * 0.3, h * 0.2, 1, '#1c5f94');
        // порты RJ-45 / serial
        for (var i = 0; i < 3; i++) {
            fillRR(x, w * (0.55 + i * 0.14), h * 0.3, w * 0.11, h * 0.26, 2, '#050607');
            fillRR(x, w * (0.565 + i * 0.14), h * 0.38, w * 0.08, h * 0.14, 1, '#8a6f2a');
            led(x, w * (0.605 + i * 0.14), h * 0.66, Math.max(1.6, h * 0.025), C_ON, true);
        }
        var comp = st.comp;
        led(x, w * 0.2, h * 0.78, Math.max(2.5, h * 0.045), comp ? pwColor(comp.power) : C_UNK, !!(comp && comp.power));
        led(x, w * 0.32, h * 0.78, Math.max(2.5, h * 0.045), C_UID, false);
        txt(x, comp && comp.model ? comp.model.slice(0, 22) : '', w * 0.62, h * 0.86, h * 0.08, 'rgba(220,224,230,.8)', 'center', '500');
        return c;
    }

    function texInlet(n, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 3, '#1b1d21');
        strokeRR(x, 1, 1, w - 2, h - 2, 3, 'rgba(255,255,255,.1)', 1);
        // C19-розетка
        fillRR(x, w * 0.2, h * 0.16, w * 0.6, h * 0.68, h * 0.1, '#08090b');
        x.fillStyle = '#2b2f35';
        x.fillRect(w * 0.3, h * 0.4, w * 0.12, h * 0.2);
        x.fillRect(w * 0.58, h * 0.4, w * 0.12, h * 0.2);
        x.fillRect(w * 0.44, h * 0.62, w * 0.12, h * 0.12);
        txt(x, 'AC ' + n, w * 0.5, h * 0.08, Math.max(7, h * 0.12), 'rgba(255,255,255,.5)');
        return c;
    }

    // ---------------------------------------------- текстуры корпуса (плиты)
    function plateBase(x, w, h) {
        x.fillStyle = lg(x, 0, 0, w, h, [[0, SK.body], [1, SK.body2]]);
        x.fillRect(0, 0, w, h);
        noise(x, w, h, 14000);
    }
    function layoutFront() {
        var L = { blades: [], psu: [], display: null };
        var fx0 = 0.014, fw = 0.972, bv0 = 0.028, bvh = 0.69, gap = 0.0025;
        var cw = fw / COLS, chh = bvh / 2;
        for (var s = 1; s <= SLOTS; s++) {
            var col = (s - 1) % COLS, row = Math.floor((s - 1) / COLS);
            L.blades.push({ n: s, u0: fx0 + col * cw + gap, u1: fx0 + (col + 1) * cw - gap, v0: bv0 + row * chh + gap * 1.6, v1: bv0 + (row + 1) * chh - gap * 1.6 });
        }
        var pw = 0.125 * fw, pg = 0.012 * fw, dw = 0.16 * fw;
        var total = 6 * pw + dw + 6 * pg;
        var u = fx0 + (fw - total) / 2;
        var v0 = 0.745, v1 = 0.955;
        for (var k = 0; k < 7; k++) {
            if (k === 3) { L.display = { u0: u, u1: u + dw, v0: v0, v1: v1 }; u += dw + pg; continue; }
            var idx = k < 3 ? k : k - 1;
            L.psu.push({ n: idx + 1, u0: u, u1: u + pw, v0: v0, v1: v1 });
            u += pw + pg;
        }
        return L;
    }
    function layoutRear() {
        var L = { fans: [], ic: [], mgr: [], inlets: [] };
        var i, col, row;
        for (i = 0; i < 10; i++) {
            col = i % 5; row = Math.floor(i / 5);
            L.fans.push({ n: i + 1, u0: 0.022 + col * 0.192 + 0.004, u1: 0.022 + (col + 1) * 0.192 - 0.004, v0: 0.02 + row * 0.172 + 0.004, v1: 0.02 + (row + 1) * 0.172 - 0.004 });
        }
        if (SYN) {
            for (i = 0; i < 6; i++) {
                col = i % 3; row = Math.floor(i / 3);
                L.ic.push({ n: i + 1, u0: 0.022 + col * 0.235 + 0.004, u1: 0.022 + (col + 1) * 0.235 - 0.004, v0: 0.385 + row * 0.18 + 0.004, v1: 0.385 + (row + 1) * 0.18 - 0.004 });
            }
            for (i = 0; i < 2; i++) {
                L.mgr.push({ n: i + 1, u0: 0.745, u1: 0.978, v0: 0.385 + i * 0.18 + 0.004, v1: 0.385 + (i + 1) * 0.18 - 0.004 });
            }
        } else {
            for (i = 0; i < 8; i++) {
                col = i % 4; row = Math.floor(i / 4);
                L.ic.push({ n: i + 1, u0: 0.022 + col * 0.239 + 0.004, u1: 0.022 + (col + 1) * 0.239 - 0.004, v0: 0.385 + row * 0.18 + 0.004, v1: 0.385 + (row + 1) * 0.18 - 0.004 });
            }
            for (i = 0; i < 2; i++) {
                L.mgr.push({ n: i + 1, u0: 0.26 + i * 0.26, u1: 0.26 + i * 0.26 + 0.22, v0: 0.775, v1: 0.9 });
            }
        }
        for (i = 0; i < 6; i++) {
            L.inlets.push({ n: i + 1, u0: 0.03 + i * 0.1617 + 0.004, u1: 0.03 + (i + 1) * 0.1617 - 0.004, v0: 0.918, v1: 0.985 });
        }
        return L;
    }
    var LF = layoutFront(), LR = layoutRear();

    function texPlate(side) {
        var w = 1024, h = Math.round(1024 * H / W), x, c = mk(w, h);
        x = c.getContext('2d');
        plateBase(x, w, h);
        var L = side === 'front' ? LF : LR;
        function box(r, label) {
            var X = r.u0 * w, Y = r.v0 * h, ww = (r.u1 - r.u0) * w, hh = (r.v1 - r.v0) * h;
            fillRR(x, X - 2, Y - 2, ww + 4, hh + 4, 4, '#0a0b0d');
            if (label) txt(x, label, X + ww / 2, Y - 8, 11, 'rgba(255,255,255,.35)');
        }
        if (side === 'front') {
            L.blades.forEach(function (r) { box(r); });
            L.psu.forEach(function (r) { box(r); });
            if (L.display) box(L.display);
            txt(x, SK.brand, w * 0.5, h * 0.978, 13, 'rgba(255,255,255,.5)', 'center', '600');
            if (SK.logo) hpeLogo(x, w * 0.02, h * 0.962, 44, 18);
        } else {
            L.fans.forEach(function (r) { box(r); });
            L.ic.forEach(function (r) { box(r); });
            L.mgr.forEach(function (r) { box(r); });
            L.inlets.forEach(function (r) { box(r); });
            // шильдик
            var sy = (SYN ? 0.775 : 0.775) * h;
            if (SYN) { fillRR(x, w * 0.1, sy, w * 0.5, h * 0.12, 4, 'rgba(0,0,0,.22)'); }
            txt(x, SK.brand, w * (SYN ? 0.35 : 0.14), h * (SYN ? 0.835 : 0.955), 13, 'rgba(255,255,255,.4)', 'center', '600');
        }
        return c;
    }
    function texSide(w, h, withHoles) {
        var c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, 0, h, [[0, SK.body], [0.5, SK.body2], [1, SK.body]]);
        x.fillRect(0, 0, w, h);
        noise(x, w, h, 12000);
        if (withHoles) {
            x.fillStyle = 'rgba(0,0,0,.62)';
            var step = 14;
            for (var yy = 40, r = 0; yy < h - 30; yy += step * 0.87, r++) {
                for (var xx = 40 + (r % 2 ? step / 2 : 0); xx < w - 30; xx += step) {
                    x.beginPath(); x.arc(xx, yy, 3.6, 0, Math.PI * 2); x.fill();
                }
            }
        }
        // рёбра жёсткости
        x.strokeStyle = 'rgba(255,255,255,.07)'; x.lineWidth = 2;
        x.strokeRect(10, 10, w - 20, h - 20);
        return c;
    }
    function texEar(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, w, 0, [[0, '#9aa0a8'], [1, '#6a7078']]);
        x.fillRect(0, 0, w, h);
        x.fillStyle = '#16181b';
        for (var i = 0; i < 12; i++) fillRR(x, w * 0.3, h * (0.04 + i * 0.08), w * 0.4, h * 0.03, 3, '#16181b');
        return c;
    }

    // ----------------------------------------------------------------- сцена
    var renderer;
    try {
        renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    } catch (e) {
        return fail('Браузер не поддерживает WebGL — 3D-вид недоступен.');
    }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    var maxAniso = renderer.capabilities.getMaxAnisotropy ? renderer.capabilities.getMaxAnisotropy() : 4;
    var dom = renderer.domElement;
    dom.style.touchAction = 'none';
    dom.style.display = 'block';
    host.innerHTML = '';
    host.appendChild(dom);
    var ov = document.createElement('canvas');
    ov.style.cssText = 'position:absolute;left:0;top:0;pointer-events:none;';
    host.appendChild(ov);
    var octx = ov.getContext('2d');

    var scene = new THREE.Scene();
    var camera = new THREE.PerspectiveCamera(30, 1, 20, 9000);
    var DEF_DIST = 2100, MIN_DIST = 450, MAX_DIST = 6000;
    var dist = DEF_DIST;
    camera.position.set(0, 0, dist);
    camera.lookAt(0, 0, 0);

    scene.add(new THREE.AmbientLight(0xffffff, 1.35));
    scene.add(new THREE.HemisphereLight(0xdfe8ff, 0x303540, 0.9));
    var key = new THREE.DirectionalLight(0xffffff, 2.4); key.position.set(500, 800, 1200); scene.add(key);
    var fill = new THREE.DirectionalLight(0xbfd4ff, 0.9); fill.position.set(-900, -300, 700); scene.add(fill);

    var rig = new THREE.Group();
    scene.add(rig);

    function tex(canvas) {
        var t = new THREE.CanvasTexture(canvas);
        t.anisotropy = maxAniso;
        if (THREE.SRGBColorSpace) t.colorSpace = THREE.SRGBColorSpace;
        t.needsUpdate = true;
        return t;
    }
    function std(o) { return new THREE.MeshStandardMaterial(o); }

    var pickables = [], modules = [];

    // Корпус
    var sideMat = std({ map: tex(texSide(1024, 560, true)), roughness: 0.55, metalness: 0.45 });
    var topMat = std({ map: tex(texSide(1024, 1024, true)), roughness: 0.55, metalness: 0.45 });
    var frontMat = std({ map: tex(texPlate('front')), roughness: 0.6, metalness: 0.35 });
    var rearMat = std({ map: tex(texPlate('rear')), roughness: 0.6, metalness: 0.35 });
    var body = new THREE.Mesh(new THREE.BoxGeometry(W, H, D), [sideMat, sideMat, topMat, topMat, frontMat, rearMat]);
    rig.add(body);

    // Крепёжные «уши»
    (function () {
        var m = std({ map: tex(texEar(64, 512)), roughness: 0.4, metalness: 0.7 });
        [-1, 1].forEach(function (s) {
            var ear = new THREE.Mesh(new THREE.BoxGeometry(24, H, 3), m);
            ear.position.set(s * (W / 2 + 12), 0, D / 2 - 30);
            rig.add(ear);
        });
    })();

    function faceXY(side, u, v) {
        return [side === 'front' ? -W / 2 + u * W : W / 2 - u * W, H / 2 - v * H];
    }

    function addModule(side, r, depth, canvas, info) {
        var w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        var t = tex(canvas);
        var faceMat = std({ map: t, roughness: 0.5, metalness: 0.25, emissive: new THREE.Color(0xffffff), emissiveMap: t, emissiveIntensity: 0 });
        var darkMat = std({ color: 0x15171a, roughness: 0.6, metalness: 0.4, emissive: new THREE.Color(0x4dabf7), emissiveIntensity: 0 });
        var mats = [darkMat, darkMat, darkMat, darkMat, darkMat, darkMat];
        mats[side === 'front' ? 4 : 5] = faceMat;
        var m = new THREE.Mesh(new THREE.BoxGeometry(w, h, depth), mats);
        var xy = faceXY(side, (r.u0 + r.u1) / 2, (r.v0 + r.v1) / 2);
        m.position.set(xy[0], xy[1], side === 'front' ? D / 2 + depth / 2 - 0.4 : -D / 2 - depth / 2 + 0.4);
        // «контур» подсветки
        var hull = new THREE.Mesh(
            new THREE.BoxGeometry(w + 7, h + 7, depth + 7),
            new THREE.MeshBasicMaterial({ color: 0x4dabf7, side: THREE.BackSide, transparent: true, opacity: 0.9, depthWrite: false })
        );
        hull.visible = false;
        m.add(hull);
        info.side = side;
        m.userData = { info: info, faceMat: faceMat, darkMat: darkMat, hull: hull, flashUntil: 0 };
        rig.add(m);
        pickables.push(m);
        modules.push(m);
        return m;
    }

    // --- поиск данных по отсекам
    function compsOf(kind) { return COMPS.filter(function (c) { return c.kind === kind; }); }
    function stateOf(kind, n, idx) {
        var list = compsOf(kind), i, f = null;
        for (i = 0; i < list.length; i++) if (list[i].bay === n) { f = list[i]; break; }
        if (!f) {
            var anyBay = list.some(function (c) { return c.bay > 0; });
            if (!anyBay && idx < list.length) f = list[idx];
        }
        if (f) return { comp: f, mode: 'present' };
        return { comp: null, mode: list.length === 0 ? 'unknown' : 'absent' };
    }
    function compInfo(kind, n, st, extraRows) {
        var info = { kind: kind, bay: n, title: '', sub: '', power: 0, rows: [], links: [] };
        if (st.comp) {
            info.title = st.comp.name;
            info.sub = KIND_TITLE[kind] + (st.comp.model ? ' · ' + st.comp.model : '');
            info.power = st.comp.power;
            info.rows = st.comp.rows || [];
            info.links = st.comp.links || [];
        } else {
            info.title = KIND_TITLE[kind] + ', отсек ' + n;
            info.sub = st.mode === 'absent' ? 'Отсек пуст / модуль не установлен' : 'Нет данных о компоненте в справочнике';
            info.power = 0;
            info.rows = [{ k: 'Отсек (bay)', v: String(n) }];
            info.links = [{ title: 'Редактировать корзину', url: '/sprsrv/bladesystemsedit-tab/' + CH.id }];
        }
        if (extraRows) info.rows = extraRows.concat(info.rows);
        info.absent = (st.mode === 'absent');
        return info;
    }

    // --- блейды и слоты (спереди)
    var bladeBySlot = {};
    BLADES.forEach(function (b) { bladeBySlot[b.slot] = b; });
    LF.blades.forEach(function (r) {
        var b = bladeBySlot[r.n];
        var w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        var cv = texBlade(b, r.n, w * 3.2, h * 3.2);
        var info;
        if (b) {
            info = { kind: 'blade', bay: r.n, title: b.name, sub: 'Слот ' + r.n + (b.model ? ' · ' + b.model : ''), power: b.power, rows: b.rows || [], links: b.links || [] };
        } else {
            info = { kind: 'blade-empty', bay: r.n, title: 'Слот ' + r.n, sub: 'Пустой отсек (заглушка)', power: 0, rows: [{ k: 'Слот', v: String(r.n) }], links: [] };
        }
        addModule('front', r, b ? 10 : 4, cv, info);
    });
    // --- БП и дисплей (спереди)
    LF.psu.forEach(function (r, i) {
        var st = stateOf('psu', r.n, i);
        var w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        addModule('front', r, st.mode === 'absent' ? 4 : 9, texPsu(st, r.n, w * 3.2, h * 3.2), compInfo('psu', r.n, st));
    });
    if (LF.display) {
        var dr = LF.display, dw = (dr.u1 - dr.u0) * W, dh = (dr.v1 - dr.v0) * H;
        addModule('front', dr, 8, texDisplay(dw * 3.2, dh * 3.2), {
            kind: 'display', title: 'Insight Display', sub: 'Дисплей на передней панели корзины', power: CH.power,
            rows: [{ k: 'Корзина', v: CH.name }, { k: 'Состояние корзины', v: pwText(CH.power) }],
            links: []
        });
    }
    // --- тыльная сторона
    LR.fans.forEach(function (r, i) {
        var st = stateOf('fan', r.n, i), w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        addModule('rear', r, st.mode === 'absent' ? 5 : 28, texFan(st, r.n, w * 3.2, h * 3.2), compInfo('fan', r.n, st));
    });
    LR.ic.forEach(function (r, i) {
        var st = stateOf('interconnect', r.n, i), w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        addModule('rear', r, st.mode === 'absent' ? 5 : 24, texInterconnect(st, r.n, w * 3.2, h * 3.2), compInfo('interconnect', r.n, st));
    });
    LR.mgr.forEach(function (r, i) {
        var st = stateOf('manager', r.n, i), w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        addModule('rear', r, st.mode === 'absent' ? 5 : 22, texManager(st, r.n, w * 3.2, h * 3.2), compInfo('manager', r.n, st));
    });
    LR.inlets.forEach(function (r, i) {
        var st = stateOf('psu', r.n, i), w = (r.u1 - r.u0) * W, h = (r.v1 - r.v0) * H;
        var info = compInfo('psu', r.n, st);
        info.kind = 'inlet';
        info.title = 'Вход питания AC ' + r.n;
        info.sub = 'Питание для блока питания ' + r.n + (st.comp && st.comp.model ? ' · ' + st.comp.model : '');
        addModule('rear', r, 8, texInlet(r.n, w * 4, h * 4), info);
    });

    // Корпус как выбираемый объект
    var chassisInfo = {
        kind: 'chassis', title: CH.name, sub: ((CH.vendor || '') + ' ' + (CH.model || '')).trim(),
        power: CH.power, rows: CH.rows || [], links: CH.links || [], side: ''
    };
    body.userData = { info: chassisInfo, isBody: true };
    pickables.push(body);

    // ------------------------------------------------------------ вид/анимация
    var Q_ID = new THREE.Quaternion();
    function qAxis(ax, deg) { return new THREE.Quaternion().setFromAxisAngle(ax, deg * Math.PI / 180); }
    var AX = new THREE.Vector3(1, 0, 0), AY = new THREE.Vector3(0, 1, 0), AZ = new THREE.Vector3(0, 0, 1);
    var VIEWS = {
        front: Q_ID.clone(),
        rear: qAxis(AY, 180),
        left: qAxis(AY, 90),
        right: qAxis(AY, -90),
        top: qAxis(AX, 90),
        iso: qAxis(AX, 22).multiply(qAxis(AY, -35)),
        isorear: qAxis(AX, 22).multiply(qAxis(AY, 180 + 35))
    };
    var anim = null;
    var dirty = true;
    function animateTo(q, d) {
        anim = { t0: performance.now(), dur: 520, q0: rig.quaternion.clone(), q1: q.clone(), d0: dist, d1: d == null ? dist : d };
        dirty = true;
    }
    function setDist(d) {
        dist = Math.max(MIN_DIST, Math.min(MAX_DIST, d));
        camera.position.set(0, 0, dist);
        if (zoomEl) zoomEl.textContent = Math.round(DEF_DIST / dist * 100) + '%';
        dirty = true;
    }
    function rotateBy(dx, dy) {
        rig.quaternion.premultiply(qAxis(AY, dx * 0.45)).premultiply(qAxis(AX, dy * 0.45)).normalize();
        anim = null; dirty = true; viewChanged();
    }
    function rollBy(rad) {
        rig.quaternion.premultiply(new THREE.Quaternion().setFromAxisAngle(AZ, rad)).normalize();
        anim = null; dirty = true; viewChanged();
    }

    // ----------------------------------------------- стороны и видимые компоненты
    var curSide = '';
    function currentSide() {
        var n = new THREE.Vector3(0, 0, 1).applyQuaternion(rig.quaternion);
        if (n.z > 0.12) return 'front';
        if (n.z < -0.12) return 'rear';
        return '';
    }
    function sideLabel(s) { return s === 'front' ? 'Спереди (блейды, блоки питания, дисплей)' : (s === 'rear' ? 'Сзади (вентиляторы, interconnect, OA/FLM, питание)' : 'Сбоку / сверху (компоненты не видны)'); }
    var viewTimer = 0;
    function viewChanged() {
        var s = currentSide();
        if (s === curSide) return;
        curSide = s;
        if (sideEl) sideEl.textContent = sideLabel(s);
        buildList();
        if (s) {
            var now = performance.now();
            modules.forEach(function (m) { if (m.userData.info.side === s) m.userData.flashUntil = now + 2200; });
            dirty = true;
        }
    }
    function dotClass(p) { return p === 1 ? 'bcv-dot-on' : (p === 2 ? 'bcv-dot-stby' : 'bcv-dot-unk'); }
    function buildList() {
        if (!listEl) return;
        var s = curSide;
        if (!s) { listEl.innerHTML = '<div class="bcv-empty-hint">Корзина повёрнута боком — доступных спереди/сзади компонентов нет. Выберите «Спереди» или «Сзади».</div>'; return; }
        var html = '';
        SIDE_KINDS[s].forEach(function (kind) {
            var items = modules.filter(function (m) {
                var i = m.userData.info; return i.side === s && i.kind === kind && !i.absent;
            });
            if (!items.length) return;
            html += '<div class="bcv-grp"><div class="bcv-grp-title">' + esc(GROUP_TITLE[kind]) + ' <span class="text-muted">(' + items.length + ')</span></div>';
            items.forEach(function (m) {
                var i = m.userData.info;
                html += '<button type="button" class="bcv-li" data-mi="' + modules.indexOf(m) + '"><span class="bcv-dot ' + dotClass(i.power) + '"></span>' +
                    esc(kind === 'blade' ? ('Слот ' + i.bay + ': ' + i.title) : (kind === 'inlet' ? i.title : i.title)) + '</button>';
            });
            html += '</div>';
        });
        listEl.innerHTML = html || '<div class="bcv-empty-hint">Нет компонентов.</div>';
    }
    if (listEl) {
        listEl.addEventListener('click', function (e) {
            var btn = e.target.closest('.bcv-li');
            if (!btn) return;
            select(modules[parseInt(btn.getAttribute('data-mi'), 10)], false);
        });
        listEl.addEventListener('mouseover', function (e) {
            var btn = e.target.closest('.bcv-li');
            hovered = btn ? modules[parseInt(btn.getAttribute('data-mi'), 10)] : null;
            dirty = true;
        });
        listEl.addEventListener('mouseleave', function () { hovered = null; dirty = true; });
    }

    // ------------------------------------------------------------ выбор/инфо
    var hovered = null, selected = null;
    function renderInfo(info) {
        if (!detailEl) return;
        if (!info) { detailEl.innerHTML = '<div class="bcv-empty-hint">Наведите курсор на компонент — подсветка и подсказка; кликните — параметры здесь.</div>'; return; }
        var h = '<div class="bcv-detail-title">' + esc(info.title) + '</div>';
        h += '<div class="bcv-detail-sub">' + esc(KIND_TITLE[info.kind] || '') + (info.sub ? ' &middot; ' + esc(info.sub) : '') + '</div>';
        h += '<div class="bcv-pw"><span class="bcv-dot ' + dotClass(info.power) + '"></span>' + esc(pwText(info.power)) + '</div>';
        var rows = info.rows || [];
        if (rows.length) {
            h += '<table>';
            rows.forEach(function (r) { h += '<tr><td>' + esc(r.k) + '</td><td>' + esc(r.v) + '</td></tr>'; });
            h += '</table>';
        }
        var links = info.links || [];
        if (links.length) {
            h += '<div class="bcv-links">';
            links.forEach(function (l) { h += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + '</a>'; });
            h += '</div>';
        }
        detailEl.innerHTML = h;
    }
    function select(m, fromScene) {
        selected = m && m.userData.isBody ? null : m;
        renderInfo(m ? m.userData.info : null);
        if (m && !m.userData.isBody && !fromScene) {
            var sd = m.userData.info.side;
            if (sd && sd !== currentSide()) animateTo(VIEWS[sd]);
        }
        dirty = true;
    }

    // --------------------------------------------------------------- рендер
    var raycaster = new THREE.Raycaster();
    var ndc = new THREE.Vector2();
    var vw = 800, vh = 560;
    function resize() {
        vw = Math.max(320, host.parentElement ? host.parentElement.clientWidth : 800);
        vh = Math.max(460, Math.min(720, Math.round(vw * 0.62)));
        renderer.setSize(vw, vh, true);
        dom.style.width = vw + 'px'; dom.style.height = vh + 'px';
        var dpr = Math.min(window.devicePixelRatio || 1, 2);
        ov.width = Math.round(vw * dpr); ov.height = Math.round(vh * dpr);
        ov.style.width = vw + 'px'; ov.style.height = vh + 'px';
        octx.setTransform(dpr, 0, 0, dpr, 0, 0);
        camera.aspect = vw / vh; camera.updateProjectionMatrix();
        dirty = true;
    }
    window.addEventListener('resize', resize);

    var corners = [];
    function computeCorners() {
        rig.updateMatrixWorld(true);
        corners.length = 0;
        for (var sx = -1; sx <= 1; sx += 2) for (var sy = -1; sy <= 1; sy += 2) for (var sz = -1; sz <= 1; sz += 2) {
            var v = new THREE.Vector3(sx * W / 2, sy * H / 2, sz * D / 2).applyMatrix4(rig.matrixWorld);
            var wz = v.z;
            v.project(camera);
            corners.push({ x: (v.x * 0.5 + 0.5) * vw, y: (-v.y * 0.5 + 0.5) * vh, sx: sx, sy: sy, sz: sz, front: wz });
        }
    }
    var hoverCorner = -1;
    function drawOverlay() {
        octx.clearRect(0, 0, vw, vh);
        computeCorners();
        octx.save();
        octx.lineWidth = 1;
        octx.strokeStyle = 'rgba(77,171,247,.38)';
        octx.setLineDash([5, 4]);
        corners.forEach(function (a, i) {
            corners.forEach(function (b, j) {
                if (j <= i) return;
                var diff = (a.sx !== b.sx) + (a.sy !== b.sy) + (a.sz !== b.sz);
                if (diff !== 1) return;
                octx.beginPath(); octx.moveTo(a.x, a.y); octx.lineTo(b.x, b.y); octx.stroke();
            });
        });
        octx.setLineDash([]);
        corners.forEach(function (c, i) {
            var r = i === hoverCorner ? 9 : 6.5;
            octx.beginPath(); octx.arc(c.x, c.y, r + 2, 0, Math.PI * 2);
            octx.fillStyle = 'rgba(10,14,20,.75)'; octx.fill();
            octx.beginPath(); octx.arc(c.x, c.y, r, 0, Math.PI * 2);
            octx.fillStyle = i === hoverCorner ? '#ffcf4d' : '#4dabf7'; octx.fill();
            octx.lineWidth = 1.5; octx.strokeStyle = '#fff'; octx.stroke();
        });
        octx.restore();
    }

    function applyHighlights(now) {
        var flashing = false;
        for (var i = 0; i < modules.length; i++) {
            var m = modules[i], u = m.userData;
            var isSel = m === selected, isHov = m === hovered, fl = u.flashUntil > now;
            var hullOn = false, color = 0x4dabf7, opacity = 0.9, emis = 0;
            if (isSel) { hullOn = true; color = 0xffcf4d; emis = 0.55; }
            else if (isHov) { hullOn = true; color = 0x4dabf7; emis = 0.38; }
            else if (fl) {
                hullOn = true; color = 0x35e0c8; flashing = true;
                var ph = (u.flashUntil - now) / 2200;
                opacity = 0.35 + 0.6 * Math.abs(Math.sin(ph * Math.PI * 4));
                emis = 0.18 * opacity;
            }
            u.hull.visible = hullOn;
            if (hullOn) { u.hull.material.color.setHex(color); u.hull.material.opacity = opacity; }
            u.faceMat.emissiveIntensity = emis;
        }
        return flashing;
    }

    function frame(now) {
        requestAnimationFrame(frame);
        if (anim) {
            var t = Math.min(1, (now - anim.t0) / anim.dur);
            var e = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
            rig.quaternion.copy(anim.q0).slerp(anim.q1, e);
            setDist(anim.d0 + (anim.d1 - anim.d0) * e);
            dirty = true;
            if (t >= 1) { anim = null; }
            viewChanged();
        }
        var fl = applyHighlights(now);
        if (fl) dirty = true;
        if (!dirty) return;
        dirty = false;
        renderer.render(scene, camera);
        drawOverlay();
    }

    // ----------------------------------------------------------- указатель
    function localXY(e) {
        var r = dom.getBoundingClientRect();
        return { x: (e.clientX - r.left) * (vw / r.width), y: (e.clientY - r.top) * (vh / r.height) };
    }
    function cornerAt(p) {
        var best = -1, bd = 15 * 15;
        corners.forEach(function (c, i) {
            var d = (c.x - p.x) * (c.x - p.x) + (c.y - p.y) * (c.y - p.y);
            if (d < bd) { bd = d; best = i; }
        });
        return best;
    }
    function pick(p) {
        ndc.set(p.x / vw * 2 - 1, -(p.y / vh) * 2 + 1);
        raycaster.setFromCamera(ndc, camera);
        var hits = raycaster.intersectObjects(pickables, false);
        return hits.length ? hits[0].object : null;
    }
    function showTip(m, p) {
        if (!tipEl) return;
        if (!m || m.userData.isBody) { tipEl.style.display = 'none'; return; }
        var i = m.userData.info;
        tipEl.innerHTML = '<b>' + esc(i.title) + '</b><br><span>' + esc(KIND_TITLE[i.kind] || '') + (i.sub && i.kind !== 'blade' ? ' · ' + esc(i.sub) : '') + '</span><br>' +
            '<span class="bcv-dot ' + dotClass(i.power) + '"></span>' + esc(pwText(i.power));
        tipEl.style.display = 'block';
        var hr = host.getBoundingClientRect();
        var rx = p.x + 16, ry = p.y + 16;
        var tw = tipEl.offsetWidth, th = tipEl.offsetHeight;
        if (rx + tw > vw - 4) rx = p.x - tw - 12;
        if (ry + th > vh - 4) ry = p.y - th - 12;
        tipEl.style.left = Math.max(2, rx) + 'px';
        tipEl.style.top = Math.max(2, ry) + 'px';
    }

    var drag = null;
    dom.addEventListener('contextmenu', function (e) { e.preventDefault(); });
    dom.addEventListener('pointerdown', function (e) {
        var p = localXY(e);
        computeCorners();
        var ci = cornerAt(p);
        var hit = ci < 0 ? pick(p) : null;
        var roll = e.shiftKey || e.button === 2;
        if (ci >= 0 || e.button === 2 || e.button === 1) {
            drag = { mode: roll ? 'roll' : 'rot', x: p.x, y: p.y, moved: 0 };
        } else if (!hit) {
            drag = { mode: roll ? 'roll' : 'rot', x: p.x, y: p.y, moved: 0 };
        } else {
            drag = { mode: 'click', x: p.x, y: p.y, moved: 0, hit: hit };
        }
        try { dom.setPointerCapture(e.pointerId); } catch (_) { /* ignore */ }
        dom.style.cursor = drag.mode === 'click' ? 'pointer' : 'grabbing';
        if (tipEl) tipEl.style.display = 'none';
        e.preventDefault();
    });
    dom.addEventListener('pointermove', function (e) {
        var p = localXY(e);
        if (drag && drag.mode !== 'click') {
            var dx = p.x - drag.x, dy = p.y - drag.y;
            drag.moved += Math.abs(dx) + Math.abs(dy);
            if (drag.mode === 'roll') {
                var cx = vw / 2, cy = vh / 2;
                var a0 = Math.atan2(drag.y - cy, drag.x - cx), a1 = Math.atan2(p.y - cy, p.x - cx);
                var da = a1 - a0;
                if (da > Math.PI) da -= 2 * Math.PI; if (da < -Math.PI) da += 2 * Math.PI;
                rollBy(da);
            } else {
                rotateBy(dx, dy);
            }
            drag.x = p.x; drag.y = p.y;
            return;
        }
        if (drag && drag.mode === 'click') {
            drag.moved += Math.abs(p.x - drag.x) + Math.abs(p.y - drag.y);
            drag.x = p.x; drag.y = p.y;
            return;
        }
        // наведение
        var ci = cornerAt(p);
        if (ci !== hoverCorner) { hoverCorner = ci; dirty = true; }
        if (ci >= 0) {
            dom.style.cursor = 'grab';
            if (hovered) { hovered = null; dirty = true; }
            showTip(null, p);
            return;
        }
        var m = pick(p);
        var hm = m && !m.userData.isBody ? m : null;
        if (hm !== hovered) { hovered = hm; dirty = true; }
        dom.style.cursor = hm ? 'pointer' : 'default';
        showTip(m, p);
    });
    function endDrag(e) {
        if (!drag) return;
        var d = drag; drag = null;
        try { dom.releasePointerCapture(e.pointerId); } catch (_) { /* ignore */ }
        dom.style.cursor = 'default';
        if (d.mode === 'click' && d.moved < 6) {
            select(d.hit, true);
        } else if (d.mode !== 'click' && d.moved < 4) {
            // клик по фону без перетаскивания снимает выделение
            select(null, true);
        }
    }
    dom.addEventListener('pointerup', endDrag);
    dom.addEventListener('pointercancel', endDrag);
    dom.addEventListener('pointerleave', function () { hoverCorner = -1; if (!drag && hovered) { hovered = null; } if (tipEl) tipEl.style.display = 'none'; dirty = true; });
    dom.addEventListener('wheel', function (e) {
        e.preventDefault();
        var k = e.deltaMode === 1 ? 33 : 1;
        setDist(dist * Math.exp(e.deltaY * k * 0.0011));
    }, { passive: false });

    // ---------------------------------------------------------- кнопки
    function bindToolbar() {
        var btns = document.querySelectorAll('[data-bcv-view]');
        for (var i = 0; i < btns.length; i++) {
            btns[i].addEventListener('click', function () {
                var v = this.getAttribute('data-bcv-view');
                if (v === 'reset') { animateTo(VIEWS.front, 1400); }
                else if (v === 'roll-l') { animateTo(rig.quaternion.clone().premultiply(qAxis(AZ, 90))); }
                else if (v === 'roll-r') { animateTo(rig.quaternion.clone().premultiply(qAxis(AZ, -90))); }
                else if (v === 'zoom-in') { setDist(dist / 1.25); }
                else if (v === 'zoom-out') { setDist(dist * 1.25); }
                else if (VIEWS[v]) { animateTo(VIEWS[v]); }
            });
        }
    }
    bindToolbar();

    // --------------------------------------------------------------- старт
    resize();
    rig.quaternion.copy(VIEWS.iso);   // старт — изометрия, видны лицевая сторона и верх
    setDist(DEF_DIST);
    viewChanged();
    renderInfo(null);
    requestAnimationFrame(frame);
    // для отладки
    window.BCV3D = { getAnim: function () { return anim; }, rig: rig, camera: camera, select: select, animateTo: animateTo, VIEWS: VIEWS, modules: modules, setDist: setDist, scene: scene, renderer: renderer };
})();
