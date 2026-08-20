<div class="hero-text mb-3">
    <h2>Справочник кластеров</h2>
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
                <input type="text" id="Name" name="Name" class="form-control" value="{{.ClusterSprFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Vcenter" class="form-label">vCenter</label>
                <input type="text" id="Vcenter" name="Vcenter" class="form-control" value="{{.ClusterSprFilter.Vcenter}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Datacenter" class="form-label">Датацентр</label>
                <input type="text" id="Datacenter" name="Datacenter" class="form-control" value="{{.ClusterSprFilter.Datacenter}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="ClusterId" class="form-label">ClusterId</label>
                <input type="text" id="ClusterId" name="ClusterId" class="form-control" value="{{.ClusterSprFilter.ClusterId}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="TypeWork" class="form-label">Среда</label>
                <select id="TypeWork" name="TypeWork" class="form-select">
                    <option value="all" {{if eq .ClusterSprFilter.TypeWork "all"}}selected{{end}}>Все</option>
                    <option value="0" {{if eq .ClusterSprFilter.TypeWork "0"}}selected{{end}}>Не указано</option>
                    <option value="1" {{if eq .ClusterSprFilter.TypeWork "1"}}selected{{end}}>Продуктовая среда</option>
                    <option value="2" {{if eq .ClusterSprFilter.TypeWork "2"}}selected{{end}}>Внутренний тест</option>
                    <option value="3" {{if eq .ClusterSprFilter.TypeWork "3"}}selected{{end}}>Внешний тест</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Stat" class="form-label">Статус</label>
                <select id="Stat" name="Stat" class="form-select">
                    <option value="all" {{if eq .ClusterSprFilter.Stat "all"}}selected{{end}}>Все</option>
                    <option value="0" {{if eq .ClusterSprFilter.Stat "0"}}selected{{end}}>Не указано</option>
                    <option value="1" {{if eq .ClusterSprFilter.Stat "1"}}selected{{end}}>Активен</option>
                    <option value="2" {{if eq .ClusterSprFilter.Stat "2"}}selected{{end}}>Выведен из работы</option>
                </select>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" name="ViewBTN" value="Просмотр" title="Просмотреть данные с учётом фильтров" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.ClusterSprCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="clFilterStickyToggle" onchange="clToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="clFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Наименование</th>
                <th>vCenter</th>
                <th>Датацентр</th>
                <th>ClusterId</th>
                <th>Среда</th>
                <th>Статус</th>
                <th>Примечание</th>
            </tr>
        </thead>
        <tbody>
            {{range $cl := .ClusterSprList}}
            <tr class="row-context-target" data-menu-id="clRowMenu{{$cl.Id}}">
                <td class="text-end">{{$cl.Id}}</td>
                <td><a href="/sprsrv/clusterspredit/{{$cl.Id}}" target="_blank">{{$cl.Name}}</a></td>
                <td>{{$cl.Vcenter}}</td>
                <td>{{$cl.Datacenter}}</td>
                <td><div style="max-width:220px; word-wrap:break-word;">{{$cl.ClusterId}}</div></td>
                <td>
                    {{if eq $cl.TypeWork 1}}Продуктовая среда
                    {{else if eq $cl.TypeWork 2}}Внутренний тест
                    {{else if eq $cl.TypeWork 3}}Внешний тест
                    {{else}}Не указано{{end}}
                </td>
                <td>
                    {{if eq $cl.Stat 1}}Активен
                    {{else if eq $cl.Stat 2}}Выведен из работы
                    {{else}}Не указано{{end}}
                </td>
                <td><div style="max-width:260px; word-wrap:break-word;">{{$cl.Annotation}}</div></td>
                <td id="clRowMenu{{$cl.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/clusterspredit/{{$cl.Id}}" target="_blank">Редактировать</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.ClusterSprCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="clPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="clPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="clRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, панели
     пагинации и контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) -
     страница полностью повторяет стиль /sprsrv/storagesystemslist и
     /sprsrv/bladesystemslist. Кнопки "Добавить" здесь нет намеренно -
     справочник кластеров наполняется автоматически, доступна только правка
     существующих записей. -->

<script>
    // Закрепление панели фильтра и sticky-заголовок таблицы - приём идентичен
    // /sprsrv/storagesystemslist (см. подробный комментарий там).
    (function () {
        var STORAGE_KEY = 'clustersprlist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('clFilterStickyToggle');
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

        window.clToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Переход по странице без потери значений фильтра - приём идентичен
    // /sprsrv/storagesystemslist.
    function clPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function clPagerChangePageSize(size) {
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
        var menu = document.getElementById('clRowContextMenu');
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
