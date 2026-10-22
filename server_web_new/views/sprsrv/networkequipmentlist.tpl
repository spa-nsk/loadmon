<div class="hero-text mb-3">
    <h2>Справочник сетевого оборудования</h2>
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
                <label for="TypeId" class="form-label">Тип оборудования</label>
                <select id="TypeId" name="TypeId" class="form-select">
                    <option value="all" {{if eq .NetworkEquipmentFilter.TypeId "all"}}selected{{end}}>Все</option>
                    <option value="0" {{if eq .NetworkEquipmentFilter.TypeId "0"}}selected{{end}}>Не указан</option>
                    {{range $t := .EquipmentTypes}}
                    <option value="{{$t.Id}}" {{if eq $.NetworkEquipmentFilter.TypeId (printf "%d" $t.Id)}}selected{{end}}>{{$t.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="ItiliumStatus" class="form-label">Статус Итилиум</label>
                <select id="ItiliumStatus" name="ItiliumStatus" class="form-select">
                    <option value="all" {{if eq .NetworkEquipmentFilter.ItiliumStatus "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .NetworkEquipmentFilter.ItiliumStatus "1"}}selected{{end}}>В работе</option>
                    <option value="2" {{if eq .NetworkEquipmentFilter.ItiliumStatus "2"}}selected{{end}}>Выведен</option>
                    <option value="3" {{if eq .NetworkEquipmentFilter.ItiliumStatus "3"}}selected{{end}}>Списан</option>
                    <option value="4" {{if eq .NetworkEquipmentFilter.ItiliumStatus "4"}}selected{{end}}>Свободно</option>
                    <option value="5" {{if eq .NetworkEquipmentFilter.ItiliumStatus "5"}}selected{{end}}>На складе</option>
                    <option value="6" {{if eq .NetworkEquipmentFilter.ItiliumStatus "6"}}selected{{end}}>Зарезервировано</option>
                    <option value="7" {{if eq .NetworkEquipmentFilter.ItiliumStatus "7"}}selected{{end}}>В ремонте</option>
                    <option value="8" {{if eq .NetworkEquipmentFilter.ItiliumStatus "8"}}selected{{end}}>В пути</option>
                    <option value="9" {{if eq .NetworkEquipmentFilter.ItiliumStatus "9"}}selected{{end}}>Черновик</option>
                    <option value="0" {{if eq .NetworkEquipmentFilter.ItiliumStatus "0"}}selected{{end}}>Не указано</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.NetworkEquipmentFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Vendor" class="form-label">Производитель</label>
                <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.NetworkEquipmentFilter.Vendor}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="SerialNum" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.NetworkEquipmentFilter.SerialNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="DatacenterSiteId" class="form-label">ЦОД / площадка</label>
                <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                    <option value="0">Все</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" {{if eq $.NetworkEquipmentFilter.DatacenterSiteId (printf "%d" $dc.Id)}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <a href="/sprsrv/networkequipmenttypelist/0" class="btn btn-outline-secondary rounded-pill px-3">Типы оборудования</a>
                <a href="/sprsrv/networkequipmentedit/0" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.NetworkEquipmentCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="swFilterStickyToggle" onchange="swToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="swFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Тип</th>
                <th>Статус</th>
                <th>Наименование</th>
                <th>Производитель / модель</th>
                <th>Серийный №</th>
                <th>IP управления</th>
                <th>ЦОД / шкаф</th>
                <th class="text-end">Юнит</th>
                <th>Сети</th>
            </tr>
        </thead>
        <tbody>
            {{range $sw := .NetworkEquipmentList}}
            <tr class="row-context-target" data-menu-id="swRowMenu{{$sw.Id}}">
                <td class="text-end">{{$sw.Id}}</td>
                <td>{{$sw.TypeName}}</td>
                <td>
                    <span class="badge {{if eq $sw.ItiliumStatus 1}}text-bg-success{{else if eq $sw.ItiliumStatus 3}}text-bg-dark{{else if or (eq $sw.ItiliumStatus 2) (eq $sw.ItiliumStatus 5)}}text-bg-secondary{{else}}text-bg-light{{end}}">
                        {{if eq $sw.ItiliumStatus 1}}В работе{{else if eq $sw.ItiliumStatus 2}}Выведен{{else if eq $sw.ItiliumStatus 3}}Списан{{else if eq $sw.ItiliumStatus 4}}Свободно{{else if eq $sw.ItiliumStatus 5}}На складе{{else if eq $sw.ItiliumStatus 6}}Зарезервировано{{else if eq $sw.ItiliumStatus 7}}В ремонте{{else if eq $sw.ItiliumStatus 8}}В пути{{else if eq $sw.ItiliumStatus 9}}Черновик{{else}}Не указано{{end}}
                    </span>
                </td>
                <td><a href="/sprsrv/networkequipmentedit/{{$sw.Id}}" target="_blank">{{$sw.Name}}</a></td>
                <td>{{$sw.Vendor}} {{$sw.Model}}</td>
                <td>{{$sw.SerialNum}}</td>
                <td>{{$sw.ManagementIp}}</td>
                <td>{{$sw.DatacenterSiteName}} {{if ne $sw.RackCabinetName ""}}/ {{$sw.RackCabinetName}}{{end}}</td>
                <td class="text-end">{{$sw.UnitText}}</td>
                <td>{{$sw.Networks}}</td>
                <td id="swRowMenu{{$sw.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/networkequipmentedit/{{$sw.Id}}" target="_blank">Редактировать</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.NetworkEquipmentCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="swPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="swPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="swPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="swPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="swPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="swRowContextMenu" class="ctx-menu d-none"></div>

<script>
    (function () {
        var STORAGE_KEY = 'networkequipmentlist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('swFilterStickyToggle');
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

        window.swToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function swPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function swPagerChangePageSize(size) {
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

    (function () {
        var menu = document.getElementById('swRowContextMenu');
        if (!menu) return;

        function hideMenu() {
            menu.classList.add('d-none');
            menu.innerHTML = '';
        }

        document.querySelectorAll('.row-context-target').forEach(function (row) {
            row.addEventListener('contextmenu', function (evt) {
                var menuId = row.getAttribute('data-menu-id');
                var source = menuId ? document.getElementById(menuId) : null;
                if (!source) return;
                evt.preventDefault();
                menu.innerHTML = source.innerHTML;
                menu.classList.remove('d-none');

                var x = evt.clientX;
                var y = evt.clientY;
                var maxX = window.innerWidth - menu.offsetWidth - 8;
                var maxY = window.innerHeight - menu.offsetHeight - 8;
                if (x > maxX) x = Math.max(8, maxX);
                if (y > maxY) y = Math.max(8, maxY);
                menu.style.left = x + 'px';
                menu.style.top = y + 'px';
            });
        });

        document.addEventListener('click', function (evt) {
            if (!menu.contains(evt.target)) hideMenu();
        });
        document.addEventListener('scroll', hideMenu, true);
        window.addEventListener('resize', hideMenu);
    })();
</script>
