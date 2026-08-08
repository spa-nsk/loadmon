<script>
$(document).ready(function () {
    $("#myInput").on("keyup", function () {
        var value = $(this).val().toLowerCase();
        $("#myTable tr").filter(function () {
            $(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
        });
    });
});

function setFocusList() {
    document.getElementById("detail{{$.ListId}}").focus();
}

// Переход на страницу N без потери значений фильтра: номер страницы
// пишется в скрытое поле формы "Update", после чего форма отправляется
// тем же POST-ом, что и обычный "Просмотр"
function pagerGoTo(page) {
    var field = document.getElementById('pageField');
    if (!field) return;
    if (page < 1) page = 1;
    field.value = page;
    document.getElementById('Update').submit();
}

// Смена количества записей на странице сбрасывает на 1-ю страницу
function pagerChangePageSize(size) {
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

document.addEventListener('DOMContentLoaded', function () {
    var menu = document.getElementById('rowContextMenu');
    if (!menu) return;

    var table = document.getElementById('myTable');

    function hideMenu() {
        menu.classList.remove('show');
    }

    function showMenuForRow(tr, x, y) {
        var srcId = tr.getAttribute('data-menu-id');
        var src = srcId ? document.getElementById(srcId) : null;
        if (!src) return;

        menu.innerHTML = src.innerHTML;
        menu.style.left = x + 'px';
        menu.style.top = y + 'px';
        menu.classList.add('show');

        // не даём меню выйти за пределы окна
        requestAnimationFrame(function () {
            var rect = menu.getBoundingClientRect();
            var newLeft = x, newTop = y;
            if (rect.right > window.innerWidth) {
                newLeft = Math.max(0, window.innerWidth - rect.width - 4);
            }
            if (rect.bottom > window.innerHeight) {
                newTop = Math.max(0, window.innerHeight - rect.height - 4);
            }
            menu.style.left = newLeft + 'px';
            menu.style.top = newTop + 'px';
        });
    }

    if (table) {
        table.addEventListener('contextmenu', function (e) {
            var tr = e.target.closest('tr.row-context-target');
            if (!tr) return;
            e.preventDefault();
            showMenuForRow(tr, e.clientX, e.clientY);
        });
    }

    document.addEventListener('click', hideMenu);
    document.addEventListener('scroll', hideMenu, true);
    window.addEventListener('resize', hideMenu);
    document.addEventListener('contextmenu', function (e) {
        if (!e.target.closest('tr.row-context-target') && !e.target.closest('#rowContextMenu')) {
            hideMenu();
        }
    });
    menu.addEventListener('click', function (e) {
        // клик по пункту меню (ссылке) не должен сразу же скрываться общим
        // обработчиком click на document раньше перехода по ссылке в новой вкладке
        e.stopPropagation();
    });
});
</script>

<style>
    /* панель постраничной навигации */
    .pager-bar {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        padding: 0.5rem 0.75rem;
    }

    /* блок пагинации закреплён внизу видимой области (нижний край
       прокручиваемого контейнера .content-panel из header.tpl), прокручивается
       только содержимое между началом страницы и этим блоком */
    .pager-bar-sticky {
        position: sticky;
        bottom: 0;
        z-index: 6;
        margin-left: -1rem;
        margin-right: -1rem;
        margin-bottom: -1rem;
        border-radius: 0;
        border-left: none;
        border-right: none;
        border-bottom: none;
        box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.15);
    }

    @media (min-width: 768px) {
        .pager-bar-sticky {
            margin-left: -1.5rem;
            margin-right: -1.5rem;
        }
    }

    .pager-nav .btn:disabled {
        color: var(--bs-secondary-color);
        opacity: 0.55;
    }

    .pager-nav .btn:not(:disabled):not(.disabled):hover {
        background-color: var(--bs-primary);
        color: #fff;
    }

    .pager-pagesize {
        min-width: 4.5rem;
    }

    /* классический вид таблицы: выделенные заголовки, разделители столбцов,
       цвета берутся из переменных Bootstrap 5 и сами переключаются
       вместе с темой (data-bs-theme) */
    .spr-classic-table {
        --bs-table-border-color: var(--bs-border-color);
    }

    .spr-classic-table thead th {
        position: sticky;
        top: 0;
        z-index: 3;
        background-color: var(--bs-tertiary-bg);
        color: var(--bs-emphasis-color);
        font-weight: 600;
        border-bottom: 2px solid var(--bs-border-color);
        vertical-align: middle;
    }

    .spr-classic-table td,
    .spr-classic-table th {
        border-color: var(--bs-border-color);
    }

    .spr-classic-table tbody tr:hover {
        --bs-table-hover-bg: var(--bs-secondary-bg);
    }

    .spr-classic-table tbody tr.row-context-target {
        cursor: context-menu;
    }

    /* светодиод питания перед именем сервера: на светлой теме - зелёный,
       на тёмной - синий (как у большинства сетевого/серверного оборудования),
       размер на 20% больше высоты текста (1.2em) */
    .power-led {
        width: 1.2em;
        height: 1.2em;
        vertical-align: -0.25em;
        margin-right: 0.3rem;
        flex-shrink: 0;
    }

    .power-led.is-on {
        color: #2fb344;
        filter: drop-shadow(0 0 2px rgba(47, 179, 68, 0.65));
    }

    [data-bs-theme="dark"] .power-led.is-on {
        color: #4dabf7;
        filter: drop-shadow(0 0 3px rgba(77, 171, 247, 0.75));
    }

    .power-led.is-off {
        color: var(--bs-secondary-color);
        opacity: 0.45;
        filter: none;
    }

    /* всплывающее меню строки (правая кнопка мыши), стиль повторяет левую
       панель навигации (sidebar-panel / btn-toggle-nav из header.tpl) */
    #rowContextMenu {
        position: fixed;
        display: none;
        min-width: 280px;
        max-width: 380px;
        max-height: 80vh;
        overflow-y: auto;
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.5rem;
        box-shadow: 0 0.5rem 1.5rem rgba(0, 0, 0, 0.3);
        padding: 0.5rem;
        z-index: 2000;
    }

    #rowContextMenu.show {
        display: block;
    }

    #rowContextMenu .ctx-menu-heading {
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        font-weight: 600;
        color: var(--bs-secondary-color);
        padding: 0.35rem 0.6rem 0.15rem;
    }

    #rowContextMenu a.ctx-menu-item {
        display: block;
        padding: 0.35rem 0.6rem;
        border-radius: 0.4rem;
        font-size: 0.9rem;
        text-decoration: none;
        color: var(--bs-emphasis-color);
        transition: background-color .15s ease, color .15s ease;
    }

    #rowContextMenu a.ctx-menu-item:hover,
    #rowContextMenu a.ctx-menu-item:focus {
        background-color: var(--bs-primary);
        color: #fff;
    }

    #rowContextMenu hr.ctx-menu-divider {
        margin: 0.4rem 0;
        border-color: var(--bs-border-color);
        opacity: 1;
    }
