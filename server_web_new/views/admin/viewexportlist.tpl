<!-- Контроллер: controllers/adminviewexport.go (AdminViewExport).
     Стиль пагинации - по образцу /admin/metricsexport
     (views/admin/metricsexport.tpl). Колонки таблицы динамические - берутся
     из результата SQL-запроса (.Columns), а не заданы в шаблоне.

     Без обёртки .table-responsive у таблицы (та же причина, что и в
     vmlist.tpl/hwviewlist.tpl - см. комментарий там): .table-responsive
     создаёт СОБСТВЕННЫЙ горизонтальный скролл-контекст внутри страницы, из-за
     чего его полоса прокрутки повисает под таблицей, а не прижимается к низу
     экрана, и заодно ломает position:sticky у thead (он "прилипает" не к
     верху окна, а к верху этого внутреннего контейнера). Единственный
     скролл-контейнер страницы - .content-panel (header.tpl,
     server_web_new/static/css/sidebars.css: overflow:auto по обеим осям) -
     когда широкая таблица переполняет его по ширине, горизонтальная полоса
     прокрутки браузера закономерно оказывается внизу ВСЕГО экрана (у
     .content-panel height:100vh), а не где-то в середине страницы под
     таблицей - то есть штатное поведение браузера здесь и даёт нужный
     "зафиксированный внизу экрана" скролл без отдельного JS.

     ДОБАВЛЕНО (2026-10-01, по запросу пользователя "для всех выгрузок через
     Выгрузку по View сделай фиксацию заголовка таблицы, фиксацию фильтра по
     переключателю как делали ранее"): закреплённый sticky-заголовок таблицы
     (.spr-classic-table thead th, уже был в loadmon.css, просто не
     использовался без пересчёта --spr-thead-top) + переключатель
     "Закреплять фильтр вверху экрана" у .viewexport-toolbar-bar - канонический
     приём из kontext/06-list-page-pattern.md, см. JS внизу файла. Та же
     страница обслуживает ВСЕ режимы выгрузки (server-hw/server-vm/cluster/
     lun/storage-systems/storage-vm и т.п., см. ModeName) - правка в одном
     шаблоне закрывает требование "для всех выгрузок" сразу. -->

<div class="viewexport-toolbar-bar p-3 mb-3 rounded border">
    <div class="d-flex flex-wrap justify-content-between align-items-center gap-2">
        <div>
            <h2 class="h4 mb-1">Выгрузка по View: {{.ModeTitle}}</h2>
            <p class="text-muted small mb-0">
                Источник - файл запроса <code>{{.ModeName}}.sql</code>. Только просмотр -
                редактирование данных на этой странице не предусмотрено.
            </p>
        </div>
        <div class="d-flex flex-wrap gap-2">
            <a href="/admin/viewexport/{{.ModeName}}?export=csv" class="btn btn-outline-primary rounded-pill px-3">Выгрузить в CSV</a>
            <a href="/admin/viewexport/{{.ModeName}}?export=xlsx" class="btn btn-outline-success rounded-pill px-3">Выгрузить в XLSX</a>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-end">
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox"
                    id="viewExportFilterStickyToggle"
                    onchange="viewExportToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted"
                    for="viewExportFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

{{if .Rows}}
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            {{range $c := .Columns}}
            <th>{{$c}}</th>
            {{end}}
        </tr>
    </thead>
    <tbody>
        {{range $row := .Rows}}
        <tr>
            {{range $v := $row}}
            <td class="text-nowrap">{{$v}}</td>
            {{end}}
        </tr>
        {{end}}
    </tbody>
</table>

<div class="pager-bar pager-bar-sticky d-flex justify-content-between align-items-center mt-2">
    <span class="text-muted small">Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.RowCount}}</span>
    <div class="d-flex align-items-center gap-2 pager-nav">
        <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize"
            style="width:auto;" onchange="viewExportPagerChangePageSize(this.value)">
            <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
            <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
            <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
        </select>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="viewExportPagerGoTo(1)"
            {{if eq .Page 1}}disabled{{end}}>&laquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="viewExportPagerGoTo({{.Page}}-1)"
            {{if eq .Page 1}}disabled{{end}}>&lsaquo;</button>
        <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="viewExportPagerGoTo({{.Page}}+1)"
            {{if eq .Page .TotalPages}}disabled{{end}}>&rsaquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="viewExportPagerGoTo({{.TotalPages}})"
            {{if eq .Page .TotalPages}}disabled{{end}}>&raquo;</button>
    </div>
</div>
{{else}}
<p class="text-muted mt-3">Нет данных.</p>
{{end}}

<script>
    // Закрепляемая верхняя панель (.viewexport-toolbar-bar) + sticky-заголовок
    // таблицы (.spr-classic-table thead th, top: var(--spr-thead-top)) - тот
    // же канонический приём, что на остальных страницах-списках, см.
    // kontext/06-list-page-pattern.md. Учтены обе ошибки, задокументированные
    // там же и в CHANGELOG (записи от 2026-09-03 (22)/(24)/(25)):
    // 1) sticky-класс навешен ТОЛЬКО на компактный div с заголовком/кнопками/
    //    переключателем - не на обёртку всей страницы и не на таблицу/пагинацию;
    // 2) таблица ниже - БЕЗ <div class="table-responsive"> (ломает position:
    //    sticky у thead, см. комментарий в начале файла) - здесь такой обёртки
    //    и не было, оставляем как есть.
    (function () {
        var STORAGE_KEY = 'viewexportlist.filterSticky';
        var bar = document.querySelector('.viewexport-toolbar-bar');
        var toggle = document.getElementById('viewExportFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('viewexport-toolbar-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('viewexport-toolbar-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');   // по умолчанию закреплено

        window.viewExportToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function viewExportPagerGoTo(page) {
        var url = new URL(window.location.href);
        url.searchParams.delete('export');
        url.searchParams.set('page', page);
        window.location.href = url.toString();
    }

    function viewExportPagerChangePageSize(size) {
        var url = new URL(window.location.href);
        url.searchParams.delete('export');
        url.searchParams.set('pagesize', size);
        url.searchParams.set('page', '1');
        window.location.href = url.toString();
    }
</script>
