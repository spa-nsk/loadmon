<div class="hero-text mb-3">
    <h2>Справочник Площадок/ЦОД</h2>
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
                <input type="text" id="Name" name="Name" class="form-control" value="{{.DatacenterSiteFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="AltName" class="form-label">Альт. наименование</label>
                <input type="text" id="AltName" name="AltName" class="form-control" value="{{.DatacenterSiteFilter.AltName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="CodeJira" class="form-label">Код Jira</label>
                <input type="text" id="CodeJira" name="CodeJira" class="form-control" value="{{.DatacenterSiteFilter.CodeJira}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" title="Просмотреть данные с учётом фильтров" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                {{if .CanEdit}}
                <button type="submit" name="SetBaseBTN" value="Редактировать выделенные" title="Внести общие данные для площадок, выделенных в списке" class="btn btn-outline-warning rounded-pill px-3">Редактировать выделенные</button>
                {{end}}
                <a href="/sprsrv/datacentersiteedit/0" title="Добавить новую площадку/ЦОД" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.DatacenterSiteCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="dcFilterStickyToggle" onchange="dcToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="dcFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center"><input class="form-check-input" type="checkbox" id="dcSelectAll" title="Выделить все на странице" onchange="dcToggleSelectAll(this.checked)"></th>
                <th class="text-center">Id</th>
                <th>Наименование</th>
                <th>Альт. наименование</th>
                <th class="text-center">ЦОД</th>
                <th>Код Jira</th>
                <th>Описание</th>
                <th class="text-end" title="Сумма по шкафам площадки">Ср. 24ч, Вт</th>
                <th class="text-end">Пик 24ч, Вт</th>
                <th class="text-end">Паспорт, Вт</th>
            </tr>
        </thead>
        <tbody>
            {{range $dc := .DatacenterSiteList}}
            <tr class="row-context-target" data-menu-id="dcRowMenu{{$dc.Id}}">
                <td class="text-center"><input class="form-check-input dc-row-select" type="checkbox" name="chSelection" value="{{$dc.Id}}" title="Выделить для группового редактирования"></td>
                <td class="text-end">{{$dc.Id}}</td>
                <td><a href="/sprsrv/datacentersiteedit/{{$dc.Id}}" target="_blank">{{$dc.Name}}</a></td>
                <td>{{$dc.AltName}}</td>
                <td class="text-center">
                    {{if $dc.IsDatacenter}}
                    <span class="badge bg-primary" title="Площадка является ЦОД - размещается активное оборудование">ЦОД</span>
                    {{else}}
                    <span class="badge bg-secondary" title="Не ЦОД - площадка размещения без активного оборудования (склад, комната и т.п.)">площадка</span>
                    {{end}}
                </td>
                <td>{{$dc.CodeJira}}</td>
                <td>{{$dc.Annotation}}</td>
                <td class="text-end">{{printf "%.0f" $dc.PowerAvg24hW}}</td>
                <td class="text-end">{{printf "%.0f" $dc.PowerPeak24hW}}</td>
                <td class="text-end">{{printf "%.0f" $dc.PowerRatedMaxW}}</td>
                <td id="dcRowMenu{{$dc.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/datacentersiteview/{{$dc.Id}}" target="_blank">Просмотр оборудования ЦОД</a>
                    <a class="ctx-menu-item" href="/sprsrv/datacentersiteedit/{{$dc.Id}}" target="_blank">Редактировать</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.DatacenterSiteCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="dcPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="dcPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="dcPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="dcPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="dcPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="dcRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, панели
     пагинации и контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) - см.
     kontext/06-list-page-pattern.md. Страница полностью повторяет стиль
     /sprsrv/bladesystemslist. -->

<script>
    function dcToggleSelectAll(checked) {
        document.querySelectorAll('.dc-row-select').forEach(function (el) { el.checked = checked; });
    }

    // Закрепление панели фильтра и sticky-заголовок таблицы - приём идентичен
    // /sprsrv/bladesystemslist (см. подробный комментарий там).
    (function () {
        var STORAGE_KEY = 'datacentersitelist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('dcFilterStickyToggle');
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

        window.dcToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function dcPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function dcPagerChangePageSize(size) {
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

    // Контекстное меню по правой кнопке мыши на строке таблицы - приём
    // идентичен /sprsrv/bladesystemslist.
    (function () {
        var menu = document.getElementById('dcRowContextMenu');
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
