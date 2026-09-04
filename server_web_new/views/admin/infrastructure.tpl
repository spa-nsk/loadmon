<div class="hero-text mb-3">
    <h2>Архитектура инфраструктуры</h2>
    <p class="text-muted small mb-0">
        Интерактивная canvas-диаграмма на основе данных VMware, HP OneView, СХД и Zabbix (только активное оборудование).
        Клик по слоту, серверу, СХД или кластеру — подсказка с деталями.
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}

<div class="infra-canvas-shell border rounded mb-3">
    <ul class="nav nav-tabs px-2 pt-2" id="infraPageTabs" role="tablist">
        <li class="nav-item" role="presentation">
            <button class="nav-link active" type="button" data-infra-page="0">Аппаратная часть</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" type="button" data-infra-page="1">Виртуализация</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" type="button" data-infra-page="2">Мониторинг</button>
        </li>
        <li class="nav-item" role="presentation">
            <button class="nav-link" type="button" data-infra-page="3">Кластеры vSphere</button>
        </li>
    </ul>

    <div class="infra-canvas-toolbar d-flex flex-wrap align-items-center gap-3 px-3 py-2 border-top border-bottom">
        <div class="d-flex flex-wrap align-items-center gap-2 infra-legend">
            <span class="small text-muted me-1">Легенда:</span>
            <span class="infra-legend-item"><img src="/static/infracanvas/icons/blade-chassis.svg" alt=""> Блэйд-корзина</span>
            <span class="infra-legend-item"><img src="/static/infracanvas/icons/blade-server.svg" alt=""> Блэйд</span>
            <span class="infra-legend-item"><img src="/static/infracanvas/icons/rack-server.svg" alt=""> Rack-сервер</span>
            <span class="infra-legend-item"><img src="/static/infracanvas/icons/storage.svg" alt=""> СХД</span>
            <span class="infra-legend-item"><span class="infra-slot-sample empty"></span> Пустой слот</span>
        </div>
        <div class="ms-auto small text-muted" id="infraCanvasStatus">Загрузка canvas…</div>
    </div>

    <div class="infra-canvas-viewport position-relative">
        <canvas id="infra-canvas" width="1280" height="720"></canvas>
        <div id="infra-canvas-tooltip" class="infra-canvas-tooltip d-none" role="tooltip">
            <div class="infra-tooltip-title fw-semibold"></div>
            <div class="infra-tooltip-body small"></div>
        </div>
    </div>
</div>

<!-- Данные передаются в base64 через data-атрибут (не текст <script>) -
     обходит поломку JSON html/template-экранированием внутри <script>, см.
     подробный комментарий в controllers/admininfrastructure.go. Раскодируются
     в infracanvas/main.go:loadData через atob()+JSON.parse. -->
<div id="infra-canvas-data" class="d-none" data-payload-b64="{{.InfraJSON}}"></div>

<script src="/static/infracanvas/wasm_exec.js"></script>
<script>
(function () {
    const statusEl = document.getElementById('infraCanvasStatus');
    const tabs = document.querySelectorAll('#infraPageTabs [data-infra-page]');
    const viewport = document.querySelector('.infra-canvas-viewport');
    const canvas = document.getElementById('infra-canvas');
    const tooltip = document.getElementById('infra-canvas-tooltip');

    function setStatus(text) {
        if (statusEl) statusEl.textContent = text;
    }

    function hideTooltip() {
        if (tooltip) tooltip.classList.add('d-none');
    }

    function resizeCanvas() {
        if (!viewport || !canvas) return;
        const w = Math.max(640, viewport.clientWidth - 4);
        const h = Math.max(360, Math.floor(w * 9 / 16));
        canvas.style.width = w + 'px';
        canvas.style.height = h + 'px';
        if (typeof window.resizeInfraCanvas === 'function') {
            window.resizeInfraCanvas(w, h);
        } else {
            canvas.width = w;
            canvas.height = h;
        }
    }

    tabs.forEach(function (btn) {
        btn.addEventListener('click', function () {
            tabs.forEach(function (b) { b.classList.remove('active'); });
            btn.classList.add('active');
            hideTooltip();
            const page = parseInt(btn.getAttribute('data-infra-page'), 10) || 0;
            if (typeof window.setInfraCanvasPage === 'function') {
                window.setInfraCanvasPage(page);
            }
        });
    });

    if (viewport) {
        viewport.addEventListener('click', function (ev) {
            if (ev.target !== canvas) hideTooltip();
        });
    }
    document.addEventListener('keydown', function (ev) {
        if (ev.key === 'Escape') hideTooltip();
    });

    window.addEventListener('resize', function () {
        resizeCanvas();
    });

    if (!WebAssembly || !WebAssembly.instantiateStreaming) {
        setStatus('WebAssembly не поддерживается браузером');
        return;
    }

    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('/static/infracanvas/infracanvas.wasm'), go.importObject)
        .then(function (result) {
            go.run(result.instance);
            resizeCanvas();
            setStatus('Canvas готов — кликните элемент для подсказки');
        })
        .catch(function (err) {
            console.error(err);
            setStatus('Не удалось загрузить WASM-модуль infracanvas');
        });
})();
</script>
