<div class="hero-text mb-3">
    <h2>Аппаратная инфраструктура ЦОД</h2>
    <p class="text-muted small mb-0">
        Обзор всех ЦОД: адреса, количество шкафов, блейд-корзин и аппаратных серверов (включая гипервизоры).
        Клик по ЦОД раскрывает его шкафы с заполнением СХД, серверами и блейд-корзинами (занятые слоты).
        Наведите курсор на элемент, чтобы увидеть его параметры и вложенные компоненты.
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}

<div class="infra-canvas-shell border rounded mb-3">
    <div class="infra-canvas-toolbar d-flex flex-wrap align-items-center gap-3 px-3 py-2 border-bottom">
        <button type="button" class="btn btn-sm btn-outline-secondary" id="hwInfraBack" disabled>&larr; Все ЦОД</button>
        <div class="d-flex flex-wrap align-items-center gap-2 infra-legend">
            <span class="small text-muted me-1">Легенда:</span>
            <span class="infra-legend-item"><img src="/static/infra2/icons/hpe-c7000.svg" alt=""> Блейд-корзина</span>
            <span class="infra-legend-item"><img src="/static/infra2/icons/server-hpe.svg" alt=""> Rack-сервер</span>
            <span class="infra-legend-item"><img src="/static/infra2/icons/storage-dell.svg" alt=""> СХД</span>
            <span class="infra-legend-item"><img src="/static/infra2/icons/esxi-host.svg" alt=""> Гипервизор</span>
            <span class="infra-legend-item"><span class="infra-slot-sample empty"></span> Пустой слот</span>
        </div>
        <div class="ms-auto small text-muted" id="hwInfraStatus">Загрузка…</div>
    </div>

    <div class="infra-canvas-viewport position-relative">
        <canvas id="hwinfra-canvas" width="1280" height="720"></canvas>
        <div id="hwinfra-tooltip" class="infra-canvas-tooltip d-none" role="tooltip">
            <div class="infra-tooltip-title fw-semibold"></div>
            <div class="infra-tooltip-body small"></div>
        </div>
    </div>
</div>

<!-- Данные передаются в base64 через data-атрибут - обходит поломку JSON
     html/template-экранированием внутри <script> (см. комментарий в
     controllers/admininfrastructure.go). -->
<div id="hwinfra-data" class="d-none" data-payload-b64="{{.HWInfraJSON}}"></div>

