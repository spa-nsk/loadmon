<div class="hero-text mb-3">
    <h2>Конфигурация серверов (справочник)</h2>
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
                <label for="PowerState" class="form-label">Состояние питания</label>
                <select id="PowerState" name="PowerState" class="form-select">
                    <option value="all" {{if eq .HWConfigFilter.PowerState "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .HWConfigFilter.PowerState "1"}}selected{{end}}>Включен</option>
                    <option value="2" {{if eq .HWConfigFilter.PowerState "2"}}selected{{end}}>Выключен</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Active" class="form-label">Статус</label>
                <select id="Active" name="Active" class="form-select">
                    <option value="all" {{if eq .HWConfigFilter.Active "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .HWConfigFilter.Active "1"}}selected{{end}}>Активен</option>
                    <option value="2" {{if eq .HWConfigFilter.Active "2"}}selected{{end}}>Выведен из эксплуатации</option>
                    <option value="3" {{if eq .HWConfigFilter.Active "3"}}selected{{end}}>Списан</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="FormFactor" class="form-label">Тип сервера</label>
                <select id="FormFactor" name="FormFactor" class="form-select">
                    <option value="all" {{if eq .HWConfigFilter.FormFactor "all"}}selected{{end}}>Все</option>
                    <option value="1" {{if eq .HWConfigFilter.FormFactor "1"}}selected{{end}}>Rack</option>
                    <option value="2" {{if eq .HWConfigFilter.FormFactor "2"}}selected{{end}}>Blade</option>
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Vendor" class="form-label">Производитель</label>
                <input type="text" id="Vendor" name="Vendor" class="form-control" value="{{.HWConfigFilter.Vendor}}" placeholder="например, HPE">
            </div>
            <div class="col-6 col-md-3">
                <label for="Model" class="form-label">Модель</label>
                <input type="text" id="Model" name="Model" class="form-control" value="{{.HWConfigFilter.Model}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="PartNum" class="form-label">Номер продукта</label>
                <input type="text" id="PartNum" name="PartNum" class="form-control" value="{{.HWConfigFilter.PartNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="CPUModel" class="form-label">Модель процессора</label>
                <input type="text" id="CPUModel" name="CPUModel" class="form-control" value="{{.HWConfigFilter.CPUModel}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="DatacenterRoom" class="form-label">ЦОД</label>
                <input type="text" id="DatacenterRoom" name="DatacenterRoom" class="form-control" value="{{.HWConfigFilter.DatacenterRoom}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="RackNum" class="form-label">Номер стойки</label>
                <input type="text" id="RackNum" name="RackNum" class="form-control" value="{{.HWConfigFilter.RackNum}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="ServerName" class="form-label">Имя сервера</label>
                <input type="text" id="ServerName" name="ServerName" class="form-control" value="{{.HWConfigFilter.ServerName}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" name="ViewBTN" value="Просмотр" title="Просмотреть данные с учётом фильтров" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель ;)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <button type="button" id="hwBulkEditBtn" title="Редактировать статус и размещение у выделенных в списке записей" class="btn btn-outline-warning rounded-pill px-3" disabled onclick="hwBulkEditGo()">Редактировать выделенные</button>
                <a href="/sprsrv/hwconfedit/0" title="Добавить новую запись конфигурации сервера" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.HWConfigCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="hwFilterStickyToggle" onchange="hwToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="hwFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
            <thead>
                <tr>
                    <th class="text-center"><input type="checkbox" id="hwSelectAll" onchange="hwToggleSelectAll(this)"></th>
                    <th class="text-center">Id</th>
                    <th>Питание</th>
                    <th>Статус</th>
                    <th>Производитель / модель</th>
                    <th>Серийный №</th>
                    <th>Процессор</th>
                    <th class="text-end">Память, МБ</th>
                    <th>Размещение</th>
                    <th>Имя</th>
                </tr>
            </thead>
            <tbody>
                {{range $hw := .HWConfigList}}
                <tr class="row-context-target" data-menu-id="hwRowMenu{{$hw.Id}}">
                    <td class="text-center"><input type="checkbox" class="hw-select" value="{{$hw.Id}}" onchange="hwUpdateBulkBtn()"></td>
                    <td class="text-end">{{$hw.Id}}</td>
                    <td>
                        <svg class="bi power-led {{if eq $hw.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $hw.PowerState 1}}Включен{{else if eq $hw.PowerState 2}}Выключен{{else}}Не указано{{end}}"><use href="#power"></use></svg>
                    </td>
                    <td>
                        <span class="badge {{if eq $hw.Active 1}}text-bg-success{{else if eq $hw.Active 2}}text-bg-secondary{{else if eq $hw.Active 3}}text-bg-dark{{else}}text-bg-light{{end}}">
                            {{if eq $hw.Active 1}}Активен{{else if eq $hw.Active 2}}Выведен{{else if eq $hw.Active 3}}Списан{{else}}Не указано{{end}}
                        </span>
                    </td>
                    <td>
                        <div style="max-width:260px; word-wrap:break-word;">
                            {{$hw.Vendor}} {{$hw.Model}}
                            <div class="small text-muted">{{$hw.PartNum}} {{if eq $hw.FormFactor 1}}(Rack){{else if eq $hw.FormFactor 2}}(Blade){{end}}</div>
                        </div>
                    </td>
                    <td><a href="/sprsrv/hwconfedit/{{$hw.Id}}" target="_blank">{{$hw.SerialNum}}</a></td>
                    <td>
                        <div style="max-width:220px; word-wrap:break-word;">
                            {{$hw.CPUModel}}
                            <div class="small text-muted">{{$hw.CPUCount}} x {{$hw.CorePerCPU}} ядер, {{$hw.CoreMHz}} МГц</div>
                        </div>
                    </td>
                    <td class="text-end">{{$hw.MemoryMb}}</td>
                    <td>{{$hw.DatacenterRoom}} {{if ne $hw.RackNum ""}}/ {{$hw.RackNum}}{{end}}</td>
                    <td>
                        {{if ne $hw.ServerName ""}}{{$hw.ServerName}}{{end}}
                        {{if ne $hw.ServerNameFQDN ""}}<div class="small text-muted">{{$hw.ServerNameFQDN}}</div>{{end}}
                    </td>
                    <td id="hwRowMenu{{$hw.Id}}" class="d-none">
                        <a class="ctx-menu-item" href="/sprsrv/hwconfedit/{{$hw.Id}}" target="_blank">Редактировать</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.HWConfigCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="hwPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="hwPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="hwRowContextMenu" class="ctx-menu d-none"></div>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы (включая
     sticky-заголовок через общую переменную --spr-thead-top), индикатора
     питания, панели пагинации и простого контекстного меню строки (.ctx-menu)
     вынесены в общий server_web_new/static/css/loadmon.css (подключён в
     header.tpl). -->

<script>
    // Закрепление панели фильтра вверху экрана можно отключить чекбоксом
    // "Закреплять фильтр вверху экрана" - панель с большим количеством полей
    // (10 фильтров) занимает много места на маленьких экранах при
    // прокрутке списка. Выбор пользователя запоминается в localStorage
    // (ключ hwconflist.filterSticky) и переживает перезагрузку страницы.
    //
    // Заголовок таблицы закреплён всегда ("в постоянном режиме"), но его
    // отступ сверху (top) подстраивается под текущее состояние панели
    // фильтра: если фильтр закреплён - заголовок прилипает сразу под ним
    // (на высоту панели), если фильтр не закреплён (прокручивается вместе
    // со страницей) - заголовок прилипает к самому верху экрана. Приём
    // идентичен /sprsrv/hwviewlist.
    (function () {
        var STORAGE_KEY = 'hwconflist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('hwFilterStickyToggle');
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

        window.hwToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    // Переход по странице без потери значений фильтра - номер страницы
    // пишется в скрытое поле формы, после чего форма отправляется тем же
    // POST-ом, что и обычный "Найти" (приём идентичен /sprsrv/view2).
    function hwPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function hwPagerChangePageSize(size) {
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

    // Групповой выбор строк для группового редактирования.
    function hwToggleSelectAll(master) {
        var boxes = document.querySelectorAll('.hw-select');
        boxes.forEach(function (b) { b.checked = master.checked; });
        hwUpdateBulkBtn();
    }

    function hwUpdateBulkBtn() {
        var checked = document.querySelectorAll('.hw-select:checked');
        var btn = document.getElementById('hwBulkEditBtn');
        if (btn) btn.disabled = checked.length === 0;
        var all = document.querySelectorAll('.hw-select');
        var master = document.getElementById('hwSelectAll');
        if (master) master.checked = all.length > 0 && checked.length === all.length;
    }

    function hwBulkEditGo() {
        var checked = document.querySelectorAll('.hw-select:checked');
        if (checked.length === 0) return;
        var ids = Array.prototype.map.call(checked, function (b) { return b.value; }).join(',');
        window.location.href = '/sprsrv/hwconfbulkedit?ids=' + encodeURIComponent(ids);
    }

    // Контекстное меню по правой кнопке мыши на строке таблицы - клонирует
    // содержимое скрытой ячейки-меню (hwRowMenu{{"{{"}}$hw.Id{{"}}"}}) во
    // всплывающий div и позиционирует его по координатам клика, приём
    // идентичен /sprsrv/view2 и /sprsrv/vm.
    (function () {
        var menu = document.getElementById('hwRowContextMenu');
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
