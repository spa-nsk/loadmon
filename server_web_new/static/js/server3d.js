/*
 * server3d.js - 3D-визуализация сервера по вендору/модели для
 * /sprsrv/hwconfvisual/:id. Бренды: HPE (ProLiant), Dell (PowerEdge),
 * Supermicro, DEPO (Storm), Lenovo, прочие. Rack 1U/2U/3U+ и блейд.
 *
 * Зависит от three.js (static/js/three.min.js), данные: window.HCV_DATA
 * (controllers/sprhwconfvisual.go: hwConfVisualData).
 *
 * Стороны: спереди (дисковые корзины, панель управления), сзади (БП,
 * PCI-адаптеры, порты управления iLO/iDRAC/IPMI), сверху (кожух снимается:
 * вентиляторы, процессоры, модули памяти).
 *
 * Управление как в 3D-виде блейд-корзины: колесо - масштаб; ЛКМ - pan
 * (сдвиг при увеличении); угловые маркеры / СКМ - вращение; Shift/ПКМ -
 * крен; наведение - подсветка и подсказка; клик - параметры компонента.
 */
(function () {
    'use strict';

    var DATA = window.HCV_DATA;
    var host = document.getElementById('hcv3dStage');
    if (!host) return;
    var detailEl = document.getElementById('hcvDetail');
    var listEl = document.getElementById('hcvVisible');
    var tipEl = document.getElementById('hcvTip');
    var zoomEl = document.getElementById('hcvZoom');
    var sideEl = document.getElementById('hcvSideLabel');
    var lidBtn = document.getElementById('hcvLidBtn');

    function fail(msg) { host.innerHTML = '<div class="hcv-empty-hint p-4">' + msg + '</div>'; }
    if (!DATA || !DATA.server) return fail('Нет данных сервера.');
    if (!window.THREE) return fail('Не загружена библиотека three.js.');
    var THREE = window.THREE;

    // ------------------------------------------------------------------ данные
    var SRV = DATA.server, COMPS = DATA.components || [];
    var BLADE = SRV.formFactor === 2;
    var U = Math.max(1, Math.min(8, SRV.sizeU || 2));
    var C_ON = '#35e06a', C_STBY = '#ffab1a', C_ERR = '#ff453a', C_UID = '#3da5ff', C_UNK = '#69707a';

    var SKINS = {
        hpe: { key: 'hpe', brand: 'HPE', series: 'ProLiant', bezel: ['#3b3e44', '#1a1c20'], body: '#25282c', ear: ['#2d3035', '#16181b'], cage: '#0d0e10', accent: '#01a982', mgmt: 'iLO', cols1: 4, drv: { body: ['#4a4d54', '#222428'], latch: ['#5a5e66', '#2c2f34'], led1: '#2fd36a', led2: '#ffab1a' } },
        dell: { key: 'dell', brand: 'DELL', series: 'PowerEdge', bezel: ['#34373d', '#18191c'], body: '#26282c', ear: ['#d6dade', '#8b9199'], cage: '#0b0c0e', accent: '#0076ce', mgmt: 'iDRAC', cols1: 5, drv: { body: ['#3f4248', '#1c1e21'], latch: ['#d0d4d9', '#7f858d'], led1: '#2fd36a', led2: '#ffab1a' } },
        supermicro: { key: 'supermicro', brand: 'SUPERMICRO', series: 'SuperServer', bezel: ['#222428', '#0d0e10'], body: '#1e2023', ear: ['#a2a8b0', '#5d636b'], cage: '#08090b', accent: '#e2231a', mgmt: 'IPMI', cols1: 5, drv: { body: ['#34373c', '#16171a'], latch: ['#aeb4bc', '#6a7078'], led1: '#3da5ff', led2: '#ff4d4d' } },
        depo: { key: 'depo', brand: 'DEPO', series: 'Storm', bezel: ['#2b2e34', '#121418'], body: '#202226', ear: ['#3a3e46', '#1c1e22'], cage: '#0b0c0e', accent: '#6b67e9', mgmt: 'BMC', cols1: 4, drv: { body: ['#3a3d44', '#191b1e'], latch: ['#6b67e9', '#3a3880'], led1: '#6bd36b', led2: '#ffab1a' } },
        lenovo: { key: 'lenovo', brand: 'Lenovo', series: 'ThinkSystem', bezel: ['#25272b', '#0f1012'], body: '#202225', ear: ['#2b2d31', '#141518'], cage: '#08080a', accent: '#e2231a', mgmt: 'XCC', cols1: 5, drv: { body: ['#3a3d42', '#17181b'], latch: ['#8a9099', '#4a4e55'], led1: '#2fd36a', led2: '#ff6a3d' } },
        generic: { key: 'generic', brand: '', series: '', bezel: ['#50545b', '#2d3035'], body: '#2d3035', ear: ['#a3a9b1', '#6c7179'], cage: '#15161a', accent: '#8d98a6', mgmt: 'BMC', cols1: 4, drv: { body: ['#4a4e55', '#202327'], latch: ['#9096a0', '#5a5f66'], led1: '#2fd36a', led2: '#ffab1a' } }
    };
    function skinOf() {
        var s = ((SRV.vendor || '') + ' ' + (SRV.model || '') + ' ' + (SRV.name || '')).toLowerCase();
        if (/hpe|hp |hewlett|proliant|\bdl\d|\bml\d|\bbl\d|synergy/.test(s)) return SKINS.hpe;
        if (/dell|poweredge|\br\d{3}/.test(s)) return SKINS.dell;
        if (/supermicro|super micro|super-micro|\bsys-|\bsys \d|\bsse-|\bssg-/.test(s)) return SKINS.supermicro;
        if (/\bdepo\b|\bstorm\s*[345]\d{3}|\bstorm\b/.test(s)) return SKINS.depo;
        if (/lenovo|thinksystem|\bsr\d{3}/.test(s)) return SKINS.lenovo;
        return SKINS.generic;
    }
    var SK = skinOf();
    var BRAND = SK.brand || (SRV.vendor || 'Server');

    // размеры, мм
    var BW = BLADE ? 56 : 432, BH = BLADE ? 217 : U * 44.45 - 1.5, BD = BLADE ? 517 : (U >= 2 ? 740 : 700);
    var FW = BLADE ? 56 : 482, FP = 4; // ширина лицевой панели, толщина

    var KIND_TITLE = {
        server: 'Сервер', disk: 'Диск', 'disk-empty': 'Пустой отсек диска', cpu: 'Процессор', ram: 'Модуль памяти',
        'ram-empty': 'Свободный слот памяти', card: 'Адаптер', psu: 'Блок питания', fan: 'Вентилятор',
        panel: 'Панель управления', ports: 'Порты ввода-вывода', mgmt: 'Контроллер управления', slot: 'Слот расширения', conn: 'Разъём корзины'
    };
    var GROUP_TITLE = {
        disk: 'Диски', panel: 'Панель управления', psu: 'Блоки питания', card: 'PCI-адаптеры', ports: 'Порты ввода-вывода', mgmt: 'Контроллер управления',
        fan: 'Вентиляторы', cpu: 'Процессоры', ram: 'Модули памяти', conn: 'Разъём корзины', slot: 'Слоты расширения'
    };
    var SIDE_KINDS = {
        front: ['disk', 'panel'],
        rear: BLADE ? ['conn'] : ['psu', 'card', 'ports', 'mgmt'],
        top: ['fan', 'cpu', 'ram', 'card']
    };
    var SIDE_NAME = { front: 'Спереди', rear: 'Сзади', top: 'Сверху' };

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (ch) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
        });
    }
    // Коды состояния: 1 включён, 2 выключен, 3 горячий резерв (диск), 4 отказ, иначе не определено
    function pwColor(p) { return p === 1 ? C_ON : (p === 2 ? C_STBY : (p === 3 ? C_UID : (p === 4 ? C_ERR : C_UNK))); }
    function pwText(p) { return p === 1 ? 'Включён' : (p === 2 ? 'Выключен' : (p === 3 ? 'Горячий резерв' : (p === 4 ? 'Отказ' : 'Не определено'))); }
    function dotClass(p) { return p === 1 ? 'hcv-dot-on' : (p === 2 ? 'hcv-dot-stby' : (p === 3 ? 'hcv-dot-spare' : (p === 4 ? 'hcv-dot-err' : 'hcv-dot-unk'))); }
    // цвет LED по health из OneView ("OK", "Warning", "Critical", ...); пусто - штатно
    function stColor(st) {
        var s = String(st || '').toLowerCase();
        if (/crit|fail|error/.test(s)) return C_ERR;
        if (/warn|degrad/.test(s)) return C_STBY;
        return C_ON;
    }
    function compsOf(kind) { return COMPS.filter(function (c) { return c.kind === kind; }); }
    function hash(n) { n = (Math.imul(n + 1, 2654435761) >>> 0); return (n % 1000) / 1000; }

    // ---------------------------------------------------------- 2D-примитивы
    function mk(w, h) {
        var c = document.createElement('canvas');
        c.width = Math.max(2, Math.round(w)); c.height = Math.max(2, Math.round(h));
        return c;
    }
    function rr(x, X, Y, w, h, r) {
        r = Math.max(0, Math.min(r, w / 2, h / 2));
        x.beginPath(); x.moveTo(X + r, Y);
        x.arcTo(X + w, Y, X + w, Y + h, r); x.arcTo(X + w, Y + h, X, Y + h, r);
        x.arcTo(X, Y + h, X, Y, r); x.arcTo(X, Y, X + w, Y, r); x.closePath();
    }
    function lg(x, x0, y0, x1, y1, stops) {
        var g = x.createLinearGradient(x0, y0, x1, y1);
        for (var i = 0; i < stops.length; i++) g.addColorStop(stops[i][0], stops[i][1]);
        return g;
    }
    function fillRR(x, X, Y, w, h, r, f) { rr(x, X, Y, w, h, r); x.fillStyle = f; x.fill(); }
    function strokeRR(x, X, Y, w, h, r, s, lw) { rr(x, X, Y, w, h, r); x.strokeStyle = s; x.lineWidth = lw || 1; x.stroke(); }
    function txt(x, s, X, Y, px, color, align, weight) {
        x.font = (weight || '600') + ' ' + Math.round(px) + 'px "Segoe UI", Arial, sans-serif';
        x.fillStyle = color; x.textAlign = align || 'center'; x.textBaseline = 'middle';
        x.fillText(s, X, Y);
    }
    // «Живой» светодиод (включённое оборудование) мерцает; статусные (ожидание/UID/ошибка) — нет.
    function ledIsLive(color) {
        return !!(color && color !== C_STBY && color !== C_ERR && color !== C_UID && color !== C_UNK &&
            color !== '#000' && color !== '#1d2024');
    }
    function led(x, cx, cy, r, color, on) {
        var live = !!on && ledIsLive(color);
        if (live) { var lc = x.canvas; (lc.__leds || (lc.__leds = [])).push({ cx: cx, cy: cy, r: r, color: color }); }
        x.save();
        if (on) { x.shadowColor = color; x.shadowBlur = r * (live ? 1.4 : 2.6); }
        if (live) x.globalAlpha = 0.72; // остальная яркость добавляется мерцающим слоем (attachLeds)
        x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
        x.fillStyle = on ? color : '#1d2024'; x.fill();
        x.restore();
        x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
        x.strokeStyle = 'rgba(0,0,0,.55)'; x.lineWidth = Math.max(1, r * 0.22); x.stroke();
    }
    function vents(x, X, Y, w, h, n, color) {
        for (var i = 0; i < n; i++) {
            var vy = Y + (i + 0.5) * (h / n);
            fillRR(x, X, vy - h / n * 0.22, w, h / n * 0.44, h / n * 0.2, color || '#07080a');
        }
    }
    function dots(x, X, Y, w, h, step, r, color) {
        x.fillStyle = color || 'rgba(0,0,0,.6)';
        for (var yy = Y + step / 2, k = 0; yy < Y + h; yy += step * 0.87, k++) {
            for (var xx = X + step / 2 + (k % 2 ? step / 2 : 0); xx < X + w; xx += step) {
                x.beginPath(); x.arc(xx, yy, r, 0, Math.PI * 2); x.fill();
            }
        }
    }
    function noise(x, w, h, n) {
        for (var i = 0; i < n; i++) {
            x.fillStyle = 'rgba(255,255,255,' + (Math.random() * 0.035).toFixed(3) + ')';
            x.fillRect(Math.random() * w, Math.random() * h, 1 + Math.random() * 2, 1);
        }
    }
    function powerButton(x, cx, cy, r, p) {
        var col = p === 1 ? C_ON : (p === 2 ? C_STBY : null);
        x.lineWidth = Math.max(1.1, r * 0.22);
        x.save();
        if (col) { x.shadowColor = col; x.shadowBlur = r * 0.9; }
        x.strokeStyle = col || '#2b2f35'; x.beginPath(); x.arc(cx, cy, r * 1.18, 0, Math.PI * 2); x.stroke();
        x.restore();
        var g = x.createRadialGradient(cx - r * 0.3, cy - r * 0.35, r * 0.1, cx, cy, r);
        g.addColorStop(0, '#f1f3f5'); g.addColorStop(1, '#868c95');
        x.fillStyle = g; x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2); x.fill();
        x.strokeStyle = '#2a2d32'; x.lineWidth = Math.max(1, r * 0.18); x.lineCap = 'round';
        x.beginPath(); x.arc(cx, cy + r * 0.05, r * 0.5, -Math.PI * 0.35, Math.PI * 1.35, false); x.stroke();
        x.beginPath(); x.moveTo(cx, cy - r * 0.62); x.lineTo(cx, cy - r * 0.05); x.stroke();
        x.lineCap = 'butt';
    }
    function drawLogo(x, kind, X, Y, h) {
        var fs = Math.max(8, h * 0.42);
        if (kind === 'hpe') {
            var w = h * 1.8;
            x.strokeStyle = '#01a982'; x.lineWidth = Math.max(1.6, h * 0.14);
            x.strokeRect(X + x.lineWidth / 2, Y + x.lineWidth / 2, w - x.lineWidth, h - x.lineWidth);
            txt(x, 'hpe', X + w / 2, Y + h / 2 + 1, fs, '#f2f5f4', 'center', '700');
        } else if (kind === 'dell') {
            var r = h * 0.5;
            x.strokeStyle = '#0076ce'; x.lineWidth = Math.max(1.6, h * 0.12);
            x.beginPath(); x.arc(X + r, Y + r, r - x.lineWidth / 2, 0, Math.PI * 2); x.stroke();
            txt(x, 'DELL', X + r, Y + r + 1, fs * 0.66, '#e8f1fb', 'center', '800');
        } else if (kind === 'supermicro') {
            txt(x, 'SUPERMICRO', X, Y + h / 2, fs * 1.0, '#eceff2', 'left', '800');
            x.fillStyle = '#e2231a'; x.fillRect(X, Y + h * 0.92, h * 1.2, Math.max(2, h * 0.08));
        } else if (kind === 'lenovo') {
            txt(x, 'Lenovo', X, Y + h / 2, fs * 1.1, '#f4f4f4', 'left', '800');
        } else if (kind === 'depo') {
            var r2 = h * 0.42, cx = X + r2, cy = Y + h / 2, segs = ['#85cfff', '#6b67e9', '#7ac943', '#f5c518'];
            for (var i = 0; i < 4; i++) {
                x.beginPath(); x.moveTo(cx, cy);
                x.arc(cx, cy, r2, -Math.PI / 2 + i * Math.PI / 2, -Math.PI / 2 + (i + 1) * Math.PI / 2);
                x.closePath(); x.fillStyle = segs[i]; x.fill();
            }
            x.beginPath(); x.arc(cx, cy, r2 * 0.38, 0, Math.PI * 2); x.fillStyle = '#121418'; x.fill();
            txt(x, 'DEPO', X + r2 * 2.2, Y + h / 2, fs, '#f2f4f8', 'left', '800');
        }
    }
    function modelText() { return String(SRV.model || '').replace(/^(HPE?|Dell|Lenovo|DEPO|Supermicro)\s+/i, ''); }

    // ---------------------------------------------------------------- раскладки
    // Лицевая сторона. u: 0..1 по ширине FW, v: 0..1 сверху вниз.
    function bayConfig(diskCount) {
        var cfg;
        // Блейд: столько отсеков, сколько дисков в карточке (минимум 2 — типичный BL460).
        if (BLADE) return { flat: false, cols: 1, rows: Math.max(2, diskCount || 2), blade: true };
        if (U <= 1) cfg = { flat: true, cols: SK.cols1, rows: 2 };
        else if (U === 2) cfg = { flat: false, cols: 24, rows: 1 };
        else cfg = { flat: false, cols: 12, rows: 2 };
        if (diskCount > cfg.cols * cfg.rows) cfg.cols = Math.max(cfg.cols, Math.ceil(diskCount / cfg.rows));
        return cfg;
    }
    var DISKS = compsOf('disk');
    var CFG = bayConfig(DISKS.length);
    // сопоставление диск -> отсек: по номеру bay из Slot ("1I:1:5"), если номера уникальны, иначе по порядку
    function diskBayMap() {
        var map = {}, used = {}, ok = true;
        DISKS.forEach(function (d, i) {
            var m = String(d.slot || '').match(/(\d+)\s*$/);
            var b = m ? parseInt(m[1], 10) - 1 : -1;
            if (b < 0 || used[b] || b >= CFG.cols * CFG.rows) ok = false;
            used[b] = true; map[i] = b;
        });
        if (!ok) DISKS.forEach(function (d, i) { map[i] = i; });
        return map;
    }
    var DMAP = diskBayMap();
    var diskAtBay = {};
    DISKS.forEach(function (d, i) { diskAtBay[DMAP[i]] = d; });

    function layoutFront() {
        var L = { bays: [], panel: null };
        if (BLADE) {
            var nBay = CFG.rows, bh = Math.min(0.12, 0.42 / nBay), gap = 0.012;
            var v0 = 0.06;
            for (var i = 0; i < nBay; i++) {
                L.bays.push({ n: i + 1, u0: 0.12, u1: 0.88, v0: v0 + i * (bh + gap), v1: v0 + i * (bh + gap) + bh });
            }
            var panelTop = v0 + nBay * (bh + gap) + 0.04;
            L.panel = { u0: 0.2, u1: 0.8, v0: Math.min(panelTop, 0.72), v1: Math.min(panelTop + 0.22, 0.94) };
            return L;
        }
        var cx0 = 0.145, cx1 = 0.80, cy0 = 0.1, cy1 = 0.9;
        var cw = (cx1 - cx0) / CFG.cols, chh = (cy1 - cy0) / CFG.rows, g = CFG.flat ? 0.0012 : 0.0006;
        for (var r = 0; r < CFG.rows; r++) for (var c = 0; c < CFG.cols; c++) {
            L.bays.push({ n: r * CFG.cols + c + 1, u0: cx0 + c * cw + g, u1: cx0 + (c + 1) * cw - g, v0: cy0 + r * chh + g * 3, v1: cy0 + (r + 1) * chh - g * 3 });
        }
        L.panel = { u0: 0.058, u1: 0.136, v0: 0.1, v1: 0.9 };
        return L;
    }
    var LF = layoutFront();

    // Карты расширения: raid, fc, nic, pci -> группы по слоту.
    // Type=9 (Devices OneView) часто дублирует Ethernet/Smart Array, уже попавшие
    // в nic/raid — такие pci отбрасываем, иначе они забивают слоты и «теряют»
    // настоящие GPU/HBA, а на 1U (всего 2 слота сзади) PCI некуда поставить.
    function buildCards() {
        var cards = [], byKey = {}, slotTaken = {};
        function slotBase(slot) { return String(slot || '').split(/,\s*порт/i)[0].replace(/\s+/g, ' ').trim(); }
        function isDupPciModel(model) {
            return /\(ethernet\)|\(storagecontroller\)|\(fibrechannel\)|\(fiberchannel\)|\(network\)/i.test(model || '')
                || /smart\s*array/i.test(model || '');
        }
        ['raid', 'fc', 'nic', 'pci'].forEach(function (kind) {
            compsOf(kind).forEach(function (c, i) {
                var base = slotBase(c.slot) || ('—' + kind + i);
                if (kind === 'pci') {
                    if (slotTaken[base]) return;
                    if (isDupPciModel(c.model)) return;
                }
                var key = kind + '|' + (kind === 'nic' || kind === 'fc' ? base : base + '#' + i);
                var card = byKey[key];
                if (!card) {
                    card = { kind: kind, key: key, slot: base, comps: [] };
                    byKey[key] = card;
                    cards.push(card);
                    slotTaken[base] = true;
                }
                card.comps.push(c);
            });
        });
        return cards;
    }
    var CARDS = buildCards();
    var CARD_KIND = { raid: 'RAID-контроллер', fc: 'FC-адаптер', nic: 'Сетевой адаптер', pci: 'PCI-устройство' };

    function layoutRear() {
        var L = { slots: [], psu: [], ports: null, ilo: null, conn: null };
        if (BLADE) { L.conn = { u0: 0.1, u1: 0.9, v0: 0.25, v1: 0.78 }; return L; }
        var i, r, c;
        if (U === 1) {
            for (i = 0; i < 2; i++) L.slots.push({ n: i + 1, u0: 0.03 + i * 0.245, u1: 0.03 + i * 0.245 + 0.235, v0: 0.1, v1: 0.9 });
            // зона портов делится: порты ввода-вывода сверху, выделенный порт управления (iLO/iDRAC) снизу
            L.ports = { u0: 0.53, u1: 0.665, v0: 0.1, v1: 0.5 };
            L.ilo = { u0: 0.53, u1: 0.665, v0: 0.52, v1: 0.9 };
            L.psu.push({ n: 1, u0: 0.685, u1: 0.82, v0: 0.1, v1: 0.9 });
            L.psu.push({ n: 2, u0: 0.835, u1: 0.97, v0: 0.1, v1: 0.9 });
        } else {
            var rows = U === 2 ? 3 : 4;
            for (c = 0; c < 2; c++) for (r = 0; r < rows; r++) {
                var hh = 0.88 / rows;
                L.slots.push({ n: c * rows + r + 1, u0: 0.025 + c * 0.255, u1: 0.025 + c * 0.255 + 0.245, v0: 0.06 + r * hh + 0.006, v1: 0.06 + (r + 1) * hh - 0.006 });
            }
            L.ports = { u0: 0.545, u1: 0.685, v0: 0.06, v1: 0.68 };
            L.ilo = { u0: 0.545, u1: 0.685, v0: 0.70, v1: 0.94 };
            L.psu.push({ n: 1, u0: 0.705, u1: 0.975, v0: 0.06, v1: 0.485 });
            L.psu.push({ n: 2, u0: 0.705, u1: 0.975, v0: 0.515, v1: 0.94 });
        }
        return L;
    }
    var LR = layoutRear();

    // Вид сверху (кожух снят): u слева направо, v от задней стенки (0) к передней (1)
    var CPUN = Math.max(1, Math.min(4, SRV.cpuCount || compsOf('cpu').length || 1));
    var RAMS = compsOf('ram');
    var FAN_COMPS_EARLY = compsOf('fan');
    function layoutTop() {
        var L = { fans: [], cpus: [], dimms: [], cards: [] };
        var i;
        if (BLADE) {
            // Раскладка по фактическим компонентам карточки (раньше было жёстко
            // 16 DIMM / 2 мезонина / 0 вентиляторов — лишние NIC/FC/RAID/RAM
            // из редактирования на 3D не попадали).
            var v = 0.03;
            var fanN = FAN_COMPS_EARLY.length;
            if (fanN > 0) {
                var fSpan = Math.min(0.14, 0.04 + fanN * 0.02), fw = 0.88 / fanN;
                for (i = 0; i < fanN; i++) {
                    L.fans.push({ n: i + 1, u0: 0.06 + i * fw + 0.004, u1: 0.06 + (i + 1) * fw - 0.004, v0: v, v1: v + fSpan });
                }
                v += fSpan + 0.02;
            }
            var banks = 2;
            var perBank = Math.max(8, Math.ceil(Math.max(RAMS.length, 1) / banks));
            if (RAMS.length === 0) perBank = 8; // пустая эстетика 2×8
            var dimmH = Math.min(0.018, 0.28 / perBank), dimmGap = 0.0035;
            var bankH = perBank * (dimmH + dimmGap);
            for (i = 0; i < banks * perBank; i++) {
                var bank = Math.floor(i / perBank), row = i % perBank;
                L.dimms.push({
                    n: i + 1, wide: true,
                    u0: 0.08, u1: 0.92,
                    v0: v + bank * (bankH + 0.02) + row * (dimmH + dimmGap),
                    v1: v + bank * (bankH + 0.02) + row * (dimmH + dimmGap) + dimmH
                });
            }
            v += banks * (bankH + 0.02) + 0.01;
            var nCpu = Math.max(1, Math.min(2, CPUN));
            var cpuH = Math.min(0.11, (0.72 - v - 0.02) / Math.max(1, nCpu + 0.5));
            for (i = 0; i < nCpu; i++) {
                L.cpus.push({ n: i + 1, u0: 0.18, u1: 0.82, v0: v + i * (cpuH + 0.015), v1: v + i * (cpuH + 0.015) + cpuH });
            }
            v += nCpu * (cpuH + 0.015) + 0.02;
            // Мезонины: по одному слоту на каждую карту (NIC/FC/RAID/PCI); минимум 2 пустых.
            var nCard = Math.max(2, CARDS.length);
            var remain = Math.max(0.08, 0.98 - v);
            var cardH = Math.min(0.09, (remain - 0.008 * (nCard - 1)) / nCard);
            for (i = 0; i < nCard; i++) {
                L.cards.push({
                    n: i + 1,
                    u0: 0.1, u1: 0.9,
                    v0: v + i * (cardH + 0.008),
                    v1: v + i * (cardH + 0.008) + cardH
                });
            }
            // Если блоков много — пропорционально вписать в 0..0.98.
            var bottom = 0;
            ['fans', 'dimms', 'cpus', 'cards'].forEach(function (k) {
                L[k].forEach(function (r) { if (r.v1 > bottom) bottom = r.v1; });
            });
            if (bottom > 0.98) {
                var sc = 0.96 / bottom;
                ['fans', 'dimms', 'cpus', 'cards'].forEach(function (k) {
                    L[k].forEach(function (r) { r.v0 *= sc; r.v1 *= sc; });
                });
            }
            return L;
        }
        var nf = U === 1 ? 7 : 6, fw = 0.92 / nf;
        for (i = 0; i < nf; i++) L.fans.push({ n: i + 1, u0: 0.04 + i * fw + 0.004, u1: 0.04 + (i + 1) * fw - 0.004, v0: 0.69, v1: 0.84 });
        var centers = CPUN === 1 ? [0.5] : (CPUN === 2 ? [0.3, 0.7] : (CPUN === 3 ? [0.2, 0.5, 0.8] : [0.14, 0.38, 0.62, 0.86]));
        var perBank = CPUN >= 3 ? 4 : Math.min(8, Math.max(6, Math.ceil(RAMS.length / (CPUN * 2))));
        var sw = CPUN >= 3 ? 0.014 : 0.0105, gp = 0.0015, n = 0;
        centers.forEach(function (cu, k) {
            L.cpus.push({ n: k + 1, u0: cu - 0.045, u1: cu + 0.045, v0: 0.38, v1: 0.52 });
            [-1, 1].forEach(function (side) {
                for (var j = 0; j < perBank; j++) {
                    var off = 0.055 + j * (sw + gp);
                    var u1 = side < 0 ? cu - off : cu + off + sw;
                    var u0 = side < 0 ? cu - off - sw : cu + off;
                    L.dimms.push({ n: ++n, cpu: k + 1, u0: u0, u1: u1, v0: 0.2, v1: 0.6 });
                }
            });
        });
        // Места под «лишние» PCIe добавляются после split CARDS → REAR/TOP.
        return L;
    }
    var LT = layoutTop();
    // Сзади — сколько влезает в раскладку; остаток (PCI/GPU/riser) — сверху при снятом кожухе.
    var REAR_CARDS = BLADE ? [] : CARDS.slice(0, LR.slots.length);
    var TOP_CARDS = BLADE ? CARDS.slice() : CARDS.slice(REAR_CARDS.length);
    if (!BLADE && TOP_CARDS.length) {
        var tc = TOP_CARDS.length, cw = Math.min(0.28, 0.9 / Math.max(tc, 1));
        for (var ti = 0; ti < tc; ti++) {
            LT.cards.push({
                n: ti + 1,
                u0: 0.04 + ti * (cw + 0.02), u1: 0.04 + ti * (cw + 0.02) + cw,
                v0: 0.04, v1: 0.14
            });
        }
    }

    // -------------------------------------------------------- текстуры плит/корпуса
    var TS = 4; // px/mm для плит
    function earSlots(x, X, Y, w, h) {
        for (var k = 0; k < U; k++) {
            var cy = Y + (k + 0.5) * h / U;
            x.fillStyle = 'rgba(0,0,0,.7)';
            x.beginPath(); x.ellipse(X + w / 2, cy, w * 0.2, h / U * 0.13, 0, 0, Math.PI * 2); x.fill();
        }
    }
    function texFrontPlate() {
        var w = Math.round(FW * TS), h = Math.round(BH * TS), c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, 0, h, [[0, SK.bezel[0]], [1, SK.bezel[1]]]); x.fillRect(0, 0, w, h);
        noise(x, w, h, 12000);
        x.fillStyle = 'rgba(255,255,255,.1)'; x.fillRect(w * 0.01, 2, w * 0.98, 2);
        if (BLADE) {
            fillRR(x, w * 0.1, h * 0.46, w * 0.8, h * 0.012, 2, SK.accent);
            vents(x, w * 0.16, h * 0.76, w * 0.68, h * 0.11, 8, '#07080a');
            drawLogo(x, SK.key === 'generic' ? '' : SK.key, w * 0.14, h * 0.915, Math.min(h * 0.045, w * 0.2));
            return c;
        }
        var ew = w * 0.052, earFill = lg(x, 0, 0, 0, h, [[0, SK.ear[0]], [1, SK.ear[1]]]);
        fillRR(x, 0, 1, ew, h - 2, 3, earFill); fillRR(x, w - ew, 1, ew, h - 2, 3, earFill);
        earSlots(x, 0, 0, ew, h); earSlots(x, w - ew, 0, ew, h);
        if (SK.accent && SK.key !== 'dell') { x.fillStyle = SK.accent; x.fillRect(w * 0.06, 3, w * 0.88, Math.max(2, h * 0.012)); }
        // корзина дисков (фон)
        fillRR(x, w * 0.145 - 3, h * 0.1 - 3, w * 0.655 + 6, h * 0.8 + 6, 4, SK.cage);
        // правая панель: вентиляция + порты
        var vx = w * 0.815, vw = w * 0.14;
        dots(x, vx, h * 0.12, vw, h * 0.76, Math.max(8, h * 0.07), Math.max(1.4, h * 0.012), 'rgba(0,0,0,.65)');
        drawLogo(x, SK.key === 'generic' ? '' : SK.key, vx + vw * 0.05, h * 0.1, Math.min(h * (U === 1 ? 0.2 : 0.12), vw * 0.4));
        // USB / VGA гнёзда
        fillRR(x, vx + vw * 0.1, h * 0.82, vw * 0.22, h * 0.07, 2, '#08090b');
        fillRR(x, vx + vw * 0.4, h * 0.82, vw * 0.22, h * 0.07, 2, '#08090b');
        fillRR(x, vx + vw * 0.68, h * 0.82, vw * 0.24, h * 0.07, 2, '#08090b');
        // логотип слева внизу, модель справа
                txt(x, modelText(), w * 0.955, h * 0.95, Math.max(12, h * (U === 1 ? 0.15 : 0.07)), 'rgba(235,238,242,.5)', 'right', '500');
        return c;
    }
    function texRearPlate() {
        var w = Math.round(BW * TS), h = Math.round(BH * TS), c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, w, h, [[0, SK.body], [1, '#101215']]); x.fillRect(0, 0, w, h);
        noise(x, w, h, 12000);
        if (BLADE) return c;
        function rect(r) { fillRR(x, r.u0 * w - 2, r.v0 * h - 2, (r.u1 - r.u0) * w + 4, (r.v1 - r.v0) * h + 4, 4, '#08090b'); }
        LR.slots.forEach(rect); LR.psu.forEach(rect); if (LR.ports) rect(LR.ports); if (LR.ilo) rect(LR.ilo);
        dots(x, w * 0.02, h * 0.02, w * 0.96, h * 0.02, 10, 1.2, 'rgba(0,0,0,.35)');
        return c;
    }
    function texSide(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, 0, h, [[0, SK.body], [0.5, '#101215'], [1, SK.body]]); x.fillRect(0, 0, w, h);
        noise(x, w, h, 12000);
        x.strokeStyle = 'rgba(255,255,255,.07)'; x.lineWidth = 2; x.strokeRect(8, 8, w - 16, h - 16);
        // вентиляционные прорези
        x.fillStyle = 'rgba(0,0,0,.55)';
        for (var i = 0; i < 28; i++) fillRR(x, w * 0.1 + i * (w * 0.03), h * 0.2, w * 0.012, h * 0.6, 3, 'rgba(0,0,0,.5)');
        return c;
    }
    function texLid() {
        var s = 2, w = Math.round(BW * s), h = Math.round(BD * s), c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, w, h, [[0, SK.body], [1, '#0d0f11']]); x.fillRect(0, 0, w, h);
        noise(x, w, h, 18000);
        // решётки над вентиляторами и процессорами (v от заднего края)
        x.save();
        dots(x, w * 0.05, h * 0.06, w * 0.9, h * 0.18, 12, 2.4, 'rgba(0,0,0,.65)');
        dots(x, w * 0.05, h * 0.62, w * 0.9, h * 0.24, 12, 2.4, 'rgba(0,0,0,.65)');
        x.restore();
        x.strokeStyle = 'rgba(255,255,255,.08)'; x.lineWidth = 3; x.strokeRect(10, 10, w - 20, h - 20);
        // наклейка
        fillRR(x, w * 0.12, h * 0.34, w * 0.76, h * 0.22, 10, 'rgba(235,238,242,.9)');
        drawLogo(x, SK.key === 'generic' ? '' : SK.key, w * 0.16, h * 0.355, Math.min(h * 0.06, w * 0.18));
        txt(x, (SRV.vendor || '') + ' ' + (SRV.model || ''), w * 0.5, h * 0.455, Math.max(14, w * 0.045), '#20242a', 'center', '700');
        txt(x, SRV.serialNum ? 'S/N ' + SRV.serialNum : '', w * 0.5, h * 0.51, Math.max(11, w * 0.035), '#444b55', 'center', '500');
        txt(x, 'Передняя панель  ↓', w * 0.5, h * 0.965, 14, 'rgba(255,255,255,.3)');
        return c;
    }
    function texBoard() {
        var s = 2, w = Math.round(BW * s), h = Math.round(BD * s), c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, 0, h, [[0, '#1b2420'], [1, '#121815']]); x.fillRect(0, 0, w, h);
        // дорожки
        x.strokeStyle = 'rgba(120,190,150,.12)'; x.lineWidth = 1.6;
        for (var i = 0; i < 90; i++) {
            var px = Math.random() * w, py = Math.random() * h;
            x.beginPath(); x.moveTo(px, py);
            x.lineTo(px + (Math.random() - 0.5) * 160, py); x.lineTo(px + (Math.random() - 0.5) * 160, py + (Math.random() - 0.5) * 90);
            x.stroke();
        }
        // дисковая корзина (передний край) и зона БП (задний)
        fillRR(x, w * 0.03, h * 0.865, w * 0.94, h * 0.12, 6, '#0a0b0d');
        txt(x, 'Дисковая корзина', w / 2, h * 0.925, 15, 'rgba(255,255,255,.25)');
        fillRR(x, w * 0.66, h * 0.03, w * 0.31, h * 0.2, 6, '#0a0b0d');
        txt(x, 'Блоки питания', w * 0.815, h * 0.13, 14, 'rgba(255,255,255,.25)');
        txt(x, 'Вентиляторы', w * 0.5, h * 0.855, 13, 'rgba(255,255,255,.22)');
        if (BLADE) txt(x, 'Системная плата блейда', w / 2, h * 0.5, 14, 'rgba(255,255,255,.18)');
        return c;
    }
    function texEar(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        x.fillStyle = lg(x, 0, 0, w, 0, [[0, '#9aa0a8'], [1, '#6a7078']]); x.fillRect(0, 0, w, h);
        return c;
    }

    // ----------------------------------------------------- текстуры компонентов
    function texDrive(vert, d, w, h, on, idx) {
        var c = mk(w, h), x = c.getContext('2d'), D = SK.drv;
        if (!d) {
            fillRR(x, 0, 0, w, h, 2, lg(x, 0, 0, w, h, [[0, '#1d1f23'], [1, '#121316']]));
            x.fillStyle = 'rgba(0,0,0,.55)';
            if (vert) for (var i = 1; i <= 4; i++) x.fillRect(w * 0.2, h * (0.2 + i * 0.13), w * 0.6, Math.max(1, h * 0.012));
            else for (var j = 1; j <= 2; j++) x.fillRect(w * 0.15, h * (0.25 + j * 0.2), w * 0.7, Math.max(1, h * 0.05));
            strokeRR(x, 1, 1, w - 2, h - 2, 2, 'rgba(255,255,255,.05)', 1);
            return c;
        }
        var isSsd = d.subType === 1, role = d.role || '';
        // LED активности: горячий резерв - синий (UID) и статичный, отказ - красный LED неисправности
        var actCol = role === 'Spare' ? C_UID : (hash(idx) > 0.4 ? D.led1 : '#ffffff');
        var faultCol = role === 'Failed' ? C_ERR : D.led2, faultOn = role === 'Failed' && on;
        if (vert) {
            fillRR(x, 0, 0, w, h, 2, lg(x, 0, 0, w, 0, [[0, D.body[1]], [0.5, D.body[0]], [1, D.body[1]]]));
            fillRR(x, w * 0.16, h * 0.72, w * 0.68, h * 0.2, 2, lg(x, 0, h * 0.72, 0, h * 0.92, [[0, D.latch[0]], [1, D.latch[1]]]));
            x.fillStyle = 'rgba(0,0,0,.5)';
            for (var k = 0; k < 6; k++) x.fillRect(w * 0.2, h * (0.2 + k * 0.08), w * 0.6, Math.max(1, h * 0.025));
            led(x, w * 0.32, h * 0.1, Math.max(1.5, w * 0.1), on && role !== 'Failed' ? actCol : '#000', on && role !== 'Failed');
            led(x, w * 0.68, h * 0.1, Math.max(1.5, w * 0.1), faultCol, faultOn);
        } else {
            fillRR(x, 0, 0, w, h, 2, lg(x, 0, 0, 0, h, [[0, D.body[0]], [1, D.body[1]]]));
            fillRR(x, w * 0.08, h * 0.14, w * 0.84, h * 0.2, 1, D.latch[1]);
            x.fillStyle = 'rgba(0,0,0,.55)';
            for (var m = 0; m < 6; m++) x.fillRect(w * 0.1 + m * (w * 0.8 / 6), h * 0.46, Math.max(1, w * 0.05), h * 0.26);
            led(x, w * 0.14, h * 0.86, Math.max(1.5, h * 0.07), on && role !== 'Failed' ? actCol : '#000', on && role !== 'Failed');
            led(x, w * 0.3, h * 0.86, Math.max(1.5, h * 0.07), faultCol, faultOn);
        }
        // метка назначения диска: HS - горячий резерв, RAW - не в массиве, C - кэш, ! - отказ (рабочие диски без метки)
        var badge = role === 'Spare' ? 'HS' : (role === 'Raw' ? 'RAW' : (role === 'Cache' ? 'C' : (role === 'Failed' ? '!' : '')));
        if (badge) {
            var bc = role === 'Spare' ? C_UID : (role === 'Failed' ? C_ERR : (role === 'Cache' ? '#9b6bff' : '#6b727c'));
            var bx = vert ? w * 0.1 : w * 0.62, by = vert ? h * 0.28 : h * 0.5, bw = vert ? w * 0.8 : w * 0.32, bh = vert ? Math.min(h * 0.14, w * 0.8) : h * 0.4;
            fillRR(x, bx, by, bw, bh, 2, bc);
            txt(x, badge, bx + bw / 2, by + bh / 2, Math.max(6, Math.min(bh * 0.7, bw * 0.72 / (0.74 * badge.length))), '#06121c', 'center', '800');
        }
        if (isSsd) { x.fillStyle = 'rgba(80,170,255,.9)'; x.fillRect(vert ? w * 0.1 : w * 0.08, vert ? h * 0.65 : h * 0.4, vert ? w * 0.8 : w * 0.84, Math.max(1, (vert ? h : w) * 0.012)); }
        strokeRR(x, 0.5, 0.5, w - 1, h - 1, 2, 'rgba(0,0,0,.8)', 1);
        return c;
    }
    function texPanel(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 3, lg(x, 0, 0, w, 0, [[0, SK.key === 'dell' ? '#c7ccd2' : '#1e2024'], [1, SK.key === 'dell' ? '#8e949c' : '#111316']]));
        var on = SRV.power === 1;
        var side = Math.min(w, h);
        if (BLADE) {
            powerButton(x, w / 2, h * 0.28, side * 0.11, SRV.power);
            var blr = side * 0.028;
            led(x, w * 0.25, h * 0.68, blr, C_UID, false); led(x, w * 0.5, h * 0.68, blr, on ? C_ON : '#1d2024', on); led(x, w * 0.75, h * 0.68, blr, C_ERR, false);
            txt(x, modelText(), w / 2, h * 0.9, Math.max(9, w * 0.09), 'rgba(235,238,242,.6)', 'center', '500');
            return c;
        }
        if (SK.key === 'dell') { // LCD-кольцо / индикатор
            fillRR(x, w * 0.14, h * 0.05, w * 0.72, h * 0.12, 3, '#10151b');
            fillRR(x, w * 0.2, h * 0.075, w * 0.6, h * 0.07, 2, '#1c5f94');
        }
        // На 1U панель почти квадратная: радиус от min(w,h), иначе кнопка
        // (раньше w*0.26 + сильное свечение) занимала почти весь торец.
        powerButton(x, w / 2, h * (SK.key === 'dell' ? 0.22 : 0.16), side * 0.10, SRV.power);
        var ly = h * 0.34, lr = side * 0.028, ls = h * 0.055;
        led(x, w * 0.5, ly, lr, C_UID, false);
        led(x, w * 0.5, ly + ls, lr, on ? C_ON : '#1d2024', on);
        led(x, w * 0.5, ly + ls * 2, lr, C_ERR, false);
        led(x, w * 0.5, ly + ls * 3, lr, C_ON, on);
        led(x, w * 0.5, ly + ls * 4, lr, C_ON, on);
        fillRR(x, w * 0.22, h * 0.78, w * 0.56, h * 0.07, 2, SK.accent);
        return c;
    }
    function texPsu(w, h, comp, n) {
        var c = mk(w, h), x = c.getContext('2d');
        var horiz = w >= h;
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, h, [[0, '#2a2d31'], [1, '#17191c']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 4, 'rgba(255,255,255,.12)', 1);
        // решётка вентилятора
        var gw = horiz ? w * 0.52 : w * 0.8, gh = horiz ? h * 0.76 : h * 0.55;
        var gx = horiz ? w * 0.06 : w * 0.1, gy = horiz ? h * 0.12 : h * 0.06;
        fillRR(x, gx, gy, gw, gh, 4, '#08090b');
        dots(x, gx + 4, gy + 4, gw - 8, gh - 8, Math.max(6, Math.min(gw, gh) * 0.1), Math.max(1.2, Math.min(gw, gh) * 0.022), 'rgba(160,166,174,.35)');
        // разъём питания C13/C14
        var ix = horiz ? w * 0.68 : w * 0.3, iy = horiz ? h * 0.3 : h * 0.68, iw = horiz ? w * 0.26 : w * 0.4, ih = horiz ? h * 0.4 : h * 0.22;
        fillRR(x, ix, iy, iw, ih, 3, '#060708');
        x.fillStyle = '#34383e';
        x.fillRect(ix + iw * 0.2, iy + ih * 0.3, iw * 0.15, ih * 0.4); x.fillRect(ix + iw * 0.65, iy + ih * 0.3, iw * 0.15, ih * 0.4);
        // ручка и LED
        fillRR(x, horiz ? w * 0.06 : w * 0.15, horiz ? h * 0.86 : h * 0.9, horiz ? w * 0.5 : w * 0.7, horiz ? h * 0.08 : h * 0.05, 2, '#9aa0a8');
        led(x, horiz ? w * 0.9 : w * 0.8, horiz ? h * 0.12 : h * 0.93, Math.max(2, Math.min(w, h) * 0.04), comp ? stColor(comp.status) : C_UNK, !!comp);
        var wattage = comp && comp.capacityGb ? Math.round(comp.capacityGb) + 'W' : (SRV.ratedW > 0 ? '' : '');
        txt(x, 'PSU ' + n + (wattage ? ' · ' + wattage : ''), horiz ? w * 0.8 : w * 0.5, horiz ? h * 0.82 : h * 0.84, Math.max(8, Math.min(w, h) * 0.1), 'rgba(235,238,242,.65)', 'center', '600');
        return c;
    }
    function portRJ45(x, X, Y, w, h, lit) {
        fillRR(x, X, Y, w, h, 2, '#050607');
        fillRR(x, X + w * 0.12, Y + h * 0.22, w * 0.76, h * 0.5, 1, '#8a6f2a');
        led(x, X + w * 0.2, Y + h * 0.1, Math.max(1.5, h * 0.08), C_ON, lit);
        led(x, X + w * 0.8, Y + h * 0.1, Math.max(1.5, h * 0.08), C_STBY, false);
    }
    function texCard(card, w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 3, lg(x, 0, 0, 0, h, [[0, '#d5d9de'], [1, '#9097a0']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 3, 'rgba(0,0,0,.5)', 1.2);
        var n = Math.max(1, card.comps.length);
        if (card.kind === 'nic') {
            var cols = Math.min(4, n), pw = (w * 0.78) / cols, ph = h * 0.42;
            for (var i = 0; i < Math.min(cols, 4); i++) portRJ45(x, w * 0.11 + i * pw + pw * 0.06, h * 0.3, pw * 0.88, ph, true);
        } else if (card.kind === 'fc') {
            var fc = Math.min(4, n), fw = (w * 0.78) / fc;
            for (var j = 0; j < fc; j++) {
                fillRR(x, w * 0.11 + j * fw + fw * 0.08, h * 0.32, fw * 0.84, h * 0.34, 2, '#050607');
                fillRR(x, w * 0.11 + j * fw + fw * 0.2, h * 0.4, fw * 0.6, h * 0.18, 1, '#2a6fbf');
                led(x, w * 0.11 + j * fw + fw * 0.5, h * 0.22, Math.max(1.5, h * 0.06), C_ON, true);
            }
        } else if (card.kind === 'raid') {
            for (var k = 0; k < 2; k++) {
                fillRR(x, w * 0.14 + k * w * 0.38, h * 0.3, w * 0.32, h * 0.4, 2, '#050607');
                x.fillStyle = '#34383e'; x.fillRect(w * 0.16 + k * w * 0.38, h * 0.42, w * 0.28, h * 0.12);
            }
            led(x, w * 0.88, h * 0.2, Math.max(1.5, h * 0.06), C_ON, true);
        } else {
            vents(x, w * 0.12, h * 0.25, w * 0.76, h * 0.5, 5, '#2b2f35');
        }
        var cap = (CARD_KIND[card.kind] || 'Адаптер');
        txt(x, cap, w * 0.5, h * 0.9, Math.max(8, h * 0.13), '#2a2e34', 'center', '600');
        return c;
    }
    function texEmptySlot(w, h, label) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 3, '#16181b');
        vents(x, w * 0.1, h * 0.2, w * 0.8, h * 0.6, 4, '#07080a');
        strokeRR(x, 1, 1, w - 2, h - 2, 3, 'rgba(255,255,255,.06)', 1);
        if (label) txt(x, label, w / 2, h * 0.5, Math.max(8, h * 0.18), 'rgba(255,255,255,.2)', 'center', '500');
        return c;
    }
    function texPorts(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, 0, h, [[0, '#2b2e33'], [1, '#17191c']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 4, 'rgba(255,255,255,.1)', 1);
        var vert = h > w * 1.2;
        var step = vert ? h / 5 : w / 5;
        function slot(i, kind) {
            var X = vert ? w * 0.1 : step * i + step * 0.1, Y = vert ? step * i + step * 0.1 : h * 0.12;
            var pw = vert ? w * 0.8 : step * 0.8, ph = vert ? step * 0.8 : h * 0.62;
            if (kind === 'rj') portRJ45(x, X, Y + ph * 0.1, pw, ph * 0.8, true);
            else if (kind === 'usb') { fillRR(x, X, Y + ph * 0.2, pw, ph * 0.6, 2, '#050607'); x.fillStyle = '#3da5ff'; x.fillRect(X + pw * 0.12, Y + ph * 0.45, pw * 0.76, ph * 0.12); }
            else if (kind === 'vga') { fillRR(x, X, Y + ph * 0.2, pw, ph * 0.6, 4, '#1b3a6a'); }
            else { fillRR(x, X, Y + ph * 0.2, pw, ph * 0.6, 3, '#0a0a0c'); }
        }
        slot(0, 'rj'); slot(1, 'rj'); slot(2, 'usb'); slot(3, 'usb'); slot(4, 'vga');
        return c;
    }
    // выделенный порт управления (iLO / iDRAC / IPMI / XCC / BMC) - отдельный модуль с данными контроллера
    function texIlo(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, 0, h, [[0, '#2b2e33'], [1, '#17191c']]));
        strokeRR(x, 1, 1, w - 2, h - 2, 4, 'rgba(255,255,255,.1)', 1);
        var horiz = w >= h * 1.1, lit = !!SRV.mgmtIp, col = SK.accent === '#e2231a' ? '#ff6a63' : '#8ee0c6';
        if (horiz) {
            portRJ45(x, w * 0.06, h * 0.12, w * 0.42, h * 0.76, lit);
            txt(x, SK.mgmt, w * 0.76, h * 0.5, Math.max(8, Math.min(h * 0.4, w * 0.17)), col, 'center', '700');
        } else {
            portRJ45(x, w * 0.1, h * 0.08, w * 0.8, h * 0.52, lit);
            txt(x, SK.mgmt, w * 0.5, h * 0.8, Math.max(8, Math.min(w * 0.22, h * 0.2)), col, 'center', '700');
        }
        return c;
    }
    function texFan(w, h, comp, n) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, h, [[0, '#33363b'], [1, '#17191c']]));
        var cx = w / 2, cy = h / 2, R = Math.min(w, h) * 0.42;
        x.beginPath(); x.arc(cx, cy, R, 0, Math.PI * 2); x.fillStyle = '#050607'; x.fill();
        x.save(); x.translate(cx, cy);
        for (var i = 0; i < 7; i++) {
            x.rotate(Math.PI * 2 / 7); x.beginPath(); x.moveTo(R * 0.2, 0);
            x.quadraticCurveTo(R * 0.7, -R * 0.35, R * 0.92, -R * 0.06); x.quadraticCurveTo(R * 0.65, R * 0.06, R * 0.2, R * 0.12);
            x.closePath(); x.fillStyle = 'rgba(78,82,90,.95)'; x.fill();
        }
        x.restore();
        x.beginPath(); x.arc(cx, cy, R * 0.22, 0, Math.PI * 2); x.fillStyle = '#4a4e55'; x.fill();
        x.beginPath(); x.arc(cx, cy, R, 0, Math.PI * 2); x.lineWidth = Math.max(2, R * 0.07); x.strokeStyle = '#70757e'; x.stroke();
        led(x, w * 0.14, h * 0.14, Math.max(2, Math.min(w, h) * 0.05), comp ? stColor(comp.status) : C_UNK, !!comp);
        txt(x, String(n), w * 0.86, h * 0.12, Math.max(8, Math.min(w, h) * 0.14), 'rgba(255,255,255,.5)');
        return c;
    }
    function texCpu(w, h, comp, n) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 4, lg(x, 0, 0, w, h, [[0, '#c4c8ce'], [1, '#7f858e']]));
        x.strokeStyle = 'rgba(0,0,0,.35)'; x.lineWidth = Math.max(1, w * 0.012);
        for (var i = 1; i < 14; i++) { x.beginPath(); x.moveTo(w * i / 14, h * 0.06); x.lineTo(w * i / 14, h * 0.94); x.stroke(); }
        strokeRR(x, 2, 2, w - 4, h - 4, 4, 'rgba(0,0,0,.6)', 2);
        fillRR(x, w * 0.25, h * 0.38, w * 0.5, h * 0.24, 3, 'rgba(18,20,24,.82)');
        txt(x, 'CPU ' + n, w / 2, h * 0.5, Math.max(9, h * 0.13), '#e9edf1', 'center', '700');
        return c;
    }
    function texDimm(populated, w, h, wide) {
        var c = mk(w, h), x = c.getContext('2d');
        if (!populated) {
            fillRR(x, 0, 0, w, h, 1, '#0c0d0f');
            x.fillStyle = '#26292e';
            if (wide) { x.fillRect(w * 0.03, h * 0.4, w * 0.94, Math.max(1, h * 0.2)); } else { x.fillRect(w * 0.4, h * 0.03, Math.max(1, w * 0.2), h * 0.94); }
            return c;
        }
        fillRR(x, 0, 0, w, h, 1, '#1a6b43');
        x.fillStyle = '#0d0f11';
        if (wide) { for (var i = 0; i < 8; i++) x.fillRect(w * (0.06 + i * 0.115), h * 0.2, w * 0.09, h * 0.6); }
        else { for (var j = 0; j < 8; j++) x.fillRect(w * 0.2, h * (0.08 + j * 0.115), w * 0.6, h * 0.09); }
        fillRR(x, wide ? 0 : 0, wide ? 0 : 0, wide ? w * 0.02 : w, wide ? h : h * 0.02, 0, '#b8bdc4');
        return c;
    }
    function texConn(w, h) {
        var c = mk(w, h), x = c.getContext('2d');
        fillRR(x, 0, 0, w, h, 3, lg(x, 0, 0, 0, h, [[0, '#2b2e33'], [1, '#15171a']]));
        for (var r = 0; r < 4; r++) for (var k = 0; k < 2; k++) {
            fillRR(x, w * (0.1 + k * 0.45), h * (0.08 + r * 0.22), w * 0.4, h * 0.17, 2, '#08090b');
            x.fillStyle = '#c9a23a';
            for (var p = 0; p < 10; p++) x.fillRect(w * (0.13 + k * 0.45) + p * w * 0.034, h * (0.12 + r * 0.22), Math.max(1, w * 0.012), h * 0.09);
        }
        return c;
    }

    // ------------------------------------------------------------------ сцена
    var renderer;
    try { renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true }); }
    catch (e) { return fail('Браузер не поддерживает WebGL — 3D-вид недоступен.'); }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    var maxAniso = renderer.capabilities.getMaxAnisotropy ? renderer.capabilities.getMaxAnisotropy() : 4;
    var dom = renderer.domElement;
    dom.style.touchAction = 'none'; dom.style.display = 'block';
    host.innerHTML = '';
    host.appendChild(dom);
    var ov = document.createElement('canvas');
    ov.style.cssText = 'position:absolute;left:0;top:0;pointer-events:none;';
    host.appendChild(ov);
    var octx = ov.getContext('2d');

    var scene = new THREE.Scene();
    var camera = new THREE.PerspectiveCamera(30, 1, 10, 9000);
    var DEF_DIST = Math.round(Math.max(BD * 2.6, FW * 3.6, BH * 4));
    var MIN_DIST = DEF_DIST * 0.2, MAX_DIST = DEF_DIST * 3;
    var dist = DEF_DIST, panX = 0, panY = 0;
    function applyCam() {
        camera.position.set(panX, panY, dist);
        camera.lookAt(panX, panY, 0);
    }
    applyCam();
    scene.add(new THREE.AmbientLight(0xffffff, 1.35));
    scene.add(new THREE.HemisphereLight(0xdfe8ff, 0x303540, 0.9));
    var key = new THREE.DirectionalLight(0xffffff, 2.4); key.position.set(500, 800, 1200); scene.add(key);
    var fillL = new THREE.DirectionalLight(0xbfd4ff, 0.9); fillL.position.set(-900, -300, 700); scene.add(fillL);
    var rig = new THREE.Group(); scene.add(rig);

    var texCache = {};
    function tex(canvas, ck) {
        if (ck && texCache[ck]) return texCache[ck];
        var t = new THREE.CanvasTexture(canvas);
        t.anisotropy = maxAniso;
        if (THREE.SRGBColorSpace) t.colorSpace = THREE.SRGBColorSpace;
        t.needsUpdate = true;
        if (ck) texCache[ck] = t;
        return t;
    }
    function std(o) { return new THREE.MeshStandardMaterial(o); }

    // --- мерцание светодиодов включённого оборудования ---------------------------
    // Светодиоды рисуются в текстуру модуля (led()), а их «дыхание» — отдельные
    // аддитивные полупрозрачные плоскости перед гранью с анимируемой opacity
    // (текстуры не перерисовываются; два слоя с разной фазой дают разнобой).
    function hexA(color, a) {
        var m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(color || '');
        if (!m) return 'rgba(255,255,255,' + a + ')';
        var h = m[1].length === 3 ? m[1].replace(/./g, '$&$&') : m[1];
        return 'rgba(' + parseInt(h.substr(0, 2), 16) + ',' + parseInt(h.substr(2, 2), 16) + ',' + parseInt(h.substr(4, 2), 16) + ',' + a + ')';
    }
    function ledBright(t, seed) {
        var s = seed * 12.9898;
        var a = 0.62 + 0.38 * (0.5 + 0.5 * Math.sin(t * 2.15 + s));
        var b = 0.78 + 0.22 * (0.5 + 0.5 * Math.sin(t * 5.4 + s * 1.73));
        var dip = 1 - 0.12 * Math.max(0, Math.sin(t * 15.5 + s * 2.9));
        return Math.min(1, Math.max(0.3, a * b * dip));
    }
    var ledMeshes = []; // { mat, seed }
    // pw/ph — размеры грани в мм; side: front | rear | top (как в addModule)
    function attachLeds(m, canvas, pw, ph, depth, side) {
        var leds = canvas.__leds;
        if (!leds || !leds.length) return;
        for (var g = 0; g < 2; g++) {
            var list = leds.filter(function (l, i) { return i % 2 === g; });
            if (!list.length) continue;
            var cv = document.createElement('canvas'); cv.width = canvas.width; cv.height = canvas.height;
            var c = cv.getContext('2d');
            c.globalCompositeOperation = 'lighter';
            list.forEach(function (l) {
                var R = l.r * 3.4, gr = c.createRadialGradient(l.cx, l.cy, 0, l.cx, l.cy, R);
                gr.addColorStop(0, hexA(l.color, 0.85)); gr.addColorStop(0.35, hexA(l.color, 0.32)); gr.addColorStop(1, hexA(l.color, 0));
                c.fillStyle = gr; c.beginPath(); c.arc(l.cx, l.cy, R, 0, Math.PI * 2); c.fill();
                c.fillStyle = 'rgba(255,255,255,0.45)';
                c.beginPath(); c.arc(l.cx - l.r * 0.2, l.cy - l.r * 0.2, l.r * 0.45, 0, Math.PI * 2); c.fill();
            });
            var mat = new THREE.MeshBasicMaterial({
                map: tex(cv), transparent: true, blending: THREE.AdditiveBlending, depthWrite: false,
                opacity: 1, toneMapped: false, polygonOffset: true, polygonOffsetFactor: -2, polygonOffsetUnits: -2
            });
            var pl = new THREE.Mesh(new THREE.PlaneGeometry(pw, ph), mat);
            pl.raycast = function () {};
            var off = depth / 2 + 0.35;
            if (side === 'rear') { pl.rotation.y = Math.PI; pl.position.z = -off; }
            else if (side === 'top') { pl.rotation.x = -Math.PI / 2; pl.position.y = off; }
            else { pl.position.z = off; }
            m.add(pl);
            ledMeshes.push({ mat: mat, seed: modules.length * 1.37 + g * 5.1 + list[0].cx * 0.011 });
        }
    }
    var ledLast = 0;
    function updateLeds(now) {
        if (!ledMeshes.length || now - ledLast < 55) return false;
        ledLast = now;
        var t = now * 0.001;
        for (var i = 0; i < ledMeshes.length; i++) ledMeshes[i].mat.opacity = ledBright(t, ledMeshes[i].seed);
        return true;
    }

    var pickables = [], modules = [];

    // корпус
    var sideMat = std({ map: tex(texSide(1024, 512)), roughness: 0.55, metalness: 0.45 });
    var lidMat = std({ map: tex(texLid()), roughness: 0.5, metalness: 0.5 });
    var boardMat = std({ map: tex(texBoard()), roughness: 0.7, metalness: 0.2 });
    var rearMat = std({ map: tex(texRearPlate()), roughness: 0.6, metalness: 0.35 });
    var darkMat = std({ color: 0x15171a, roughness: 0.6, metalness: 0.4 });
    var bodyMats = [sideMat, sideMat, lidMat, sideMat, darkMat, rearMat];
    var body = new THREE.Mesh(new THREE.BoxGeometry(BW, BH, BD), bodyMats);
    rig.add(body);
    // лицевая панель (с «ушами» стойки)
    var frontMat = std({ map: tex(texFrontPlate()), roughness: 0.55, metalness: 0.35 });
    var front = new THREE.Mesh(new THREE.BoxGeometry(FW, BH, FP), [darkMat, darkMat, darkMat, darkMat, frontMat, darkMat]);
    front.position.set(0, 0, BD / 2 + FP / 2);
    rig.add(front);
    var FZ = BD / 2 + FP;

    function faceW(side) { return side === 'front' ? FW : BW; }
    function addModule(side, r, depth, canvas, info, ck) {
        var fw = faceW(side), w = (r.u1 - r.u0) * fw;
        var uc = (r.u0 + r.u1) / 2, vc = (r.v0 + r.v1) / 2;
        var geo, idx, px, py, pz;
        if (side === 'top') {
            var len = (r.v1 - r.v0) * BD;
            geo = new THREE.BoxGeometry(w, depth, len); idx = 2;
            px = -BW / 2 + uc * BW; py = BH / 2 + depth / 2 - 0.4; pz = -BD / 2 + vc * BD;
        } else {
            var h = (r.v1 - r.v0) * BH;
            geo = new THREE.BoxGeometry(w, h, depth); idx = side === 'front' ? 4 : 5;
            px = side === 'front' ? -FW / 2 + uc * FW : BW / 2 - uc * BW; py = BH / 2 - vc * BH;
            pz = side === 'front' ? FZ + depth / 2 - 0.4 : -BD / 2 - depth / 2 + 0.4;
        }
        var t = tex(canvas, ck);
        var faceMat = std({ map: t, roughness: 0.5, metalness: 0.25, emissive: new THREE.Color(0xffffff), emissiveMap: t, emissiveIntensity: 0 });
        var dm = std({ color: 0x15171a, roughness: 0.6, metalness: 0.4 });
        var mats = [dm, dm, dm, dm, dm, dm]; mats[idx] = faceMat;
        var m = new THREE.Mesh(geo, mats);
        m.position.set(px, py, pz);
        var bb = geo.parameters;
        var hull = new THREE.Mesh(new THREE.BoxGeometry(bb.width + 5, bb.height + 5, bb.depth + 5),
            new THREE.MeshBasicMaterial({ color: 0x4dabf7, side: THREE.BackSide, transparent: true, opacity: 0.9, depthWrite: false }));
        hull.visible = false; m.add(hull);
        info.side = side;
        m.userData = { info: info, faceMat: faceMat, hull: hull, flashUntil: 0 };
        attachLeds(m, canvas, bb.width, side === 'top' ? bb.depth : bb.height, side === 'top' ? bb.height : bb.depth, side);
        m.visible = side !== 'top';
        rig.add(m); pickables.push(m); modules.push(m);
        return m;
    }
    function sz(r, side) { return { w: (r.u1 - r.u0) * faceW(side), h: (side === 'top' ? (r.v1 - r.v0) * BD : (r.v1 - r.v0) * BH) }; }
    function info(kind, title, sub, power, rows, links) {
        return { kind: kind, title: title, sub: sub || '', power: power || 0, rows: rows || [], links: links || [] };
    }
    var K = function (k, v) { return { k: k, v: v }; };
    var S3 = 3.2;

    // --- лицевая сторона: диски
    var on = SRV.power === 1;
    LF.bays.forEach(function (r, i) {
        var d = BLADE ? (DISKS[i] || null) : (diskAtBay[i] || null);
        var s = sz(r, 'front'), vert = BLADE ? false : !CFG.flat;
        var cv = texDrive(vert, d, s.w * S3, s.h * S3, on, i);
        var inf;
        if (d) {
            var dpw = d.role === 'Spare' ? 3 : (d.role === 'Failed' ? 4 : (on ? 1 : 2));
            inf = info('disk', d.name, 'Отсек ' + (i + 1) + (d.subTypeName ? ' · ' + d.subTypeName : '') + (d.capacityGb > 0 ? ' · ' + d.capacityGb.toFixed(1) + ' ГБ' : '') + (d.roleName ? ' · ' + d.roleName : ''), dpw, d.rows || []);
        } else {
            inf = info('disk-empty', 'Отсек диска ' + (i + 1), 'Диск не установлен / нет данных', 0, [K('Отсек', String(i + 1))]);
            inf.absent = true;
        }
        addModule('front', r, d ? 7 : 3, cv, inf, d ? null : 'drive-empty-' + vert + '-' + Math.round(s.w) + 'x' + Math.round(s.h));
    });
    if (LF.panel) {
        var ps = sz(LF.panel, 'front');
        addModule('front', LF.panel, 6, texPanel(ps.w * S3, ps.h * S3),
            info('panel', 'Панель управления', 'Кнопка питания, индикаторы UID / health', SRV.power,
                [K('Питание', pwText(SRV.power)), K('Сервер', SRV.name)]));
    }
    // --- тыльная сторона
    var psuComps = compsOf('psu');
    if (BLADE) {
        var cs = sz(LR.conn, 'rear');
        addModule('rear', LR.conn, 6, texConn(cs.w * S3, cs.h * S3),
            info('conn', 'Разъём подключения к корзине', 'Midplane блейд-корзины', 0, [K('Сервер', SRV.name)]));
    } else {
        LR.psu.forEach(function (r, i) {
            var comp = psuComps[i] || null, s = sz(r, 'rear');
            var inf = comp ? info('psu', comp.name, 'Блок питания ' + (i + 1), /crit|fail|error/i.test(comp.status || '') ? 4 : 1, comp.rows || [])
                : info('psu', 'Блок питания ' + (i + 1), 'Нет данных о блоке питания в справочнике', 0,
                    [K('Отсек', String(i + 1))].concat(SRV.powerRows || [], [K('Данные БП', 'OneView не отдаёт состав БП серверов в стойке (нужен опрос iLO / Redfish)')]));
            addModule('rear', r, 14, texPsu(s.w * S3, s.h * S3, comp, i + 1), inf);
        });
        LR.slots.forEach(function (r, i) {
            var card = REAR_CARDS[i], s = sz(r, 'rear');
            if (card) {
                var first = card.comps[0], rows = [K('Слот', card.slot), K('Тип', CARD_KIND[card.kind])];
                var model = first.model || first.name;
                rows.push(K('Производитель', first.vendor)); rows.push(K('Модель', model)); rows.push(K('Серийный номер', first.serialNum));
                if (card.comps.length > 1 || card.kind === 'nic' || card.kind === 'fc') {
                    rows.push(K('Портов', String(card.comps.length)));
                    card.comps.forEach(function (c, j) { rows.push(K('Порт ' + (j + 1), c.slot || c.name)); });
                }
                rows = rows.filter(function (q) { return q.v; });
                var inf = info('card', model || CARD_KIND[card.kind], CARD_KIND[card.kind] + (card.slot ? ' · ' + card.slot : ''), 0, rows);
                addModule('rear', r, 12, texCard(card, s.w * S3, s.h * S3), inf);
            } else {
                var emp = info('slot', 'Слот расширения ' + (i + 1), 'Свободен / нет данных', 0, [K('Слот', String(i + 1))]);
                emp.absent = true;
                addModule('rear', r, 3, texEmptySlot(s.w * S3, s.h * S3, ''), emp, 'slot-empty-' + Math.round(s.w) + 'x' + Math.round(s.h));
            }
        });
        if (LR.ports) {
            var pr = LR.ports, s2 = sz(pr, 'rear');
            addModule('rear', pr, 10, texPorts(s2.w * S3, s2.h * S3),
                info('ports', 'Порты ввода-вывода', 'LOM, USB, VGA', 0, [K('Порты', 'LOM 2 × RJ45, USB 2, VGA')]));
        }
        if (LR.ilo) {
            var ir = LR.ilo, s4 = sz(ir, 'rear');
            // данные контроллера собирает Go (mgmtRows: IP, модель, прошивка, BIOS/ROM); запасной вариант - из mgmtIp/mgmtModel
            var mrows = (SRV.mgmtRows && SRV.mgmtRows.length) ? SRV.mgmtRows.slice() : [K('IP адрес', SRV.mgmtIp), K('Модель контроллера', SRV.mgmtModel)];
            mrows = [K('Порт управления', SK.mgmt)].concat(mrows).filter(function (q) { return q.v; });
            if (mrows.length < 2) mrows.push(K('Данные', 'OneView не вернул данные контроллера управления'));
            var mlinks = SRV.mgmtIp ? [{ title: 'Открыть ' + SK.mgmt + ' (https://' + SRV.mgmtIp + ')', url: 'https://' + SRV.mgmtIp }] : [];
            addModule('rear', ir, 10, texIlo(s4.w * S3, s4.h * S3),
                info('mgmt', SK.mgmt + (SRV.mgmtIp ? ' · ' + SRV.mgmtIp : ''), 'Выделенный порт управления', SRV.mgmtIp ? 1 : 0, mrows, mlinks));
        }
    }
    // --- вид сверху (кожух снят)
    var fanComps = FAN_COMPS_EARLY;
    // Число слотов вентиляторов: по данным (в т.ч. оценка из Go), иначе типовое по U.
    if (!BLADE && fanComps.length > LT.fans.length) {
        var need = fanComps.length, fw2 = 0.92 / need;
        LT.fans = [];
        for (var fi = 0; fi < need; fi++) {
            LT.fans.push({ n: fi + 1, u0: 0.04 + fi * fw2 + 0.004, u1: 0.04 + (fi + 1) * fw2 - 0.004, v0: 0.69, v1: 0.84 });
        }
    }
    LT.fans.forEach(function (r, i) {
        var comp = fanComps[i] || null, s = sz(r, 'top');
        var inf = comp ? info('fan', comp.name, comp.model || ('Вентилятор ' + (i + 1)), /crit|fail|error/i.test(comp.status || '') ? 4 : 1, comp.rows || [])
            : info('fan', 'Вентилятор ' + (i + 1), 'Модуль охлаждения (нет данных в справочнике)', 0,
                [K('Позиция', String(i + 1)), K('Данные вентиляторов', 'OneView не отдаёт состав вентиляторов серверов в стойке (нужен опрос iLO / Redfish)')]);
        addModule('top', r, 22, texFan(s.w * S3, s.h * S3, comp, i + 1), inf);
    });
    var cpuComps = compsOf('cpu');
    LT.cpus.forEach(function (r, i) {
        var comp = cpuComps[i] || null, s = sz(r, 'top');
        var rows = comp ? comp.rows.slice() : [K('Сокет', String(i + 1))];
        if (!comp && SRV.cpuCount > 0) rows.push(K('Всего процессоров', String(SRV.cpuCount)));
        addModule('top', r, 14, texCpu(s.w * S3, s.h * S3, comp, i + 1),
            info('cpu', comp ? comp.name : 'Процессор ' + (i + 1), 'Сокет ' + (i + 1), 0, rows));
    });
    LT.dimms.forEach(function (r, i) {
        var comp = RAMS[i] || null, s = sz(r, 'top');
        var inf;
        if (comp) inf = info('ram', comp.name, 'Слот памяти ' + (i + 1) + (comp.capacityGb > 0 ? ' · ' + Math.round(comp.capacityGb) + ' ГБ' : ''), 0, comp.rows || []);
        else {
            inf = info('ram-empty', 'Слот памяти ' + (i + 1), RAMS.length ? 'Свободен' : 'Нет данных о модулях памяти', 0,
                [K('Слот', String(i + 1))].concat(SRV.memoryGb > 0 ? [K('Всего памяти', SRV.memoryGb + ' ГБ')] : []));
            inf.absent = !!RAMS.length;
        }
        addModule('top', r, 6, texDimm(!!comp, Math.max(8, s.w * S3), Math.max(8, s.h * S3), !!r.wide), inf, 'dimm-' + (comp ? 1 : 0) + (r.wide ? 'w' : 'v') + '-' + Math.round(s.w) + 'x' + Math.round(s.h));
    });
    LT.cards.forEach(function (r, i) {
        // Блейд: все мезонины сверху. Rack: сверху — адаптеры, не влезшие сзади (часто PCI Type=9).
        var card = TOP_CARDS[i] || null;
        var s = sz(r, 'top');
        if (card) {
            var rows = [K('Слот', card.slot), K('Тип', CARD_KIND[card.kind]), K('Модель', card.comps[0].model || card.comps[0].name)].filter(function (q) { return q.v; });
            if (card.comps.length > 1) {
                rows.push(K('Портов / частей', String(card.comps.length)));
                card.comps.forEach(function (c, j) { rows.push(K('Часть ' + (j + 1), c.slot || c.name)); });
            }
            if (!BLADE) rows.push(K('Расположение на 3D', 'Внутренний/ризерный слот (вид сверху)'));
            addModule('top', r, 8, texCard(card, s.w * S3, s.h * S3),
                info('card', card.comps[0].model || CARD_KIND[card.kind], (BLADE ? 'Мезонин: ' : 'PCIe: ') + CARD_KIND[card.kind], 0, rows));
        } else if (BLADE) {
            var empty = info('slot', 'Мезонин ' + (r.n || (i + 1)), 'Свободен / нет данных в справочнике', 0, [K('Мезонин', String(r.n || (i + 1)))]);
            empty.absent = true;
            addModule('top', r, 4, texEmptySlot(s.w * S3, s.h * S3, 'Mezz ' + (r.n || (i + 1))), empty, 'mezz-empty-' + Math.round(s.w) + 'x' + Math.round(s.h));
        }
    });

    // корпус как объект
    var serverInfo = { kind: 'server', title: SRV.name, sub: ((SRV.vendor || '') + ' ' + (SRV.model || '')).trim() + (BLADE ? '' : ' · ' + U + 'U'), power: SRV.power, rows: SRV.rows || [], links: SRV.links || [], side: '' };
    body.userData = { info: serverInfo, isBody: true };
    front.userData = { info: serverInfo, isBody: true };
    pickables.push(body); pickables.push(front);

    // кожух
    var lidOn = true;
    function setLid(onFlag) {
        lidOn = onFlag;
        bodyMats[2] = onFlag ? lidMat : boardMat;
        body.material = bodyMats; // обновим ссылку на массив
        modules.forEach(function (m) { if (m.userData.info.side === 'top') m.visible = !onFlag; });
        if (selected && !selected.visible) selected = null;
        if (hovered && !hovered.visible) hovered = null;
        if (lidBtn) lidBtn.textContent = onFlag ? 'Снять кожух' : 'Установить кожух';
        curSide = '#'; viewChanged();
        dirty = true;
    }

    // --------------------------------------------------------- виды/анимация
    var Q_ID = new THREE.Quaternion();
    function qAxis(ax, deg) { return new THREE.Quaternion().setFromAxisAngle(ax, deg * Math.PI / 180); }
    var AX = new THREE.Vector3(1, 0, 0), AY = new THREE.Vector3(0, 1, 0), AZ = new THREE.Vector3(0, 0, 1);
    var VIEWS = {
        front: Q_ID.clone(), rear: qAxis(AY, 180), left: qAxis(AY, 90), right: qAxis(AY, -90), top: qAxis(AX, 90),
        iso: qAxis(AX, 24).multiply(qAxis(AY, -35)), isorear: qAxis(AX, 24).multiply(qAxis(AY, 180 + 35)),
        isotop: qAxis(AX, 52).multiply(qAxis(AY, -30))
    };
    var anim = null, dirty = true;
    function animateTo(q, d) {
        anim = { t0: performance.now(), dur: 520, q0: rig.quaternion.clone(), q1: q.clone(), d0: dist, d1: d == null ? dist : d, px0: panX, py0: panY };
        dirty = true;
    }
    function setDist(d) {
        dist = Math.max(MIN_DIST, Math.min(MAX_DIST, d));
        applyCam();
        if (zoomEl) zoomEl.textContent = Math.round(DEF_DIST / dist * 100) + '%';
        dirty = true;
    }
    function panBy(dx, dy) {
        var k = dist * Math.tan(camera.fov * Math.PI / 360) * 2 / Math.max(1, vh);
        panX -= dx * k * (camera.aspect || 1);
        panY += dy * k;
        applyCam();
        anim = null; dirty = true;
    }
    function resetPan() { panX = 0; panY = 0; applyCam(); dirty = true; }
    function rotateBy(dx, dy) { rig.quaternion.premultiply(qAxis(AY, dx * 0.45)).premultiply(qAxis(AX, dy * 0.45)).normalize(); anim = null; dirty = true; viewChanged(); }
    function rollBy(rad) { rig.quaternion.premultiply(new THREE.Quaternion().setFromAxisAngle(AZ, rad)).normalize(); anim = null; dirty = true; viewChanged(); }

    // --------------------------------------------------------- стороны и список
    var curSide = '#';
    function currentSide() {
        var q = rig.quaternion, f = new THREE.Vector3(0, 0, 1).applyQuaternion(q).z,
            t = new THREE.Vector3(0, 1, 0).applyQuaternion(q).z;
        var cand = [['front', f], ['rear', -f], ['top', t]].sort(function (a, b) { return b[1] - a[1]; })[0];
        return cand[1] > 0.3 ? cand[0] : '';
    }
    function sideLabel(s) {
        if (!s) return 'Сбоку (компоненты не видны)';
        if (s === 'front') return 'Спереди (диски, панель управления)';
        if (s === 'rear') return BLADE ? 'Сзади (разъём корзины)' : 'Сзади (БП, адаптеры, порты управления)';
        return lidOn ? 'Сверху (кожух установлен — снимите кожух)' : 'Сверху (вентиляторы, CPU, память)';
    }
    function viewChanged() {
        var s = currentSide();
        if (s === curSide) return;
        curSide = s;
        if (sideEl) sideEl.textContent = sideLabel(s);
        buildList();
        if (s) {
            var now = performance.now();
            modules.forEach(function (m) { if (m.visible && m.userData.info.side === s && !m.userData.info.absent) m.userData.flashUntil = now + 2200; });
            dirty = true;
        }
    }
    function buildList() {
        if (!listEl) return;
        var s = curSide;
        if (!s) { listEl.innerHTML = '<div class="hcv-empty-hint">Сервер повёрнут боком — доступных компонентов нет. Выберите «Спереди», «Сзади» или «Сверху».</div>'; return; }
        if (s === 'top' && lidOn) { listEl.innerHTML = '<div class="hcv-empty-hint">Кожух установлен. Нажмите «Снять кожух», чтобы увидеть и выбрать вентиляторы, процессоры и память.</div>'; return; }
        var html = '';
        SIDE_KINDS[s].forEach(function (kind) {
            var items = modules.filter(function (m) { var i = m.userData.info; return m.visible && i.side === s && i.kind === kind && !i.absent; });
            if (!items.length) return;
            html += '<div class="hcv-grp"><div class="hcv-grp-title">' + esc(GROUP_TITLE[kind]) + ' <span class="text-muted">(' + items.length + ')</span></div>';
            items.forEach(function (m) {
                var i = m.userData.info;
                html += '<button type="button" class="hcv-li" data-mi="' + modules.indexOf(m) + '"><span class="hcv-dot ' + dotClass(i.power) + '"></span>' + esc(i.title) + '</button>';
            });
            html += '</div>';
        });
        listEl.innerHTML = html || '<div class="hcv-empty-hint">Нет компонентов.</div>';
    }
    if (listEl) {
        listEl.addEventListener('click', function (e) {
            var b = e.target.closest('.hcv-li'); if (!b) return;
            select(modules[parseInt(b.getAttribute('data-mi'), 10)], false);
        });
        listEl.addEventListener('mouseover', function (e) {
            var b = e.target.closest('.hcv-li');
            hovered = b ? modules[parseInt(b.getAttribute('data-mi'), 10)] : null; dirty = true;
        });
        listEl.addEventListener('mouseleave', function () { hovered = null; dirty = true; });
    }

    // ------------------------------------------------------------- выбор/инфо
    var hovered = null, selected = null;
    function renderInfo(inf) {
        if (!detailEl) return;
        if (!inf) { detailEl.innerHTML = '<div class="hcv-empty-hint">Наведите курсор на компонент — подсветка и подсказка; кликните — параметры здесь. Клик по корпусу показывает данные сервера.</div>'; return; }
        var h = '<div class="hcv-detail-title">' + esc(inf.title) + '</div>';
        h += '<div class="hcv-detail-sub">' + esc(KIND_TITLE[inf.kind] || '') + (inf.sub ? ' &middot; ' + esc(inf.sub) : '') + '</div>';
        if (inf.kind !== 'server' || true) h += '<div class="hcv-pw"><span class="hcv-dot ' + dotClass(inf.power) + '"></span>' + esc(pwText(inf.power)) + '</div>';
        if (inf.rows && inf.rows.length) {
            h += '<table>';
            inf.rows.forEach(function (r) { h += '<tr><td>' + esc(r.k) + '</td><td>' + esc(r.v) + '</td></tr>'; });
            h += '</table>';
        }
        if (inf.links && inf.links.length) {
            h += '<div class="hcv-links">';
            inf.links.forEach(function (l) { h += '<a href="' + esc(l.url) + '" target="_blank" rel="noopener">' + esc(l.title) + '</a>'; });
            h += '</div>';
        }
        detailEl.innerHTML = h;
    }
    function select(m, fromScene) {
        selected = m && m.userData.isBody ? null : m;
        renderInfo(m ? m.userData.info : serverInfo);
        if (m && !m.userData.isBody && !fromScene) {
            var sd = m.userData.info.side;
            if (sd && sd !== currentSide()) animateTo(VIEWS[sd]);
        }
        dirty = true;
    }

    // ----------------------------------------------------------------- рендер
    var raycaster = new THREE.Raycaster(), ndc = new THREE.Vector2();
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

    var corners = [], hoverCorner = -1;
    function computeCorners() {
        rig.updateMatrixWorld(true);
        corners.length = 0;
        for (var sx = -1; sx <= 1; sx += 2) for (var sy = -1; sy <= 1; sy += 2) for (var sz_ = -1; sz_ <= 1; sz_ += 2) {
            var v = new THREE.Vector3(sx * FW / 2, sy * BH / 2, sz_ * (BD / 2 + (sz_ > 0 ? FP : 0))).applyMatrix4(rig.matrixWorld);
            v.project(camera);
            corners.push({ x: (v.x * 0.5 + 0.5) * vw, y: (-v.y * 0.5 + 0.5) * vh, sx: sx, sy: sy, sz: sz_ });
        }
    }
    function drawOverlay() {
        octx.clearRect(0, 0, vw, vh);
        computeCorners();
        octx.save(); octx.lineWidth = 1; octx.strokeStyle = 'rgba(77,171,247,.38)'; octx.setLineDash([5, 4]);
        corners.forEach(function (a, i) {
            corners.forEach(function (b, j) {
                if (j <= i) return;
                if (((a.sx !== b.sx) + (a.sy !== b.sy) + (a.sz !== b.sz)) !== 1) return;
                octx.beginPath(); octx.moveTo(a.x, a.y); octx.lineTo(b.x, b.y); octx.stroke();
            });
        });
        octx.setLineDash([]);
        corners.forEach(function (c, i) {
            var r = i === hoverCorner ? 9 : 6.5;
            octx.beginPath(); octx.arc(c.x, c.y, r + 2, 0, Math.PI * 2); octx.fillStyle = 'rgba(10,14,20,.75)'; octx.fill();
            octx.beginPath(); octx.arc(c.x, c.y, r, 0, Math.PI * 2); octx.fillStyle = i === hoverCorner ? '#ffcf4d' : '#4dabf7'; octx.fill();
            octx.lineWidth = 1.5; octx.strokeStyle = '#fff'; octx.stroke();
        });
        octx.restore();
    }
    function applyHighlights(now) {
        var flashing = false;
        for (var i = 0; i < modules.length; i++) {
            var m = modules[i], u = m.userData;
            if (!m.visible) { u.hull.visible = false; continue; }
            var isSel = m === selected, isHov = m === hovered, fl = u.flashUntil > now;
            var hullOn = false, color = 0x4dabf7, opacity = 0.9, emis = 0;
            if (isSel) { hullOn = true; color = 0xffcf4d; emis = 0.55; }
            else if (isHov) { hullOn = true; emis = 0.38; }
            else if (fl) {
                hullOn = true; color = 0x35e0c8; flashing = true;
                var ph = (u.flashUntil - now) / 2200;
                opacity = 0.35 + 0.6 * Math.abs(Math.sin(ph * Math.PI * 4)); emis = 0.18 * opacity;
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
            panX = (anim.px0 || 0) * (1 - e);
            panY = (anim.py0 || 0) * (1 - e);
            setDist(anim.d0 + (anim.d1 - anim.d0) * e);
            dirty = true;
            if (t >= 1) { panX = 0; panY = 0; applyCam(); anim = null; }
            viewChanged();
        }
        if (applyHighlights(now)) dirty = true;
        if (updateLeds(now)) dirty = true;
        if (!dirty) return;
        dirty = false;
        renderer.render(scene, camera);
        drawOverlay();
    }

    // -------------------------------------------------------------- указатель
    function localXY(e) { var r = dom.getBoundingClientRect(); return { x: (e.clientX - r.left) * (vw / r.width), y: (e.clientY - r.top) * (vh / r.height) }; }
    function cornerAt(p) {
        var best = -1, bd = 15 * 15;
        corners.forEach(function (c, i) { var d = (c.x - p.x) * (c.x - p.x) + (c.y - p.y) * (c.y - p.y); if (d < bd) { bd = d; best = i; } });
        return best;
    }
    function pick(p) {
        ndc.set(p.x / vw * 2 - 1, -(p.y / vh) * 2 + 1);
        raycaster.setFromCamera(ndc, camera);
        var list = pickables.filter(function (m) { return m.visible; });
        var hits = raycaster.intersectObjects(list, false);
        return hits.length ? hits[0].object : null;
    }
    function showTip(m, p) {
        if (!tipEl) return;
        if (!m || m.userData.isBody) { tipEl.style.display = 'none'; return; }
        var i = m.userData.info;
        tipEl.innerHTML = '<b>' + esc(i.title) + '</b><br><span>' + esc(KIND_TITLE[i.kind] || '') + (i.sub ? ' · ' + esc(i.sub) : '') + '</span>';
        tipEl.style.display = 'block';
        var rx = p.x + 16, ry = p.y + 16, tw = tipEl.offsetWidth, th = tipEl.offsetHeight;
        if (rx + tw > vw - 4) rx = p.x - tw - 12;
        if (ry + th > vh - 4) ry = p.y - th - 12;
        tipEl.style.left = Math.max(2, rx) + 'px'; tipEl.style.top = Math.max(2, ry) + 'px';
    }
    var drag = null;
    dom.addEventListener('contextmenu', function (e) { e.preventDefault(); });
    dom.addEventListener('pointerdown', function (e) {
        var p = localXY(e); computeCorners();
        var ci = cornerAt(p), hit = ci < 0 ? pick(p) : null;
        var roll = e.shiftKey || e.button === 2;
        // Углы / СКМ — вращение; ПКМ/Shift — крен; ЛКМ — pan (для увеличенного вида),
        // короткий клик по модулю — выбор.
        if (ci >= 0) drag = { mode: roll ? 'roll' : 'rot', x: p.x, y: p.y, moved: 0 };
        else if (roll) drag = { mode: 'roll', x: p.x, y: p.y, moved: 0 };
        else if (e.button === 1) drag = { mode: 'rot', x: p.x, y: p.y, moved: 0 };
        else if (hit && !hit.userData.isBody) drag = { mode: 'click', x: p.x, y: p.y, moved: 0, hit: hit };
        else drag = { mode: 'pan', x: p.x, y: p.y, moved: 0 };
        try { dom.setPointerCapture(e.pointerId); } catch (_) { /* ignore */ }
        dom.style.cursor = drag.mode === 'click' ? 'pointer' : 'grabbing';
        if (tipEl) tipEl.style.display = 'none';
        e.preventDefault();
    });
    dom.addEventListener('pointermove', function (e) {
        var p = localXY(e);
        if (drag && drag.mode === 'click' && drag.moved >= 6) {
            drag.mode = 'pan';
            dom.style.cursor = 'grabbing';
        }
        if (drag && drag.mode !== 'click') {
            var dx = p.x - drag.x, dy = p.y - drag.y;
            drag.moved += Math.abs(dx) + Math.abs(dy);
            if (drag.mode === 'roll') {
                var cx = vw / 2, cy = vh / 2, da = Math.atan2(p.y - cy, p.x - cx) - Math.atan2(drag.y - cy, drag.x - cx);
                if (da > Math.PI) da -= 2 * Math.PI; if (da < -Math.PI) da += 2 * Math.PI;
                rollBy(da);
            } else if (drag.mode === 'pan') {
                panBy(dx, dy);
            } else {
                rotateBy(dx, dy);
            }
            drag.x = p.x; drag.y = p.y; return;
        }
        if (drag) { drag.moved += Math.abs(p.x - drag.x) + Math.abs(p.y - drag.y); drag.x = p.x; drag.y = p.y; return; }
        var ci = cornerAt(p);
        if (ci !== hoverCorner) { hoverCorner = ci; dirty = true; }
        if (ci >= 0) { dom.style.cursor = 'grab'; if (hovered) { hovered = null; dirty = true; } showTip(null, p); return; }
        var m = pick(p), hm = m && !m.userData.isBody ? m : null;
        if (hm !== hovered) { hovered = hm; dirty = true; }
        dom.style.cursor = hm ? 'pointer' : 'grab';
        showTip(m, p);
    });
    function endDrag(e) {
        if (!drag) return;
        var d = drag; drag = null;
        try { dom.releasePointerCapture(e.pointerId); } catch (_) { /* ignore */ }
        dom.style.cursor = 'default';
        if (d.mode === 'click' && d.moved < 6) select(d.hit, true);
        else if (d.mode === 'pan' && d.moved < 4) select(null, true);
        else if (d.mode === 'rot' && d.moved < 4) select(null, true);
    }
    dom.addEventListener('pointerup', endDrag);
    dom.addEventListener('pointercancel', endDrag);
    dom.addEventListener('pointerleave', function () { hoverCorner = -1; if (!drag && hovered) hovered = null; if (tipEl) tipEl.style.display = 'none'; dirty = true; });
    dom.addEventListener('wheel', function (e) { e.preventDefault(); setDist(dist * Math.exp(e.deltaY * (e.deltaMode === 1 ? 33 : 1) * 0.0011)); }, { passive: false });

    // ---------------------------------------------------------------- кнопки
    (function () {
        var btns = document.querySelectorAll('[data-hcv-view]');
        for (var i = 0; i < btns.length; i++) {
            btns[i].addEventListener('click', function () {
                var v = this.getAttribute('data-hcv-view');
                if (v === 'reset') animateTo(VIEWS.front, DEF_DIST * 0.62);
                else if (v === 'roll-l') animateTo(rig.quaternion.clone().premultiply(qAxis(AZ, 90)));
                else if (v === 'roll-r') animateTo(rig.quaternion.clone().premultiply(qAxis(AZ, -90)));
                else if (v === 'zoom-in') setDist(dist / 1.25);
                else if (v === 'zoom-out') setDist(dist * 1.25);
                else if (v === 'lid') setLid(!lidOn);
                else if (VIEWS[v]) {
                    if ((v === 'top' || v === 'isotop') && lidOn) setLid(false);
                    animateTo(VIEWS[v]);
                }
            });
        }
    })();

    // ------------------------------------------------------------------ старт
    resize();
    rig.quaternion.copy(VIEWS.iso);
    setDist(DEF_DIST);
    setLid(true);
    renderInfo(null);
    requestAnimationFrame(frame);
    window.HCV3D = { rig: rig, camera: camera, select: select, animateTo: animateTo, VIEWS: VIEWS, modules: modules, setDist: setDist, setLid: setLid, scene: scene, renderer: renderer, getAnim: function () { return anim; } };
})();
