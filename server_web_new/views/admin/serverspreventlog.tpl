<div class="hero-text mb-3">
    <h2>Журнал изменений карточек серверов</h2>
    <p class="text-muted small mb-0">Ручные правки формы /sprsrv/updatebase (создание и изменение полей), с указанием автора. Автоматические обновления из VMware/OneView/Zabbix сюда не пишутся (см. поле "Источник" в самой карточке). Хранится 180 суток.</p>
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
                <label for="filterType" class="form-label">Тип события</label>
                <select id="filterType" name="type" class="form-select" onchange="document.getElementById('pageField').value=1;document.getElementById('Update').submit();">
                    <option value="all" {{if eq .CurType "all"}}selected{{end}}>Все</option>
                    {{range $t := .TypeList}}
                    <option value="{{$t.Value}}" {{if eq $.CurType (printf "%d" $t.Value)}}selected{{end}}>{{$t.Label}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="filterServer" class="form-label">Id сервера</label>
                <input type="number" id="filterServer" name="server" class="form-control" value="{{if gt .CurServerId 0}}{{.CurServerId}}{{end}}" placeholder="все">
            </div>
            <div class="col-6 col-md-3">
                <label for="filterFrom" class="form-label">Дата с</label>
                <input type="date" id="filterFrom" name="from" class="form-control" value="{{.FromDate}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="filterTo" class="form-label">Дата по</label>
                <input type="date" id="filterTo" name="to" class="form-control" value="{{.ToDate}}">
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Найти</button>
                <span class="text-muted small">Найдено записей: {{.TotalRecords}}</span>
                <div class="d-flex gap-2 ms-auto">
                    <a href="#" onclick="serverspreventlogExportXLSX(); return false;" class="btn btn-outline-success btn-sm rounded-pill px-3">Экспорт в XLSX</a>
                    <a href="/admin/serverspreventlog" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
                </div>
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="serverspreventlogFilterStickyToggle" onchange="serverspreventlogToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="serverspreventlogFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
            <thead>
                <tr>
                    <th class="text-center">Id</th>
                    <th>Время</th>
                    <th>Тип события</th>
                    <th>Сервер</th>
                    <th>Поле</th>
                    <th>Было / стало</th>
                    <th>Автор</th>
                    <th>Сообщение</th>
                </tr>
            </thead>
            <tbody>
                {{range $row := .Rows}}
                <tr>
                    <td class="text-end">{{$row.Id}}</td>
                    <td>{{$row.Time.Format "2006-01-02 15:04:05"}}</td>
                    <td>{{$row.TypeLabel}}</td>
                    <td><a href="/sprsrv/updatebase/all/{{$row.ServerId}}" target="_blank">{{$row.Key}}</a></td>
                    <td>{{if $row.FieldName}}{{$row.FieldName}}{{else}}&mdash;{{end}}</td>
                    <td style="max-width:260px; word-wrap:break-word;">
                        {{if $row.FieldName}}
                            <span class="text-muted">{{$row.OldValue}}</span> &rarr; <strong>{{$row.NewValue}}</strong>
                        {{else}}
                            <span class="text-muted">&mdash;</span>
                        {{end}}
                    </td>
                    <td>{{if $row.UserName}}{{$row.UserName}}{{else}}&mdash;{{end}}</td>
                    <td style="word-break:break-all;">{{$row.Message}}</td>
                </tr>
                {{else}}
                <tr>
                    <td colspan="8" class="text-center text-muted py-4">По выбранным условиям фильтра записи не найдены</td>
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
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="serverspreventlogPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="serverspreventlogPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="serverspreventlogPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="serverspreventlogPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="serverspreventlogPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<!-- Общие стили закрепляемой панели фильтра (.hw-filter-bar), "классической"
     таблицы со sticky-заголовком (--spr-thead-top) и панели пагинации
     (.pager-bar-sticky) вынесены в server_web_new/static/css/loadmon.css,
     страница построена по образцу /admin/hardwareeventlog. -->

<script>
    (function () {
        var STORAGE_KEY = 'serverspreventlog.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('serverspreventlogFilterStickyToggle');
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

        window.serverspreventlogToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Экспорт в XLSX учитывает текущие значения фильтра прямо из полей формы -
    // тот же приём, что и /admin/hardwareeventlog (см. controllers/adminserverspreventlog.go).
    function serverspreventlogExportXLSX() {
        var type = document.getElementById('filterType').value;
        var server = document.getElementById('filterServer').value;
        var from = document.getElementById('filterFrom').value;
        var to = document.getElementById('filterTo').value;
        var params = new URLSearchParams();
        params.set('type', type);
        if (server) params.set('server', server);
        if (from) params.set('from', from);
        if (to) params.set('to', to);
        params.set('export', 'xlsx');
        window.location.href = '/admin/serverspreventlog?' + params.toString();
    }

    function serverspreventlogPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function serverspreventlogPagerChangePageSize(size) {
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
