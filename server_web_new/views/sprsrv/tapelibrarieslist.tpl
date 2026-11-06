<div class="hero-text mb-3">
    <h2>Справочник ленточных библиотек</h2>
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
                <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                    <option value="all" {{if eq .TapeLibrariesFilter.ItiliumStatus "all"}}selected{{end}}>Все</option>
                    {{range $s := .ItiliumStatuses}}
                    <option value="{{$s.Value}}" {{if eq $.TapeLibrariesFilter.ItiliumStatus (printf "%d" $s.Value)}}selected{{end}}>{{$s.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Name" class="form-label">Название</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.TapeLibrariesFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.TapeLibrariesFilter.Model}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="SerialNum" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.TapeLibrariesFilter.SerialNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="TechNumberIT" class="form-label">Технический номер ИТ</label>
                <input type="text" id="TechNumberIT" name="TechNumberIT" class="form-control" value="{{.TapeLibrariesFilter.TechNumberIT}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="PcName" class="form-label">Имя ПК</label>
                <input type="text" id="PcName" name="PcName" class="form-control" value="{{.TapeLibrariesFilter.PcName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="tlFilterDatacenterSiteId" class="form-label">Площадка/ЦОД</label>
                <select id="tlFilterDatacenterSiteId" class="form-select">
                    <option value="0">Все</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}">{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="RackCabinetId" class="form-label">Шкаф</label>
                <select id="RackCabinetId" name="RackCabinetId" class="form-select">
                    <option value="0">Все</option>
                    {{range $rc := .RackCabinets}}
                    <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.TapeLibrariesFilter.RackCabinetId (printf "%d" $rc.Id)}}selected{{end}}>{{$rc.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" name="ViewBTN" value="Просмотр" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <a href="/sprsrv/tapelibrariesedit/0" title="Добавить новую ленточную библиотеку" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12">
                <span class="text-muted small">Найдено записей: {{.TapeLibrariesCount}}</span>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Статус</th>
                <th>Название</th>
                <th>Модель</th>
                <th>Серийный №</th>
                <th>Инв. №</th>
                <th>IP</th>
                <th>Сервер (Имя ПК)</th>
                <th>Размещение</th>
                <th class="text-end">Юнитов</th>
            </tr>
        </thead>
        <tbody>
            {{range $tl := .TapeLibrariesList}}
            <tr class="row-context-target" data-menu-id="tlRowMenu{{$tl.Id}}">
                <td class="text-end">{{$tl.Id}}</td>
                <td>
                    <span class="badge {{if eq $tl.ItiliumStatus 1}}text-bg-success{{else if eq $tl.ItiliumStatus 3}}text-bg-dark{{else if or (eq $tl.ItiliumStatus 2) (eq $tl.ItiliumStatus 5)}}text-bg-secondary{{else}}text-bg-light{{end}}">
                        {{if eq $tl.ItiliumStatus 1}}В работе{{else if eq $tl.ItiliumStatus 2}}Выведен{{else if eq $tl.ItiliumStatus 3}}Списан{{else if eq $tl.ItiliumStatus 4}}Свободно{{else if eq $tl.ItiliumStatus 5}}На складе{{else if eq $tl.ItiliumStatus 6}}Зарезервировано{{else if eq $tl.ItiliumStatus 7}}В ремонте{{else if eq $tl.ItiliumStatus 8}}В пути{{else if eq $tl.ItiliumStatus 9}}Черновик{{else}}Не указано{{end}}
                    </span>
                </td>
                <td><a href="/sprsrv/tapelibrariesedit/{{$tl.Id}}" target="_blank">{{if $tl.Name}}{{$tl.Name}}{{else}}{{$tl.TechnicalNumberIT}}{{end}}</a></td>
                <td>{{$tl.Model}}</td>
                <td>{{$tl.SerialNum}}</td>
                <td>{{$tl.InventoryNum}}</td>
                <td>{{$tl.IPAddress}}</td>
                <td>
                    {{if $tl.ServerName}}<a href="/sprsrv/updatebase-tab/{{$tl.ServerId}}" target="_blank">{{$tl.ServerName}}</a>{{else}}{{$tl.PcName}}{{end}}
                </td>
                <td>{{$tl.Placement}}</td>
                <td class="text-end">{{if gt $tl.SizeUnit 0}}{{$tl.SizeUnit}}{{end}}</td>
                <td id="tlRowMenu{{$tl.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/tapelibrariesedit/{{$tl.Id}}" target="_blank">Редактировать</a>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="10" class="text-center text-muted py-4">По выбранным условиям фильтра записи не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.TapeLibrariesCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="tlPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="tlPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="tlPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="tlPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="tlPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="tlRowContextMenu" class="ctx-menu d-none"></div>

<script>
    function tlPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function tlPagerChangePageSize(size) {
        var f = document.createElement('input');
        f.type = 'hidden';
        f.name = 'pagesize';
        f.value = size;
        var form = document.getElementById('Update');
        form.appendChild(f);
        document.getElementById('pageField').value = 1;
        form.submit();
    }

    // Контекстное меню по правой кнопке мыши на строке таблицы - приём как в /sprsrv/datacentersitelist.
    (function () {
        var menu = document.getElementById('tlRowContextMenu');
        if (!menu) return;
        function hideMenu() { menu.classList.add('d-none'); menu.innerHTML = ''; }
        document.querySelectorAll('.row-context-target').forEach(function (row) {
            row.addEventListener('contextmenu', function (evt) {
                var source = document.getElementById(row.getAttribute('data-menu-id'));
                if (!source) return;
                evt.preventDefault();
                menu.innerHTML = source.innerHTML;
                menu.classList.remove('d-none');
                var x = evt.clientX, y = evt.clientY;
                var maxX = window.innerWidth - menu.offsetWidth - 8, maxY = window.innerHeight - menu.offsetHeight - 8;
                if (x > maxX) x = Math.max(8, maxX);
                if (y > maxY) y = Math.max(8, maxY);
                menu.style.left = x + 'px';
                menu.style.top = y + 'px';
            });
        });
        document.addEventListener('click', function (evt) { if (!menu.contains(evt.target)) hideMenu(); });
        document.addEventListener('scroll', hideMenu, true);
        window.addEventListener('resize', hideMenu);
    })();

    // Каскадный фильтр "Площадка/ЦОД -> Шкаф": select ЦОД сам не отправляется, он только
    // сужает список опций "Шкаф" по data-dc (как в storagesystemslist.tpl).
    (function () {
        var dcSelect = document.getElementById('tlFilterDatacenterSiteId');
        var rackSelect = document.getElementById('RackCabinetId');
        if (!dcSelect || !rackSelect) return;
        var allOptions = Array.prototype.slice.call(rackSelect.options);

        function applyFilter() {
            var dc = dcSelect.value;
            var current = rackSelect.value;
            rackSelect.innerHTML = '';
            allOptions.forEach(function (opt) {
                var optDc = opt.getAttribute('data-dc') || '0';
                if (opt.value === '0' || dc === '0' || optDc === dc) rackSelect.appendChild(opt);
            });
            var present = Array.prototype.some.call(rackSelect.options, function (o) { return o.value === current; });
            rackSelect.value = present ? current : '0';
        }

        var sel = allOptions.find(function (opt) { return opt.selected; });
        if (sel && sel.value !== '0') dcSelect.value = sel.getAttribute('data-dc') || '0';
        applyFilter();
        dcSelect.addEventListener('change', applyFilter);
    })();
</script>
