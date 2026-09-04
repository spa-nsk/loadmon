<div class="hero-text mb-3">
    <h2>Справочник блейд-корзин</h2>
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
                <label for="Vendor" class="form-label">Производитель</label>
                <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.BladeSystemsFilter.Vendor}}" placeholder="например, HPE">
            </div>
            <div class="col-6 col-md-3">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.BladeSystemsFilter.Model}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Type" class="form-label">Тип</label>
                <input type="text" id="Type" name="Type" class="form-control" value="{{.BladeSystemsFilter.Type}}" placeholder="например, c7000">
            </div>
            <div class="col-6 col-md-3">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.BladeSystemsFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="SerialNum" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.BladeSystemsFilter.SerialNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.BladeSystemsFilter.InventoryNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="DatacenterRoom" class="form-label">ЦОД / серверная</label>
                <input type="text" id="DatacenterRoom" name="DatacenterRoom" class="form-control" value="{{.BladeSystemsFilter.DatacenterRoom}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="RackNum" class="form-label">Номер стойки</label>
                <input type="text" id="RackNum" name="RackNum" class="form-control" value="{{.BladeSystemsFilter.RackNum}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" name="ViewBTN" value="Просмотр" title="Просмотреть данные с учётом фильтров" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <a href="/sprsrv/bladesystemsedit/0" title="Добавить новую запись блейд-корзины" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.BladeSystemsCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="bsFilterStickyToggle" onchange="bsToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="bsFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Наименование</th>
                <th>Производитель / модель</th>
                <th>Тип</th>
                <th>Серийный №</th>
                <th class="text-end">Слотов</th>
                <th class="text-end">Занято</th>
                <th>Размещение</th>
            </tr>
        </thead>
        <tbody>
            {{range $bs := .BladeSystemsList}}
            <tr class="row-context-target" data-menu-id="bsRowMenu{{$bs.Id}}">
                <td class="text-end">{{$bs.Id}}</td>
                <td><a href="/sprsrv/bladesystemsedit/{{$bs.Id}}" target="_blank">{{$bs.Name}}</a></td>
                <td>
                    <div style="max-width:260px; word-wrap:break-word;">
                        {{$bs.Vendor}} {{$bs.Model}}
                        <div class="small text-muted">{{$bs.PartNum}}</div>
                    </div>
                </td>
                <td>{{$bs.Type}}</td>
                <td>{{$bs.SerialNum}}</td>
                <td class="text-end">{{$bs.Size}}</td>
                <td class="text-end">{{$bs.Used}}</td>
                <td>{{$bs.DatacenterRoom}} {{if ne $bs.RackNum ""}}/ {{$bs.RackNum}}{{end}}</td>
                <td id="bsRowMenu{{$bs.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/bladesystemsedit/{{$bs.Id}}" target="_blank">Редактировать</a>
                    <a class="ctx-menu-item" href="/sprsrv/bladesystemserverslist/{{$bs.Id}}" target="_blank">Серверы в корзине</a>
                </td>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.BladeSystemsCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="bsPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="bsPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="bsPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="bsPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="bsPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="bsRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, панели
     пагинации и контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) -
     страница полностью повторяет стиль /sprsrv/storagesystemslist и
     /sprsrv/hwconflist. -->

<script>
    // Закрепление панели фильтра и sticky-заголовок таблицы - приём идентичен
    // /sprsrv/storagesystemslist (см. подробный комментарий там).
    (function () {
        var STORAGE_KEY = 'bladesystemslist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('bsFilterStickyToggle');
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

        window.bsToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Переход по странице без потери значений фильтра - приём идентичен
    // /sprsrv/storagesystemslist.
    function bsPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function bsPagerChangePageSize(size) {
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
    // идентичен /sprsrv/storagesystemslist.
    (function () {
        var menu = document.getElementById('bsRowContextMenu');
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
