<style>
    /* стиль выпадающего списка поиска ИТ-системы - скопирован из
       views/sprsrv/view2.tpl (образец для этого поля, initITSystemPicker
       ниже - тот же код 1:1, см. комментарий там). */
    .it-system-dropdown {
        z-index: 1000;
        max-height: 280px;
        overflow-y: auto;
        box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, .25);
    }

    .it-system-dropdown .list-group-item {
        cursor: pointer;
    }

    .it-system-dropdown .list-group-item small {
        display: block;
        color: var(--bs-secondary-color);
    }
</style>

<div class="hero-text mb-3">
    <h2>Выгрузка метрик утилизации</h2>
    <p class="text-muted">
        Единообразная выгрузка метрик CPU / память / диск / сеть по виртуальным машинам и
        аппаратным серверам, зарегистрированным в Zabbix, за произвольный период.
        Период до {{.DetailedPeriodLimitDays}} суток включительно выгружается подробно
        (минутная сетка, значение "продлевается" от последнего известного замера), более
        длинный период - по дням (среднее за календарные сутки). Детальные метрики
        виртуальных машин хранятся только {{.DetailedPeriodLimitDays}} суток и не пишутся
        для рабочих станций Windows - подробности в комментарии к
        <code>AdminMetricsUnifiedExport</code> (controllers/adminmetricsexport.go) и в
        kontext/03-data-model.md.
    </p>
    {{if .Detailed}}
    <p class="small text-success mb-0">Выбранный период ({{.FromDate}} — {{.ToDate}}) уложится в подробный (минутный) режим выгрузки.</p>
    {{else}}
    <p class="small text-warning mb-0">Выбранный период ({{.FromDate}} — {{.ToDate}}) длиннее {{.DetailedPeriodLimitDays}} суток - выгрузка будет по дням (среднее за сутки), не по минутам.</p>
    {{end}}
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="MetricsExportForm" class="metricsexport-filter-bar mb-3 p-3 rounded border" method="GET">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="filterFrom" class="form-label">Период с</label>
            <input type="date" id="filterFrom" name="from" class="form-control" value="{{.FromDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterTo" class="form-label">Период по</label>
            <input type="date" id="filterTo" name="to" class="form-control" value="{{.ToDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterName" class="form-label">Имя сервера/ВМ содержит</label>
            <input type="text" id="filterName" name="name" class="form-control"
                placeholder="необязательно" value="{{.CurName}}">
        </div>
        <div class="col-6 col-md-3 position-relative">
            <label for="ITSystemSearch" class="form-label">ИТ система (код или название)</label>
            <input type="text" id="ITSystemSearch" class="form-control"
                placeholder="Поиск по коду или названию..." autocomplete="off">
            <div id="ITSystemSearchResults" class="list-group position-absolute w-100 d-none it-system-dropdown"></div>
            <!-- фактический фильтр, отправляемый на сервер - код ИТ-системы;
                 подставляется либо выбором из выпадающего списка выше, либо
                 вводится вручную напрямую (см. initITSystemPicker ниже, код
                 скопирован 1:1 из views/sprsrv/view2.tpl - образец для этого
                 поля). -->
            <input type="text" id="ITSystemKey" name="itsystem" class="form-control form-control-sm mt-1"
                placeholder="код ИТ-системы, необязательно" value="{{.CurITSystem}}" autocomplete="off">
            <div id="ITSystemKeyHint" class="form-text"></div>
        </div>
    </div>
    <div class="row g-3 align-items-center mt-1">
        <div class="col-auto">
            <div class="form-check">
                <input class="form-check-input" type="checkbox" id="includeVM" name="includevm"
                    value="on" {{if .IncludeVM}}checked{{end}}>
                <label class="form-check-label" for="includeVM">Виртуальные машины (по данным vm_record)</label>
            </div>
        </div>
        <div class="col-auto">
            <div class="form-check">
                <input class="form-check-input" type="checkbox" id="includeZabbix" name="includezabbix"
                    value="on" {{if .IncludeZabbix}}checked{{end}}>
                <label class="form-check-label" for="includeZabbix">Аппаратные серверы в Zabbix (по данным zabbix_item)</label>
            </div>
        </div>
    </div>
    <div class="row mt-3">
        <div class="col-12 d-flex flex-wrap gap-2 align-items-center">
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="metricsExportSubmit('')"
                title="Обновить таблицу на экране по текущему фильтру">
                Показать
            </button>
            <button type="button" class="btn btn-success rounded-pill px-3" onclick="metricsExportSubmit('xlsx')"
                title="Сформировать и скачать XLSX по всей выборке за указанный период и фильтр (не только текущей странице)">
                Сформировать выгрузку (XLSX)
            </button>
            <a href="/admin/metricsexport" class="btn btn-outline-secondary rounded-pill px-3">Сбросить фильтр</a>
            <div class="form-check form-switch mb-0 ms-auto">
                <input class="form-check-input" type="checkbox"
                    id="metricsExportFilterStickyToggle"
                    onchange="metricsExportToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted"
                    for="metricsExportFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12">
            <span class="text-muted small">
                Совет: чем длиннее период и шире фильтр (или без фильтра - выгрузка по всем
                серверам сразу), тем больше строк. Таблица на экране разбита по страницам, но
                XLSX-выгрузка всегда формируется по всей отфильтрованной выборке целиком - для
                больших выгрузок сужайте период и/или указывайте имя сервера/ИТ-системы.
            </span>
        </div>
    </div>
