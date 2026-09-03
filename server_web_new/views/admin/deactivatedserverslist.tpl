<!-- Контроллер: controllers/admindeactivatedservers.go (AdminDeactivatedServersList).
     Стиль фильтра/пагинации - по образцу /admin/metricsexport
     (views/admin/metricsexport.tpl), чекбоксы выбора и кнопка "Восстановить
     выделенные" - по образцу /sprsrv/view2 → /sprsrv/updateselectedbase
     (см. controllers/sprsrvview2.go, sprsrvupdateselectedbase.go). -->

<div class="hero-text mb-3">
    <h2>Серверы, выведенные из учёта</h2>
    <p class="text-muted">
        Записи справочника серверов со статусом Stat=0 - автоматическая сверка с
        VMware/HPE OneView/Zabbix посчитала, что сервер пропал из соответствующей
        системы, и вывела запись из учёта (PowerState=выключен, Active=выведен из
        эксплуатации). Часть таких записей может быть выведена ошибочно (временная
        недоступность источника, смена идентификатора и т.п.) - здесь их можно
        вернуть в работу, по одной или группой. Возврат в работу происходит только
        когда явно выбраны значения "Включен" и "Активен" - тогда Stat автоматически
        становится 1, и правятся связанные записи по ВМ/аппаратному серверу.
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="DeactivatedServersForm" method="POST" class="deactivated-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-4">
            <label for="filterName" class="form-label">Имя сервера содержит</label>
            <input type="text" id="filterName" name="name" class="form-control"
                placeholder="необязательно" value="{{.CurName}}">
        </div>
        <div class="col-6 col-md-4">
            <label for="filterType" class="form-label">Тип</label>
            <select id="filterType" name="type" class="form-select">
                <option value="all" {{if or (eq .CurType "") (eq .CurType "all")}}selected{{end}}>Все типы</option>
                <option value="0" {{if eq .CurType "0"}}selected{{end}}>Аппаратный сервер</option>
                <option value="1" {{if eq .CurType "1"}}selected{{end}}>Виртуальный сервер</option>
                <option value="2" {{if eq .CurType "2"}}selected{{end}}>Гипервизор</option>
            </select>
        </div>
        <div class="col-6 col-md-4">
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="deactivatedServersFind()">Найти</button>
        </div>
    </div>
    <input type="hidden" name="page" id="pageFieldHidden" value="{{.Page}}">

    {{if .Rows}}
    <div class="table-responsive mt-3">
        <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
            <thead>
                <tr>
                    <th class="text-center">Выбор</th>
                    <th class="text-center">Тип</th>
                    <th>Имя сервера</th>
                    <th>vCenter / Datacenter / Cluster</th>
                    <th>Серийный номер</th>
                    <th class="text-center">Zabbix</th>
                    <th class="text-center">PowerState</th>
                    <th class="text-center">Active</th>
                    <th class="text-center">Источник (SourceUpdateType)</th>
                    <th>Выведен из учёта</th>
                    <th class="text-center">Действие</th>
                </tr>
            </thead>
            <tbody>
                {{range $r := .Rows}}
                <tr>
                    <td class="text-center"><input name="chSelection{{$r.Id}}" class="form-check-input" type="checkbox" id="Selection{{$r.Id}}" value="on"></td>
                    <td class="text-center">
                        {{if eq $r.Type 1}}виртуальный{{else if eq $r.Type 2}}гипервизор{{else}}аппаратный{{end}}
                    </td>
                    <td>
                        {{$r.Name}}
                        {{if ne $r.NameFQDN ""}}<div class="small text-muted">{{$r.NameFQDN}}</div>{{end}}
                    </td>
                    <td class="small text-muted">{{$r.Vcenter}} {{if ne $r.Datacenter ""}}/ {{$r.Datacenter}}{{end}} {{if ne $r.Cluster ""}}/ {{$r.Cluster}}{{end}}</td>
                    <td>{{$r.SerialNumber}}</td>
                    <td class="text-center">{{if gt $r.ZabbixHostId 0}}<span class="badge rounded-pill text-bg-info">Zabbix</span>{{end}}</td>
                    <td class="text-center">{{if eq $r.PowerState 1}}Включен{{else if eq $r.PowerState 2}}Выключен{{else}}—{{end}}</td>
                    <td class="text-center">{{if eq $r.Active 1}}Активен{{else if eq $r.Active 2}}Выведен{{else}}—{{end}}</td>
                    <td class="text-center">
                        {{if eq $r.SourceUpdateType 1}}Вручную{{else if eq $r.SourceUpdateType 2}}VMware{{else if eq $r.SourceUpdateType 3}}HPE OneView{{else if eq $r.SourceUpdateType 4}}Zabbix{{else}}не указано{{end}}
                    </td>
                    <td class="small">{{if not $r.TimeDeActiveState.IsZero}}{{$r.TimeDeActiveState.Format "2006-01-02 15:04"}}{{else}}—{{end}}</td>
                    <td class="text-center">
                        <a href="/admin/deactivatedservers/restore/{{$r.Id}}" class="btn btn-sm btn-outline-success rounded-pill">Восстановить</a>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <div class="row mt-2">
        <div class="col-12 d-flex flex-wrap gap-2 align-items-center">
            <button type="submit" name="RestoreSelectedBTN" value="1" class="btn btn-success rounded-pill px-3">
                Восстановить выделенные
            </button>
        </div>
    </div>

    <div class="pager-bar d-flex justify-content-between align-items-center mt-2">
        <span class="text-muted small">Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.RowCount}}</span>
        <div class="d-flex align-items-center gap-2 pager-nav">
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize"
                style="width:auto;" onchange="deactivatedServersPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="deactivatedServersPagerGoTo(1)"
                {{if eq .Page 1}}disabled{{end}}>&laquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="deactivatedServersPagerGoTo({{.Page}}-1)"
                {{if eq .Page 1}}disabled{{end}}>&lsaquo;</button>
            <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="deactivatedServersPagerGoTo({{.Page}}+1)"
                {{if eq .Page .TotalPages}}disabled{{end}}>&rsaquo;</button>
            <button type="button" class="btn btn-outline-secondary btn-sm" onclick="deactivatedServersPagerGoTo({{.TotalPages}})"
                {{if eq .Page .TotalPages}}disabled{{end}}>&raquo;</button>
        </div>
    </div>
    {{else}}
    <p class="text-muted mt-3">Нет серверов, выведенных из учёта, по текущему фильтру.</p>
    {{end}}
</form>

<script>
    function deactivatedServersFind() {
        var pageField = document.getElementById('pageFieldHidden');
        if (pageField) pageField.value = '1';
        document.getElementById('DeactivatedServersForm').submit();
    }

    function deactivatedServersPagerGoTo(page) {
        var url = new URL(window.location.href);
        url.searchParams.set('page', page);
        url.searchParams.set('name', document.getElementById('filterName').value);
        url.searchParams.set('type', document.getElementById('filterType').value);
        window.location.href = url.toString();
    }

    function deactivatedServersPagerChangePageSize(size) {
        var url = new URL(window.location.href);
        url.searchParams.set('pagesize', size);
        url.searchParams.set('page', '1');
        url.searchParams.set('name', document.getElementById('filterName').value);
        url.searchParams.set('type', document.getElementById('filterType').value);
        window.location.href = url.toString();
    }
</script>
