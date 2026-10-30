<div class="hero-text mb-3">
    <h2>Справочник шкафов</h2>
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
            <div class="col-12 col-md-3 position-relative">
                <label for="LocalitySearch" class="form-label">Населённый пункт</label>
                <input type="text" id="LocalitySearch" class="form-control" autocomplete="off"
                       placeholder="начните вводить название, например: Новосиб" value="{{.FilterLocalityName}}">
                <input type="hidden" id="LocalityCode" name="LocalityCode" value="{{.RackCabinetFilter.LocalityCode}}">
                <div id="LocalityList" class="list-group position-absolute w-100 shadow d-none" style="z-index: 1050; max-height: 320px; overflow-y: auto;"></div>
            </div>
            <div class="col-6 col-md-3">
                <label for="DatacenterSiteId" class="form-label">Площадка/ЦОД</label>
                <select id="DatacenterSiteId" name="DatacenterSiteId" class="form-select">
                    <option value="0">Все</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" data-locality="{{$dc.LocalityCode}}" {{if eq $.RackCabinetFilter.DatacenterSiteId (printf "%d" $dc.Id)}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-3">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.RackCabinetFilter.Name}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="AltName" class="form-label">Альт. наименование</label>
                <input type="text" id="AltName" name="AltName" class="form-control" value="{{.RackCabinetFilter.AltName}}">
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
                <a href="/sprsrv/rackcabinetedit/0" class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.RackCabinetCount}}</span>
                <div class="form-check mb-0">
                    <input class="form-check-input" type="checkbox" id="ShowUnnamed" name="ShowUnnamed" value="1" {{if eq .RackCabinetFilter.ShowUnnamed "1"}}checked{{end}} onchange="document.getElementById('pageField').value=1; this.form.submit();">
                    <label class="form-check-label small text-muted" for="ShowUnnamed">Показывать шкафы без наименований</label>
                </div>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="rcFilterStickyToggle" onchange="rcToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="rcFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>Наименование</th>
                <th>Альт. наименование</th>
                <th>ЦОД</th>
                <th class="text-end">Высота, U</th>
                <th class="text-end" title="Rack-серверы + корзины (без блейдов)">Ср. 24ч, Вт</th>
                <th class="text-end">Пик 24ч, Вт</th>
                <th class="text-end">Паспорт, Вт</th>
            </tr>
        </thead>
        <tbody>
            {{range $rc := .RackCabinetList}}
            <tr class="row-context-target" data-menu-id="rcRowMenu{{$rc.Id}}">
                <td class="text-end">{{$rc.Id}}</td>
                <td><a href="/sprsrv/rackcabinetedit/{{$rc.Id}}" target="_blank">{{$rc.Name}}</a></td>
                <td>{{$rc.AltName}}</td>
                <td>{{$rc.DatacenterSiteName}}</td>
                <td class="text-end">{{$rc.UHeight}}</td>
                <td class="text-end">{{printf "%.0f" $rc.PowerAvg24hW}}</td>
                <td class="text-end">{{printf "%.0f" $rc.PowerPeak24hW}}</td>
                <td class="text-end">{{printf "%.0f" $rc.PowerRatedMaxW}}</td>
                <td id="rcRowMenu{{$rc.Id}}" class="d-none">
                    <a class="ctx-menu-item" href="/sprsrv/rackcabinetedit/{{$rc.Id}}" target="_blank">Редактировать</a>
                    <a class="ctx-menu-item" href="/sprsrv/rackcabinetview/{{$rc.Id}}" target="_blank">Вид шкафа с оборудованием</a>
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
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.RackCabinetCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="rcPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="rcPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="rcPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="rcPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="rcPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<div id="rcRowContextMenu" class="ctx-menu d-none"></div>

<script>
    // Фильтр по населённому пункту: подсказка /sprsrv/localitysearch (как в
    // справочнике площадок/ЦОД); сужает select «Площадка/ЦОД» по data-locality.
    (function () {
        var search = document.getElementById('LocalitySearch');
        var list = document.getElementById('LocalityList');
        var code = document.getElementById('LocalityCode');
        var siteSelect = document.getElementById('DatacenterSiteId');
        if (!search || !list || !code) return;
        var timer = null, seq = 0;
        var allSiteOptions = siteSelect ? Array.prototype.slice.call(siteSelect.options) : [];

        function filterSitesByLocality() {
            if (!siteSelect) return;
            var loc = code.value || '';
            var cur = siteSelect.value;
            siteSelect.innerHTML = '';
            allSiteOptions.forEach(function (opt) {
                if (opt.value === '0' || opt.value === '') {
                    siteSelect.appendChild(opt.cloneNode(true));
                    return;
                }
                if (!loc || opt.getAttribute('data-locality') === loc) {
                    siteSelect.appendChild(opt.cloneNode(true));
                }
            });
            var still = false;
            Array.prototype.forEach.call(siteSelect.options, function (o) {
                if (o.value === cur) still = true;
            });
            siteSelect.value = still ? cur : '0';
        }

        function hide() { list.classList.add('d-none'); list.innerHTML = ''; }

        function pick(item) {
            code.value = item.oktmo;
            search.value = item.label;
            hide();
            filterSitesByLocality();
        }

        function render(items) {
            list.innerHTML = '';
            if (!items.length) {
                var empty = document.createElement('div');
                empty.className = 'list-group-item text-muted small';
                empty.textContent = 'Ничего не найдено';
                list.appendChild(empty);
            }
            items.forEach(function (it) {
                var a = document.createElement('button');
                a.type = 'button';
                a.className = 'list-group-item list-group-item-action small';
                a.textContent = it.label + '  [' + it.oktmo + ']';
                a.addEventListener('mousedown', function (e) { e.preventDefault(); pick(it); });
                list.appendChild(a);
            });
            list.classList.remove('d-none');
        }

        function query() {
            var q = search.value.trim();
            if (q.length < 2) { hide(); return; }
            var my = ++seq;
            fetch('/sprsrv/localitysearch?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
                .then(function (r) { return r.json(); })
                .then(function (items) { if (my === seq) render(items); })
                .catch(function () { hide(); });
        }

        search.addEventListener('input', function () {
            if (search.value.trim() === '') {
                code.value = '';
                filterSitesByLocality();
            }
            clearTimeout(timer);
            timer = setTimeout(query, 200);
        });
        search.addEventListener('blur', function () { setTimeout(hide, 150); });
        search.addEventListener('keydown', function (e) { if (e.key === 'Escape') hide(); });
        filterSitesByLocality();
    })();

    (function () {
        var STORAGE_KEY = 'rackcabinetlist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('rcFilterStickyToggle');
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

        window.rcToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function rcPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function rcPagerChangePageSize(size) {
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
        var menu = document.getElementById('rcRowContextMenu');
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
