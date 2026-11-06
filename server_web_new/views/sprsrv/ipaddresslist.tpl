<div class="hero-text mb-3">
    <h2>Справочник IP-адресов</h2>
    <p class="text-muted small mb-0">
        Строится автоматически по данным справочника серверов (поля "IP-адрес"/"MAC-адрес"/"Сеть") -
        раз в сутки и сразу при запуске приложения. Раздел только для просмотра, редактирование
        недоступно - чтобы изменить IP/MAC/сеть сервера, откройте его карточку.
    </p>
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
                <label for="IPAddress" class="form-label">IP-адрес</label>
                <input type="text" id="IPAddress" name="IPAddress" class="form-control" value="{{.IPAddressFilter.IPAddress}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="MacAddress" class="form-label">MAC-адрес</label>
                <input type="text" id="MacAddress" name="MacAddress" class="form-control" value="{{.IPAddressFilter.MacAddress}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="ServerName" class="form-label">Сервер</label>
                <input type="text" id="ServerName" name="ServerName" class="form-control" value="{{.IPAddressFilter.ServerName}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="NetworkId" class="form-label">Сеть</label>
                <select id="NetworkId" name="NetworkId" class="form-select">
                    <option value="0">- любая -</option>
                    {{range $nw := .NetworkList}}
                    <option value="{{$nw.Id}}" {{if eq $.IPAddressFilter.NetworkId (printf "%d" $nw.Id)}}selected{{end}}>{{$nw.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-12 d-flex flex-wrap align-items-center gap-2">
                <button type="submit" class="btn btn-primary rounded-pill px-3" onclick="document.getElementById('pageField').value=1;">Просмотр</button>
                <button type="submit" name="ExportBTN" value="Экспорт в CSV" formtarget="_blank" title="Экспорт данных с учётом фильтров в CSV (utf8, разделитель |)" class="btn btn-outline-success rounded-pill px-3">Экспорт в CSV</button>
                <button type="submit" name="ExportBTN2" value="Экспорт в XLSX" formtarget="_blank" title="Экспорт данных с учётом фильтров в XLSX" class="btn btn-outline-success rounded-pill px-3">Экспорт в XLSX</button>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex flex-wrap align-items-center gap-3">
                <span class="text-muted small">Найдено записей: {{.IPCount}}</span>
                <div class="form-check form-switch ms-auto mb-0">
                    <input class="form-check-input" type="checkbox" id="ipFilterStickyToggle" onchange="ipToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="ipFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Id</th>
                <th>IP-адрес</th>
                <th>MAC-адрес</th>
                <th>Сеть</th>
                <th>Сервер</th>
                <th>Обновлено</th>
            </tr>
        </thead>
        <tbody>
            {{range $ip := .IPList}}
            <tr>
                <td class="text-end">{{$ip.Id}}</td>
                <td>{{$ip.IPAddress}}</td>
                <td>{{if $ip.MacAddress}}{{$ip.MacAddress}}{{else}}&mdash;{{end}}</td>
                <td>{{if $ip.NetworkId}}{{index $.NetworkNameById $ip.NetworkId}}{{else}}&mdash;{{end}}</td>
                <td><a href="/sprsrv/updatebase-tab/{{$ip.ServerSprId}}" target="_blank">{{$ip.ServerName}}</a></td>
                <td>{{$ip.Time_Mfy.Format "2006-01-02 15:04"}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="6" class="text-center text-muted py-4">По выбранным условиям фильтра записи не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.IPCount}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="ipPagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="ipPagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="ipPagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="ipPagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="ipPagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
</form>

<script>
    (function () {
        var STORAGE_KEY = 'ipaddresslist.filterSticky';
        var bar = document.querySelector('.hw-filter-bar');
        var toggle = document.getElementById('ipFilterStickyToggle');
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

        window.ipToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };

        window.addEventListener('resize', updateTheadOffset);
        window.addEventListener('load', updateTheadOffset);
    })();

    function ipPagerGoTo(page) {
        var field = document.getElementById('pageField');
        if (!field) return;
        if (page < 1) page = 1;
        field.value = page;
        document.getElementById('Update').submit();
    }

    function ipPagerChangePageSize(size) {
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
</script>
