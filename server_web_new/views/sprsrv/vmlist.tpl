<div class="hero-text mb-3">
    <h2>Список виртуальных машин</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form id="vmFilterForm" class="vm-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="filterVcenter" class="form-label">vCenter</label>
            <select id="filterVcenter" class="form-select" onchange="vmFilterNavigate('vcenter')">
                <option value="all" {{if eq .CurVcenter "all"}}selected{{end}}>Все</option>
                {{range $v := .VCenterList}}
                <option value="{{$v}}" {{if eq $.CurVcenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="filterDatacenter" class="form-label">Датацентр</label>
            <select id="filterDatacenter" class="form-select" onchange="vmFilterNavigate('datacenter')">
                <option value="all" {{if eq .CurDatacenter "all"}}selected{{end}}>Все</option>
                {{range $v := .DatacenterList}}
                <option value="{{$v}}" {{if eq $.CurDatacenter $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="filterCluster" class="form-label">Кластер</label>
            <select id="filterCluster" class="form-select" onchange="vmFilterNavigate('cluster')">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $v := .ClusterList}}
                <option value="{{$v}}" {{if eq $.CurCluster $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="filterVM" class="form-label">Виртуальная машина</label>
            <select id="filterVM" class="form-select" onchange="vmFilterNavigate('vm')">
                <option value="all" {{if eq .CurVM "all"}}selected{{end}}>Все</option>
                {{range $v := .VMNameList}}
                <option value="{{$v}}" {{if eq $.CurVM $v}}selected{{end}}>{{$v}}</option>
                {{end}}
            </select>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex justify-content-between align-items-center">
            <span class="text-muted small">Найдено виртуальных машин: {{.VMCount}}</span>
            <a href="/sprsrv/vm/all/all/all/all" class="btn btn-outline-secondary btn-sm rounded-pill px-3">Сбросить фильтр</a>
        </div>
    </div>
</form>

<div class="table-responsive">
    <table class="table table-hover table-sm align-middle vm-list-table">
        <thead>
            <tr>
                <th>Имя ВМ</th>
                <th class="text-center">Питание</th>
                <th>vCenter</th>
                <th>Датацентр</th>
                <th>Кластер</th>
                <th>Хост</th>
                <th>Гостевая ОС</th>
                <th>IP адрес</th>
                <th class="text-end">vCPU</th>
                <th class="text-end">Память, ГБ</th>
                <th class="text-end">CPU средн., %</th>
                <th class="text-end">Память средн., %</th>
                <th>VMware Tools</th>
            </tr>
        </thead>
        <tbody>
            {{range $vm := .VMList}}
            <tr>
                <td class="fw-semibold">{{$vm.Name}}</td>
                <td class="text-center">
                    {{if $vm.PowerState}}
                    <span class="badge rounded-pill text-bg-success">Включена</span>
                    {{else}}
                    <span class="badge rounded-pill text-bg-secondary">Выключена</span>
                    {{end}}
                </td>
                <td>{{$vm.Vcenter}}</td>
                <td>{{$vm.Datacenter}}</td>
                <td>{{$vm.Cluster}}</td>
                <td>{{$vm.HostName}}</td>
                <td>{{$vm.GuestFullName}}</td>
                <td>{{$vm.IPAddress}}</td>
                <td class="text-end">{{$vm.NumCPU}}</td>
                <td class="text-end">{{printf "%.1f" $vm.MemorySizeGB}}</td>
                <td class="text-end">{{printf "%.1f" $vm.CPUUsageAverage}}</td>
                <td class="text-end">{{printf "%.1f" $vm.MemUsageAverage}}</td>
                <td>
                    {{if eq $vm.ToolsStatus "toolsOk"}}
                    <span class="badge text-bg-success">{{$vm.ToolsStatus}}</span>
                    {{else if eq $vm.ToolsStatus ""}}
                    <span class="text-muted">&mdash;</span>
                    {{else}}
                    <span class="badge text-bg-warning">{{$vm.ToolsStatus}}</span>
                    {{end}}
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="13" class="text-center text-muted py-4">По выбранным условиям фильтра виртуальные машины не найдены</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<style>
    /* фильтр закреплён вверху списка и не уезжает при прокрутке таблицы */
    .vm-filter-bar {
        position: sticky;
        top: 0;
        z-index: 5;
        background-color: var(--bs-body-bg);
        border-color: var(--bs-border-color) !important;
        box-shadow: 0 2px 6px rgba(0, 0, 0, .08);
    }

    .vm-list-table thead th {
        position: sticky;
        top: calc(2px + var(--vm-filter-bar-height, 0px));
        background-color: var(--bs-body-bg);
        z-index: 4;
    }
</style>

<script>
    function vmFilterNavigate(changed) {
        var vc = document.getElementById('filterVcenter').value || 'all';
        var dc = document.getElementById('filterDatacenter').value || 'all';
        var cl = document.getElementById('filterCluster').value || 'all';
        var vm = document.getElementById('filterVM').value || 'all';

        // при смене вышестоящего уровня фильтра нижестоящие сбрасываются в "Все",
        // так как их старые значения могли относиться к другому vcenter/датацентру/кластеру
        if (changed === 'vcenter') { dc = 'all'; cl = 'all'; vm = 'all'; }
        if (changed === 'datacenter') { cl = 'all'; vm = 'all'; }
        if (changed === 'cluster') { vm = 'all'; }

        var enc = encodeURIComponent;
        window.location.href = '/sprsrv/vm/' + enc(vc) + '/' + enc(dc) + '/' + enc(cl) + '/' + enc(vm);
    }
</script>
