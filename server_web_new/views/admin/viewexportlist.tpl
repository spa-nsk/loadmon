<!-- Контроллер: controllers/adminviewexport.go (AdminViewExport).
     Стиль пагинации - по образцу /admin/metricsexport
     (views/admin/metricsexport.tpl). Колонки таблицы динамические - берутся
     из результата SQL-запроса (.Columns), а не заданы в шаблоне. -->

<div class="hero-text mb-3">
    <h2>Выгрузка по View: {{.ModeTitle}}</h2>
    <p class="text-muted">
        Источник - файл запроса <code>{{.ModeName}}.sql</code>. Только просмотр -
        редактирование данных на этой странице не предусмотрено.
        <a href="/admin/viewexport">&laquo; К списку режимов</a>
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<div class="row mt-2 mb-3">
    <div class="col-12 d-flex flex-wrap gap-2 align-items-center">
        <a href="/admin/viewexport/{{.ModeName}}?export=csv" class="btn btn-outline-primary rounded-pill px-3">Выгрузить в CSV</a>
        <a href="/admin/viewexport/{{.ModeName}}?export=xlsx" class="btn btn-outline-success rounded-pill px-3">Выгрузить в XLSX</a>
    </div>
</div>

{{if .Rows}}
<div class="table-responsive">
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
                <td>{{$v}}</td>
                {{end}}
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<div class="pager-bar d-flex justify-content-between align-items-center mt-2">
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
