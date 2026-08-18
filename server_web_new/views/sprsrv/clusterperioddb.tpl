<script>
$(document).ready(function () {
    $('[data-toggle="toggle-st"]').change(function () {
        $(this).parents().next('.hidestorage').toggle();
    });
});

// Смена значения любого из полей фильтра (vcenter/датацентр/кластер/период)
// сразу перезагружает страницу с новыми GET-параметрами - без отдельной
// кнопки "Применить" для селектов, аналогично /sprsrv/listclusterview.
function clusterPeriodNavigate() {
    var vc = document.getElementById('vcid').value || 'all';
    var dc = document.getElementById('dcid').value || 'all';
    var cl = document.getElementById('clid').value || 'all';
    var from = document.getElementById('fromDate').value;
    var to = document.getElementById('toDate').value;
    var enc = encodeURIComponent;
    window.location.href = '/sprsrv/clusterperioddb?vcenter=' + enc(vc) + '&datacenter=' + enc(dc) + '&cluster=' + enc(cl) + '&from=' + enc(from) + '&to=' + enc(to);
}

// Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
// приём идентичен /sprsrv/cluster (сохраняется отдельным ключом в localStorage).
(function () {
    var STORAGE_KEY = 'clusterperioddb.filterSticky';

    function init() {
        var bar = document.querySelector('.cluster-filter-bar');
        var toggle = document.getElementById('clusterPeriodFilterStickyToggle');
        if (!bar || !toggle) return;

        function applySticky(sticky) {
            bar.classList.toggle('cluster-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.clusterPeriodToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
</script>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы, индикатора
     питания и т.д. вынесены в общий server_web_new/static/css/loadmon.css
     (подключён в header.tpl); здесь остаются только правила, специфичные для
     этой страницы (идентичны /sprsrv/cluster, см. views/sprsrv/cluster.tpl). -->
<style>
.label tr td label {
    display: block;
}

.hidestorage {
    display: none;
}
</style>

<div class="container-fluid d-grid gap-2">
<div class="hero-text">
    <h3>Список кластеров и одиночных хостов - метрики за период (по данным БД)</h3>
    <div class="text-muted small">
        Данные посчитаны по суточным записям в таблицах cluster_record/host_record/vm_record
        (заполняются ежесуточно), а не по текущему состоянию vSphere - в отличие от
        <a href="/sprsrv/cluster/all/all/all">"Загрузка кластеров и хостов с 00:00"</a>, где
        показывается состояние "здесь и сейчас".
    </div>
    {{if .flash.error}}
    <blockquote>{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote>{{.flash.notice}}</blockquote>
    {{end}}
</div>

<div class="cluster-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3">
        <div class="col-6 col-md-3">
            <label for="vcid" class="form-label">Сервер управления</label>
            <select id="vcid" class="form-select" onchange="clusterPeriodNavigate()">
                <option value="all" {{if eq .CurVcenter "all"}}selected{{end}}>Все</option>
                {{range $vc := .VcenterList}}
                <option value="{{$vc}}" {{if eq $.CurVcenter $vc}}selected{{end}}>{{$vc}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="dcid" class="form-label">Датацентр</label>
            <select id="dcid" class="form-select" onchange="clusterPeriodNavigate()">
                <option value="all" {{if eq .CurDatacenter "all"}}selected{{end}}>Все</option>
                {{range $dc := .DatacenterList}}
                <option value="{{$dc}}" {{if eq $.CurDatacenter $dc}}selected{{end}}>{{$dc}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-2">
            <label for="clid" class="form-label">Кластер</label>
            <select id="clid" class="form-select" onchange="clusterPeriodNavigate()">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $cl := .ClusterList2}}
                <option value="{{$cl}}" {{if eq $.CurCluster $cl}}selected{{end}}>{{$cl}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-2">
            <label for="fromDate" class="form-label">Период с</label>
            <input type="date" id="fromDate" class="form-control" value="{{.FromDate}}">
        </div>
        <div class="col-6 col-md-2">
            <label for="toDate" class="form-label">по</label>
            <input type="date" id="toDate" class="form-control" value="{{.ToDate}}">
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex flex-wrap align-items-center justify-content-between gap-2">
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="clusterPeriodNavigate()">Найти</button>
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" id="clusterPeriodFilterStickyToggle" onchange="clusterPeriodToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted" for="clusterPeriodFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
</div>

<!-- Стиль таблицы приведён к образцу /sprsrv/view2 (см. views/sprsrv/view2.tpl
     и .spr-classic-table в loadmon.css): настоящая <table> с <thead>/<tbody>
     вместо раскладки на div+float, ширина столбцов задаётся через <colgroup>
     в процентах от ширины таблицы, чтобы столбцы шапки и данных совпадали и
     не наезжали друг на друга при любой ширине окна. -->
<div style="overflow-x:auto">
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <colgroup>
        <col style="width:4%">
        <col style="width:26%">
        <col style="width:9%">
        <col style="width:7%">
        <col style="width:8%">
        <col style="width:8%">
        <col style="width:12%">
        <col style="width:12%">
        <col style="width:8%">
        <col style="width:6%">
    </colgroup>
    <thead>
        <tr>
            <th></th>
            <th>Кластеры (одиночные хосты)</th>
            <th class="text-center">Процессоры / Ядра (частота МГц)</th>
            <th class="text-center">Память (Гб)</th>
            <th class="text-center">Процессор (макс. %)</th>
            <th class="text-center">Память (макс. %)</th>
            <th class="text-center">Диски всего (Гб)</th>
            <th class="text-center">Диски свободно (Гб)</th>
            <th class="text-center">Диски занято (%)</th>
            <th class="text-center">Записей за период</th>
        </tr>
    </thead>
    {{range $index, $cl := .ClusterList}}
    <tbody class="label">
        <tr>
            <td>
                <label for="clst{{$index}}">
                    <input type="checkbox" name="clusterst{{$index}}" id="clst{{$index}}" data-toggle="toggle-st">
                </label>
            </td>
            <td>{{$cl.Vcenter}} / {{$cl.Datacenter}} / {{$cl.Name}}</td>
            <td class="text-center">{{printf "%d/%d (%d)" $cl.AllNumCpuPkgs $cl.AllNumCpuCores $cl.CpuMhzAvg}}</td>
            <td class="text-center">{{printf "%.0f" $cl.AllMemorySizeGB}}</td>
            {{if and (ge $cl.CPU.Max 0.0) (le $cl.CPU.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $cl.CPU.Max}}</td>
            {{else if and (gt $cl.CPU.Max 75.0) (le $cl.CPU.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $cl.CPU.Max}}</td>
            {{else if (gt $cl.CPU.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $cl.CPU.Max}}</td>
            {{else}}<td class="text-end">{{printf "%2.2f" $cl.CPU.Max}}</td>{{end}}
            {{if and (ge $cl.Mem.Max 0.0) (le $cl.Mem.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $cl.Mem.Max}}</td>
            {{else if and (gt $cl.Mem.Max 75.0) (le $cl.Mem.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $cl.Mem.Max}}</td>
            {{else if (gt $cl.Mem.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $cl.Mem.Max}}</td>
            {{else}}<td class="text-end">{{printf "%2.2f" $cl.Mem.Max}}</td>{{end}}
            <td class="text-end">{{printf "%.1f" $cl.AllCapacityGb}}</td>
            <td class="text-end">{{printf "%.1f" $cl.AllFreeSpaceGb}}</td>
            <td class="text-end">{{printf "%2.2f" $cl.AllPercentUsed}}</td>
            <td class="text-end">{{$cl.RecordCount}}</td>
        </tr>
    </tbody>
    <tbody class="hidestorage">
        {{if gt $cl.HostCount 0}}
        <tr>
            <td colspan="10">
                <div class="text-muted small mb-1">Хосты кластера</div>
                <table class="table table-sm table-bordered table-hover align-middle spr-classic-table mb-2">
                    <colgroup>
                        <col style="width:15%">
                        <col style="width:9%">
                        <col style="width:6%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                    </colgroup>
                    <thead>
                        <tr>
                            <th>Хосты кластера</th>
                            <th class="text-center">Процессоры / Ядра (частота МГц)</th>
                            <th class="text-center">Память (Гб)</th>
                            <th class="text-center">Процессор (макс. %)</th>
                            <th class="text-center">Процессор (ср. %)</th>
                            <th class="text-center">Процессор (мин. %)</th>
                            <th class="text-center">Память (макс. %)</th>
                            <th class="text-center">Память (ср. %)</th>
                            <th class="text-center">Память (мин. %)</th>
                            <th class="text-center">Диск (макс. Мб/сек)</th>
                            <th class="text-center">Диск (ср. Мб/сек)</th>
                            <th class="text-center">Диск (мин. Мб/сек)</th>
                            <th class="text-center">Сеть (макс. Мб/сек)</th>
                            <th class="text-center">Сеть (ср. Мб/сек)</th>
                            <th class="text-center">Сеть (мин. Мб/сек)</th>
                        </tr>
                    </thead>
                    <tbody>
                    {{range $idx, $hi := $cl.Hosts}}
                    <tr>
                        <td>{{if ne $hi.SerialNumber ""}}<a href="/sprsrv/hwconfeditbysn/{{$hi.SerialNumber}}" target="_blank" title="Редактировать конфигурацию сервера">{{$hi.Name}}</a>{{else}}<a href="/sprsrv/hwconfeditbyname/{{$hi.Name}}" target="_blank" title="Редактировать конфигурацию сервера">{{$hi.Name}}</a>{{end}}</td>
                        <td class="text-center">{{printf "%d/%d (%d)" $hi.NumCpuPkgs $hi.NumCpuCores $hi.CpuMhz}}</td>
                        <td class="text-center">{{printf "%.0f" $hi.MemorySizeGB}}</td>
                        {{if and (ge $hi.CPU.Max 0.0) (le $hi.CPU.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.CPU.Max}}</td>
                        {{else if and (gt $hi.CPU.Max 75.0) (le $hi.CPU.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $hi.CPU.Max}}</td>
                        {{else if (gt $hi.CPU.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $hi.CPU.Max}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.CPU.Max}}</td>{{end}}
                        {{if and (ge $hi.CPU.Avg 0.0) (le $hi.CPU.Avg 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.CPU.Avg}}</td>
                        {{else if and (gt $hi.CPU.Avg 75.0) (le $hi.CPU.Avg 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $hi.CPU.Avg}}</td>
                        {{else if (gt $hi.CPU.Avg 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $hi.CPU.Avg}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.CPU.Avg}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $hi.CPU.Min}}</td>
                        {{if and (ge $hi.Mem.Max 0.0) (le $hi.Mem.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Max}}</td>
                        {{else if and (gt $hi.Mem.Max 75.0) (le $hi.Mem.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $hi.Mem.Max}}</td>
                        {{else if (gt $hi.Mem.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $hi.Mem.Max}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Max}}</td>{{end}}
                        {{if and (ge $hi.Mem.Avg 0.0) (le $hi.Mem.Avg 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Avg}}</td>
                        {{else if and (gt $hi.Mem.Avg 75.0) (le $hi.Mem.Avg 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $hi.Mem.Avg}}</td>
                        {{else if (gt $hi.Mem.Avg 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $hi.Mem.Avg}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Avg}}</td>{{end}}
                        {{if and (ge $hi.Mem.Min 0.0) (le $hi.Mem.Min 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Min}}</td>
                        {{else if and (gt $hi.Mem.Min 75.0) (le $hi.Mem.Min 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $hi.Mem.Min}}</td>
                        {{else if (gt $hi.Mem.Min 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $hi.Mem.Min}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.Mem.Min}}</td>{{end}}
                        <td class="text-end">{{printf "%.1f" $hi.Disk.Max}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.Disk.Avg}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.Disk.Min}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.Net.Max}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.Net.Avg}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.Net.Min}}</td>
                    </tr>
                    {{end}}
                    {{if gt $cl.HostCount 1}}
                    <tr class="table-secondary">
                        <td><b>Итого (по кластеру)</b></td>
                        <td class="text-center"><b>{{printf "%d/%d (%d)" $cl.AllNumCpuPkgs $cl.AllNumCpuCores $cl.CpuMhzAvg}}</b></td>
                        <td class="text-center"><b>{{printf "%.0f" $cl.AllMemorySizeGB}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPU.Max}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPU.Avg}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPU.Min}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.Mem.Max}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.Mem.Avg}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.Mem.Min}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Disk.Max}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Disk.Avg}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Disk.Min}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Net.Max}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Net.Avg}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.Net.Min}}</b></td>
                    </tr>
                    {{end}}
                    </tbody>
                </table>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.VMCount 0}}
        {{if $cl.VMDataMissing}}
        <tr>
            <td colspan="10">
                <div class="text-muted fst-italic py-1">Виртуальные машины кластера ({{$cl.VMCount}} шт.) - за выбранный период нет сохранённых данных в БД (vm_record). Попробуйте расширить диапазон дат.</div>
            </td>
        </tr>
        {{else}}
        <tr>
            <td colspan="10">
                <div class="text-muted small mb-1">Виртуальные машины кластера</div>
                <table class="table table-sm table-bordered table-hover align-middle spr-classic-table mb-2">
                    <colgroup>
                        <col style="width:15%">
                        <col style="width:9%">
                        <col style="width:6%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                        <col style="width:5.83%"><col style="width:5.83%"><col style="width:5.83%">
                    </colgroup>
                    <thead>
                        <tr>
                            <th>Виртуальные машины кластера</th>
                            <th class="text-center">Процессоры / Ядра</th>
                            <th class="text-center">Память (Гб)</th>
                            <th class="text-center">Процессор (макс. %)</th>
                            <th class="text-center">Процессор (ср. %)</th>
                            <th class="text-center">Процессор (мин. %)</th>
                            <th class="text-center">Память (макс. %)</th>
                            <th class="text-center">Память (ср. %)</th>
                            <th class="text-center">Память (мин. %)</th>
                            <th class="text-center">Диск (макс. Мб/сек)</th>
                            <th class="text-center">Диск (ср. Мб/сек)</th>
                            <th class="text-center">Диск (мин. Мб/сек)</th>
                            <th class="text-center">Сеть (макс. Мб/сек)</th>
                            <th class="text-center">Сеть (ср. Мб/сек)</th>
                            <th class="text-center">Сеть (мин. Мб/сек)</th>
                        </tr>
                    </thead>
                    <tbody>
                    {{range $idx, $vm := $cl.VM}}
                    <tr>
                        <td>{{if ne $vm.Name ""}}<a href="/sprsrv/vmeditbyname/{{$vm.Name}}" target="_blank" title="Общие данные виртуальной машины">{{$vm.Name}}</a>{{else}}&mdash;{{end}}</td>
                        <td class="text-center">{{printf "%d/%d" $vm.NumCPU $vm.NumCores}}</td>
                        <td class="text-center">{{printf "%.0f" $vm.MemorySizeGB}}</td>
                        {{if and (ge $vm.CPU.Max 0.0) (le $vm.CPU.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.CPU.Max}}</td>
                        {{else if and (gt $vm.CPU.Max 75.0) (le $vm.CPU.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $vm.CPU.Max}}</td>
                        {{else if (gt $vm.CPU.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $vm.CPU.Max}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.CPU.Max}}</td>{{end}}
                        {{if and (ge $vm.CPU.Avg 0.0) (le $vm.CPU.Avg 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.CPU.Avg}}</td>
                        {{else if and (gt $vm.CPU.Avg 75.0) (le $vm.CPU.Avg 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $vm.CPU.Avg}}</td>
                        {{else if (gt $vm.CPU.Avg 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $vm.CPU.Avg}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.CPU.Avg}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $vm.CPU.Min}}</td>
                        {{if and (ge $vm.Mem.Max 0.0) (le $vm.Mem.Max 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.Mem.Max}}</td>
                        {{else if and (gt $vm.Mem.Max 75.0) (le $vm.Mem.Max 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $vm.Mem.Max}}</td>
                        {{else if (gt $vm.Mem.Max 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $vm.Mem.Max}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.Mem.Max}}</td>{{end}}
                        {{if and (ge $vm.Mem.Avg 0.0) (le $vm.Mem.Avg 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.Mem.Avg}}</td>
                        {{else if and (gt $vm.Mem.Avg 75.0) (le $vm.Mem.Avg 90.0)}}<td class="metric-warn text-end">{{printf "%2.2f" $vm.Mem.Avg}}</td>
                        {{else if (gt $vm.Mem.Avg 90.0)}}<td class="metric-crit text-end">{{printf "%2.2f" $vm.Mem.Avg}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.Mem.Avg}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $vm.Mem.Min}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Disk.Max}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Disk.Avg}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Disk.Min}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Net.Max}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Net.Avg}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.Net.Min}}</td>
                    </tr>
                    {{end}}
                    </tbody>
                </table>
            </td>
        </tr>
        {{end}}
        {{end}}
    </tbody>
    {{else}}
    <tbody>
        <tr>
            <td colspan="10" class="text-muted py-3">По выбранным условиям фильтра и периоду данные в БД не найдены.</td>
        </tr>
    </tbody>
    {{end}}
</table>
</div>
</div>
