<div class="hero-text mb-3">
    <h2>Справочник населённых пунктов</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="Update" method="POST">
    <input type="hidden" name="page" id="pageField" value="{{.Page}}">

    <div class="hw-filter-bar mb-3 p-3 rounded border">
        <div class="row g-3 align-items-end">
            <div class="col-6 col-md-3">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.LocalityFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Oktmo" class="form-label">Код ОКТМО (начало)</label>
                <input type="text" id="Oktmo" name="Oktmo" class="form-control" value="{{.LocalityFilter.Oktmo}}">
            </div>
            <div class="col-6 col-md-2">
                <label for="RegionCode" class="form-label">Код региона</label>
                <input type="text" id="RegionCode" name="RegionCode" class="form-control" value="{{.LocalityFilter.RegionCode}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="RegionName" class="form-label">Субъект РФ</label>
                <input type="text" id="RegionName" name="RegionName" class="form-control" value="{{.LocalityFilter.RegionName}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <span class="text-muted small">Найдено записей: {{.LocalityCount}}</span>
                <a href="/admin/importlocalities" class="btn btn-outline-secondary rounded-pill px-3 ms-auto">Импорт из ОКТМО</a>
                <a href="/admin/localityedit/0" class="btn btn-success rounded-pill px-3">Добавить</a>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Код ОКТМО</th>
                <th>Тип</th>
                <th>Наименование</th>
                <th>Муниципальное образование</th>
                <th>Субъект РФ</th>
                <th class="text-center">Код региона</th>
            </tr>
        </thead>
        <tbody>
            {{range $l := .LocalityList}}
            <tr>
                <td class="text-end">{{$l.Id}}</td>
                <td><a href="/admin/localityedit/{{$l.Id}}">{{$l.Oktmo}}</a></td>
                <td>{{$l.TypeShort}}</td>
                <td>{{$l.Name}}</td>
                <td>{{$l.Municipality}}</td>
                <td>{{$l.RegionName}}</td>
                <td class="text-center">{{$l.RegionCode}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="7" class="text-center text-muted py-4">По выбранным условиям фильтра записи не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.LocalityCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="locPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="locPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="locPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="locPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="locPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<script>
    function locPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function locPagerChangePageSize(size) {
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
