<script>
// Смена значения любого из полей фильтра (vcenter/датацентр/кластер/период)
// сразу перезагружает страницу с новыми GET-параметрами - без отдельной
// кнопки "Применить" для селектов, аналогично /sprsrv/clusterperioddb.
function vmMetricsExportNavigate() {
    var vc = document.getElementById('vcid').value || 'all';
    var dc = document.getElementById('dcid').value || 'all';
    var cl = document.getElementById('clid').value || 'all';
    var from = document.getElementById('fromDate').value;
    var to = document.getElementById('toDate').value;
    var enc = encodeURIComponent;
    window.location.href = '/sprsrv/vmmetricsexport?vcenter=' + enc(vc) + '&datacenter=' + enc(dc) + '&cluster=' + enc(cl) + '&from=' + enc(from) + '&to=' + enc(to);
}

// Экспорт (CSV/XLSX) - по всей отфильтрованной выборке за период, а не
// только по текущей странице предпросмотра (kontext/06-list-page-pattern.md, п.5).
function vmMetricsExportDownload(format) {
    var vc = document.getElementById('vcid').value || 'all';
    var dc = document.getElementById('dcid').value || 'all';
    var cl = document.getElementById('clid').value || 'all';
    var from = document.getElementById('fromDate').value;
    var to = document.getElementById('toDate').value;
    var enc = encodeURIComponent;
    window.location.href = '/sprsrv/vmmetricsexport?vcenter=' + enc(vc) + '&datacenter=' + enc(dc) + '&cluster=' + enc(cl) + '&from=' + enc(from) + '&to=' + enc(to) + '&export=' + enc(format);
}

function vmMetricsExportPagerGoTo(page) {
    var url = new URL(window.location.href);
    url.searchParams.set('page', page);
    window.location.href = url.toString();
}

function vmMetricsExportPagerChangePageSize(size) {
    var url = new URL(window.location.href);
    url.searchParams.set('pagesize', size);
    url.searchParams.set('page', '1');
    window.location.href = url.toString();
}
</script>

