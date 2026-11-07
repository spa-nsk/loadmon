<style>
    .bcv-layout { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
    .bcv-side { flex: 0 0 340px; max-width: 100%; display: flex; flex-direction: column; gap: 0.75rem; }
    .bcv-main { flex: 1 1 520px; min-width: 0; }
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
    .bcv-toolbar { display: flex; flex-wrap: wrap; gap: 0.35rem; align-items: center; margin-bottom: 0.5rem; }
    .bcv-toolbar .btn { padding: 0.2rem 0.65rem; font-size: 0.85rem; }
    .bcv-toolbar .bcv-sep { width: 1px; align-self: stretch; background: var(--bs-border-color); margin: 0 0.25rem; }
    .bcv-stage {
        position: relative; width: 100%; line-height: 0; border-radius: 0.6rem; overflow: hidden;
        border: 1px solid var(--bs-border-color);
        background: radial-gradient(ellipse at 50% 35%, #2a3140 0%, #12151b 70%, #0a0c10 100%);
    }
    .bcv-stage canvas { max-width: 100%; }
    .bcv-hud {
        display: flex; flex-wrap: wrap; justify-content: space-between; gap: 0.5rem;
        color: var(--bs-secondary-color); font-size: 0.82rem; margin-top: 0.4rem;
    }
    .bcv-tip {
        display: none; position: absolute; z-index: 5; pointer-events: none; line-height: 1.35;
        background: rgba(12, 16, 22, 0.94); color: #e8ecf1; border: 1px solid #4dabf7;
        border-radius: 0.4rem; padding: 0.35rem 0.55rem; font-size: 0.8rem; max-width: 280px;
        box-shadow: 0 4px 14px rgba(0, 0, 0, 0.45);
    }
    .bcv-tip span { color: #aab4c0; }
    .bcv-detail-title { font-weight: 700; font-size: 1rem; color: var(--bs-emphasis-color); word-break: break-word; }
    .bcv-detail-sub { color: var(--bs-secondary-color); margin-bottom: 0.45rem; font-size: 0.9rem; }
    .bcv-detail table { width: 100%; margin: 0.35rem 0; font-size: 0.875rem; }
    .bcv-detail td { padding: 2px 0; vertical-align: top; }
    .bcv-detail td:first-child { color: var(--bs-secondary-color); white-space: nowrap; padding-right: 0.65rem; }
    .bcv-detail .bcv-pw { font-weight: 600; }
    .bcv-detail .bcv-links { display: flex; flex-direction: column; gap: 0.2rem; margin-top: 0.5rem; border-top: 1px solid var(--bs-border-color); padding-top: 0.5rem; }
    .bcv-empty-hint { color: var(--bs-secondary-color); font-size: 0.9rem; }
    .bcv-visible { max-height: 320px; overflow: auto; }
    .bcv-grp { margin-bottom: 0.5rem; }
    .bcv-grp-title { font-weight: 600; font-size: 0.85rem; margin-bottom: 0.2rem; color: var(--bs-emphasis-color); }
    .bcv-li {
        display: block; width: 100%; text-align: left; background: transparent; border: 0;
        color: var(--bs-body-color); font-size: 0.82rem; padding: 1px 4px; border-radius: 0.25rem;
        white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    }
    .bcv-li:hover { background: rgba(77, 171, 247, 0.18); }
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
            <h6>Выделенный элемент</h6>
            <div id="bcvDetail" class="bcv-detail">
                <div class="bcv-empty-hint">Наведите курсор на компонент — подсветка и подсказка; кликните — параметры здесь.</div>
            </div>
        </div>
        <div class="bcv-card">
            <h6>Доступно с этой стороны</h6>
            <div class="small text-muted mb-2" id="bcvSideLabel"></div>
            <div id="bcvVisible" class="bcv-visible"></div>
        </div>
        <div class="bcv-card">
            <h6>Состояние</h6>
            <ul class="bcv-legend small">
                <li><span class="bcv-dot bcv-dot-on"></span> Включено / OK</li>
                <li><span class="bcv-dot bcv-dot-stby"></span> Выключено / неисправно</li>
                <li><span class="bcv-dot bcv-dot-unk"></span> Не определено</li>
            </ul>
            <div class="text-muted small">
                Колесо мыши — масштаб. Тяните <b>угловые маркеры</b> (или фон) — вращение; с <b>Shift</b> / правой кнопкой — крен.
                При повороте на другую сторону доступные компоненты кратко подсвечиваются.
            </div>
        </div>
    </aside>
    <div class="bcv-main" id="bcvMain">
        <div class="bcv-toolbar">
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="front">Спереди</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="rear">Сзади</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="left">Слева</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="right">Справа</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="top">Сверху</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="iso">Изометрия</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="isorear">Изометрия сзади</button>
            <span class="bcv-sep"></span>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="roll-l" title="Крен на 90° влево">&#8634;</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="roll-r" title="Крен на 90° вправо">&#8635;</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="zoom-in" title="Приблизить">+</button>
            <button type="button" class="btn btn-outline-secondary" data-bcv-view="zoom-out" title="Отдалить">&minus;</button>
            <button type="button" class="btn btn-outline-primary" data-bcv-view="reset">Сброс</button>
        </div>
        <div style="position:relative;">
            <div class="bcv-stage" id="bcv3dStage"></div>
            <div id="bcvTip" class="bcv-tip"></div>
        </div>
        <div class="bcv-hud">
            <span>Масштаб: <b id="bcvZoom">100%</b></span>
            <span>Изображение построено по виду заводского исполнения вендора (HPE c7000 / Synergy 12000)</span>
        </div>
    </div>
</div>

<script src="/static/js/three.min.js?v={{.StaticVer}}"></script>
<script>window.BCV_DATA = {{.ChassisJSON}};</script>
<script src="/static/js/bladechassis3d.js?v={{.StaticVer}}"></script>
