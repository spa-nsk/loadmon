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
        <div class="col-6 col-md-3">
            <label for="clid" class="form-label">Кластер</label>
            <select id="clid" class="form-select" onchange="clusterPeriodNavigate()">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $cl := .ClusterList2}}
                <option value="{{$cl}}" {{if eq $.CurCluster $cl}}selected{{end}}>{{$cl}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3 d-flex align-items-end gap-2">
            <div class="flex-grow-1">
                <label for="fromDate" class="form-label">Период с</label>
                <input type="date" id="fromDate" class="form-control" value="{{.FromDate}}">
            </div>
            <div class="flex-grow-1">
                <label for="toDate" class="form-label">по</label>
                <input type="date" id="toDate" class="form-control" value="{{.ToDate}}">
            </div>
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="clusterPeriodNavigate()">Найти</button>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-end">
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" id="clusterPeriodFilterStickyToggle" onchange="clusterPeriodToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted" for="clusterPeriodFilterStickyToggle">Закреплять фильтр вверху экрана</label>
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
                    <div style="width:90px; float:left; text-align: center">Диски всего (Гб)</div>
                    <div style="width:100px; float:left; text-align: center">Диски свободно (Гб)</div>
                    <div style="width:80px; float:left; text-align: center">Диски занято (%)</div>
                    <div style="width:80px; float:left; text-align: center">Записей за период</div>
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
                    <div style="width:400px; float:left">{{$cl.Vcenter}} / {{$cl.Datacenter}} / {{$cl.Name}}</div>
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d (%d)" $cl.AllNumCpuPkgs $cl.AllNumCpuCores $cl.CpuMhzAvg}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $cl.AllMemorySizeGB}}</div>
                    {{if and (ge $cl.CPU.Max 0.0) (le $cl.CPU.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPU.Max}}</div>
                    {{else if and (gt $cl.CPU.Max 75.0) (le $cl.CPU.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPU.Max}}</div>
                    {{else if (gt $cl.CPU.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPU.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.CPU.Max}}</div>{{end}}
                    {{if and (ge $cl.Mem.Max 0.0) (le $cl.Mem.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.Mem.Max}}</div>
                    {{else if and (gt $cl.Mem.Max 75.0) (le $cl.Mem.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.Mem.Max}}</div>
                    {{else if (gt $cl.Mem.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.Mem.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.Mem.Max}}</div>{{end}}
                    <div style="width:90px; float:left; text-align: right">{{printf "%.3f" $cl.AllCapacityGb}}</div>
                    <div style="width:100px; float:left; text-align: right">{{printf "%.3f" $cl.AllFreeSpaceGb}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $cl.AllPercentUsed}}</div>
                    <div style="width:80px; float:left; text-align: right">{{$cl.RecordCount}}</div>
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
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d (%d)" $hi.NumCpuPkgs $hi.NumCpuCores $hi.CpuMhz}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $hi.MemorySizeGB}}</div>
                    {{if and (ge $hi.CPU.Max 0.0) (le $hi.CPU.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Max}}</div>
                    {{else if and (gt $hi.CPU.Max 75.0) (le $hi.CPU.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Max}}</div>
                    {{else if (gt $hi.CPU.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Max}}</div>{{end}}
                    {{if and (ge $hi.CPU.Avg 0.0) (le $hi.CPU.Avg 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Avg}}</div>
                    {{else if and (gt $hi.CPU.Avg 75.0) (le $hi.CPU.Avg 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Avg}}</div>
                    {{else if (gt $hi.CPU.Avg 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Avg}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Avg}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.CPU.Min}}</div>
                    {{if and (ge $hi.Mem.Max 0.0) (le $hi.Mem.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Max}}</div>
                    {{else if and (gt $hi.Mem.Max 75.0) (le $hi.Mem.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Max}}</div>
                    {{else if (gt $hi.Mem.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Max}}</div>{{end}}
                    {{if and (ge $hi.Mem.Avg 0.0) (le $hi.Mem.Avg 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Avg}}</div>
                    {{else if and (gt $hi.Mem.Avg 75.0) (le $hi.Mem.Avg 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Avg}}</div>
                    {{else if (gt $hi.Mem.Avg 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Avg}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Avg}}</div>{{end}}
                    {{if and (ge $hi.Mem.Min 0.0) (le $hi.Mem.Min 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Min}}</div>
                    {{else if and (gt $hi.Mem.Min 75.0) (le $hi.Mem.Min 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Min}}</div>
                    {{else if (gt $hi.Mem.Min 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Min}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $hi.Mem.Min}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Disk.Max}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Disk.Avg}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Disk.Min}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Net.Max}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Net.Avg}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $hi.Net.Min}}</div>
                    <div style="clear: both"></div>
                    {{end}}
                    {{if gt $cl.HostCount 1}}
                    <div style="width:220px; float:left"><b>Итого (по кластеру)</b></div>
                    <div style="width:120px; float:left; text-align: center"><b>{{printf "%d/%d (%d)" $cl.AllNumCpuPkgs $cl.AllNumCpuCores $cl.CpuMhzAvg}}</b></div>
                    <div style="width:70px; float:left; text-align: center"><b>{{printf "%.0f" $cl.AllMemorySizeGB}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPU.Max}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPU.Avg}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.CPU.Min}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.Mem.Max}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.Mem.Avg}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%2.2f" $cl.Mem.Min}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Disk.Max}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Disk.Avg}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Disk.Min}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Net.Max}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Net.Avg}}</b></div>
                    <div style="width:80px; float:left; text-align: right"><b>{{printf "%.3f" $cl.Net.Min}}</b></div>
                    <div style="clear: both"></div>
                    {{end}}
                </div>
            </td>
        </tr>
        {{end}}
        {{if gt $cl.VMCount 0}}
        {{if $cl.VMDataMissing}}
        <tr>
            <td>
                <div class="text-muted fst-italic py-1">Виртуальные машины кластера ({{$cl.VMCount}} шт.) - за выбранный период нет сохранённых данных в БД (vm_record). Попробуйте расширить диапазон дат.</div>
            </td>
        </tr>
        {{else}}
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
                    <div style="width:220px; float:left">{{if ne $vm.Name ""}}<a href="/sprsrv/vmeditbyname/{{$vm.Name}}" target="_blank" title="Общие данные виртуальной машины">{{$vm.Name}}</a>{{else}}&mdash;{{end}}</div>
                    <div style="width:120px; float:left; text-align: center">{{printf "%d/%d" $vm.NumCPU $vm.NumCores}}</div>
                    <div style="width:70px; float:left; text-align: center">{{printf "%.0f" $vm.MemorySizeGB}}</div>
                    {{if and (ge $vm.CPU.Max 0.0) (le $vm.CPU.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Max}}</div>
                    {{else if and (gt $vm.CPU.Max 75.0) (le $vm.CPU.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Max}}</div>
                    {{else if (gt $vm.CPU.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Max}}</div>{{end}}
                    {{if and (ge $vm.CPU.Avg 0.0) (le $vm.CPU.Avg 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Avg}}</div>
                    {{else if and (gt $vm.CPU.Avg 75.0) (le $vm.CPU.Avg 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Avg}}</div>
                    {{else if (gt $vm.CPU.Avg 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Avg}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Avg}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.CPU.Min}}</div>
                    {{if and (ge $vm.Mem.Max 0.0) (le $vm.Mem.Max 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Max}}</div>
                    {{else if and (gt $vm.Mem.Max 75.0) (le $vm.Mem.Max 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Max}}</div>
                    {{else if (gt $vm.Mem.Max 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Max}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Max}}</div>{{end}}
                    {{if and (ge $vm.Mem.Avg 0.0) (le $vm.Mem.Avg 75.0)}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Avg}}</div>
                    {{else if and (gt $vm.Mem.Avg 75.0) (le $vm.Mem.Avg 90.0)}}<div style="background-color: #ffff00; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Avg}}</div>
                    {{else if (gt $vm.Mem.Avg 90.0)}}<div style="background-color: #fc2824; width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Avg}}</div>
                    {{else}}<div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Avg}}</div>{{end}}
                    <div style="width:80px; float:left; text-align: right">{{printf "%2.2f" $vm.Mem.Min}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Disk.Max}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Disk.Avg}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Disk.Min}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Net.Max}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Net.Avg}}</div>
                    <div style="width:80px; float:left; text-align: right">{{printf "%.3f" $vm.Net.Min}}</div>
                    <div style="clear: both"></div>
                    {{end}}
                </div>
            </td>
        </tr>
        {{end}}
        {{end}}
    </tbody>
    {{else}}
    <tbody>
        <tr>
            <td class="text-muted py-3">По выбранным условиям фильтра и периоду данные в БД не найдены.</td>
        </tr>
    </tbody>
    {{end}}
    <tfoot>
        <tr>
            <td></td>
        </tr>
    </tfoot>
</table>
</div>
</div>
