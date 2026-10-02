<!-- Контроллер: controllers/admindeactivatedservers.go (AdminDeactivatedServersList).
     Стиль фильтра/пагинации/закрепления - по образцу /sprsrv/hwviewlist
     (закрепляемая панель фильтра с чекбоксом отключения, закреплённый
     заголовок таблицы через общую переменную --spr-thead-top, закреплённая
     внизу экрана пагинация, контекстное меню строки по правой кнопке мыши
     класса .ctx-menu - см. server_web_new/views/sprsrv/hwviewlist.tpl и
     комментарии в конце этого файла). Кнопка "Восстановить выделенные" - по
     образцу кнопки "Редактировать выделенные" на /sprsrv/view2 (тот же
     класс btn-outline-warning и то же место - в тулбаре над таблицей, а не
     под ней, см. controllers/sprsrvview2.go, server_web_new/views/sprsrv/view2.tpl).

     Без обёртки .table-responsive у таблицы - как и на /admin/viewexport/:name
     (см. комментарий там же) и на /sprsrv/hwviewlist/vmlist.tpl: она создаёт
     собственный горизонтальный скролл-контекст, из-за которого
     position:sticky у thead прилипал бы не к верху окна, а к верху этого
     внутреннего контейнера. -->

<div class="hero-text mb-3">
    <h2>Серверы, выведенные из эксплуатации</h2>
    <p class="text-muted">
        Записи справочника серверов с Active=2 (выведен из эксплуатации). Автоматически
        так помечается сервер, который не найден ни в одном источнике (VMware, HPE OneView,
        Zabbix, SCOM) при условии, что все источники отработали без ошибок; ручные записи и
        данные из Итиллиум автоматически не выводятся. Здесь записи можно вернуть в работу,
        по одной или группой: выберите "Активен" - вместе с записью обновятся связанные
        записи по ВМ/аппаратному серверу. Если сервер так и не появится в источниках,
        автоматика снова выведет его из эксплуатации.
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="DeactivatedServersForm" method="POST">
    <div class="deactivated-filter-bar mb-3 p-3 rounded border">
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
        {{if .Rows}}
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap justify-content-between align-items-center gap-2">
                <button type="submit" name="RestoreSelectedBTN" value="1" class="btn btn-outline-warning rounded-pill px-3"
                    title="Применить возврат в учёт ко всем строкам с отмеченным чекбоксом">
                    Восстановить выделенные
                </button>
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="deactFilterStickyToggle" onchange="deactToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="deactFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
        {{end}}
    </div>
    <input type="hidden" name="page" id="pageFieldHidden" value="{{.Page}}">

    {{if .Rows}}
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
                <th>Выведен из эксплуатации</th>
            </tr>
        </thead>
        <tbody>
            {{range $ind, $r := .Rows}}
            <tr class="row-context-target" data-menu-id="deactRowMenu{{$ind}}" title="Правая кнопка мыши - меню действий">
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
                    {{if eq $r.SourceUpdateType 1}}Вручную{{else if eq $r.SourceUpdateType 2}}VMware{{else if eq $r.SourceUpdateType 3}}HPE OneView{{else if eq $r.SourceUpdateType 4}}Zabbix{{else if eq $r.SourceUpdateType 5}}SCOM{{else}}не указано{{end}}
                </td>
                <td class="small">{{if not $r.TimeDeActiveState.IsZero}}{{$r.TimeDeActiveState.Format "2006-01-02 15:04"}}{{else}}—{{end}}</td>
                <td id="deactRowMenu{{$ind}}" class="d-none">
                    <a class="ctx-menu-item" href="/admin/deactivatedservers/restore/{{$r.Id}}">Восстановить</a>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <div class="pager-bar pager-bar-sticky d-flex justify-content-between align-items-center mt-2">
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
    <p class="text-muted mt-3">Нет серверов, выведенных из эксплуатации, по текущему фильтру.</p>
    {{end}}
</form>

<div id="deactRowContextMenu" class="ctx-menu d-none"></div>

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

    // Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
    // приём идентичен /sprsrv/hwviewlist, выбор запоминается отдельным
    // ключом в localStorage. Заголовок таблицы закреплён всегда, но его
    // отступ сверху (--spr-thead-top) подстраивается под текущее состояние
    // панели фильтра - см. подробный комментарий в hwviewlist.tpl.
    (function () {
        var STORAGE_KEY = 'deactivatedserverslist.filterSticky';
        var bar = document.querySelector('.deactivated-filter-bar');
        var toggle = document.getElementById('deactFilterStickyToggle');
        if (!bar || !toggle) return;

        function updateTheadOffset() {
            var offset = 0;
            if (!bar.classList.contains('deactivated-filter-bar-unpinned')) {
                offset = bar.offsetHeight;
            }
            document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
        }

        function applySticky(sticky) {
            bar.classList.toggle('deactivated-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
            updateTheadOffset();
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.deactToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Контекстное меню по правой кнопке мыши на строке таблицы - приём
    // идентичен /sprsrv/hwviewlist: содержимое скрытого
    // <td id="deactRowMenu..."> строки копируется в общий плавающий блок
    // #deactRowContextMenu и позиционируется под курсором.
    (function () {
        var menu = document.getElementById('deactRowContextMenu');
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
