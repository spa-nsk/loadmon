<!-- Контроллер: controllers/admindeactivatedservers.go (AdminDeactivatedServersRestore).
     Стиль формы - по образцу /sprsrv/updateselectedbase
     (views/sprsrv/updateselectedbase.tpl) - список выбранных записей сверху,
     форма применяемых значений снизу. -->

<div class="hero-text mb-3">
    <h2>Возврат в учёт: {{.SelectedCount}} серверов</h2>
    <p class="text-muted">
        Выбранные записи будут обновлены значениями ниже. Stat автоматически станет
        1 (в учёте) только если итоговые PowerState и Active означают "Включен" и
        "Активен" одновременно - иначе запись останется выведенной из учёта с новыми
        значениями PowerState/Active. При фактическом возврате в учёт также
        обновляются связанные записи: у виртуальных машин - признак включения
        (Vm_PowerState), у аппаратных серверов - PowerState/Active и источник
        обновления (SourceUpdateType наследуется от записи справочника серверов).
    </p>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

{{if .SelectedRecords}}
<div class="table-responsive mb-3">
    <table class="table table-sm table-bordered align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Тип</th>
                <th>Имя сервера</th>
                <th>vCenter / Cluster</th>
                <th class="text-center">Текущий PowerState</th>
                <th class="text-center">Текущий Active</th>
                <th class="text-center">Источник (SourceUpdateType)</th>
            </tr>
        </thead>
        <tbody>
            {{range $r := .SelectedRecords}}
            <tr>
                <td class="text-center">{{if eq $r.Type 1}}виртуальный{{else if eq $r.Type 2}}гипервизор{{else}}аппаратный{{end}}</td>
                <td>{{$r.Name}}</td>
                <td class="small text-muted">{{$r.Vcenter}} {{if ne $r.Cluster ""}}/ {{$r.Cluster}}{{end}}</td>
                <td class="text-center">{{if eq $r.PowerState 1}}Включен{{else if eq $r.PowerState 2}}Выключен{{else}}—{{end}}</td>
                <td class="text-center">{{if eq $r.Active 1}}Активен{{else if eq $r.Active 2}}Выведен{{else}}—{{end}}</td>
                <td class="text-center">
                    {{if eq $r.SourceUpdateType 1}}Вручную{{else if eq $r.SourceUpdateType 2}}VMware{{else if eq $r.SourceUpdateType 3}}HPE OneView{{else if eq $r.SourceUpdateType 4}}Zabbix{{else}}не указано{{end}}
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<form role="form" method="POST" class="p-3 rounded border">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-4">
            <label for="PowerState" class="form-label">PowerState</label>
            <select id="PowerState" name="PowerState" class="form-select">
                <option value="0">не менять</option>
                <option value="1" selected>Включен</option>
                <option value="2">Выключен</option>
            </select>
        </div>
        <div class="col-6 col-md-4">
            <label for="Active" class="form-label">Active</label>
            <select id="Active" name="Active" class="form-select">
                <option value="0">не менять</option>
                <option value="1" selected>Активен</option>
                <option value="2">Выведен из эксплуатации</option>
            </select>
        </div>
        <div class="col-12 col-md-4">
            <button type="submit" class="btn btn-success rounded-pill px-3">Применить</button>
            <a href="/admin/deactivatedservers" class="btn btn-outline-secondary rounded-pill px-3">Отмена</a>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12">
            <span class="text-muted small">
                SourceUpdateType записи справочника серверов этой формой не меняется
                (наследуется как есть) - при возврате в учёт связанного аппаратного
                сервера то же значение SourceUpdateType переносится и на него.
            </span>
        </div>
    </div>
</form>
{{else}}
<p class="text-muted">Ни одна из выбранных записей не найдена.</p>
{{end}}
