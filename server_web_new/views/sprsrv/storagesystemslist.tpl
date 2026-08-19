<div class="hero-text mb-3">
    <h2>Справочник СХД</h2>
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
                <label for="TypeWork" class="form-label">Среда</label>
                <select id="TypeWork" name="TypeWork" class="form-select">
                    <option value="all" {{if eq .StorageSystemsFilter.TypeWork "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .StorageSystemsFilter.TypeWork "1"}}selected{{end}}>Продуктовая</option>
                    <option value="2" {{if eq .StorageSystemsFilter.TypeWork "2"}}selected{{end}}>Внутренний тест</option>
                    <option value="3" {{if eq .StorageSystemsFilter.TypeWork "3"}}selected{{end}}>Внешний тест / Бэкап</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Status" class="form-label">Статус</label>
                <select id="Status" name="Status" class="form-select">
                    <option value="all" {{if eq .StorageSystemsFilter.Status "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .StorageSystemsFilter.Status "1"}}selected{{end}}>Используется</option>
                    <option value="2" {{if eq .StorageSystemsFilter.Status "2"}}selected{{end}}>Выключен</option>
                    <option value="3" {{if eq .StorageSystemsFilter.Status "3"}}selected{{end}}>Не используется</option>
                    <option value="4" {{if eq .StorageSystemsFilter.Status "4"}}selected{{end}}>Тестирование</option>
                    <option value="5" {{if eq .StorageSystemsFilter.Status "5"}}selected{{end}}>Приёмка</option>
                    <option value="6" {{if eq .StorageSystemsFilter.Status "6"}}selected{{end}}>Демонтирован</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Vendor" class="form-label">Производитель</label>
                <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.StorageSystemsFilter.Vendor}}" placeholder="например, Huawei">
            </div>
            <div class="col-6 col-md-3">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.StorageSystemsFilter.Model}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="ShortName" class="form-label">Мнемоническое имя</label>
                <input type="text" id="ShortName" name="ShortName" class="form-control" value="{{.StorageSystemsFilter.ShortName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="SerialNum" class="form-label">Серийный номер</label>
                <input type="text" id="SerialNum" name="SerialNum" class="form-control" value="{{.StorageSystemsFilter.SerialNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="InventoryNum" class="form-label">Инвентарный номер</label>
                <input type="text" id="InventoryNum" name="InventoryNum" class="form-control" value="{{.StorageSystemsFilter.InventoryNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="DatacenterRoom" class="form-label">ЦОД / серверная</label>
                <input type="text" id="DatacenterRoom" name="DatacenterRoom" class="form-control" value="{{.StorageSystemsFilter.DatacenterRoom}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="RackNum" class="form-label">Номер стойки</label>
                <input type="text" id="RackNum" name="RackNum" class="form-control" value="{{.StorageSystemsFilter.RackNum}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" name="ViewBTN" value="Просмотр" title="Просмотреть данные с учётом фильтров" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <a href="/sprsrv/storagesystemsedit/0" title="Добавить новую запись СХД" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.StorageSystemsCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="stFilterStickyToggle" onchange="stToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="stFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Питание</th>
                <th>Статус</th>
                <th>Среда</th>
                <th>Производитель / модель</th>
                <th>Мнемоническое имя</th>
                <th>Серийный №</th>
                <th class="text-end">Ёмкость (сырая)</th>
                <th class="text-end">Занято, %</th>
                <th>Размещение</th>
            </tr>
        </thead>
        <tbody>
            {{range $st := .StorageSystemsList}}
            <tr class="row-context-target" data-menu-id="stRowMenu{{$st.Id}}">
                <td class="text-end">{{$st.Id}}</td>
                <td>
                    <svg class="bi power-led {{if eq $st.Power 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $st.Power 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg>
                </td>
                <td>
                    <span class="badge {{if eq $st.Status 1}}text-bg-success{{else if eq $st.Status 3}}text-bg-secondary{{else if eq $st.Status 6}}text-bg-dark{{else}}text-bg-light{{end}}">
                        {{if eq $st.Status 1}}Используется{{else if eq $st.Status 2}}Выключен{{else if eq $st.Status 3}}Не используется{{else if eq $st.Status 4}}Тестирование{{else if eq $st.Status 5}}Приёмка{{else if eq $st.Status 6}}Демонтирован{{else}}Не указано{{end}}
                    </span>
                </td>
                <td>{{if eq $st.TypeWork 1}}Продуктовая{{else if eq $st.TypeWork 2}}Внутр. тест{{else if eq $st.TypeWork 3}}Внеш. тест/бэкап{{else}}—{{end}}</td>
                <td>
                    <div style="max-width:260px; word-wrap:break-word;">
                        {{$st.Vendor}} {{$st.Model}}
                        <div class="small text-muted">{{$st.PartNum}}</div>
                    </div>
                </td>
                <td><a href="/sprsrv/storagesystemsedit/{{$st.Id}}" target="_blank">{{$st.ShortName}}</a></td>
                <td>{{$st.SerialNum}}</td>
                <td class="text-end">{{printf "%.1f" $st.TotalSizeRAW}}</td>
                <td class="text-end">{{printf "%.1f" $st.UsedPercent}}</td>
                <td>{{$st.DatacenterRoom}} {{if ne $st.RackNum ""}}/ {{$st.RackNum}}{{end}}</td>
                <td id="stRowMenu{{$st.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/storagesystemsedit/{{$st.Id}}" target="_blank">Редактировать</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.StorageSystemsCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="stPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="stPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="stPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="stPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="stPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="stRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, индикатора
     питания, панели пагинации и контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) -
     страница полностью повторяет стиль /sprsrv/hwconflist. -->

<script>
    // Закрепление панели фильтра и sticky-заголовок таблицы - приём идентичен
    // /sprsrv/hwconflist (см. подробный комментарий там).
    (function () {
        var STORAGE_KEY = 'storagesystemslist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('stFilterStickyToggle');
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

        window.stToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Переход по странице без потери значений фильтра - номер страницы
    // пишется в скрытое поле формы, после чего форма отправляется тем же
    // POST-ом, что и обычный "Просмотр" (приём идентичен /sprsrv/hwconflist).
    function stPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function stPagerChangePageSize(size) {
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
    // идентичен /sprsrv/hwconflist.
    (function () {
        var menu = document.getElementById('stRowContextMenu');
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
