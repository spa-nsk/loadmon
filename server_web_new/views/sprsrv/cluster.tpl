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

<style>
.label tr td label {
    display: block;
}

.hidestorage {
    display: none;
}

.cluster-filter-bar {
    position: sticky;
    top: 0;
    z-index: 5;
    background-color: var(--bs-body-bg);
    border-color: var(--bs-border-color) !important;
    box-shadow: 0 2px 6px rgba(0, 0, 0, .08);
}

.cluster-filter-bar.cluster-filter-bar-unpinned {
    position: static;
    box-shadow: none;
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
        <div class="row">
            <div class="form-group col-sm-4">
                <label for="vcenterId">Сервер управления</label>
                <select id="vcid" name="vcenterId" size="1" class="form-control" onchange="selectvcenter(this.value)" value="{{.vcenterId}}">
                    <option value="all" {{if eq .vcenterId "all"}}selected{{end}}>Все</option>
                    {{range $vc := .VcenterList}}
                    <option value="{{$vc.Hostname}}" {{if eq $vc.Hostname $.vcenterId}}selected{{end}}>{{$vc.Hostname}}</option>
                    {{end}}
                </select>
            </div>
            <div class="form-group col-sm-4">
                <label for="datacenterId">Датацентр</label>
                <select id="dcid" name="datacenterId" size="1" class="form-control" onchange="selectdatacenter(this.value)" value="{{.datacenterId}}">
                    <option value="all" {{if eq .datacenterId "all"}}selected{{end}}>Все</option>
                    {{range $dc := .DatacenterList}}
                    <option value="{{$dc.Name}}" {{if eq $dc.Name $.datacenterId}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="form-group col-sm-4">
                <label for="clusterId">Кластер</label>
                <select id="clid" name="clusterId" size="1" class="form-control" onchange="selectcluster(this.value)" value="{{.clusterId}}">
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

<div style="overflow-x:auto">
<table class="table">
    <tbody>
        <tr>
            <td>
                <div style="width:1220px">
                    <div style="width:40px; float:left"></div>
                    <div style="width:400px; float:left">Кластеры (одиночные хосты)</div>
                    <div style="width:120px; float:left; text-align: center">Процессоры / Ядра (частота МГц)</div>
                    <div style="width:70px; float:left; text-align: center">Память (Гб)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (макс. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (макс. Мб/сек)</div>
                    <div style="width:90px; float:left; text-align: center">Диски всего (Гб)</div>
                    <div style="width:100px; float:left; text-align: center">Диски свободно (Гб)</div>
                    <div style="width:80px; float:left; text-align: center">Диски занято (%)</div>
                    <div style="clear: both"></div>
                </div>
            </td>
        </tr>
    </tbody>
    {{range $index, $cl := .ClusterList}}
    <tbody class="label">
        <tr>
            <td>
                <div style="width:1220px">
                    <div style="width:40px; float:left">
                        <label for="clst{{$index}}">
                            <input type="checkbox" name="clusterst{{$index}}" id="clst{{$index}}" data-toggle="toggle-st">
                        </label>
                    </div>
                    <div style="width:400px; float:left">{{$cl.Vcenter}} / {{$cl.Datacenter}} / {{if $cl.HostFlag}}{{if $cl.Hosts}}{{$h0 := index $cl.Hosts 0}}{{if ne $h0.SerialNumber ""}}<a href="/sprsrv/hwconfeditbysn/{{$h0.SerialNumber}}" target="_blank" title="Редактировать конфигурацию сервера">{{$cl.Name}}</a>{{else}}<a href="/sprsrv/hwconfeditbyname/{{$h0.Name}}" target="_blank" title="Редактировать конфигурацию сервера">{{$cl.Name}}</a>{{end}}{{else}}{{$cl.Name}}{{end}}{{else}}{{$cl.Name}}{{end}}</div>
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d (%d)" $cl.AllNumCPUPkgs $cl.AllNumCPUCores $cl.CPUMhzAvg}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $cl.AllMemorySizeGB}}</div>
                    {{if and (ge $cl.CPUUsageMaximum 0.0) (le $cl.CPUUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPUUsageMaximum}}</div>
                    {{else if and (gt $cl.CPUUsageMaximum 75.0) (le $cl.CPUUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPUUsageMaximum}}</div>
                    {{else if (gt $cl.CPUUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPUUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPUUsageMaximum}}</div>{{end}}
                    {{if and (ge $cl.MemUsageMaximum 0.0) (le $cl.MemUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.MemUsageMaximum}}</div>
                    {{else if and (gt $cl.MemUsageMaximum 75.0) (le $cl.MemUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.MemUsageMaximum}}</div>
                    {{else if (gt $cl.MemUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.MemUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.MemUsageMaximum}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $cl.DiskUsageMaximum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $cl.NetUsageMaximum}}</div>
                    <div style="width:90px; float:left; text-align: right">{{printf "%.3f" $cl.AllCapacityGb}}</div>
                    <div style="width:100px; float:left; text-align: right">{{printf "%.3f" $cl.AllFreeSpaceGb}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.AllPercentUsed}}</div>
                    <div style="clear: both"></div>
                </div>
            </td>
        </tr>
    </tbody>
    <tbody class="hidestorage">
        {{if gt $cl.HostCount 0}}
        <tr>
            <td>
                <div style="width:1440px">
                    <div style="width:220px; float:left">Хосты кластера</div>
                    <div style="width:120px; float:left; text-align: center">Процессоры / Ядра (частота МГц)</div>
                    <div style="width:70px; float:left; text-align: center">Память (Гб)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (ср. %)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (мин. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (ср. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (мин. %)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (макс. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (ср. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (мин. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (макс. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (ср. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (мин. Мб/сек)</div>
                    <div style="clear: both"></div>
                </div>
            </td>
        </tr>
        <tr>
            <td>
                <div style="width:1440px">
                    {{range $idx, $hi := $cl.Hosts}}
                    <div style="width:220px; float:left">{{if ne $hi.SerialNumber ""}}<a href="/sprsrv/hwconfeditbysn/{{$hi.SerialNumber}}" target="_blank" title="Редактировать конфигурацию сервера">{{$hi.Name}}</a>{{else}}<a href="/sprsrv/hwconfeditbyname/{{$hi.Name}}" target="_blank" title="Редактировать конфигурацию сервера">{{$hi.Name}}</a>{{end}}</div>
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d (%d)" $hi.NumCPUPkgs $hi.NumCPUCores $hi.CPUMhz}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $hi.MemorySizeGB}}</div>
                    {{if and (ge $hi.CPUUsageMaximum 0.0) (le $hi.CPUUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageMaximum}}</div>
                    {{else if and (gt $hi.CPUUsageMaximum 75.0) (le $hi.CPUUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageMaximum}}</div>
                    {{else if (gt $hi.CPUUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageMaximum}}</div>{{end}}
                    {{if and (ge $hi.CPUUsageAverage 0.0) (le $hi.CPUUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageAverage}}</div>
                    {{else if and (gt $hi.CPUUsageAverage 75.0) (le $hi.CPUUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageAverage}}</div>
                    {{else if (gt $hi.CPUUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageAverage}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageAverage}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPUUsageMinimum}}</div>
                    {{if and (ge $hi.MemUsageMaximum 0.0) (le $hi.MemUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMaximum}}</div>
                    {{else if and (gt $hi.MemUsageMaximum 75.0) (le $hi.MemUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMaximum}}</div>
                    {{else if (gt $hi.MemUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMaximum}}</div>{{end}}
                    {{if and (ge $hi.MemUsageAverage 0.0) (le $hi.MemUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageAverage}}</div>
                    {{else if and (gt $hi.MemUsageAverage 75.0) (le $hi.MemUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageAverage}}</div>
                    {{else if (gt $hi.MemUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageAverage}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageAverage}}</div>{{end}}
                    {{if and (ge $hi.MemUsageMinimum 0.0) (le $hi.MemUsageMinimum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMinimum}}</div>
                    {{else if and (gt $hi.MemUsageMinimum 75.0) (le $hi.MemUsageMinimum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMinimum}}</div>
                    {{else if (gt $hi.MemUsageMinimum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMinimum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.MemUsageMinimum}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.DiskUsageMaximum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.DiskUsageAverage}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.DiskUsageMinimum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.NetUsageMaximum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.NetUsageAverage}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.NetUsageMinimum}}</div>
                    <div style="clear: both"></div>
                    {{end}}
                    {{if gt $cl.HostCount 1}}
                    <div style="width:220px; float:left"><b>Итого</b></div>
                    <div style="width:120px; float:left; text-align: center"><b>{{printf "%d/%d (%d)" $cl.AllNumCPUPkgs $cl.AllNumCPUCores $cl.CPUMhzAvg}}</b></div>
                    <div style="width:70px; float:left; text-align: center"><b>{{printf "%.0f" $cl.AllMemorySizeGB}}</b></div>
                    {{if and (ge $cl.CPUUsageMaximum 0.0) (le $cl.CPUUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageMaximum}}</b></div>
                    {{else if and (gt $cl.CPUUsageMaximum 75.0) (le $cl.CPUUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageMaximum}}</b></div>
                    {{else if (gt $cl.CPUUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageMaximum}}</b></div>
                    {{else}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageMaximum}}</b></div>{{end}}
                    {{if and (ge $cl.CPUUsageAverage 0.0) (le $cl.CPUUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageAverage}}</b></div>
                    {{else if and (gt $cl.CPUUsageAverage 75.0) (le $cl.CPUUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageAverage}}</b></div>
                    {{else if (gt $cl.CPUUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageAverage}}</b></div>
                    {{else}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageAverage}}</b></div>{{end}}
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPUUsageMinimum}}</b></div>
                    {{if and (ge $cl.MemUsageMaximum 0.0) (le $cl.MemUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMaximum}}</b></div>
                    {{else if and (gt $cl.MemUsageMaximum 75.0) (le $cl.MemUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMaximum}}</b></div>
                    {{else if (gt $cl.MemUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMaximum}}</b></div>
                    {{else}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMaximum}}</b></div>{{end}}
                    {{if and (ge $cl.MemUsageAverage 0.0) (le $cl.MemUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageAverage}}</b></div>
                    {{else if and (gt $cl.MemUsageAverage 75.0) (le $cl.MemUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageAverage}}</b></div>
                    {{else if (gt $cl.MemUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageAverage}}</b></div>
                    {{else}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageAverage}}</b></div>{{end}}
                    {{if and (ge $cl.MemUsageMinimum 0.0) (le $cl.MemUsageMinimum 75.0)}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMinimum}}</b></div>
                    {{else if and (gt $cl.MemUsageMinimum 75.0) (le $cl.MemUsageMinimum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMinimum}}</b></div>
                    {{else if (gt $cl.MemUsageMinimum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMinimum}}</b></div>
                    {{else}}<div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.MemUsageMinimum}}</b></div>{{end}}
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.DiskUsageMaximum}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.DiskUsageAverage}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.DiskUsageMinimum}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.NetUsageMaximum}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.NetUsageAverage}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.NetUsageMinimum}}</b></div>
                    <div style="clear: both"></div>
                    {{end}}
                </div>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.VMCount 0}}
        <tr>
            <td>
                <div style="width:1440px">
                    <div style="width:220px; float:left">Виртуальные машины кластера</div>
                    <div style="width:120px; float:left; text-align: center">Процессоры / Ядра</div>
                    <div style="width:70px; float:left; text-align: center">Память (Гб)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (ср. %)</div>
                    <div style="width:80px; float:left; text-align: center">Процессор (мин. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (макс. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (ср. %)</div>
                    <div style="width:80px; float:left; text-align: center">Память (мин. %)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (макс. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (ср. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Диск (мин. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (макс. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (ср. Мб/сек)</div>
                    <div style="width:80px; float:left; text-align: center">Сеть (мин. Мб/сек)</div>
                    <div style="clear: both"></div>
                </div>
            </td>
        </tr>
        <tr>
            <td>
                <div style="width:1440px">
                    {{range $idx, $vm := $cl.VM}}
                    <div style="width:20px; float:left; text-align: center">{{if gt $vm.PerfCount 0}}<a href="/sprsrv/vmperfday/{{$vm.Vcenter}}/{{$vm.Datacenter}}/{{$vm.Cluster}}/{{$vm.Name}}" target="_blank"><svg class="bi cluster-activity-icon" aria-hidden="true"><use href="#activity"></use></svg></a>{{else}}<span class="glyphicon glyphicon-off" title="виртуальная машина выключена"></span>{{end}}</div>
                    <div style="width:200px; float:left"><a href="/sprsrv/vmeditbyname/{{$vm.Name}}" target="_blank" title="Общие данные виртуальной машины">{{$vm.Name}}</a></div>
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d" $vm.NumCPU $vm.NumCores}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $vm.MemorySizeGB}}</div>
                    {{if and (ge $vm.CPUUsageMaximum 0.0) (le $vm.CPUUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageMaximum}}</div>
                    {{else if and (gt $vm.CPUUsageMaximum 75.0) (le $vm.CPUUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageMaximum}}</div>
                    {{else if (gt $vm.CPUUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageMaximum}}</div>{{end}}
                    {{if and (ge $vm.CPUUsageAverage 0.0) (le $vm.CPUUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageAverage}}</div>
                    {{else if and (gt $vm.CPUUsageAverage 75.0) (le $vm.CPUUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageAverage}}</div>
                    {{else if (gt $vm.CPUUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageAverage}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageAverage}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPUUsageMinimum}}</div>
                    {{if and (ge $vm.MemUsageMaximum 0.0) (le $vm.MemUsageMaximum 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageMaximum}}</div>
                    {{else if and (gt $vm.MemUsageMaximum 75.0) (le $vm.MemUsageMaximum 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageMaximum}}</div>
                    {{else if (gt $vm.MemUsageMaximum 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageMaximum}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageMaximum}}</div>{{end}}
                    {{if and (ge $vm.MemUsageAverage 0.0) (le $vm.MemUsageAverage 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageAverage}}</div>
                    {{else if and (gt $vm.MemUsageAverage 75.0) (le $vm.MemUsageAverage 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageAverage}}</div>
                    {{else if (gt $vm.MemUsageAverage 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageAverage}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageAverage}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.MemUsageMinimum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.DiskUsageMaximum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.DiskUsageAverage}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.DiskUsageMinimum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.NetUsageMaximum}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.NetUsageAverage}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.NetUsageMinimum}}</div>
                    <div style="clear: both"></div>
                    {{end}}
                </div>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.StorageCount 0}}
        <tr>
            <td>
                <div style="width:1440px">
                    <div style="width:600px; float:left">Хранилища кластера / Виртуальные машины</div>
                    <div style="width:90px; float:left; text-align: center">Ёмкость (Гб)</div>
                    <div style="width:100px; float:left; text-align: center">Свободно (Гб)</div>
                    <div style="width:100px; float:left; text-align: center">Занято (%)</div>
                    <div style="width:100px; float:left; text-align: center">Занято VM (Гб)</div>
                    <div style="width:120px; float:left; text-align: center">Использовано в хранилище (%)</div>
                    <div style="clear: both"></div>
                    {{range $st := $cl.Storage}}
                    <div style="width:600px; float:left">{{$st.StorageName}}</div>
                    <div style="width:90px; float:left; text-align: right">{{printf "%.3f" $st.CapacityGb}}</div>
                    <div style="width:100px; float:left; text-align: right">{{printf "%.3f" $st.FreeSpaceGb}}</div>
                    <div style="width:100px; float:left; text-align: right">{{printf "%2.2f" $st.PercentUsed}}</div>
                    <div style="clear: both"></div>
                    {{range $idx, $vm := $st.VM}}
                    <div style="width:50px; float:left; text-align: center">{{$idx}}</div>
                    <div style="width:820px; float:left">{{$vm.Name}}</div>
                    {{range $j, $usds := $vm.UsageDatastore}}
                    {{if eq $st.StorageName $usds.StorageName}}
                    <div style="width:100px; float:left; text-align: right">{{printf "%.3f" $usds.CommittedGb}}</div>
                    <div style="width:120px; float:left; text-align: right">{{printf "%.3f" $usds.PercentAllocateStorage}}</div>
                    {{end}}
                    {{end}}
                    <div style="clear: both"></div>
                    {{end}}
                    {{end}}
                    {{if gt $cl.StorageCount 1}}
                    <div style="width:600px; float:left"><b>Итого</b></div>
                    <div style="width:90px; float:left; text-align: right"><b>{{printf "%.3f" $cl.AllCapacityGb}}</b></div>
                    <div style="width:100px; float:left; text-align: right"><b>{{printf "%.3f" $cl.AllFreeSpaceGb}}</b></div>
                    <div style="width:100px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.AllPercentUsed}}</b></div>
                    <div style="clear: both"></div>
                    {{end}}
                </div>
            </td>
        </tr>
        {{end}}
    </tbody>
    {{end}}
    <tfoot>
        <tr>
            <td></td>
        </tr>
    </tfoot>
</table>
</div>
</form>
</div>