</style>

<form role="form" id="Update" method="POST">
    <div class="container-fluid d-grid gap-2">
        <div class="hero-text">
            <h3>Фильтр для списка серверов/рабочих станций</h3>
        </div>
        <div class="hero-text">
            <p>Фильтр состоит из 2 независимых блоков, помечая блок вы включаете обработку конкретного блока. Данные выборок консолидируются, исключая повторения серверов в списке. </p>
        </div>

        {{if .flash.error}}
        <blockquote>{{.flash.error}}</blockquote>
        {{end}}

        {{if .flash.notice}}
        <blockquote>{{.flash.notice}}</blockquote>
        {{end}}

        <div class="row">
            <div class="form-group col-sm-6">
                <input name="chBaseFilter" checked class="form-check-input" type="checkbox" id="inlineCheckbox00" value="on">
                <label class="fw-semibold" style="color:#a52a2a;" for="chBaseFilter">Основной блок фильтра</label>
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-2">
                <label for="Active">Состояние</label>
                <select name="Active" size="1" class="form-control" value="{{.record.Active}}">
                    <option value="0" {{if eq .record.Active 0}}selected{{end}}>Не выбрано</option>
                    <option value="1" {{if eq .record.Active 1}}selected{{end}}>Активен</option>
                    <option value="2" {{if eq .record.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                    <option value="999" {{if eq .record.Active 999}}selected{{end}}>Все</option>
                </select>
            </div>
            <div class="form-group col-sm-2">
                <label for="PowerState">Вкл./Выкл.</label>
                <select name="PowerState" size="1" class="form-control" value="{{.record.PowerState}}">
                    <option value="0" {{if eq .record.PowerState 0}}selected{{end}}>Не выбрано</option>
                    <option value="1" {{if eq .record.PowerState 1}}selected{{end}}>Включен</option>
                    <option value="2" {{if eq .record.PowerState 2}}selected{{end}}>Выключен</option>
                </select>
            </div>
            <div class="form-group col-sm-2">
                <label for="Name">Имя сервера</label>
                <input name="Name" class="form-control" type="text" value="{{.record.Name}}"
                    title="Список фрагментов имён серверов через разделитель ; (точка с запятой)" />
            </div>
            <div class="form-group col-sm-3">
                <label for="NameFQDN">Имя сервера полное</label>
                <input name="NameFQDN" class="form-control" type="text" value="{{.record.NameFQDN}}"
                    title="Список фрагментов полных имён серверов через разделитель ; (точка с запятой)" />
            </div>
            <div class="form-group col-sm-3">
                <label for="NameInfo">Альтернативное наименование</label>
                <input name="NameInfo" class="form-control" type="text" value="{{.record.NameInfo}}"
                    title="Список фрагментов альтернативных имён серверов через разделитель ; (точка с запятой)" />
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-2">
                <label for="Type">Тип сервера</label>
                <select name="Type" size="1" class="form-control" value="{{.record.Type}}">
                    <option value="0" {{if eq .record.Type 0}}selected{{end}}>Аппаратный сервер</option>
                    <option value="1" {{if eq .record.Type 1}}selected{{end}}>Виртуальная машина</option>
                    <option value="2" {{if eq .record.Type 2}}selected{{end}}>Гипервизор VMWARE</option>
                    <option value="3" {{if eq .record.Type 3}}selected{{end}}>Сервер во внешнем облаке</option>
                    <option value="999" {{if eq .record.Type 999}}selected{{end}}>Все</option>
                </select>
            </div>
            <div class="form-group col-sm-2">
                <label for="TypeMicroservice">Роль в микросервисах</label>
                <select name="TypeMicroservice" size="1" class="form-control" value="{{.record.TypeMicroservice}}">
                    <option value="0" {{if eq .record.TypeMicroservice 0}}selected{{end}}>не микросервис</option>
                    <option value="1" {{if eq .record.TypeMicroservice 1}}selected{{end}}>Docker</option>
                    <option value="2" {{if eq .record.TypeMicroservice 2}}selected{{end}}>Kubernetes</option>
                    <option value="999" {{if eq .record.TypeMicroservice 999}}selected{{end}}>Все</option>
                </select>
            </div>
            <div class="form-group col-sm-2">
                <label for="RoleType">APP/DB сервер</label>
                <select name="RoleType" size="1" class="form-control" value="{{.record.RoleType}}">
                    <option value="0" {{if eq .record.RoleType 0}}selected{{end}}>не указано</option>
                    <option value="1" {{if eq .record.RoleType 1}}selected{{end}}>APP сервер</option>
                    <option value="2" {{if eq .record.RoleType 2}}selected{{end}}>DB сервер</option>
                    <option value="3" {{if eq .record.RoleType 3}}selected{{end}}>DB+APP сервер</option>
                </select>
            </div>
            <div class="form-group col-sm-3">
                <label for="TypeWork">Тип среды</label>
                <select name="TypeWork" size="1" class="form-control" value="{{.record.TypeWork}}">
                    <option value="0" {{if eq .record.TypeWork 0}}selected{{end}}>не указано</option>
                    <option value="1" {{if eq .record.TypeWork 1}}selected{{end}}>Продуктовая</option>
                    <option value="2" {{if eq .record.TypeWork 2}}selected{{end}}>Внутренний тест</option>
                    <option value="3" {{if eq .record.TypeWork 3}}selected{{end}}>Внешний тест</option>
                </select>
            </div>
            <div class="form-group col-sm-3">
                <label for="CriticalLevel">Уровень критичности</label>
                <select name="CriticalLevel" size="1" class="form-control" value="{{.record.CriticalLevel}}">
                    <option value="0" {{if eq .record.CriticalLevel 0}}selected{{end}}>не указано</option>
                    <option value="1" {{if eq .record.CriticalLevel 1}}selected{{end}}>Vital</option>
                    <option value="2" {{if eq .record.CriticalLevel 2}}selected{{end}}>Critical</option>
                    <option value="3" {{if eq .record.CriticalLevel 3}}selected{{end}}>Secondary</option>
                    <option value="4" {{if eq .record.CriticalLevel 4}}selected{{end}}>Non Priority</option>
                </select>
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-12">
                <label for="ITSystemKey">ИТ система, в которой используется серверы</label>
                <select name="ITSystemKey" size="1" class="form-control" value="{{.record.ITSystemKey}}">
                    <option value="" {{if eq .record.ITSystemKey "" }}selected{{end}}>не выбрано</option>
                    {{range $its := .ITSystemSpr}}
                    <option value="{{$its.ObjectKey}}" {{if eq $its.ObjectKey $.record.ITSystemKey}}selected{{end}}>
                        {{$its.ObjectKey}} {{$its.Label}}: {{$its.FullName}} {{$its.Status}} {{$its.SystemType}} {{$its.BusinessColumn}} {{$its.CFO}}
                    </option>
                    {{end}}
                </select>
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-4">
                <label for="OS">Операционная система</label>
                <input name="OS" class="form-control" type="text" value="{{.record.OS}}" />
            </div>
            <div class="form-group col-sm-4">
                <label for="Role">Роль</label>
                <input name="Role" class="form-control" type="text" value="{{.record.Role}}" />
            </div>
            <div class="form-group col-sm-4">
                <label for="Anotation">Назначение</label>
                <input name="Anotation" class="form-control" type="text" value="{{.record.Anotation}}" />
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-4">
                <label for="Network">Сеть</label>
                <input name="Network" class="form-control" type="text" value="{{.record.Network}}" />
            </div>
            <div class="form-group col-sm-4">
                <label for="IPAddress">IP адреса</label>
                <input name="IPAddress" class="form-control" type="text" value="{{.record.IPAddress}}" />
            </div>
            <div class="form-group col-sm-4">
                <label for="Owner">Ответственный</label>
                <input name="Owner" class="form-control" type="text" value="{{.record.Owner}}"
                    title="Список владельцев через разделитель ;(точка с запятой)" />
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-6">
                <label for="DPCOffice">ЦОД: адрес и площадка</label>
                <select name="DPCOffice" size="1" class="form-control" value="{{.record.DPCOffice}}">
                    <option value="0" {{if eq .record.DPCOffice 0}}selected{{end}}>не выбрано</option>
                    <option value="101" {{if eq .record.DPCOffice 101}}selected{{end}}>623к, г. Новосибирск, ул. Советская 37
                    </option>
                    <option value="102" {{if eq .record.DPCOffice 102}}selected{{end}}>624к, г. Новосибирск, ул. Советская 37
                    </option>
                    <option value="103" {{if eq .record.DPCOffice 103}}selected{{end}}>316к, г. Новосибирск, ул. Советская 37
                    </option>
                    <option value="121" {{if eq .record.DPCOffice 121}}selected{{end}}>Мегафон,г. Новосибирск, ул. Толмачевская 19а к3</option>
                    <option value="131" {{if eq .record.DPCOffice 131}}selected{{end}}>Ростелеком,г. Новосибирск, ул. Менделеева 1</option>
                    <option value="201" {{if eq .record.DPCOffice 201}}selected{{end}}>3-02 г. Москва, ул. Мира 69</option>
                    <option value="202" {{if eq .record.DPCOffice 202}}selected{{end}}>4-19 г. Москва, ул. Мира 69</option>
                    <option value="281" {{if eq .record.DPCOffice 281}}selected{{end}}>DSP г. Москва, ул.Шарикоподшипниковская улица 11/9</option>
                    <option value="291" {{if eq .record.DPCOffice 291}}selected{{end}}>М1 г. Москва, ул. Варшавское шоссе 125</option>
                </select>
            </div>
        </div>

        <div class="row">
            <div class="form-group col-sm-6">
                <input name="chClusters" class="form-check-input" type="checkbox" value="on" checked>
                <label style="color:#800000;" class="form-check-label" for="chClusters">Дополнение фильтра по кластерам</label>
            </div>
        </div>

        <div class="row">
            {{range $cluster := .ClusterList}}
            <div class="form-group col-sm-3">
                <input name="ch{{$cluster.Name}}" class="form-check-input" type="checkbox" value="on" {{if eq $cluster.Id 1}}checked{{end}}>
                <label class="form-check-label" for="ch{{$cluster.Name}}">{{$cluster.Name}}</label>
            </div>
            {{end}}
        </div>

        <div class="row">
            <div class="col-12 d-flex flex-wrap gap-2">
                <input type="submit" name="ViewBTN" title="Просмотреть данные с учётом фильтров" value="Просмотр" class="btn btn-primary rounded-pill px-3" />
                <input type="submit" name="ExportBTN" title="Экспорт данных с учётом фильтров в CSV (utf8) (разделитель |)" value="Экспорт в CSV" formtarget="_blank" class="btn btn-outline-success rounded-pill px-3" />
                <input type="submit" name="ExportBTN2" title="Экспорт данных с учётом фильтров в XLSX" value="Экспорт в XLSX" formtarget="_blank" class="btn btn-outline-success rounded-pill px-3" />
                {{if eq .ViewOn 1}}
                <input type="submit" name="SetBaseBTN" title="Внести общие данные для группы серверов, выделенных в списке"
                    value="Редактировать выделенные" class="btn btn-outline-warning rounded-pill px-3" />
                {{end}}
                <a href="/sprsrv/addserver/{{.backmode}}/new" title="Добавить новый сервер"
                    class="btn btn-success rounded-pill px-3 ms-auto">Добавить</a>
            </div>
        </div>
        <div class="row">
            &nbsp;
        </div>
    </div>

    {{if eq .ViewOn 1}}
    {{if eq .ExportCSV 1}}

    {{else}}

    {{if gt .CountRecords 0}}
    <div class="hero-text">
        <h3>Выбрано серверов: {{.CountRecords}}</h3>
    </div>

    <input type="hidden" name="page" id="pageField" value="{{.Page}}">

    <table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
        <thead>
            <tr>
                <th class="text-center">Выбор</th>
                <th class="text-center">Тип</th>
                <th class="text-center">Заполнение %</th>
                <th>Имя сервера</th>
                <th>Сеть</th>
                <th>IP адрес</th>
                <th>Ответственный</th>
                <th>Предупреждение</th>
                <th class="text-center">Статус</th>
            </tr>
        </thead>
        <tbody id="myTable">
            {{range $record := .records}}
            {{if ne $record.S.Stat 666}}
            <tr class="row-context-target" data-menu-id="rowMenu{{$record.S.Id}}" title="Правая кнопка мыши - меню действий">
                <td class="text-center"><input name="chSelection{{$record.S.Id}}" title="chSelection{{$record.S.Id}}" class="form-check-input" type="checkbox" id="Selection{{$record.S.Id}}" value="on"></td>
                <td class="text-center">
                    {{if eq $record.S.Type 1}}{{if $record.VM.WorkStation}}виртуальная рабочая станция{{else}}виртуальный сервер{{end}}{{else if eq $record.S.Type 2}}гипервизор{{else if eq $record.S.Type 3}}сервер во внешнем облаке{{else}}аппаратный сервер {{if gt $record.S.ZabbixHostId 0}}<br>(Zabbix){{end}} {{end}}{{if eq $record.S.Stat 666}}<del>{{end}}
                </td>
                <td class="text-end">{{printf "%.2f" $record.PercentFilledField}}</td>
                <td>
                    {{if ne $record.S.NameInfo ""}} [ {{$record.S.NameInfo}} ] {{end}}
                    {{if ne $record.S.Active 2}}
                    {{if eq $record.S.Type 1}}
                    <div style="width:20px; float:left; text-align: center;"><a href="/sprsrv/vmperfperiodbysrvid/{{$record.S.Id}}" target="_blank" title="График загрузки виртуальной машины за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}"><span class="glyphicon glyphicon-tasks"></span></a></div>
                    <div style="width:20px; float:left; text-align: center;"><a href="/sprsrv/vmperfkeytest/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}/{{$record.VM.Vm_Name}}" target="_blank" title="Данные для анализа утилизации ресурсов {{$record.VM.Vm_Name}} в https://bcs-gpt.global.bcs/ базовый промпт: В файле приложены данные по загрузке ресурсов процессора, оперативной памяти, дисков и сети, данные получены из vmware. Имя виртуальной машины в наименовании листа. Виртуальная машина использует N ядра процессора NN гигабайта оперативной памяти и диск NNNгб. Проведи анализ утилизации ресурсов и выдай рекомендации по оптимизации. "><span class="glyphicon glyphicon-tasks"></span></a></div>
                    {{else if eq $record.S.Type 2}}
                    <div style="width:20px; float:left; text-align: center"><a href="/sprsrv/hostperfbyserverid/filter/{{$record.S.Id}}" target="_blank" title="График загрузки хоста за период из фильтра с {{printf "%02d.%02d.%04d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%04d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}"><span class="glyphicon glyphicon-tasks"></span></a></div>
                    {{else}}
                    <div style="width:20px; float:left; text-align: center"><a href="/details/filter/{{$record.S.Name}}" target="_blank" title="График загрузки сервера за период из фильтра с {{printf "%02d.%02d.%04d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%04d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}"><span class="glyphicon glyphicon-tasks"></span></a></div>
                    {{end}}
                    <div style="width:300px;" class="d-inline-flex align-items-center {{if eq $record.S.Type 2}}text-success{{else if eq $record.S.Type 1}}text-danger{{end}}"><svg class="bi power-led {{if eq $record.S.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $record.S.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg><a href="/sprsrv/updatebase/all/{{$record.S.Id}}" target="_blank" class="{{if eq $record.S.Type 2}}text-success{{else if eq $record.S.Type 1}}text-danger{{else if eq $record.S.Type 0}}link-body-emphasis{{end}}" title="Общие данные по серверу">{{$record.S.NameFQDN}}</a> {{if gt $record.VM.Id 0 }} ( <a href="/sprsrv/clusterperiod/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}" target="_blank" title="Информация по кластеру">{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}</a> ){{end}}{{if gt $record.Hardware.H.Id 0 }} ( sn:<a href="/sprsrv/hwconfedit/{{$record.Hardware.H.Id}}" target="_blank" title="Конфигурация сервера">{{$record.Hardware.H.SerialNum}}</a> ){{end}}</div>
                    {{else}}
                    {{if gt $record.VM.Id 0 }}
                    <div style="width:300px;" class="d-inline-flex align-items-center text-body-secondary"><svg class="bi power-led {{if eq $record.S.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $record.S.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg><a href="/sprsrv/updatebase/all/{{$record.S.Id}}" target="_blank" title="Общие данные по серверу">{{$record.S.Name}}</a> ( {{$record.VM.Vcenter}} / {{$record.VM.Datacenter}} / {{$record.VM.Cluster}} ) </div>
                    {{else}}
                    <div style="width:300px;" class="d-inline-flex align-items-center text-body-secondary"><svg class="bi power-led {{if eq $record.S.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $record.S.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg><a href="/sprsrv/updatebase/all/{{$record.S.Id}}" target="_blank" title="Общие данные по серверу">{{$record.S.Name}}</a> {{if gt $record.Hardware.H.Id 0 }} ( sn:<a href="/sprsrv/hwconfedit/{{$record.Hardware.H.Id}}" target="_blank" title="Конфигурация сервера">{{$record.Hardware.H.SerialNum}}</a> ){{end}}</div>
                    {{end}}
                    {{end}}
                </td>
                <td id="rowMenu{{$record.S.Id}}" class="d-none">
                    <div class="ctx-menu-heading">Подробнее</div>
                    <a class="ctx-menu-item" href="/sprsrv/updatebase/{{$.backmode}}/{{$record.S.Id}}" target="_blank">Общие данные</a>
                    <a class="ctx-menu-item" href="/sprsrv/updatecfo/{{$.backmode}}/edit/{{$record.S.Id}}" target="_blank">Аллокация</a>
                    {{if or (eq $record.S.Type 0) (eq $record.S.Type 2) }}<a class="ctx-menu-item" href="/sprsrv/updatehw/{{$.backmode}}/{{$record.S.Id}}" target="_blank">Конфигурация сервера</a>{{end}}
                    {{if eq $record.S.Type 1}}
                    <a class="ctx-menu-item" href="/sprsrv/updatevm/{{$.backmode}}/{{$record.S.Id}}" target="_blank">Детали по виртуальной машине</a>
                    <a class="ctx-menu-item" href="/sprsrv/updatevmcluster/{{$.backmode}}/{{$record.S.Id}}" target="_blank">Размещение виртуальной машины</a>
                    {{end}}
                    <hr class="ctx-menu-divider">
                    <div class="ctx-menu-heading">Загрузка</div>
                    {{if eq $record.S.Type 1}}
                    <a class="ctx-menu-item" href="/detailskeyvm/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}/{{$record.VM.Vm_Name}}" target="_blank">График загрузки виртуальной машины за день с 0:00 - 24:00</a>
                    <a class="ctx-menu-item" href="/sprsrv/vmperfkeytest/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}/{{$record.VM.Vm_Name}}" target="_blank" title="Данные для анализа утилизации ресурсов {{$record.VM.Vm_Name}} за день с 0:00 до текущего времени в https://bcs-gpt.global.bcs/ базовый промпт: В файле приложены данные по загрузке ресурсов процессора, оперативной памяти, дисков и сети, данные получены из vmware. Имя виртуальной машины в наименовании листа. Виртуальная машина использует N ядра процессора NN гигабайта оперативной памяти и диск NNNгб. Проведи анализ утилизации ресурсов и выдай рекомендации по оптимизации.">Данные для анализа утилизации ресурсов {{$record.VM.Vm_Name}} за день с 0:00 до текущего времени в BCS GPT</a>
                    <a class="ctx-menu-item" href="/detailsvmid/all/{{$record.S.VMSprId}}" target="_blank">График загрузки за весь период</a>
                    <a class="ctx-menu-item" href="/sprsrv/vmperfperiodbysrvid/{{$record.S.Id}}" target="_blank">График загрузки за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>
                    {{if gt $record.S.OldId 0}}<a class="ctx-menu-item" href="/sprsrv/vmperfperiodbysrvid/{{$record.S.OldId}}" target="_blank">График загрузки архивной версии за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>{{end}}
                    {{else if eq $record.S.Type 2}}
                    <a class="ctx-menu-item" href="/sprsrv/hostperfbyserverid/month/{{$record.S.Id}}" target="_blank">График загрузки за последний месяц</a>
                    <a class="ctx-menu-item" href="/sprsrv/hostperfbyserverid/all/{{$record.S.Id}}" target="_blank">График загрузки за весь период</a>
                    <a class="ctx-menu-item" href="/sprsrv/hostperfbyserverid/filter/{{$record.S.Id}}" target="_blank">График загрузки за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>
                    {{else}}
                    <a class="ctx-menu-item" href="/details/month/{{$record.S.Name}}" target="_blank">График загрузки за последний месяц</a>
                    <a class="ctx-menu-item" href="/details/all/{{$record.S.Name}}" target="_blank">График загрузки за весь период</a>
                    <a class="ctx-menu-item" href="/details/filter/{{$record.S.Name}}" target="_blank">График загрузки за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>
                    {{end}}
                    {{if gt $record.S.ZabbixHostId 0}}
                    <a class="ctx-menu-item" href="/sprsrv/zabbixperfperiod/{{$record.S.ZabbixHostId}}" target="_blank">График загрузки по данным Zabbix за период из фильтра с {{printf "%02d.%02d.%4d" $.Filter_save.FromTime.Day $.Filter_save.FromTime.Month $.Filter_save.FromTime.Year}} по {{printf "%02d.%02d.%4d" $.Filter_save.ToTime.Day $.Filter_save.ToTime.Month $.Filter_save.ToTime.Year}}</a>
                    {{end}}
                </td>
                <td>
                    <div style="width:180px">{{$record.S.Network}}</div>
                </td>
                <td>
                    <div style="width:180px">{{$record.S.IPAddress}}</div>
                </td>
                <td>{{$record.S.Owner}}</td>
                <td class="text-center">{{$record.S.Warning}}</td>
                {{if and (ge $record.S.Level 0.0) (le $record.S.Level 75.0)}} <td class="text-center"> </td>
                {{ else if and (gt $record.S.Level 75.0) (le $record.S.Level 90.0)}} <td class="text-center table-warning">{{printf "%.0f" $record.S.Level}} </td>
                {{ else if (gt $record.S.Level 90.0)}} <td class="text-center table-danger">{{printf "%.0f" $record.S.Level}} </td>
                {{ else }} <td class="text-end">{{printf "%.0f" $record.S.Level}}</td> {{end}}
            </tr>
            {{end}}
            {{end}}
        </tbody>
        <tfoot>
            <tr>
            </tr>
        </tfoot>
    </table>
    <div id="rowContextMenu"></div>

    <div class="pager-bar pager-bar-sticky d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">
        <div class="text-muted small">
            Показано {{.PageFrom}}&ndash;{{.PageTo}} из {{.CountRecords}}
        </div>
        <div class="d-flex align-items-center gap-2">
            <label for="pageSizeSelect" class="small text-muted mb-0">На странице:</label>
            <select id="pageSizeSelect" class="form-select form-select-sm pager-pagesize" style="width:auto;" onchange="pagerChangePageSize(this.value)">
                <option value="25" {{if eq .PageSize 25}}selected{{end}}>25</option>
                <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .PageSize 100}}selected{{end}}>100</option>
                <option value="200" {{if eq .PageSize 200}}selected{{end}}>200</option>
            </select>
            <div class="btn-group pager-nav" role="group" aria-label="Постраничная навигация">
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="pagerGoTo(1)" {{if eq .Page 1}}disabled{{end}} title="Первая страница">&laquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="pagerGoTo({{.Page}}-1)" {{if eq .Page 1}}disabled{{end}} title="Предыдущая страница">&lsaquo;</button>
                <span class="btn btn-sm disabled border-0 px-2">Стр. {{.Page}} из {{.TotalPages}}</span>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="pagerGoTo({{.Page}}+1)" {{if eq .Page .TotalPages}}disabled{{end}} title="Следующая страница">&rsaquo;</button>
                <button type="button" class="btn btn-outline-secondary btn-sm" onclick="pagerGoTo({{.TotalPages}})" {{if eq .Page .TotalPages}}disabled{{end}} title="Последняя страница">&raquo;</button>
            </div>
        </div>
    </div>
    {{else}}
    <div class="hero-text">
        <h3>Нет данных по фильтру</h3>
    </div>
    <div class="row">
        &nbsp;
    </div>
    {{end}}
    {{end}}
    {{end}}
</form>
