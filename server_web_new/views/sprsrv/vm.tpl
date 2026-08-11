<script>
function selectvcenter(val) {
    //document.getElementById('itSystem')
    //alert("Выбрана ИТ система" + val)
    document.getElementById("Update").submit();
}

function selectdatacenter(val) {
    //document.getElementById('itSystem')
    //alert("Выбрана ИТ система" + val)
    document.getElementById("Update").submit();
}

function selectcluster(val) {
    //document.getElementById('itSystem')
    //alert("Выбрана ИТ система" + val)
    document.getElementById("Update").submit();
}

$(document).ready(function () {
    $("#myInput").on("keyup", function () {
        var value = $(this).val().toLowerCase();
        $("#myTable tr").filter(function () {
            $(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
        });
    });
});
</script>

<div class="container-fluid d-grid gap-2" style="width:100%; overflow:scroll">
<form role="form" id="Update" method="POST">
    <div class="hero-text">
        <h3>Список виртуальных машин</h3>
        {{if .flash.error}}
        <blockquote>{{.flash.error}}</blockquote>
        {{end}}

        {{if .flash.notice}}
        <blockquote>{{.flash.notice}}</blockquote>
        {{end}}
    </div>
    <div class="row">
        <div class="form-group col-sm-6">
            <label for="vcenterId">Сервер управления</label>
            <select id="vcid" name="vcenterId" size="1" class="form-control" onchange="selectvcenter(this.value)" value="{{.vcenterId}}">
                <option value="all" {{if eq .vcenterId "all" }}selected{{end}}>Все</option>
                {{range $vc := .VCenterList}}
                <option value="{{$vc.Hostname}}" {{if eq $vc.Hostname $.vcenterId}}selected{{end}}>{{$vc.Hostname}}</option>
                {{end}}
            </select>
        </div>
        <div class="form-group col-sm-6">
            <label for="datacenterId">Датацентр</label>
            <select id="dcid" name="datacenterId" size="1" class="form-control" onchange="selectdatacenter(this.value)" value="{{.datacenterId}}">
                <option value="all" {{if eq .datacenterId "all" }}selected{{end}}>Все</option>
                {{range $dc := .DatacenterList}}
                <option value="{{$dc.Name}}" {{if eq $dc.Name $.datacenterId}}selected{{end}}>{{$dc.Name}}</option>
                {{end}}
            </select>
        </div>
    </div>

    <div class="row">
        <div class="form-group col-sm-6">
            <label for="cluserId">Кластер</label>
            <select id="clid" name="clusterId" size="1" class="form-control" onchange="selectcluster(this.value)" value="{{.clusterId}}">
                <option value="all" {{if eq .clusterId "all" }}selected{{end}}>Все</option>
                {{range $cli := .ClusterList}}
                <option value="{{$cli.Name}}" {{if eq $cli.Name $.clusterId}}selected{{end}}>{{$cli.Name}}</option>
                {{end}}
            </select>
        </div>
        <div class="form-group col-sm-6">
            <label for="vmnameId">Виртуальная машина</label>
            <input id="vmid" name="vmnameId" class="form-control" type="text" value="{{.vmnameId}}" />
        </div>
    </div>

    <div class="row">
        <div class="form-group col-sm-2">
            <input name="Find" type="submit" title="Найти" value="Найти" class="form-control btn btn-sm btn-default btn-outline-secondary rounded-pill" />
        </div>
    </div>

    <div class="row">
{{if eq .ViewOn 1 }}
        <table class="table">
        <thead>
        <tr>
        <th style="text-align: center;width: 40px;">Id</th>
        <th style="text-align: center;width: 120px;">VCenter/Datacenter/Cluster/Имя</th>
        <th style="text-align: center;width: 70px;">Процессоры / Ядра</th>
        <th style="text-align: center;width: 70px;">Память (GB)</th>
        <th style="text-align: center;width: 120px;">Сеть</th>
        <th style="text-align: center;width: 120px;">IP адрес</th>
        <th style="text-align: center;width: 350px;">Диски (путь, размер, занято (%), свободно)</th>
        <th></th>
        </tr>
        </thead>
        <tbody id="myTable">
        {{range $index, $vm := .VMList}}
        <tr>
        <td style="text-align: right">{{$vm.Id}}</td>
        <td style="word-wrap: break-word; width: 200px;">
        {{if $vm.Vm_PowerState}}<span style="color:#00b300">{{end}}
        <b>{{$vm.Vcenter}}</b> / <b>{{$vm.Datacenter}}</b> / <b>{{$vm.Cluster}}</b> / <b>{{$vm.Vm_Name}}</b> ({{$vm.Vm_HostName}})
        {{if $vm.WorkStation}}(Рабочая станция Windows){{end}}
        {{if $vm.Vm_PowerState}}</span>{{end}}
        </td>
        {{if and (ge $vm.Cpu_usage_maximum 0.0) (le $vm.Cpu_usage_maximum 75.00)}} <td style="text-align: right">
        {{$vm.Vm_NumCPU}} / {{$vm.Vm_NumCores}} ({{printf "%.0f" $vm.Cpu_usage_maximum}}%)</td>
        {{ else if and (gt $vm.Cpu_usage_maximum 75.00) (le $vm.Cpu_usage_maximum 90.00)}} <td
        style="background-color: #ffff00; text-align: right">{{$vm.Vm_NumCPU}} / {{$vm.Vm_NumCores}} ({{printf
        "%.0f"
        $vm.Cpu_usage_maximum}}%) </td>
        {{ else if (gt $vm.Cpu_usage_maximum 90.00 ) }} <td style="background-color: #fc2824; text-align: right">
        {{$vm.Vm_NumCPU}} / {{$vm.Vm_NumCores}} ({{printf "%.0f" $vm.Cpu_usage_maximum}}%) </td>
        {{ else }} <td style="text-align: right"> {{$vm.Vm_NumCPU}} / {{$vm.Vm_NumCores}} ({{printf "%.0f"
        $vm.Cpu_usage_maximum}}%)</td> {{end}}

        {{if and (ge $vm.Mem_usage_maximum 0.0) (le $vm.Mem_usage_maximum 75.00)}} <td style="text-align: right">
        {{printf "%.0f" $vm.Vm_MemorySizeGB}} ({{printf "%.0f" $vm.Mem_usage_maximum}}%)</td>
        {{ else if and (gt $vm.Mem_usage_maximum 75.00) (le $vm.Mem_usage_maximum 90.00)}} <td
        style="background-color: #ffff00; text-align: right">{{printf "%.0f" $vm.Vm_MemorySizeGB}} ({{printf
        "%.0f"
        $vm.Mem_usage_maximum}}%) </td>
        {{ else if (gt $vm.Mem_usage_maximum 90.00 ) }} <td style="background-color: #fc2824; text-align: right">
        {{printf "%.0f" $vm.Vm_MemorySizeGB}} ({{printf "%.0f" $vm.Mem_usage_maximum}}%) </td>
        {{ else }} <td style="text-align: right"> {{printf "%.0f" $vm.Vm_MemorySizeGB}} ({{printf "%.0f"
        $vm.Mem_usage_maximum}}%)</td> {{end}}

        <td ><div style="text-align: left;width:120px;word-wrap: break-word;">{{$vm.Vm_Network}}</div> </td>
        <td ><div style="text-align: left;width:120px;word-wrap: break-word;">{{$vm.Vm_IpAddress}}</div> </td>

        <td style="text-align: left;vertical-align:top;">
          <div style="width:600px;">
          Путь к виртуальной машине <br>{{printf "%s" $vm.Vm_PathName}}
          {{if gt $vm.Vm_CapacityGb 0.001 }}
          <table>
            <tr>
              <th>
                <div style="width:200px;word-wrap: break-word;"><b>все диски</b></div>
              </th>
              <th style="text-align: right;">
                <div style="width:80px;">{{printf "%.3fGb" $vm.Vm_CapacityGb}}</div>
              </th>
              {{if gt $vm.Vm_PercentUsed 90.0}}
              <th style="background-color: #fc2824; text-align: right;">
                <div style="width:60px;">{{printf "%2.2f" $vm.Vm_PercentUsed}}%</div>
              </th>
              {{else if lt $vm.Vm_PercentUsed 1.0}}
              <th style="background-color: #00ffff; text-align: right;">
                <div style="width:60px;">{{printf "%2.2f" $vm.Vm_PercentUsed}}%</div>
              </th>
              {{else}}
              <th style="text-align: right;">
                <div style="width:60px;">{{printf "%2.2f" $vm.Vm_PercentUsed}}%</div>
              </th>
              {{end}}
              <th style="text-align: right;">
                <div style="width:80px;">{{printf "%.3fGb" $vm.Vm_FreeSpaceGb}}</div>
              </th>
            </tr>
            <br>{{range $dsk := $vm.Vm_Disk}}
            <tr>
              <td>
                <div style="width:200px;word-wrap: break-word;"><b>{{$dsk.DiskPath}}</b></div>
              </td>
              <td style="text-align: right;">
                <div style="width:80px;">{{printf "%.3fGb" $dsk.CapacityGb}}</div>
              </td>

              {{if gt $dsk.PercentUsed 90.0}}
              <td style="background-color: #fc2824; text-align: right;">
                <div style="width:60px;">{{printf "%.2f" $dsk.PercentUsed}}%</div>
              </td>
              {{else if lt $dsk.PercentUsed 1.0}}
              <td style="background-color: #00ffff; text-align: right;">
                <div style="width:60px;">{{printf "%.2f" $dsk.PercentUsed}}%</div>
              </th>
              {{else}}
              <td style="text-align: right;">
                <div style="width:60px;">{{printf "%.2f" $dsk.PercentUsed}}%</div>
              </td>
              {{end}}

              <td style="text-align: right;">
                <div style="width:80px;">{{printf "%.3fGb" $dsk.FreeSpaceGb}}</div>
              </td>
            </tr>
            {{end}}

            {{if $vm.UsageDatastore}}
            <tr>
                <td style="text-align: center;">
                    <div style="width:200px;word-wrap: break-word;"><b>Хранилища и RDM диски</b></div>
                </td>
                <td style="text-align: center;">
                    <div style="width:80px;"><b>Выделено</b></div>
                </td>
                <td style="text-align: center;">
                    <div style="width:60px;"><b>% от общего</b></div>
                </td>
                <td>
                    <div style="width:60px;"></div>
                </td>
            </tr>
            {{end}}

            {{range $rdm := $vm.UsageDatastore}}
            <tr>
              {{if gt $rdm.PercentAllocateStorage 0.0}}
              <td> <b>{{$rdm.StorageName}}</b></td>
              {{else}}
              <td> <b>{{$rdm.StorageName}} {{$rdm.Datastore}}</b></td>
              {{end}}
              <td style="text-align: right;">{{printf "%.3fGb" $rdm.CommittedGb}}</td>
              {{if gt $rdm.PercentAllocateStorage 0.0}}
              <td style="text-align: right;">{{printf "%.2f" $rdm.PercentAllocateStorage}}% </td>
              {{else}}
              <td> </td>
              {{end}}
              <td> </td>
            </tr>
            {{end}}

            {{if $vm.VmFiles}}
            <tr>
              <td style="text-align: center;background-color: #ffff00;">
                <div style="width:200px;word-wrap: break-word;"><b>Снапшоты</b></div>
              </td>
              <td style="text-align: center;background-color: #ffff00;">
                <div style="width:80px;"><b>Размер</b></div>
              </td>
              <td style="text-align: center;">
                <div style="width:60px;"><b></b></div>
              </td>
              <td>
                <div style="width:60px;"></div>
              </td>
            </tr>
            {{end}}
            {{range $vmf := $vm.VmFiles}}
            <tr>
              {{if eq $vmf.Type "snapshotData"}}
              <td style="background-color: #ffff00;">{{$vmf.Name}}</td>
              <td style="background-color: #ffff00;text-align: right;">{{printf "%.3fGb" $vmf.Size}}</td>
              <td> </td>
              <td> </td>
              {{end}}
            </tr>
            {{end}}

          </table>
          {{else}}
            <div style="width:350px;word-wrap: break-word;"><b>Нет данных о структуре дисков в виртуальной машине.
            Выделено {{printf "%.3fGb" $vm.Vm_DiskCommittedGB}}</b></div>
          {{end}}
          </div>
        </td>
        <td>
            <div class="row">
                <a class="btn btn-default btn-sm btn-outline-secondary rounded-pill {{if eq $vm.PerfCount 0}}disabled{{end}}"
                href="/sprsrv/vmperfkey/{{$vm.KeyName}}" target="_blank">Загрузка</a>
            </div>

            {{if eq $.Privilage 3 }}
            <div class="row">
                <a class="btn btn-default btn-sm btn-outline-secondary rounded-pill {{if eq $vm.PerfCount 0}}disabled{{end}}"
                href="/sprsrv/vmperfkeytest/{{$vm.KeyName}}" target="_blank">тест1</a>
            </div>
            <div class="row">
                <a class="btn btn-default btn-sm btn-outline-secondary rounded-pill {{if eq $vm.PerfCount 0}}disabled{{end}}"
                href="/sprsrv/vmdelperfkey/{{$vm.KeyName}}" target="_blank">тест2</a>
            </div>
            {{end}}
            <div class="row">
                <a class="btn btn-default btn-sm btn-outline-secondary rounded-pill {{if eq $vm.PerfCount 0}}disabled{{end}}"
                href="/sprsrv/vmperfperiod/{{$vm.Vcenter}}/{{$vm.Datacenter}}/{{$vm.Cluster}}/{{$vm.Vm_HostName}}"
                target="_blank">Загрузка с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day
                $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d"
                $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>
            </div>
        </td>
        </tr>
        {{end}}
        </tbody>
        <tfoot>
        <tr>
        </tr>
        </tfoot>
        </table>
{{end}}
    </div>
</form>
</div>