</form>

<!-- источник данных справочника ИТ-систем для поиска на клиенте - сами
     элементы скрыты, JS читает их data-атрибуты один раз при загрузке
     страницы; html/template экранирует значения -->
<div id="ITSystemDataList" class="d-none">
    {{range $its := .ITSystemSpr}}
    <div class="it-system-item"
        data-key="{{$its.ObjectKey}}"
        data-label="{{$its.Label}}"
        data-full="{{$its.FullName}}"
        data-status="{{$its.Status}}"
        data-type="{{$its.SystemType}}"
        data-biz="{{$its.BusinessColumn}}"
        data-cfo="{{$its.CFO}}"></div>
    {{end}}
</div>

{{if .Rows}}
    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th>Время</th>
                <th>ИТ система</th>
                <th>Имя сервера</th>
                <th>Утилизация CPU %</th>
                <th>Утилизация RAM %</th>
                <th>Диск чтение МБ/сек</th>
                <th>Диск запись МБ/сек</th>
                <th>Сеть приём МБ/сек</th>
                <th>Сеть передача МБ/сек</th>
            </tr>
        </thead>
        <tbody>
            {{range $r := .Rows}}
            <tr>
                <td>{{$r.Timestamp.Format "2006-01-02 15:04:05"}}</td>
                <td>{{$r.ITSystem}}</td>
                <td>{{$r.ObjectName}}</td>
                <td>{{if $r.CPUPercent}}{{$r.CPUPercent}}{{else}}—{{end}}</td>
                <td>{{if $r.MemPercent}}{{$r.MemPercent}}{{else}}—{{end}}</td>
                <td>{{if $r.DiskReadMBs}}{{$r.DiskReadMBs}}{{else}}—{{end}}</td>
                <td>{{if $r.DiskWriteMBs}}{{$r.DiskWriteMBs}}{{else}}—{{end}}</td>
                <td>{{if $r.NetRecvMBs}}{{$r.NetRecvMBs}}{{else}}—{{end}}</td>
                <td>{{if $r.NetTransMBs}}{{$r.NetTransMBs}}{{else}}—{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

<div class="pager-bar pager-bar-sticky d-flex justify-content-between align-items-center mt-2">
    <span class="text-muted small">Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.RowCount}}</span>
    <div class="d-flex align-items-center gap-2 pager-nav">
        <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize"
            style="width:auto;" onchange="metricsExportPagerChangePageSize(this.value)">
            <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
            <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
            <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
        </select>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="metricsExportPagerGoTo(1)"
            {{if eq .Page 1}}disabled{{end}}>&laquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="metricsExportPagerGoTo({{.Page}}-1)"
            {{if eq .Page 1}}disabled{{end}}>&lsaquo;</button>
        <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="metricsExportPagerGoTo({{.Page}}+1)"
            {{if eq .Page .TotalPages}}disabled{{end}}>&rsaquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="metricsExportPagerGoTo({{.TotalPages}})"
            {{if eq .Page .TotalPages}}disabled{{end}}>&raquo;</button>
    </div>
