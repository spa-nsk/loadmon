<script>
$(document).ready(function () {
    $('[data-toggle="toggle-st"]').change(function () {
        $(this).parents().next('.hidestorage').toggle();
    });
});

function selectvcenter(val) {
    document.getElementById("Update").submit();
}

function selectdatacenter(val) {
    document.getElementById("Update").submit();
}

function selectcluster(val) {
    document.getElementById("Update").submit();
}

// Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
// приём идентичен /sprsrv/hwviewlist и /sprsrv/hwconflist/0, выбор
// запоминается отдельным ключом в localStorage.
(function () {
    var STORAGE_KEY = 'cluster.filterSticky';

    function init() {
        var bar = document.querySelector('.cluster-filter-bar');
        var toggle = document.getElementById('clusterFilterStickyToggle');
        if (!bar || !toggle) return;

        function applySticky(sticky) {
            bar.classList.toggle('cluster-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.clusterToggleFilterSticky = function (sticky) {
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

<!-- Стили закрепляемой панели фильтра вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl); здесь
     остаются только правила, специфичные для этой страницы. -->
<style>
.label tr td label {
    display: block;
}

.hidestorage {
    display: none;
}

/* svg.bi без явного размера (в отличие от .power-led на других страницах)
   рендерится в размере viewBox самого символа #activity, а не в размере
   текста рядом - из-за этого иконка "график загрузки ВМ" выглядела в разы
   крупнее остальных элементов таблицы. */
.cluster-activity-icon {
    width: 1rem;
    height: 1rem;
    vertical-align: -0.125em;
}
</style>

<div class="container-fluid d-grid gap-2">
<div class="hero-text">
    <h3>Список кластеров и одиночных хостов виртуальных машин</h3>
    {{if .flash.error}}
    <blockquote>{{.flash.error}}</blockquote>
    {{end}}

    {{if .flash.notice}}
    <blockquote>{{.flash.notice}}</blockquote>
    {{end}}
</div>

<form role="form" id="Update" method="POST">
    <div class="cluster-filter-bar mb-3 p-3 rounded border">
        <div class="row g-3">
            <div class="col-6 col-md-4">
                <label for="vcenterId" class="form-label">Сервер управления</label>
                <select id="vcid" name="vcenterId" size="1" class="form-select" onchange="selectvcenter(this.value)" value="{{.vcenterId}}">
                    <option value="all" {{if eq .vcenterId "all"}}selected{{end}}>Все</option>
                    {{range $vc := .VcenterList}}
                    <option value="{{$vc.Hostname}}" {{if eq $vc.Hostname $.vcenterId}}selected{{end}}>{{$vc.Hostname}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-4">
                <label for="datacenterId" class="form-label">Датацентр</label>
                <select id="dcid" name="datacenterId" size="1" class="form-select" onchange="selectdatacenter(this.value)" value="{{.datacenterId}}">
                    <option value="all" {{if eq .datacenterId "all"}}selected{{end}}>Все</option>
                    {{range $dc := .DatacenterList}}
                    <option value="{{$dc.Name}}" {{if eq $dc.Name $.datacenterId}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="col-6 col-md-4">
                <label for="clusterId" class="form-label">Кластер</label>
                <select id="clid" name="clusterId" size="1" class="form-select" onchange="selectcluster(this.value)" value="{{.clusterId}}">
                    <option value="all" {{if eq .clusterId "all"}}selected{{end}}>Все</option>
                    {{range $cli := .ClusterList2}}
                    <option value="{{$cli.Name}}" {{if eq $cli.Name $.clusterId}}selected{{end}}>{{$cli.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <div class="row mt-2">
            <div class="col-12 d-flex justify-content-end">
                <div class="form-check form-switch mb-0">
                    <input class="form-check-input" type="checkbox" id="clusterFilterStickyToggle" onchange="clusterToggleFilterSticky(this.checked)">
                    <label class="form-check-label small text-muted" for="clusterFilterStickyToggle">Закреплять фильтр вверху экрана</label>
                </div>
            </div>
        </div>
    </div>

<!-- Стиль таблицы приведён к образцу /sprsrv/view2 (см. views/sprsrv/view2.tpl
     и .spr-classic-table в loadmon.css) и аналогичен /sprsrv/clusterperioddb -
     настоящая <table> с <colgroup> в процентах вместо раскладки на div+float.
     Колонкам "Диски всего"/"Диски свободно" здесь намеренно отдано больше
     места (по 14% каждой), чем остальным числовым колонкам - значения в них
     не помещались в ячейку. -->
<div style="overflow-x:auto">
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <colgroup>
        <col style="width:3%">
        <col style="width:20%">
        <col style="width:9%">
        <col style="width:6%">
        <col style="width:7%">
        <col style="width:7%">
        <col style="width:7%">
        <col style="width:7%">
        <col style="width:14%">
        <col style="width:14%">
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
            <th class="text-center">Диск (макс. Мб/сек)</th>
            <th class="text-center">Сеть (макс. Мб/сек)</th>
            <th class="text-center">Диски всего (Гб)</th>
            <th class="text-center">Диски свободно (Гб)</th>
            <th class="text-center">Диски занято (%)</th>
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
            <td>{{$cl.Vcenter}} / {{$cl.Datacenter}} / {{if $cl.HostFlag}}{{if $cl.Hosts}}{{$h0 := index $cl.Hosts 0}}{{if ne $h0.SerialNumber ""}}<a href="/sprsrv/hwconfeditbysn/{{$h0.SerialNumber}}" target="_blank" title="Редактировать конфигурацию сервера">{{$cl.Name}}</a>{{else}}<a href="/sprsrv/hwconfeditbyname/{{$h0.Name}}" target="_blank" title="Редактировать конфигурацию сервера">{{$cl.Name}}</a>{{end}}{{else}}{{$cl.Name}}{{end}}{{else}}{{$cl.Name}}{{end}}</td>
            <td class="text-center">{{printf "%d/%d (%d)" $cl.AllNumCPUPkgs $cl.AllNumCPUCores $cl.CPUMhzAvg}}</td>
            <td class="text-center">{{printf "%.0f" $cl.AllMemorySizeGB}}</td>
            {{if and (ge $cl.CPUUsageMaximum 0.0) (le $cl.CPUUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $cl.CPUUsageMaximum}}</td>
            {{else if and (gt $cl.CPUUsageMaximum 75.0) (le $cl.CPUUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $cl.CPUUsageMaximum}}</td>
            {{else if (gt $cl.CPUUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $cl.CPUUsageMaximum}}</td>
            {{else}}<td class="text-end">{{printf "%2.2f" $cl.CPUUsageMaximum}}</td>{{end}}
            {{if and (ge $cl.MemUsageMaximum 0.0) (le $cl.MemUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $cl.MemUsageMaximum}}</td>
            {{else if and (gt $cl.MemUsageMaximum 75.0) (le $cl.MemUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $cl.MemUsageMaximum}}</td>
            {{else if (gt $cl.MemUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $cl.MemUsageMaximum}}</td>
            {{else}}<td class="text-end">{{printf "%2.2f" $cl.MemUsageMaximum}}</td>{{end}}
            <td class="text-end">{{printf "%.1f" $cl.DiskUsageMaximum}}</td>
            <td class="text-end">{{printf "%.1f" $cl.NetUsageMaximum}}</td>
            <td class="text-end">{{printf "%.1f" $cl.AllCapacityGb}}</td>
            <td class="text-end">{{printf "%.1f" $cl.AllFreeSpaceGb}}</td>
            <td class="text-end">{{printf "%2.2f" $cl.AllPercentUsed}}</td>
        </tr>
    </tbody>
    <tbody class="hidestorage">
        {{if gt $cl.HostCount 0}}
        <tr>
            <td colspan="11">
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
                       C:\work\loadmon\server_web_new\views\sprsrv\cluster.tpl     <th class="text-center">Память (Гб)</th>
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
                        <td class="text-center">{{printf "%d/%d (%d)" $hi.NumCPUPkgs $hi.NumCPUCores $hi.CPUMhz}}</td>
                        <td class="text-center">{{printf "%.0f" $hi.MemorySizeGB}}</td>
                        {{if and (ge $hi.CPUUsageMaximum 0.0) (le $hi.CPUUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.CPUUsageMaximum}}</td>
                        {{else if and (gt $hi.CPUUsageMaximum 75.0) (le $hi.CPUUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $hi.CPUUsageMaximum}}</td>
                        {{else if (gt $hi.CPUUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $hi.CPUUsageMaximum}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.CPUUsageMaximum}}</td>{{end}}
                        {{if and (ge $hi.CPUUsageAverage 0.0) (le $hi.CPUUsageAverage 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.CPUUsageAverage}}</td>
                        {{else if and (gt $hi.CPUUsageAverage 75.0) (le $hi.CPUUsageAverage 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $hi.CPUUsageAverage}}</td>
                        {{else if (gt $hi.CPUUsageAverage 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $hi.CPUUsageAverage}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.CPUUsageAverage}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $hi.CPUUsageMinimum}}</td>
                        {{if and (ge $hi.MemUsageMaximum 0.0) (le $hi.MemUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageMaximum}}</td>
                        {{else if and (gt $hi.MemUsageMaximum 75.0) (le $hi.MemUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $hi.MemUsageMaximum}}</td>
                        {{else if (gt $hi.MemUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $hi.MemUsageMaximum}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageMaximum}}</td>{{end}}
                        {{if and (ge $hi.MemUsageAverage 0.0) (le $hi.MemUsageAverage 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageAverage}}</td>
                        {{else if and (gt $hi.MemUsageAverage 75.0) (le $hi.MemUsageAverage 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $hi.MemUsageAverage}}</td>
                        {{else if (gt $hi.MemUsageAverage 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $hi.MemUsageAverage}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageAverage}}</td>{{end}}
                        {{if and (ge $hi.MemUsageMinimum 0.0) (le $hi.MemUsageMinimum 75.0)}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageMinimum}}</td>
                        {{else if and (gt $hi.MemUsageMinimum 75.0) (le $hi.MemUsageMinimum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $hi.MemUsageMinimum}}</td>
                        {{else if (gt $hi.MemUsageMinimum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $hi.MemUsageMinimum}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $hi.MemUsageMinimum}}</td>{{end}}
                        <td class="text-end">{{printf "%.1f" $hi.DiskUsageMaximum}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.DiskUsageAverage}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.DiskUsageMinimum}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.NetUsageMaximum}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.NetUsageAverage}}</td>
                        <td class="text-end">{{printf "%.1f" $hi.NetUsageMinimum}}</td>
                    </tr>
                    {{end}}
                    {{if gt $cl.HostCount 1}}
                    <tr>
                        <td><b>Итого</b></td>
                        <td class="text-center"><b>{{printf "%d/%d (%d)" $cl.AllNumCPUPkgs $cl.AllNumCPUCores $cl.CPUMhzAvg}}</b></td>
                        <td class="text-center"><b>{{printf "%.0f" $cl.AllMemorySizeGB}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPUUsageMaximum}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPUUsageAverage}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.CPUUsageMinimum}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.MemUsageMaximum}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.MemUsageAverage}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.MemUsageMinimum}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.DiskUsageMaximum}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.DiskUsageAverage}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.DiskUsageMinimum}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.NetUsageMaximum}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.NetUsageAverage}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.NetUsageMinimum}}</b></td>
                    </tr>
                    {{end}}
                    </tbody>
                </table>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.VMCount 0}}
        <tr>
            <td colspan="11">
                <div class="text-muted small mb-1">Виртуальные машины кластера</div>
                <table class="table table-sm table-bordered table-hover align-middle spr-classic-table mb-2">
                    <colgroup>
                        <col style="width:4%">
                        <col style="width:11%">
                        <col style="width:8%">
                        <col style="width:5%">
                        <col style="width:6%"><col style="width:6%"><col style="width:6%">
                        <col style="width:6%"><col style="width:6%"><col style="width:6%">
                        <col style="width:6%"><col style="width:6%"><col style="width:6%">
                        <col style="width:6%"><col style="width:6%"><col style="width:6%">
                    </colgroup>
                    <thead>
                        <tr>
                            <th></th>
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
                        <td class="text-center">{{if gt $vm.PerfCount 0}}<a href="/sprsrv/vmperfday/{{$vm.Vcenter}}/{{$vm.Datacenter}}/{{$vm.Cluster}}/{{$vm.Name}}" target="_blank"><svg class="bi cluster-activity-icon" aria-hidden="true"><use href="#activity"></use></svg></a>{{else}}<svg class="bi power-led is-off" aria-hidden="true" title="виртуальная машина выключена"><use href="#power"></use></svg>{{end}}</td>
                        <td><a href="/sprsrv/vmeditbyname/{{$vm.Name}}" target="_blank" title="Общие данные виртуальной машины">{{$vm.Name}}</a></td>
                        <td class="text-center">{{printf "%d/%d" $vm.NumCPU $vm.NumCores}}</td>
                        <td class="text-center">{{printf "%.0f" $vm.MemorySizeGB}}</td>
                        {{if and (ge $vm.CPUUsageMaximum 0.0) (le $vm.CPUUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.CPUUsageMaximum}}</td>
                        {{else if and (gt $vm.CPUUsageMaximum 75.0) (le $vm.CPUUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $vm.CPUUsageMaximum}}</td>
                        {{else if (gt $vm.CPUUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $vm.CPUUsageMaximum}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.CPUUsageMaximum}}</td>{{end}}
                        {{if and (ge $vm.CPUUsageAverage 0.0) (le $vm.CPUUsageAverage 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.CPUUsageAverage}}</td>
                        {{else if and (gt $vm.CPUUsageAverage 75.0) (le $vm.CPUUsageAverage 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $vm.CPUUsageAverage}}</td>
                        {{else if (gt $vm.CPUUsageAverage 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $vm.CPUUsageAverage}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.CPUUsageAverage}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $vm.CPUUsageMinimum}}</td>
                        {{if and (ge $vm.MemUsageMaximum 0.0) (le $vm.MemUsageMaximum 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.MemUsageMaximum}}</td>
                        {{else if and (gt $vm.MemUsageMaximum 75.0) (le $vm.MemUsageMaximum 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $vm.MemUsageMaximum}}</td>
                        {{else if (gt $vm.MemUsageMaximum 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $vm.MemUsageMaximum}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.MemUsageMaximum}}</td>{{end}}
                        {{if and (ge $vm.MemUsageAverage 0.0) (le $vm.MemUsageAverage 75.0)}}<td class="text-end">{{printf "%2.2f" $vm.MemUsageAverage}}</td>
                        {{else if and (gt $vm.MemUsageAverage 75.0) (le $vm.MemUsageAverage 90.0)}}<td class="table-warning text-end">{{printf "%2.2f" $vm.MemUsageAverage}}</td>
                        {{else if (gt $vm.MemUsageAverage 90.0)}}<td class="table-danger text-end">{{printf "%2.2f" $vm.MemUsageAverage}}</td>
                        {{else}}<td class="text-end">{{printf "%2.2f" $vm.MemUsageAverage}}</td>{{end}}
                        <td class="text-end">{{printf "%2.2f" $vm.MemUsageMinimum}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.DiskUsageMaximum}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.DiskUsageAverage}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.DiskUsageMinimum}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.NetUsageMaximum}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.NetUsageAverage}}</td>
                        <td class="text-end">{{printf "%.1f" $vm.NetUsageMinimum}}</td>
                    </tr>
                    {{end}}
                    </tbody>
                </table>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.StorageCount 0}}
        <tr>
            <td colspan="11">
                <div class="text-muted small mb-1">Хранилища кластера / Виртуальные машины</div>
                <table class="table table-sm table-bordered table-hover align-middle spr-classic-table mb-2">
                    <colgroup>
                        <col style="width:44%">
                        <col style="width:11%">
                        <col style="width:11%">
                        <col style="width:11%">
                        <col style="width:11%">
                        <col style="width:12%">
                    </colgroup>
                    <thead>
                        <tr>
                            <th>Хранилища кластера / Виртуальные машины</th>
                            <th class="text-center">Ёмкость (Гб)</th>
                            <th class="text-center">Свободно (Гб)</th>
                            <th class="text-center">Занято (%)</th>
                            <th class="text-center">Занято VM (Гб)</th>
                            <th class="text-center">Использовано в хранилище (%)</th>
                        </tr>
                    </thead>
                    <tbody>
                    {{range $st := $cl.Storage}}
                    <tr>
                        <td><b>{{$st.StorageName}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $st.CapacityGb}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $st.FreeSpaceGb}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $st.PercentUsed}}</b></td>
                        <td></td>
                        <td></td>
                    </tr>
                    {{range $idx, $vm := $st.VM}}
                    <tr>
                        <td class="ps-4">{{$idx}}. {{$vm.Name}}</td>
                        <td></td>
                        <td></td>
                        <td></td>
                        {{range $j, $usds := $vm.UsageDatastore}}
                        {{if eq $st.StorageName $usds.StorageName}}
                        <td class="text-end">{{printf "%.1f" $usds.CommittedGb}}</td>
                        <td class="text-end">{{printf "%.1f" $usds.PercentAllocateStorage}}</td>
                        {{end}}
                        {{end}}
                    </tr>
                    {{end}}
                    {{end}}
                    {{if gt $cl.StorageCount 1}}
                    <tr>
                        <td><b>Итого</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.AllCapacityGb}}</b></td>
                        <td class="text-end"><b>{{printf "%.1f" $cl.AllFreeSpaceGb}}</b></td>
                        <td class="text-end"><b>{{printf "%2.2f" $cl.AllPercentUsed}}</b></td>
                        <td></td>
                        <td></td>
                    </tr>
                    {{end}}
                    </tbody>
                </table>
            </td>
        </tr>
        {{end}}
    </tbody>
    {{end}}
</table>
</div>
</form>
</div>
