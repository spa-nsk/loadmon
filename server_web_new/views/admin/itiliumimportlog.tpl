<div class="hero-text mb-3">
    <h2>Журнал импорта Итилиум (getCU)</h2>
    <p class="text-muted small mb-0">
        Результат последнего прогона <code>SyncGetCU</code>. Таблица перезаписывается при каждом импорте.
        {{if .HasLastRun}}Последний импорт: {{.LastRun.Format "2006-01-02 15:04:05"}}.{{else}}Импорт ещё не выполнялся.{{end}}
    </p>
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
            <div class="col-6 col-md-3">
                <label for="filterClass" class="form-label">Класс КЕ</label>
                <select id="filterClass" name="cuclass" class="form-select" onchange="document.getElementById('pageField').value=1;document.getElementById('Update').submit();">
                    <option value="all" {{if eq .CurClass "all"}}selected{{end}}>Все</option>
                    {{range $t := .ClassList}}
                    <option value="{{$t.Value}}" {{if eq $.CurClass (printf "%d" $t.Value)}}selected{{end}}>{{$t.Label}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="filterResult" class="form-label">Результат</label>
                <select id="filterResult" name="result" class="form-select" onchange="document.getElementById('pageField').value=1;document.getElementById('Update').submit();">
                    <option value="all" {{if eq .CurResult "all"}}selected{{end}}>Все</option>
                    {{range $t := .ResultList}}
                    <option value="{{$t.Value}}" {{if eq $.CurResult (printf "%d" $t.Value)}}selected{{end}}>{{$t.Label}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="filterKey" class="form-label">Ключ (подстрока)</label>
                <input type="text" id="filterKey" name="key" class="form-control" value="{{.KeyFilter}}" placeholder="code / SN">
            </div>
            <div class="col-6 col-md-3">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Найти</button>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <div class="form-check mb-0">
                    <input class="form-check-input" type="checkbox" id="filterOnlyErr" name="onlyerr" value="1" {{if .OnlyErr}}checked{{end}}
                        onchange="document.getElementById('pageField').value=1;document.getElementById('Update').submit();">
                    <label class="form-check-label small" for="filterOnlyErr">Только ошибки</label>
                </div>
                <span class="text-muted small">Найдено записей: {{.TotalRecords}}</span>
                <div class="d-flex gap-2 ms-auto">
                    <a href="#" onclick="itiliumimportlogExportXLSX(); return false;" class="btn btn-outline-success btn-sm rounded-pill px-3">Экспорт в XLSX</a>
                    <a href="/admin/itiliumimportlog" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
                </div>
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="itiliumimportlogFilterStickyToggle" onchange="itiliumimportlogToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="itiliumimportlogFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Время</th>
                <th>Класс КЕ</th>
                <th>Результат</th>
                <th>Ключ</th>
                <th>Сообщение</th>
            </tr>
        </thead>
        <tbody>
            {{range $row := .Rows}}
            <tr>
                <td class="text-end">{{$row.Id}}</td>
                <td>{{$row.Time.Format "2006-01-02 15:04:05"}}</td>
                <td>{{$row.ClassLabel}}</td>
                <td>
                    {{if eq $row.Result 4}}
                        <span class="badge text-bg-danger">{{$row.ResultLabel}}</span>
                    {{else if eq $row.Result 1}}
                        <span class="badge text-bg-success">{{$row.ResultLabel}}</span>
                    {{else if eq $row.Result 3}}
                        <span class="badge text-bg-secondary">{{$row.ResultLabel}}</span>
                    {{else}}
                        <span class="badge text-bg-primary">{{$row.ResultLabel}}</span>
                    {{end}}
                </td>
                <td style="word-break:break-all;">{{$row.Key}}</td>
                <td style="word-break:break-all;">{{$row.Message}}</td>
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
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="itiliumimportlogPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="itiliumimportlogPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="itiliumimportlogPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="itiliumimportlogPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="itiliumimportlogPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<script>
    (function () {
        var STORAGE_KEY = 'itiliumimportlog.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('itiliumimportlogFilterStickyToggle');
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

        window.itiliumimportlogToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function itiliumimportlogExportXLSX() {
        var params = new URLSearchParams();
        params.set('cuclass', document.getElementById('filterClass').value);
        params.set('result', document.getElementById('filterResult').value);
        var key = document.getElementById('filterKey').value;
        if (key) params.set('key', key);
        if (document.getElementById('filterOnlyErr').checked) params.set('onlyerr', '1');
        params.set('export', 'xlsx');
        window.location.href = '/admin/itiliumimportlog?' + params.toString();
    }

    function itiliumimportlogPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function itiliumimportlogPagerChangePageSize(size) {
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
