<div class="hero-text mb-3">
    <h2>Журнал действий пользователей</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="Update" method="GET">
    <input type="hidden" name="page" id="pageField" value="{{.Page}}">

    <div class="hw-filter-bar mb-3 p-3 rounded border">
        <div class="row g-3 align-items-end">
            <div class="col-6 col-md-4">
                <label for="filterType" class="form-label">Тип операции</label>
                <select id="filterType" name="type" class="form-select" onchange="document.getElementById('pageField').value=1;document.getElementById('Update').submit();">
                    <option value="all" {{if eq .CurType "all"}}selected{{end}}>Все</option>
                    {{range $t := .TypeList}}
                    <option value="{{$t.Value}}" {{if eq $.CurType (printf "%d" $t.Value)}}selected{{end}}>{{$t.Label}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <span class="text-muted small">Найдено записей: {{.TotalRecords}}</span>
                <div class="d-flex gap-2 ms-auto">
                    <a href="/admin/userlog?type={{.CurType}}&export=xlsx" class="btn btn-outline-success btn-sm rounded-pill px-3">Экспорт в XLSX</a>
                    <a href="/admin/userlog" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
                </div>
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="userlogFilterStickyToggle" onchange="userlogToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="userlogFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
            <thead>
                <tr>
                    <th class="text-center">Id</th>
                    <th>Пользователь</th>
                    <th>Время входа</th>
                    <th>Время операции</th>
                    <th>Тип операции</th>
                    <th>Запрос</th>
                </tr>
            </thead>
            <tbody>
                {{range $row := .Rows}}
                <tr>
                    <td class="text-end">{{$row.Id}}</td>
                    <td>{{$row.UserName}}</td>
                    <td>{{$row.Login.Format "2006-01-02 15:04:05"}}</td>
                    <td>{{$row.Time.Format "2006-01-02 15:04:05"}}</td>
                    <td>{{$row.TypeLabel}}</td>
                    <td style="word-break:break-all;">{{$row.Request}}</td>
                </tr>
                {{else}}
                <tr>
                    <td colspan="6" class="text-center text-muted py-4">По выбранным условиям фильтра записи не найдены</td>
                </tr>
                {{end}}
            </tbody>
        </table>

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.TotalRecords}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="userlogPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="userlogPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="userlogPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="userlogPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="userlogPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<!-- Общие стили закрепляемой панели фильтра (.hw-filter-bar), "классической"
     таблицы со sticky-заголовком (--spr-thead-top) и панели пагинации
     (.pager-bar-sticky) вынесены в server_web_new/static/css/loadmon.css,
     приём идентичен /sprsrv/hwconflist. -->

<script>
    // Закрепление панели фильтра вверху экрана (переключается чекбоксом,
    // выбор запоминается в localStorage) - высота панели пересчитывается в
    // CSS-переменную --spr-thead-top, чтобы заголовок таблицы прилипал сразу
    // под ней, а не перекрывался ею. Приём идентичен /sprsrv/hwconflist.
    (function () {
        var STORAGE_KEY = 'userlog.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('userlogFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('hw-filter-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('hw-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.userlogToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function userlogPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function userlogPagerChangePageSize(size) {
        var pageSizeField = document.createElement('input');
        pageSizeField.type = 'hidden';
        pageSizeField.name = 'pagesize';
        pageSizeField.value = size;
        var form = document.getElementById('Update');
        form.appendChild(pageSizeField);
        var field = document.getElementById('pageField');
        if (field) field.value = 1;
        form.submit();
    }
</script>
