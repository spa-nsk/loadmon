<div class="hero-text mb-3">
    <h2>Архитектура инфраструктуры</h2>
    <p class="text-muted small mb-0">
        Интерактивная canvas-диаграмма на основе данных VMware, HP OneView, СХД и Zabbix (только активное оборудование).
        Отрисовка выполняется WASM-модулем <code>go-canvas</code>.
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

    <div class="infra-canvas-viewport">
        <canvas id="infra-canvas" width="1280" height="760"></canvas>
    </div>
</div>

<script type="application/json" id="infra-canvas-data">{{.InfraJSON}}</script>

<script src="/static/infracanvas/wasm_exec.js"></script>
<script>
(function () {
    const statusEl = document.getElementById('infraCanvasStatus');
    const tabs = document.querySelectorAll('#infraPageTabs [data-infra-page]');

    function setStatus(text) {
        if (statusEl) statusEl.textContent = text;
    }

    tabs.forEach(function (btn) {
        btn.addEventListener('click', function () {
            tabs.forEach(function (b) { b.classList.remove('active'); });
            btn.classList.add('active');
            const page = parseInt(btn.getAttribute('data-infra-page'), 10) || 0;
            if (typeof window.setInfraCanvasPage === 'function') {
                window.setInfraCanvasPage(page);
            }
        });
    });

    let raw = document.getElementById('infra-canvas-data');
    try {
        window.infraCanvasData = JSON.parse(raw ? raw.textContent : '{}');
    } catch (e) {
        window.infraCanvasData = {};
        setStatus('Ошибка разбора JSON данных');
    }

    if (!WebAssembly || !WebAssembly.instantiateStreaming) {
        setStatus('WebAssembly не поддерживается браузером');
        return;
    }

    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('/static/infracanvas/infracanvas.wasm'), go.importObject)
        .then(function (result) {
            go.run(result.instance);
            setStatus('Canvas готов');
        })
        .catch(function (err) {
            console.error(err);
            setStatus('Не удалось загрузить WASM-модуль infracanvas');
        });
})();
</script>
