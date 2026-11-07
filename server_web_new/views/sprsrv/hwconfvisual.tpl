<style>
    .hcv-layout { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
    .hcv-side { flex: 0 0 360px; max-width: 100%; display: flex; flex-direction: column; gap: 0.75rem; }
    .hcv-main { flex: 1 1 520px; min-width: 0; }
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
    .hcv-dot-spare { background: #3da5ff; box-shadow: 0 0 6px #3da5ff; }
    .hcv-dot-err { background: #ff453a; box-shadow: 0 0 6px #ff453a; }
    .hcv-toolbar { display: flex; flex-wrap: wrap; gap: 0.35rem; align-items: center; margin-bottom: 0.5rem; }
    .hcv-toolbar .btn { padding: 0.2rem 0.65rem; font-size: 0.85rem; }
    .hcv-toolbar .hcv-sep { width: 1px; align-self: stretch; background: var(--bs-border-color); margin: 0 0.25rem; }
    .hcv-stage {
        position: relative; width: 100%; line-height: 0; border-radius: 0.6rem; overflow: hidden;
        border: 1px solid var(--bs-border-color);
        background: radial-gradient(ellipse at 50% 35%, #2a3140 0%, #12151b 70%, #0a0c10 100%);
    }
    .hcv-stage canvas { max-width: 100%; }
    .hcv-hud {
        display: flex; flex-wrap: wrap; justify-content: space-between; gap: 0.5rem;
        color: var(--bs-secondary-color); font-size: 0.82rem; margin-top: 0.4rem;
    }
    .hcv-tip {
        display: none; position: absolute; z-index: 5; pointer-events: none; line-height: 1.35;
        background: rgba(12, 16, 22, 0.94); color: #e8ecf1; border: 1px solid #4dabf7;
        border-radius: 0.4rem; padding: 0.35rem 0.55rem; font-size: 0.8rem; max-width: 280px;
        box-shadow: 0 4px 14px rgba(0, 0, 0, 0.45);
    }
    .hcv-tip span { color: #aab4c0; }
    .hcv-detail-title { font-weight: 700; font-size: 1rem; color: var(--bs-emphasis-color); word-break: break-word; }
    .hcv-detail-sub { color: var(--bs-secondary-color); margin-bottom: 0.45rem; font-size: 0.9rem; }
    .hcv-detail table { width: 100%; margin: 0.35rem 0; font-size: 0.875rem; }
    .hcv-detail td { padding: 2px 0; vertical-align: top; }
    .hcv-detail td:first-child { color: var(--bs-secondary-color); white-space: nowrap; padding-right: 0.65rem; }
    .hcv-detail .hcv-pw { font-weight: 600; }
    .hcv-detail .hcv-links { display: flex; flex-direction: column; gap: 0.2rem; margin-top: 0.5rem; border-top: 1px solid var(--bs-border-color); padding-top: 0.5rem; }
    .hcv-empty-hint { color: var(--bs-secondary-color); font-size: 0.9rem; }
    .hcv-visible { max-height: 320px; overflow: auto; }
    .hcv-grp { margin-bottom: 0.5rem; }
    .hcv-grp-title { font-weight: 600; font-size: 0.85rem; margin-bottom: 0.2rem; color: var(--bs-emphasis-color); }
    .hcv-li {
        display: block; width: 100%; text-align: left; background: transparent; border: 0;
        color: var(--bs-body-color); font-size: 0.82rem; padding: 1px 4px; border-radius: 0.25rem;
        white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    }
    .hcv-li:hover { background: rgba(77, 171, 247, 0.18); }
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
                <div class="hcv-empty-hint">Наведите курсор на компонент — подсветка и подсказка; кликните — параметры здесь. Клик по корпусу показывает данные сервера.</div>
            </div>
        </div>
        <div class="hcv-card">
            <h6>Доступно с этой стороны</h6>
            <div class="small text-muted mb-2" id="hcvSideLabel"></div>
            <div id="hcvVisible" class="hcv-visible"></div>
        </div>
        <div class="hcv-card">
            <h6>Состояние</h6>
            <ul class="hcv-legend small">
                <li><span class="hcv-dot hcv-dot-on"></span> Включено / OK</li>
                <li><span class="hcv-dot hcv-dot-stby"></span> Выключено</li>
                <li><span class="hcv-dot hcv-dot-unk"></span> Не определено</li>
                <li><span class="hcv-dot hcv-dot-spare"></span> Диск — горячий резерв (метка HS)</li>
                <li><span class="hcv-dot hcv-dot-err"></span> Диск / компонент — отказ</li>
            </ul>
            <div class="text-muted small">
                Колесо мыши — масштаб. Тяните <b>угловые маркеры</b> (или фон) — вращение; с <b>Shift</b> / правой кнопкой — крен.
                Внешний вид строится по бренду (HPE, Dell, Supermicro, DEPO). Вентиляторы, процессоры и память видны при снятом кожухе.
            </div>
        </div>
    </aside>
    <div class="hcv-main" id="hcvMain">
        <div class="hcv-toolbar">
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="front">Спереди</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="rear">Сзади</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="top">Сверху</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="left">Слева</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="right">Справа</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="iso">Изометрия</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="isorear">Изометрия сзади</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="isotop">Изометрия сверху</button>
            <span class="hcv-sep"></span>
            <button type="button" class="btn btn-outline-warning" id="hcvLidBtn" data-hcv-view="lid">Снять кожух</button>
            <span class="hcv-sep"></span>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="roll-l" title="Крен на 90° влево">&#8634;</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="roll-r" title="Крен на 90° вправо">&#8635;</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="zoom-in" title="Приблизить">+</button>
            <button type="button" class="btn btn-outline-secondary" data-hcv-view="zoom-out" title="Отдалить">&minus;</button>
            <button type="button" class="btn btn-outline-primary" data-hcv-view="reset">Сброс</button>
        </div>
        <div style="position:relative;">
            <div class="hcv-stage" id="hcv3dStage"></div>
            <div id="hcvTip" class="hcv-tip"></div>
        </div>
        <div class="hcv-hud">
            <span>Масштаб: <b id="hcvZoom">100%</b></span>
            <span>Модель построена по виду серверов HPE / Dell / Supermicro / DEPO</span>
        </div>
    </div>
</div>

<script src="/static/js/three.min.js?v={{.StaticVer}}"></script>
<script>window.HCV_DATA = {{.VisualJSON}};</script>
<script src="/static/js/server3d.js?v={{.StaticVer}}"></script>