</div>
{{else}}
<p class="text-muted">Нет данных для отображения за выбранный период и фильтр.</p>
{{end}}

<!-- Скрытые "невидимые" поля checkbox для чекбоксов, снятых пользователем - HTML не
     отправляет невыставленный чекбокс вообще, а не значение "off"; контроллер
     (AdminMetricsUnifiedExport) трактует отсутствие параметра includevm/includezabbix как
     "включено" (this.GetString(...) != "off"), поэтому нужно явно передать "off" при снятой
     галочке - иначе снятие чекбокса в форме визуально не будет иметь эффекта. -->
<script>
    function metricsExportSubmit(exportFormat) {
        var form = document.getElementById('MetricsExportForm');

        function ensureHiddenFlag(checkboxId, hiddenName) {
            var cb = document.getElementById(checkboxId);
            var existing = document.getElementById(hiddenName + 'Hidden');
            if (existing) existing.remove();
            if (cb && !cb.checked) {
                var hidden = document.createElement('input');
                hidden.type = 'hidden';
                hidden.id = hiddenName + 'Hidden';
                hidden.name = hiddenName;
                hidden.value = 'off';
                form.appendChild(hidden);
            }
        }
        ensureHiddenFlag('includeVM', 'includevm');
        ensureHiddenFlag('includeZabbix', 'includezabbix');

        var exportField = document.getElementById('exportFieldHidden');
        if (exportField) exportField.remove();
        if (exportFormat) {
            var hidden = document.createElement('input');
            hidden.type = 'hidden';
            hidden.id = 'exportFieldHidden';
            hidden.name = 'export';
            hidden.value = exportFormat;
            form.appendChild(hidden);
        }

        // смена фильтра всегда сбрасывает постраничный просмотр на 1-ю страницу
        var pageField = document.getElementById('pageFieldHidden');
        if (pageField) pageField.remove();
        pageField = document.createElement('input');
        pageField.type = 'hidden';
        pageField.id = 'pageFieldHidden';
        pageField.name = 'page';
        pageField.value = '1';
        form.appendChild(pageField);

        form.submit();
    }

    // Переход по страницам/смена размера страницы - переносим текущие
    // значения фильтра формы через query-параметры (GET), без пересборки
    // всей формы (см. kontext/06-list-page-pattern.md, п.4).
    function metricsExportPagerGoTo(page) {
        var url = new URL(window.location.href);
        url.searchParams.set('page', page);
        window.location.href = url.toString();
    }

    function metricsExportPagerChangePageSize(size) {
        var url = new URL(window.location.href);
        url.searchParams.set('pagesize', size);
        url.searchParams.set('page', '1');
        window.location.href = url.toString();
    }

    // Поле "ИТ система" - поиск по справочнику (код/название/статус/тип/
    // бизнес-направление/ЦФО), выбор элемента списка подставляет его код в
    // ITSystemKey. Код скопирован 1:1 из views/sprsrv/view2.tpl
    // (initITSystemPicker там же, образец для этого поля) - при изменении
    // логики поддерживать в обоих местах синхронно.
    function initITSystemPicker() {
        var keyInput = document.getElementById('ITSystemKey');
        var hint = document.getElementById('ITSystemKeyHint');
        var searchInput = document.getElementById('ITSystemSearch');
        var results = document.getElementById('ITSystemSearchResults');
        var dataList = document.getElementById('ITSystemDataList');
        if (!keyInput || !searchInput || !results || !dataList) return;

        var items = Array.prototype.map.call(dataList.querySelectorAll('.it-system-item'), function (el) {
            return {
                key: el.getAttribute('data-key') || '',
                label: el.getAttribute('data-label') || '',
                full: el.getAttribute('data-full') || '',
                status: el.getAttribute('data-status') || '',
                type: el.getAttribute('data-type') || '',
                biz: el.getAttribute('data-biz') || '',
                cfo: el.getAttribute('data-cfo') || ''
            };
        });

        function findByKey(key) {
            if (!key) return null;
            var lower = key.toLowerCase();
            for (var i = 0; i < items.length; i++) {
                if (items[i].key.toLowerCase() === lower) return items[i];
            }
            return null;
        }

        function describe(item) {
            var parts = [item.key];
            if (item.label) parts.push(item.label);
            if (item.full) parts.push(item.full);
            return parts.join(' — ');
        }

        function updateHint() {
            var match = findByKey(keyInput.value.trim());
            if (match) {
                hint.textContent = describe(match) + (match.status ? ' (' + match.status + ')' : '');
                hint.classList.remove('text-danger');
            } else if (keyInput.value.trim() !== '') {
                hint.textContent = 'Код не найден в справочнике - будет использован как есть';
                hint.classList.add('text-danger');
            } else {
                hint.textContent = '';
                hint.classList.remove('text-danger');
            }
        }

        function hideResults() {
            results.classList.add('d-none');
            results.innerHTML = '';
        }

        function renderResults(query) {
            var q = query.trim().toLowerCase();
            if (q === '') {
                hideResults();
                return;
            }
            var matches = items.filter(function (item) {
                return item.key.toLowerCase().indexOf(q) !== -1 ||
                    item.label.toLowerCase().indexOf(q) !== -1 ||
                    item.full.toLowerCase().indexOf(q) !== -1 ||
                    item.status.toLowerCase().indexOf(q) !== -1 ||
                    item.type.toLowerCase().indexOf(q) !== -1 ||
                    item.biz.toLowerCase().indexOf(q) !== -1 ||
                    item.cfo.toLowerCase().indexOf(q) !== -1;
            }).slice(0, 50);

            if (matches.length === 0) {
                results.innerHTML = '<div class="list-group-item text-muted">Ничего не найдено</div>';
                results.classList.remove('d-none');
                return;
            }

            results.innerHTML = '';
            matches.forEach(function (item) {
                var row = document.createElement('button');
                row.type = 'button';
                row.className = 'list-group-item list-group-item-action';
                row.innerHTML = '<div>' + escapeHtml(item.key) + (item.label ? ' — ' + escapeHtml(item.label) : '') + '</div>' +
                    (item.full ? '<small>' + escapeHtml(item.full) + '</small>' : '');
                row.addEventListener('click', function () {
                    keyInput.value = item.key;
                    searchInput.value = describe(item);
                    updateHint();
                    hideResults();
                });
                results.appendChild(row);
            });
            results.classList.remove('d-none');
        }

        function escapeHtml(s) {
            var div = document.createElement('div');
            div.textContent = s;
            return div.innerHTML;
        }

        searchInput.addEventListener('input', function () {
            renderResults(searchInput.value);
        });
        searchInput.addEventListener('focus', function () {
            if (searchInput.value.trim() !== '') renderResults(searchInput.value);
        });
        document.addEventListener('click', function (evt) {
            if (evt.target !== searchInput && !results.contains(evt.target)) hideResults();
        });

        keyInput.addEventListener('input', updateHint);
        keyInput.addEventListener('blur', updateHint);

        var initial = findByKey(keyInput.value.trim());
        if (initial) {
            searchInput.value = describe(initial);
        }
        updateHint();
    }

    // Закрепляемая панель фильтра (kontext/06-list-page-pattern.md, п.1) -
    // тот же приём, что и на /sprsrv/vm/... (vmlist.tpl), свой ключ
    // localStorage и пересчёт --spr-thead-top для sticky-заголовка таблицы.
    (function () {
        var STORAGE_KEY = 'metricsexport.filterSticky';
        var bar = document.querySelector('.metricsexport-filter-bar');
        var toggle = document.getElementById('metricsExportFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('metricsexport-filter-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('metricsexport-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.metricsExportToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    document.addEventListener('DOMContentLoaded', function () {
        initITSystemPicker();
    });
</script>