<div class="container-fluid d-grid gap-2">
<div class="hero-text">
    <h3>Выгрузка метрик виртуальных машин за период</h3>
    <div class="text-muted small">
        Данные посчитаны по суточным записям в таблице vm_record (заполняются ежесуточно),
        дополнены в начале колонками Vcenter/Datacenter/Cluster/Vm_Name из связанной записи
        справочника vm_spr_record, а также ITSystemKey/TypeWork/TagName/TagDescription из
        справочника серверов server_spr_records (по связи vm_record.ServerId). Экспорт
        (CSV/XLSX) выгружает весь набор по текущему фильтру и периоду, а не только показанную
        на экране страницу.
    </div>
    {{if .flash.error}}
    <blockquote>{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote>{{.flash.notice}}</blockquote>
    {{end}}
</div>

<div class="vm-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3">
        <div class="col-6 col-md-3">
            <label for="vcid" class="form-label">Сервер управления</label>
            <select id="vcid" class="form-select" onchange="vmMetricsExportNavigate()">
                <option value="all" {{if eq .CurVcenter "all"}}selected{{end}}>Все</option>
                {{range $vc := .VcenterList}}
                <option value="{{$vc}}" {{if eq $.CurVcenter $vc}}selected{{end}}>{{$vc}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-3">
            <label for="dcid" class="form-label">Датацентр</label>
            <select id="dcid" class="form-select" onchange="vmMetricsExportNavigate()">
                <option value="all" {{if eq .CurDatacenter "all"}}selected{{end}}>Все</option>
                {{range $dc := .DatacenterList}}
                <option value="{{$dc}}" {{if eq $.CurDatacenter $dc}}selected{{end}}>{{$dc}}</option>
                {{end}}
            </select>
        </div>
        <div class="col-6 col-md-2">
            <label for="clid" class="form-label">Кластер</label>
            <select id="clid" class="form-select" onchange="vmMetricsExportNavigate()">
                <option value="all" {{if eq .CurCluster "all"}}selected{{end}}>Все</option>
                {{range $cl := .ClusterList}}
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
            <div class="d-flex flex-wrap gap-2">
                <button type="button" class="btn btn-primary rounded-pill px-3" onclick="vmMetricsExportNavigate()">Найти</button>
                <button type="button" class="btn btn-outline-success rounded-pill px-3" onclick="vmMetricsExportDownload('csv')"
                    title="Выгрузить весь отфильтрованный набор за период в CSV">Экспорт в CSV</button>
                <button type="button" class="btn btn-outline-success rounded-pill px-3" onclick="vmMetricsExportDownload('xlsx')"
                    title="Выгрузить весь отфильтрованный набор за период в XLSX">Экспорт в XLSX</button>
            </div>
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" id="vmMetricsExportFilterStickyToggle" onchange="vmMetricsExportToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted" for="vmMetricsExportFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
</div>

<script>
// Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
// приём идентичен /sprsrv/clusterperioddb (сохраняется отдельным ключом в
// localStorage). Скрипт размещён здесь, ПОСЛЕ .vm-filter-bar в разметке -
// document.querySelector должен найти уже существующий в DOM элемент (см.
// такое же размещение в /sprsrv/hwviewlist.tpl); если разместить его раньше
// панели фильтра, bar/toggle будут пойманы как null безвозвратно, даже
// внутри обработчика DOMContentLoaded.
(function () {
    var STORAGE_KEY = 'vmmetricsexport.filterSticky';
    var bar = document.querySelector('.vm-filter-bar');
    var toggle = document.getElementById('vmMetricsExportFilterStickyToggle');
    if (!bar || !toggle) return;

    function updateTheadOffset() {
        var offset = 0;
        if (!bar.classList.contains('vm-filter-bar-unpinned')) {
            offset = bar.offsetHeight;
        }
        document.documentElement.style.setProperty('--spr-thead-top', offset + 'px');
    }

    function applySticky(sticky) {
        bar.classList.toggle('vm-filter-bar-unpinned', !sticky);
        toggle.checked = sticky;
        updateTheadOffset();
    }

    var stored = null;
    try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
    applySticky(stored === null ? true : stored === '1');

    window.vmMetricsExportToggleFilterSticky = function (sticky) {
        applySticky(sticky);
        try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
    };

    window.addEventListener('resize', updateTheadOffset);
    window.addEventListener('load', updateTheadOffset);
})();
</script>

{{if .Rows}}
<!-- Без обёртки overflow-x:auto у таблицы - см. пояснение в
     views/admin/viewexportlist.tpl: единственный скролл-контейнер страницы -
     .content-panel (header.tpl), и обёртка вокруг таблицы ломает
     position:sticky у thead и у панели фильтра выше. -->
<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th>Vcenter</th>
            <th>Datacenter</th>
            <th>Cluster</th>
            <th>Vm_Name</th>
            <th>ITSystemKey</th>
            <th>TypeWork</th>
            <th>TagName</th>
            <th>TagDescription</th>
            <th>Дата</th>
            <th class="text-end">VM DISK (Gb)</th>
            <th class="text-end">VM DISK Free (Gb)</th>
            <th class="text-end">VM DISK Used %</th>
            <th class="text-end">VM CPU max %</th>
            <th class="text-end">VM CPU avg %</th>
            <th class="text-end">VM CPU min %</th>
            <th class="text-end">VM MEM max %</th>
            <th class="text-end">VM MEM avg %</th>
            <th class="text-end">VM MEM min %</th>
            <th class="text-end">VM DISK Usage max %</th>
            <th class="text-end">VM DISK Usage avg %</th>
            <th class="text-end">VM DISK Usage min %</th>
            <th class="text-end">VM CPU Count</th>
            <th class="text-end">VM CPU Cores</th>
            <th class="text-end">VM Memory (GB)</th>
            <th>VM IP Address</th>
            <th>VM Network</th>
            <th class="text-end">VM Ethernet Cards</th>
            <th class="text-end">VM Virtual Disks</th>
            <th class="text-end">VM Disk Committed (GB)</th>
            <th class="text-end">VM Disk Uncommitted (GB)</th>
            <th class="text-end">VM DISK Read Latency max (ms)</th>
            <th class="text-end">VM DISK Read Latency avg (ms)</th>
            <th class="text-end">VM DISK Read Latency min (ms)</th>
            <th class="text-end">VM DISK Write Latency max (ms)</th>
            <th class="text-end">VM DISK Write Latency avg (ms)</th>
            <th class="text-end">VM DISK Write Latency min (ms)</th>
            <th class="text-end">VM DISK Write max (KB/s)</th>
            <th class="text-end">VM DISK Write avg (KB/s)</th>
            <th class="text-end">VM DISK Write min (KB/s)</th>
            <th class="text-end">VM DISK Read max (KB/s)</th>
            <th class="text-end">VM DISK Read avg (KB/s)</th>
            <th class="text-end">VM DISK Read min (KB/s)</th>
            <th class="text-end">VM NET Usage max (KB/s)</th>
            <th class="text-end">VM NET Usage avg (KB/s)</th>
            <th class="text-end">VM NET Usage min (KB/s)</th>
            <th class="text-end">VM CPU Billing (MHz*h)</th>
            <th class="text-end">VM MEM Billing (MB*h)</th>
            <th class="text-end">VM DISK Billing (GB*h)</th>
            <th class="text-end">VM NET Billing (MB*h)</th>
        </tr>
    </thead>
    <tbody>
        {{range $r := .Rows}}
        <tr>
            <td>{{$r.Vcenter}}</td>
            <td>{{$r.Datacenter}}</td>
            <td>{{$r.Cluster}}</td>
            <td>{{$r.Vm_Name}}</td>
            <td>{{$r.ITSystemKey}}</td>
            <td>{{$r.TypeWork}}</td>
            <td>{{$r.TagName}}</td>
            <td>{{$r.TagDescription}}</td>
            <td>{{$r.Date.Format "2006-01-02"}}</td>
            <td class="text-end">{{printf "%.1f" $r.Vm_CapacityGb}}</td>
            <td class="text-end">{{printf "%.1f" $r.Vm_FreeSpaceGb}}</td>
            <td class="text-end">{{printf "%.2f" $r.Vm_PercentUsed}}</td>
            <td class="text-end">{{printf "%.2f" $r.Cpu_usage_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Cpu_usage_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Cpu_usage_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Mem_usage_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Mem_usage_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Mem_usage_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_usage_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_usage_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_usage_minimum}}</td>
            <td class="text-end">{{$r.Vm_NumCPU}}</td>
            <td class="text-end">{{$r.Vm_NumCores}}</td>
            <td class="text-end">{{printf "%.1f" $r.Vm_MemorySizeGB}}</td>
            <td>{{$r.Vm_IpAddress}}</td>
            <td>{{$r.Vm_Network}}</td>
            <td class="text-end">{{$r.Vm_NumEthernetCards}}</td>
            <td class="text-end">{{$r.Vm_NumVirtualDisks}}</td>
            <td class="text-end">{{printf "%.1f" $r.Vm_DiskCommittedGB}}</td>
            <td class="text-end">{{printf "%.1f" $r.Vm_DiskUncommittedGB}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalReadLatency_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalReadLatency_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalReadLatency_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalWriteLatency_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalWriteLatency_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_totalWriteLatency_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_write_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_write_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_write_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_read_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_read_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Disk_read_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Net_usage_maximum}}</td>
            <td class="text-end">{{printf "%.2f" $r.Net_usage_average}}</td>
            <td class="text-end">{{printf "%.2f" $r.Net_usage_minimum}}</td>
            <td class="text-end">{{printf "%.2f" $r.CpuBilling}}</td>
            <td class="text-end">{{printf "%.2f" $r.MemBilling}}</td>
            <td class="text-end">{{printf "%.2f" $r.DiskBilling}}</td>
            <td class="text-end">{{printf "%.2f" $r.NetBilling}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div class="pager-bar pager-bar-sticky d-flex justify-content-between align-items-center mt-2">
    <span class="text-muted small">Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.RowCount}}</span>
    <div class="d-flex align-items-center gap-2 pager-nav">
        <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize"
            style="width:auto;" onchange="vmMetricsExportPagerChangePageSize(this.value)">
            <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
            <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
            <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
        </select>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmMetricsExportPagerGoTo(1)"
            {{if eq .Page 1}}disabled{{end}}>&laquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmMetricsExportPagerGoTo({{.Page}}-1)"
            {{if eq .Page 1}}disabled{{end}}>&lsaquo;</button>
        <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmMetricsExportPagerGoTo({{.Page}}+1)"
            {{if eq .Page .TotalPages}}disabled{{end}}>&rsaquo;</button>
        <button type="button" class="btn btn-outline-secondary btn-sm" onclick="vmMetricsExportPagerGoTo({{.TotalPages}})"
            {{if eq .Page .TotalPages}}disabled{{end}}>&raquo;</button>
    </div>
</div>
{{else}}
<p class="text-muted">По выбранным условиям фильтра и периоду данные в БД не найдены.</p>
{{end}}
</div>
