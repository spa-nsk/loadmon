<div class="hero-text mb-3">
    <h2>Справочник томов</h2>
    <p class="text-muted small mb-0">Список записей StorageVolumes с расшифровкой СХД. Заполняется автосинхронизацией (vSphere/API СХД) - без редактирования, детали тома доступны по правому клику на строке.</p>
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
                <label for="Status" class="form-label">Статус</label>
                <select id="Status" name="Status" class="form-select">
                    <option value="all" {{if eq .StorageVolumesFilter.Status "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .StorageVolumesFilter.Status "1"}}selected{{end}}>В виртуализации</option>
                    <option value="2" {{if eq .StorageVolumesFilter.Status "2"}}selected{{end}}>Прочее (активен)</option>
                    <option value="0" {{if eq .StorageVolumesFilter.Status "0"}}selected{{end}}>Выведен/не найден</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="ShortName" class="form-label">СХД (мнемоника)</label>
                <input type="text" id="ShortName" name="ShortName" class="form-control" value="{{.StorageVolumesFilter.ShortName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="StorageName" class="form-label">Имя тома на СХД</label>
                <input type="text" id="StorageName" name="StorageName" class="form-control" value="{{.StorageVolumesFilter.StorageName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="VirtualName" class="form-label">Имя хранилища в vSphere</label>
                <input type="text" id="VirtualName" name="VirtualName" class="form-control" value="{{.StorageVolumesFilter.VirtualName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Vcenter" class="form-label">vCenter</label>
                <input type="text" id="Vcenter" name="Vcenter" class="form-control" value="{{.StorageVolumesFilter.Vcenter}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Cluster" class="form-label">Кластер</label>
                <input type="text" id="Cluster" name="Cluster" class="form-control" value="{{.StorageVolumesFilter.Cluster}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="DiskType" class="form-label">Тип дисков</label>
                <input type="text" id="DiskType" name="DiskType" class="form-control" value="{{.StorageVolumesFilter.DiskType}}">
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
                <span class="text-muted small">Найдено записей: {{.StorageVolumesCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="svFilterStickyToggle" onchange="svToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="svFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Статус</th>
                <th>СХД</th>
                <th>Имя тома</th>
                <th>Кластер</th>
                <th>Тип дисков</th>
                <th class="text-end">Ёмкость, Gb</th>
                <th class="text-end">Свободно, Gb</th>
                <th class="text-end">Занято, %</th>
                <th>Сервер(-а)</th>
            </tr>
        </thead>
        <tbody>
            {{range $sv := .StorageVolumesList}}
            <tr class="row-context-target" data-menu-id="svRowMenu{{$sv.Id}}">
                <td class="text-end">{{$sv.Id}}</td>
                <td>
                    <span class="badge {{if eq $sv.Status 1}}text-bg-success{{else if eq $sv.Status 2}}text-bg-info{{else}}text-bg-secondary{{end}}">
                        {{if eq $sv.Status 1}}В виртуализации{{else if eq $sv.Status 2}}Активен{{else}}Выведен/не найден{{end}}
                    </span>
                </td>
                <td>{{if $sv.StorageSystemLabel}}{{$sv.StorageSystemLabel}}{{else}}<span class="text-muted">&mdash;</span>{{end}}</td>
                <td>
                    <div style="max-width:260px; word-wrap:break-word;">
                        {{if $sv.StorageName}}{{$sv.StorageName}}{{else}}{{$sv.VirtualName}}{{end}}
                    </div>
                </td>
                <td>{{$sv.Vcenter}} {{if ne $sv.Cluster ""}}/ {{$sv.Cluster}}{{end}}</td>
                <td>{{$sv.DiskType}}</td>
                <td class="text-end">{{printf "%.1f" $sv.CapacityGb}}</td>
                <td class="text-end">{{printf "%.1f" $sv.FreeSpaceGb}}</td>
                <td class="text-end">{{printf "%.1f" $sv.PercentUsed}}</td>
                <td style="max-width:220px; word-wrap:break-word;">{{$sv.Server}}</td>
                <td id="svRowMenu{{$sv.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/storagevolumeview/{{$sv.Id}}" target="_blank">Просмотр</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.StorageVolumesCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="svPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="svPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="svPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="svPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="svPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="svRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, панели
     пагинации и контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) -
     страница полностью повторяет стиль /sprsrv/storagesystemslist. -->

<script>
    // Закрепление панели фильтра и sticky-заголовок таблицы - приём идентичен
    // /sprsrv/storagesystemslist (см. подробный комментарий там / в
    // /sprsrv/hwconflist).
    (function () {
        var STORAGE_KEY = 'storagevolumeslist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('svFilterStickyToggle');
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

        window.svToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function svPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function svPagerChangePageSize(size) {
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
    // идентичен /sprsrv/storagesystemslist. Единственный пункт - переход на
    // read-only карточку /sprsrv/storagevolumeview/:Id (редактирования нет).
    (function () {
        var menu = document.getElementById('svRowContextMenu');
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