<script>
(function () {
    'use strict';

    // ---- Данные ----
    // atob() возвращает "бинарную строку" (1 код = 1 байт), а не UTF-8
    // текст - JSON.parse на ней ломает кириллицу (каждый байт multi-byte
    // UTF-8 последовательности читается как отдельный символ Latin1).
    // Нужно явно перегнать байты через TextDecoder('utf-8').
    function b64ToUtf8(b64) {
        var binary = atob(b64);
        var bytes = new Uint8Array(binary.length);
        for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
        return new TextDecoder('utf-8').decode(bytes);
    }
    var raw = document.getElementById('hwinfra-data').getAttribute('data-payload-b64') || '';
    var DATA = { datacenters: [] };
    try { DATA = JSON.parse(b64ToUtf8(raw)); } catch (e) { console.error('hwinfra: bad payload', e); }
    DATA.datacenters = DATA.datacenters || [];

    var canvas = document.getElementById('hwinfra-canvas');
    var ctx = canvas.getContext('2d', { alpha: false });
    var tooltip = document.getElementById('hwinfra-tooltip');
    var statusEl = document.getElementById('hwInfraStatus');
    var backBtn = document.getElementById('hwInfraBack');
    var viewport = document.querySelector('.infra-canvas-viewport');

    var dpr = window.devicePixelRatio || 1;
    var cssW = 1280, cssH = 720;
    var view = 'overview'; // 'overview' | 'datacenter'
    var currentDcIdx = -1;
    var hitRegions = []; // {x,y,w,h,payload}

    // ---- Иконки (drawImage, см. kontext/08-svg-infra-icon-library.md) ----
    var ICONS = {};
    function loadIcon(name, src) {
        var img = new Image();
        img.src = src;
        ICONS[name] = img;
    }
    loadIcon('site', '/static/infra2/icons/datacenter-site.svg');
    loadIcon('chassis', '/static/infra2/icons/hpe-c7000.svg');
    loadIcon('server', '/static/infra2/icons/server-hpe.svg');
    loadIcon('storage', '/static/infra2/icons/storage-dell.svg');
    loadIcon('host', '/static/infra2/icons/esxi-host.svg');
    loadIcon('switch', '/static/infra2/icons/switch-cisco.svg');

    // ---- Цвета (light/dark - см. kontext/07-color-scheme.md) ----
    function cssVar(name, fallback) {
        var v = getComputedStyle(document.documentElement).getPropertyValue(name);
        return v && v.trim() ? v.trim() : fallback;
    }
    function colors() {
        var dark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
        return {
            bg: dark ? '#1a1d20' : '#ffffff',
            card: dark ? '#22262a' : '#f8f9fa',
            border: dark ? '#3a3f44' : '#dee2e6',
            text: dark ? '#e9ecef' : '#212529',
            muted: dark ? '#adb5bd' : '#6c757d',
            accent: dark ? '#4dabf7' : '#0d6efd',
            ok: '#2fb344',
            warn: '#f0ad4e',
            crit: '#dc3545',
            slotEmpty: dark ? '#2c3034' : '#e9ecef'
        };
    }

    function resize() {
        if (!viewport) return;
        var w = Math.max(640, viewport.clientWidth - 4);
        var h = Math.max(420, Math.floor(w * 9 / 16));
        cssW = w; cssH = h;
        dpr = window.devicePixelRatio || 1;
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        draw();
    }
    window.addEventListener('resize', resize);

    function roundRect(x, y, w, h, r) {
        ctx.beginPath();
        ctx.moveTo(x + r, y);
        ctx.arcTo(x + w, y, x + w, y + h, r);
        ctx.arcTo(x + w, y + h, x, y + h, r);
        ctx.arcTo(x, y + h, x, y, r);
        ctx.arcTo(x, y, x + w, y, r);
        ctx.closePath();
    }

    // ==== Обзор всех ЦОД (карточки) ====
    function drawOverview() {
        var C = colors();
        ctx.fillStyle = C.bg;
        ctx.fillRect(0, 0, cssW, cssH);
        hitRegions = [];

        var pad = 16, cardW = 300, cardH = 150, gap = 16;
        var perRow = Math.max(1, Math.floor((cssW - pad) / (cardW + gap)));
        var x = pad, y = pad, col = 0;

        if (DATA.datacenters.length === 0) {
            ctx.fillStyle = C.muted;
            ctx.font = '14px system-ui, sans-serif';
            ctx.fillText('Нет данных по ЦОД (справочник DatacenterSite пуст либо оборудование не привязано к шкафам).', pad, 30);
            return;
        }

        DATA.datacenters.forEach(function (dc, idx) {
            roundRect(Math.round(x), Math.round(y), cardW, cardH, 10);
            ctx.fillStyle = C.card;
            ctx.fill();
            ctx.strokeStyle = C.border;
            ctx.lineWidth = 1;
            ctx.stroke();

            if (ICONS.site && ICONS.site.complete) {
                ctx.drawImage(ICONS.site, x + 10, y + 10, 28, 22);
            }
            ctx.fillStyle = C.text;
            ctx.font = '600 15px system-ui, sans-serif';
            ctx.fillText(dc.name || '—', x + 44, y + 26);

            ctx.font = '12px system-ui, sans-serif';
            ctx.fillStyle = C.muted;
            var addr = dc.address ? dc.address : 'адрес не указан';
            ctx.fillText(trunc(addr, 34), x + 12, y + 50);

            var lines = [
                'Шкафов: ' + (dc.rackCount || 0),
                'Блейд-корзин: ' + (dc.chassisCount || 0) + ', СХД: ' + (dc.storageCount || 0),
                'Аппаратных серверов: ' + (dc.serverCount || 0) + ' (гипервизоров: ' + (dc.hvCount || 0) + ')'
            ];
            ctx.font = '13px system-ui, sans-serif';
            ctx.fillStyle = C.text;
            lines.forEach(function (line, i) {
                ctx.fillText(line, x + 12, y + 74 + i * 20);
            });

            ctx.fillStyle = C.accent;
            ctx.font = '12px system-ui, sans-serif';
            ctx.fillText('Открыть структуру →', x + 12, y + cardH - 12);

            hitRegions.push({
                x: x, y: y, w: cardW, h: cardH,
                onClick: function () { openDatacenter(idx); },
                tooltip: {
                    title: dc.name,
                    lines: [
                        dc.address ? ('Адрес: ' + dc.address) : 'Адрес не указан',
                        'Шкафов: ' + (dc.rackCount || 0),
                        'Блейд-корзин: ' + (dc.chassisCount || 0),
                        'СХД: ' + (dc.storageCount || 0),
                        'Аппаратных серверов: ' + (dc.serverCount || 0),
                        'Из них гипервизоров: ' + (dc.hvCount || 0)
                    ]
                }
            });

            col++;
            if (col >= perRow) {
                col = 0; x = pad; y += cardH + gap;
            } else {
                x += cardW + gap;
            }
        });
    }

    // ==== Структура одного ЦОД: шкафы (rack-elevation) ====
    var UPX = 12; // пикселей на 1 юнит высоты шкафа

    function drawDatacenter(dc) {
        var C = colors();
        ctx.fillStyle = C.bg;
        ctx.fillRect(0, 0, cssW, cssH);
        hitRegions = [];

        ctx.fillStyle = C.text;
        ctx.font = '600 16px system-ui, sans-serif';
        ctx.fillText(dc.name + (dc.address ? ' — ' + dc.address : ''), 16, 26);

        var pad = 16, rackW = 230, gap = 20;
        var x = pad, y = 46;
        var rowMaxH = 0;
        var perRow = Math.max(1, Math.floor((cssW - pad) / (rackW + gap)));
        var col = 0;

        (dc.racks || []).forEach(function (rack) {
            var uHeight = rack.uHeight || 42;
            var bodyH = uHeight * UPX;
            var totalH = bodyH + 40;
            drawRack(x, y, rackW, rack, C);
            rowMaxH = Math.max(rowMaxH, totalH + 10);
            col++;
            if (col >= perRow) {
                col = 0; x = pad; y += rowMaxH; rowMaxH = 0;
            } else {
                x += rackW + gap;
            }
        });
    }

    function unitToY(top, uHeight, startUnit, sizeUnit) {
        // Юниты нумеруются снизу вверх (1 = нижний) - типовая практика.
        var bottomFromTop = (uHeight - (startUnit - 1)) * UPX;
        var topY = bottomFromTop - sizeUnit * UPX;
        return top + Math.max(0, topY);
    }

    function drawRack(x, y, w, rack, C) {
        var uHeight = rack.uHeight || 42;
        var headerH = 24;
        var bodyH = uHeight * UPX;
        var bodyTop = y + headerH;

        ctx.fillStyle = C.text;
        ctx.font = '600 13px system-ui, sans-serif';
        ctx.fillText(trunc(rack.name || 'Шкаф', 26), x, y + 16);

        roundRect(x, bodyTop, w, bodyH, 4);
        ctx.fillStyle = C.card;
        ctx.fill();
        ctx.strokeStyle = C.border;
        ctx.stroke();

        hitRegions.push({
            x: x, y: y, w: w, h: headerH + bodyH,
            tooltip: {
                title: rack.name,
                lines: [
                    'Высота: ' + uHeight + ' U',
                    rack.model ? ('Модель: ' + rack.model) : null,
                    rack.annotation ? rack.annotation : null,
                    'СХД: ' + ((rack.storage || []).length) + ', серверов: ' + ((rack.servers || []).length) + ', корзин: ' + ((rack.chassis || []).length)
                ].filter(Boolean)
            }
        });

        var items = [];
        (rack.storage || []).forEach(function (s) { items.push({ type: 'storage', data: s, startUnit: s.startUnit || 1, sizeUnit: s.sizeUnit || 4 }); });
        (rack.servers || []).forEach(function (s) { items.push({ type: 'server', data: s, startUnit: s.startUnit || 1, sizeUnit: s.sizeUnit || 2 }); });
        (rack.chassis || []).forEach(function (s) { items.push({ type: 'chassis', data: s, startUnit: s.startUnit || 1, sizeUnit: s.sizeUnit || 10 }); });

        // Раскладка "снизу вверх" по startUnit; если startUnit не задан,
        // элементы стекуются последовательно снизу, чтобы не перекрываться.
        var cursorUnit = 1;
        items.sort(function (a, b) { return (a.startUnit || 0) - (b.startUnit || 0); });
        items.forEach(function (it) {
            var su = it.startUnit > 0 ? it.startUnit : cursorUnit;
            var itemY = unitToY(bodyTop, uHeight, su, it.sizeUnit);
            var itemH = it.sizeUnit * UPX;
            cursorUnit = su + it.sizeUnit;

            var iconKey = it.type === 'storage' ? 'storage' : it.type === 'chassis' ? 'chassis' : (it.data.hypervisor ? 'host' : 'server');
            var fillOk = it.data.powerOn === false ? C.muted : C.ok;

            roundRect(x + 6, itemY, w - 12, Math.max(itemH - 3, 8), 3);
            ctx.fillStyle = C.bg;
            ctx.fill();
            ctx.strokeStyle = fillOk;
            ctx.lineWidth = 1.5;
            ctx.stroke();

            if (ICONS[iconKey] && ICONS[iconKey].complete && itemH >= 14) {
                var iconH = Math.min(itemH - 4, 22);
                ctx.drawImage(ICONS[iconKey], x + 9, itemY + (itemH - iconH) / 2, iconH * 1.2, iconH);
            }
            ctx.fillStyle = C.text;
            ctx.font = '11px system-ui, sans-serif';
            var label = it.data.name || '';
            ctx.fillText(trunc(label, 20), x + 34, itemY + itemH / 2 + 4);

            var region = {
                x: x + 6, y: itemY, w: w - 12, h: Math.max(itemH - 3, 8),
                tooltip: itemTooltip(it)
            };
            if (it.type === 'chassis') {
                region.onClick = function () { openChassisDetail(it.data); };
            }
            hitRegions.push(region);
        });
    }

    function itemTooltip(it) {
        var d = it.data;
        if (it.type === 'storage') {
            return {
                title: d.name + ' (СХД)',
                lines: [
                    d.vendor ? (d.vendor + ' ' + d.model) : d.model,
                    'Дисков: ' + (d.diskUsed || 0) + ' / ' + (d.diskMax || 0),
                    'Объём: ' + (d.totalSizeTb ? d.totalSizeTb.toFixed(1) + ' ТБ' : '—'),
                    'Использовано: ' + (d.usedPercent ? d.usedPercent.toFixed(1) + '%' : '—'),
                    'Питание: ' + (d.powerOn ? 'включено' : 'выключено')
                ]
            };
        }
        if (it.type === 'server') {
            return {
                title: d.name + (d.hypervisor ? ' (гипервизор ESXi)' : ' (сервер)'),
                lines: [
                    d.vendor ? (d.vendor + ' ' + d.model) : d.model,
                    d.serialNum ? ('S/N: ' + d.serialNum) : null,
                    d.cpuModel ? ('CPU: ' + d.cpuCount + ' × ' + d.corePerCpu + ' ядер (' + d.cpuModel + ')') : null,
                    d.memoryMb ? ('Память: ' + Math.round(d.memoryMb / 1024) + ' ГБ') : null,
                    'Питание: ' + (d.powerOn ? 'включено' : 'выключено')
                ].filter(Boolean)
            };
        }
        // chassis
        var used = d.used || 0, total = d.slotCount || 0;
        return {
            title: d.name + ' (блейд-корзина)',
            lines: [
                d.model ? d.model : null,
                d.serialNum ? ('S/N: ' + d.serialNum) : null,
                'Занято слотов: ' + used + ' / ' + total,
                'Клик — показать занятые слоты'
            ].filter(Boolean)
        };
    }

    // ==== Детали блейд-корзины: занятые слоты ====
    var chassisDetail = null;
    function openChassisDetail(chassis) {
        chassisDetail = chassis;
        view = 'chassis';
        draw();
    }

    function drawChassisDetail(chassis) {
        var C = colors();
        ctx.fillStyle = C.bg;
        ctx.fillRect(0, 0, cssW, cssH);
        hitRegions = [];

        ctx.fillStyle = C.text;
        ctx.font = '600 16px system-ui, sans-serif';
        ctx.fillText('Блейд-корзина: ' + chassis.name + (chassis.model ? ' (' + chassis.model + ')' : ''), 16, 26);
        ctx.font = '12px system-ui, sans-serif';
        ctx.fillStyle = C.muted;
        ctx.fillText('Занято слотов: ' + (chassis.used || 0) + ' / ' + (chassis.slotCount || 0), 16, 46);

        var slots = chassis.slots || [];
        var pad = 16, slotW = 150, slotH = 60, gap = 10;
        var perRow = Math.max(1, Math.floor((cssW - pad) / (slotW + gap)));
        var x = pad, y = 64, col = 0;

        slots.forEach(function (slot) {
            var occupied = !!slot.occupied;
            roundRect(x, y, slotW, slotH, 6);
            ctx.fillStyle = occupied ? C.card : C.slotEmpty;
            ctx.fill();
            ctx.strokeStyle = occupied ? (slot.powerOn === false ? C.muted : C.ok) : C.border;
            ctx.lineWidth = occupied ? 1.5 : 1;
            ctx.stroke();

            ctx.fillStyle = C.muted;
            ctx.font = '10px system-ui, sans-serif';
            ctx.fillText('Слот ' + slot.num, x + 8, y + 14);

            if (occupied) {
                var iconKey = slot.hypervisor ? 'host' : 'server';
                if (ICONS[iconKey] && ICONS[iconKey].complete) {
                    ctx.drawImage(ICONS[iconKey], x + 8, y + 20, 24, 18);
                }
                ctx.fillStyle = C.text;
                ctx.font = '11px system-ui, sans-serif';
                ctx.fillText(trunc(slot.serverName || '—', 16), x + 38, y + 34);
                if (slot.model) {
                    ctx.font = '10px system-ui, sans-serif';
                    ctx.fillStyle = C.muted;
                    ctx.fillText(trunc(slot.model, 20), x + 38, y + 48);
                }
            } else {
                ctx.fillStyle = C.muted;
                ctx.font = '11px system-ui, sans-serif';
                ctx.fillText('пусто', x + 8, y + 40);
            }

            hitRegions.push({
                x: x, y: y, w: slotW, h: slotH,
                tooltip: occupied ? {
                    title: slot.serverName + (slot.hypervisor ? ' (гипервизор)' : ''),
                    lines: [
                        'Слот: ' + slot.num,
                        slot.model ? ('Модель: ' + slot.model) : null,
                        'Питание: ' + (slot.powerOn ? 'включено' : 'выключено')
                    ].filter(Boolean)
                } : { title: 'Слот ' + slot.num, lines: ['Свободен'] }
            });

            col++;
            if (col >= perRow) { col = 0; x = pad; y += slotH + gap; } else { x += slotW + gap; }
        });
    }

    function trunc(s, n) {
        s = String(s || '');
        return s.length > n ? s.slice(0, n - 1) + '…' : s;
    }

    function openDatacenter(idx) {
        currentDcIdx = idx;
        view = 'datacenter';
        chassisDetail = null;
        backBtn.disabled = false;
        draw();
    }
    backBtn.addEventListener('click', function () {
        if (view === 'chassis') {
            view = 'datacenter';
            chassisDetail = null;
        } else {
            view = 'overview';
            currentDcIdx = -1;
            backBtn.disabled = true;
        }
        hideTooltip();
        draw();
    });

    function draw() {
        if (view === 'overview') {
            drawOverview();
            statusEl.textContent = DATA.datacenters.length + ' ЦОД';
        } else if (view === 'datacenter' && currentDcIdx >= 0) {
            drawDatacenter(DATA.datacenters[currentDcIdx]);
            statusEl.textContent = DATA.datacenters[currentDcIdx].name;
        } else if (view === 'chassis' && chassisDetail) {
            drawChassisDetail(chassisDetail);
            statusEl.textContent = 'Корзина ' + chassisDetail.name;
        }
    }

    // ---- Взаимодействие: hover-подсказка + клик ----
    function regionAt(px, py) {
        for (var i = hitRegions.length - 1; i >= 0; i--) {
            var r = hitRegions[i];
            if (px >= r.x && px <= r.x + r.w && py >= r.y && py <= r.y + r.h) return r;
        }
        return null;
    }
    function hideTooltip() { tooltip.classList.add('d-none'); }
    function showTooltip(r, clientX, clientY) {
        if (!r.tooltip) { hideTooltip(); return; }
        var titleEl = tooltip.querySelector('.infra-tooltip-title');
        var bodyEl = tooltip.querySelector('.infra-tooltip-body');
        titleEl.textContent = r.tooltip.title || '';
        bodyEl.innerHTML = '';
        (r.tooltip.lines || []).forEach(function (line) {
            var div = document.createElement('div');
            div.textContent = line;
            bodyEl.appendChild(div);
        });
        var rect = viewport.getBoundingClientRect();
        var left = clientX - rect.left + 14;
        var top = clientY - rect.top + 14;
        tooltip.style.left = left + 'px';
        tooltip.style.top = top + 'px';
        tooltip.classList.remove('d-none');
    }

    canvas.addEventListener('mousemove', function (ev) {
        var rect = canvas.getBoundingClientRect();
        var px = ev.clientX - rect.left;
        var py = ev.clientY - rect.top;
        var r = regionAt(px, py);
        canvas.style.cursor = r && r.onClick ? 'pointer' : 'default';
        if (r) showTooltip(r, ev.clientX, ev.clientY); else hideTooltip();
    });
    canvas.addEventListener('mouseleave', hideTooltip);
    canvas.addEventListener('click', function (ev) {
        var rect = canvas.getBoundingClientRect();
        var px = ev.clientX - rect.left;
        var py = ev.clientY - rect.top;
        var r = regionAt(px, py);
        if (r && r.onClick) { hideTooltip(); r.onClick(); }
    });

    Object.keys(ICONS).forEach(function (k) { ICONS[k].onload = function () { draw(); }; });

    resize();
})();
</script>
