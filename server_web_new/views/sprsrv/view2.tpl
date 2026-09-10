<style>
    /* стиль выпадающего списка поиска ИТ-системы - скопирован из
       views/sprsrv/updatebase.tpl (образец для этого поля) */
    .it-system-dropdown {
        z-index: 1000;
        max-height: 280px;
        overflow-y: auto;
        box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, .25);
    }

    .it-system-dropdown .list-group-item {
        cursor: pointer;
    }

    .it-system-dropdown .list-group-item small {
        display: block;
        color: var(--bs-secondary-color);
    }
</style>

<script>
$(document).ready(function () {
    $("#myInput").on("keyup", function () {
        var value = $(this).val().toLowerCase();
        $("#myTable tr").filter(function () {
            $(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
        });
    });
});

document.addEventListener('DOMContentLoaded', function () {
    initITSystemPicker();
});

// Поле "ИТ система" - два синхронизированных способа ввода: прямой ввод
// кода в ITSystemKey и поиск по справочнику (ITSystemSearch) с фильтрацией
// списка по вводимым символам (код/название/статус/тип/бизнес-направление/
// ЦФО) - выбор элемента списка подставляет его код в ITSystemKey. Код
// скопирован 1:1 из views/sprsrv/updatebase.tpl (initITSystemPicker там же,
// образец для этого поля) - при изменении логики поддерживать в обоих
// местах синхронно.
function initITSystemPicker() {
    var keyInput = document.getElementById('ITSystemKey');
    var hint = document.getElementById('ITSystemKeyHint');
    var searchInput = document.getElementById('ITSystemSearch');
    var results = document.getElementById('ITSystemSearchResults');
    var dataList = document.getElementById('ITSystemDataList');
    if (!keyInput || !searchInput || !results || !dataList) return;

    var items = Array.prototype.map.call(dataList.querySelectorAll('.it-system-item'), function (el) {
        return {
            key: el.getAttribute('data-key') || '',
            label: el.getAttribute('data-label') || '',
            full: el.getAttribute('data-full') || '',
            status: el.getAttribute('data-status') || '',
            type: el.getAttribute('data-type') || '',
            biz: el.getAttribute('data-biz') || '',
            cfo: el.getAttribute('data-cfo') || ''
        };
    });

    function findByKey(key) {
        if (!key) return null;
        var lower = key.toLowerCase();
        for (var i = 0; i < items.length; i++) {
            if (items[i].key.toLowerCase() === lower) return items[i];
        }
        return null;
    }

    function describe(item) {
        var parts = [item.key];
        if (item.label) parts.push(item.label);
        if (item.full) parts.push(item.full);
        return parts.join(' — ');
    }

    function updateHint() {
        var match = findByKey(keyInput.value.trim());
        if (match) {
            hint.textContent = describe(match) + (match.status ? ' (' + match.status + ')' : '');
            hint.classList.remove('text-danger');
        } else if (keyInput.value.trim() !== '') {
            hint.textContent = 'Код не найден в справочнике - будет сохранён как есть';
            hint.classList.add('text-danger');
        } else {
            hint.textContent = '';
            hint.classList.remove('text-danger');
        }
    }

    function hideResults() {
        results.classList.add('d-none');
        results.innerHTML = '';
    }

    function renderResults(query) {
        var q = query.trim().toLowerCase();
        if (q === '') {
            hideResults();
            return;
        }
        var matches = items.filter(function (item) {
            return item.key.toLowerCase().indexOf(q) !== -1 ||
                item.label.toLowerCase().indexOf(q) !== -1 ||
                item.full.toLowerCase().indexOf(q) !== -1 ||
                item.status.toLowerCase().indexOf(q) !== -1 ||
                item.type.toLowerCase().indexOf(q) !== -1 ||
                item.biz.toLowerCase().indexOf(q) !== -1 ||
                item.cfo.toLowerCase().indexOf(q) !== -1;
        }).slice(0, 50);

        if (matches.length === 0) {
            results.innerHTML = '<div class="list-group-item text-muted">Ничего не найдено</div>';
            results.classList.remove('d-none');
            return;
        }

        results.innerHTML = '';
        matches.forEach(function (item) {
            var row = document.createElement('button');
            row.type = 'button';
            row.className = 'list-group-item list-group-item-action';
            row.innerHTML = '<div>' + escapeHtml(item.key) + (item.label ? ' — ' + escapeHtml(item.label) : '') + '</div>' +
                (item.full ? '<small>' + escapeHtml(item.full) + '</small>' : '');
            row.addEventListener('click', function () {
                keyInput.value = item.key;
                searchInput.value = describe(item);
                updateHint();
                hideResults();
            });
            results.appendChild(row);
        });
        results.classList.remove('d-none');
    }

    function escapeHtml(s) {
        var div = document.createElement('div');
        div.textContent = s;
        return div.innerHTML;
    }

    searchInput.addEventListener('input', function () {
        renderResults(searchInput.value);
    });
    searchInput.addEventListener('focus', function () {
        if (searchInput.value.trim() !== '') renderResults(searchInput.value);
    });
    document.addEventListener('click', function (evt) {
        if (evt.target !== searchInput && !results.contains(evt.target)) hideResults();
    });

    keyInput.addEventListener('input', updateHint);
    keyInput.addEventListener('blur', updateHint);

    // первичная инициализация: если фильтр уже содержит код ИТ-системы,
    // сразу показать расшифровку и заполнить строку поиска для наглядности
    var initial = findByKey(keyInput.value.trim());
    if (initial) {
        searchInput.value = describe(initial);
    }
    updateHint();
}

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

<!-- Стили панели пагинации, "классической" таблицы, индикатора питания и
     всплывающего контекстного меню строки вынесены в общий
     server_web_new/static/css/loadmon.css (подключён в header.tpl) - эта
     страница является для них образцом оформления. -->

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
                    <option value="4" {{if eq .record.Type 4}}selected{{end}}>Аппаратный сервер в Zabbix</option>
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
                <label class="form-label">ИТ система, в которой используется серверы</label>
                <!-- Поле "ИТ система" в этом фильтре переделано с <select> (плохо
                     масштабируется на большой справочник ИТ-систем и не позволяет
                     искать по названию/статусу) на тот же ввод с поиском, что и в
                     /sprsrv/updatebase/all - код вводится напрямую в ITSystemKey
                     (значение уходит в форму как есть, пустое значение = фильтр не
                     применяется) либо подбирается через ITSystemSearch, который
                     фильтрует список .ITSystemSpr на клиенте и подставляет код
                     выбранной записи - см. initITSystemPicker() ниже, код скопирован
                     1:1 из views/sprsrv/updatebase.tpl (образец для этого поля). -->
                <div class="row g-2">
                    <div class="col-md-4">
                        <input type="text" name="ITSystemKey" id="ITSystemKey" class="form-control"
                            value="{{.record.ITSystemKey}}" placeholder="CMDB-" autocomplete="off">
                        <div id="ITSystemKeyHint" class="form-text"></div>
                    </div>
                    <div class="col-md-8 position-relative">
                        <input type="text" id="ITSystemSearch" class="form-control"
                            placeholder="Поиск по справочнику: код, название, статус..." autocomplete="off">
                        <div id="ITSystemSearchResults" class="list-group position-absolute w-100 d-none it-system-dropdown"></div>
                    </div>
                </div>
                <!-- источник данных справочника ИТ-систем для поиска на клиенте -
                     сами элементы скрыты, JS читает их data-атрибуты один раз при
                     загрузке страницы; html/template экранирует значения -->
                <div id="ITSystemDataList" class="d-none">
                    {{range $its := .ITSystemSpr}}
                    <div class="it-system-item"
                        data-key="{{$its.ObjectKey}}"
                        data-label="{{$its.Label}}"
                        data-full="{{$its.FullName}}"
                        data-status="{{$its.Status}}"
                        data-type="{{$its.SystemType}}"
                        data-biz="{{$its.BusinessColumn}}"
                        data-cfo="{{$its.CFO}}"></div>
                    {{end}}
                </div>
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

        <!-- ИЗМЕНЕНО (2026-09-08, по запросу пользователя "замени в форме
             /sprsrv/view2 ... схему с выбором ЦОД и Шкафа вместо ЦОД: адрес и
             площадка по старым данным"): вместо select "DPCOffice" (жёстко
             зашитый в HTML список адресов, не связанный со справочниками) -
             каскадный выбор ЦОД -> Шкаф из models.DatacenterSite/
             models.RackCabinet, тот же приём, что уже в bladesystemsedit.tpl
             и /sprsrv/hwconflist (см. sprsrvview2.go:placementServerSprIds
             про правила определения размещения для разных типов записей). -->
        <div class="row">
            <div class="form-group col-sm-6">
                <label for="view2DatacenterSiteId">Площадка/ЦОД</label>
                <!-- ИЗМЕНЕНО (2026-09-10, по запросу пользователя "фильтр по
                     Площадка/ЦОД и Шкаф не работает"): добавлен
                     name="DatacenterSiteId" - раньше select был чисто
                     клиентским (сужал только список "Шкаф"), выбор ЦОД БЕЗ
                     конкретного шкафа никак не фильтровал список (см.
                     sprsrvview2.go: datacenterSiteId используется как
                     запасной уровень фильтра, когда "Шкаф" оставлен "Все"). -->
                <select id="view2DatacenterSiteId" name="DatacenterSiteId" class="form-control">
                    <option value="0">Все</option>
                    {{range $dc := .DatacenterSites}}
                    <option value="{{$dc.Id}}" {{if eq $.FilterDatacenterSiteId (printf "%d" $dc.Id)}}selected{{end}}>{{$dc.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div class="form-group col-sm-6">
                <label for="RackCabinetId">Шкаф</label>
                <select id="RackCabinetId" name="RackCabinetId" size="1" class="form-control">
                    <option value="0">Все</option>
                    {{range $rc := .RackCabinets}}
                    <option value="{{$rc.Id}}" data-dc="{{$rc.DatacenterSiteId}}" {{if eq $.RackCabinetId (printf "%d" $rc.Id)}}selected{{end}}>{{$rc.Name}}</option>
                    {{end}}
                </select>
            </div>
        </div>
        <script>
            // Каскадный фильтр "ЦОД -> Шкаф" - select ЦОД
            // (view2DatacenterSiteId, name="DatacenterSiteId" - с 2026-09-10
            // тоже отправляется на сервер и используется КАК ЗАПАСНОЙ
            // уровень фильтра, если "Шкаф" не выбран, см. sprsrvview2.go)
            // ещё и сужает список опций select "Шкаф" (RackCabinetId - если
            // выбран конкретный шкаф, он в приоритете) по атрибуту data-dc
            // каждой опции - идентично /sprsrv/hwconflist и
            // bladesystemsedit.tpl.
            (function () {
                var dcSelect = document.getElementById('view2DatacenterSiteId');
                var rackSelect = document.getElementById('RackCabinetId');
                if (!dcSelect || !rackSelect) return;

                var allRackOptions = Array.prototype.slice.call(rackSelect.options);

                function applyFilter(preserveSelection) {
                    var dc = dcSelect.value;
                    var currentValue = rackSelect.value;
                    rackSelect.innerHTML = '';
                    allRackOptions.forEach(function (opt) {
                        var optDc = opt.getAttribute('data-dc') || '0';
                        if (opt.value === '0' || dc === '0' || optDc === dc) {
                            rackSelect.appendChild(opt);
                        }
                    });
                    if (preserveSelection) {
                        var stillPresent = Array.prototype.some.call(rackSelect.options, function (opt) {
                            return opt.value === currentValue;
                        });
                        rackSelect.value = stillPresent ? currentValue : '0';
                    }
                }

                var selectedOption = allRackOptions.find(function (opt) { return opt.selected; });
                if (selectedOption && selectedOption.value !== '0') {
                    dcSelect.value = selectedOption.getAttribute('data-dc') || '0';
                }
                applyFilter(true);

                dcSelect.addEventListener('change', function () { applyFilter(true); });
            })();
        </script>

        <div class="row">
            <div class="form-group col-sm-6">
                <input id="chClusters" name="chClusters" class="form-check-input" type="checkbox" value="on" onchange="toggleClusterList()">
                <label style="color:#800000;" class="form-check-label" for="chClusters">Дополнительные фильтры по кластерам</label>
            </div>
        </div>

        <div class="row" id="clusterListRow" style="display:none;">
            {{range $cluster := .ClusterList}}
            <div class="form-group col-sm-3">
                <input id="ch{{$cluster.Name}}" name="ch{{$cluster.Name}}" class="form-check-input" type="checkbox" value="on" {{if eq $cluster.ID 1}}checked{{end}}>
                <label class="form-check-label" for="ch{{$cluster.Name}}">{{$cluster.Name}}</label>
            </div>
            {{end}}
        </div>
        <script>
            function toggleClusterList() {
                var ch = document.getElementById('chClusters');
                var row = document.getElementById('clusterListRow');
                if (ch && row) {
                    row.style.display = ch.checked ? '' : 'none';
                }
            }
            document.addEventListener('DOMContentLoaded', toggleClusterList);
        </script>

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
                    {{if eq $record.S.Type 1}}{{if $record.VM.WorkStation}}виртуальная рабочая станция{{else}}виртуальный сервер{{end}}{{else if eq $record.S.Type 2}}гипервизор{{else if eq $record.S.Type 3}}сервер во внешнем облаке{{else}}аппаратный сервер {{end}}{{if gt $record.S.ZabbixHostId 0}}<br><span class="badge rounded-pill text-bg-info" title="Сервер зарегистрирован в Zabbix (ZabbixHostId={{$record.S.ZabbixHostId}}) - метрики загрузки поступают из Zabbix">Zabbix</span>{{end}}{{if eq $record.S.Stat 666}}<del>{{end}}
                </td>
                <td class="text-end">{{printf "%.2f" $record.PercentFilledField}}</td>
                <td>
                    {{if ne $record.S.NameInfo ""}} [ {{$record.S.NameInfo}} ] {{end}}
                    {{if ne $record.S.Active 2}}
                    {{if eq $record.S.Type 1}}
                    <!-- Иконки /sprsrv/vmperfperiodbysrvid и /sprsrv/vmperfkeytest вели на методы,
                         не реализованные в controllers (маршруты закомментированы в
                         routers/routers.go) - убраны. -->
                    {{else if eq $record.S.Type 2}}
                    <!-- Иконка /sprsrv/hostperfbyserverid вела на метод HostPerfByServerId,
                         не реализованный в controllers (маршрут закомментирован в
                         routers/routers.go) - убрана. -->
                    {{else}}
                    <!-- Иконка/ссылка на /details/filter/... убрана по требованию. -->
                    {{end}}
                    <!-- Для ВМ (Type 1) и гипервизоров (Type 2) имя сервера показывается в 2
                         строки: 1-я - индикатор питания + имя (+ sn для гипервизора), 2-я -
                         vcenter/датацентр/кластер (models.ServerSprRecords.Vcenter/Datacenter/
                         Cluster заполняются именно для Type 1 и 2 - см. комментарий к полям в
                         models/models_add.go); раньше vcenter/dc/cluster брались из $record.VM
                         (догружается отдельным запросом только для ВМ), из-за чего у
                         гипервизоров эта информация не показывалась вовсе. Для остальных типов
                         (физический сервер, внешнее облако) вторая строка не нужна и не выводится. -->
                    <div style="width:300px;" class="srv-name-cell {{if eq $record.S.Type 2}}text-success{{else if eq $record.S.Type 1}}text-danger{{end}}">
                        <div class="d-inline-flex align-items-center">
                            <svg class="bi power-led {{if eq $record.S.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $record.S.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg>
                            <a href="/sprsrv/updatebase/all/{{$record.S.Id}}" target="_blank" class="{{if eq $record.S.Type 2}}text-success{{else if eq $record.S.Type 1}}text-danger{{else if eq $record.S.Type 0}}link-body-emphasis{{end}}" title="Общие данные по серверу">{{$record.S.NameFQDN}}</a>
                            {{if gt $record.Hardware.H.Id 0 }} ( sn:<a href="/sprsrv/hwconfedit/{{$record.Hardware.H.Id}}" target="_blank" title="Конфигурация сервера">{{$record.Hardware.H.SerialNum}}</a> ){{end}}
                        </div>
                        {{if or (eq $record.S.Type 1) (eq $record.S.Type 2)}}{{if ne $record.S.Vcenter "" }}
                        <!-- Раньше ссылка вела на /sprsrv/clusterperiod/... (метод SprClusterPeriod
                             не был реализован, маршрут закомментирован в routers/routers.go) -
                             теперь ведёт на его рабочий аналог /sprsrv/clusterperfperiod/... -
                             графики загрузки кластера за период по данным БД
                             (controllers/sprsrvclusterperfperiod.go, ClusterRecord/ClusterPerf),
                             реализованный по тому же принципу, что и /sprsrv/vmperfperiod для ВМ. -->
                        <a href="/sprsrv/clusterperfperiod/{{$record.S.Vcenter}}/{{$record.S.Datacenter}}/{{$record.S.Cluster}}" target="_blank" title="Загрузка кластера за период" class="text-success">{{$record.S.Vcenter}} / {{$record.S.Datacenter}} / {{$record.S.Cluster}}</a>
                        {{end}}{{end}}
                    </div>
                    {{else}}
                    <div style="width:300px;" class="srv-name-cell text-body-secondary">
                        <div class="d-inline-flex align-items-center">
                            <svg class="bi power-led {{if eq $record.S.PowerState 1}}is-on{{else}}is-off{{end}}" aria-hidden="true" title="{{if eq $record.S.PowerState 1}}Включен{{else}}Выключен{{end}}"><use href="#power"></use></svg>
                            <a href="/sprsrv/updatebase/all/{{$record.S.Id}}" target="_blank" title="Общие данные по серверу">{{$record.S.Name}}</a>
                            {{if gt $record.Hardware.H.Id 0 }} ( sn:<a href="/sprsrv/hwconfedit/{{$record.Hardware.H.Id}}" target="_blank" title="Конфигурация сервера">{{$record.Hardware.H.SerialNum}}</a> ){{end}}
                        </div>
                        {{if or (eq $record.S.Type 1) (eq $record.S.Type 2)}}{{if ne $record.S.Vcenter "" }}
                        <div class="text-success">{{$record.S.Vcenter}} / {{$record.S.Datacenter}} / {{$record.S.Cluster}}</div>
                        {{end}}{{end}}
                    </div>
                    {{end}}
                </td>
                <td id="rowMenu{{$record.S.Id}}" class="d-none">
                    <!-- Меню объединено в один плоский список без заголовков секций
                         "Подробнее"/"Загрузка" и разделителя между ними - по запросу. -->
                    <a class="ctx-menu-item" href="/sprsrv/updatebase/{{$.backmode}}/{{$record.S.Id}}" target="_blank">Редактировать</a>
                    {{if and (or (eq $record.S.Type 0) (eq $record.S.Type 2)) (ne $record.S.SerialNumber "") }}<a class="ctx-menu-item" href="/sprsrv/hwconfeditbysn/{{$record.S.SerialNumber}}" target="_blank">Конфигурация сервера</a>{{end}}
                    <!-- Пункты "Детали по виртуальной машине"/"Размещение виртуальной машины"
                         (controllers.SprSrvUpdateVM/SprSrvUpdateVMCluster) убраны - методы не
                         реализованы, маршруты закомментированы в routers/routers.go. -->
                    <!-- Раньше весь блок "Загрузка" был обёрнут в {{"{{if ne $record.S.Type 0}}"}},
                         из-за чего он никогда не показывался для аппаратных серверов (Type 0) -
                         даже уже готовая ссылка на /sprsrv/zabbixperfperiod ниже была недостижима,
                         хотя ZabbixHostId заполняется именно для физических серверов
                         (zabbix.Get10MinutesHistory опрашивает только группу "физические серверы").
                         Условие снято.
                         Для аппаратных серверов (Type 0) все старые ссылки на /details/... (report)
                         убраны по требованию - для них показывается только график по данным
                         Zabbix (см. общий блок ZabbixHostId ниже); для остальных типов (Type 3 -
                         внешнее облако) старые ссылки /details/... сохранены без изменений. -->
                    {{if eq $record.S.Type 1}}
                    <!-- Для ВМ (Type 1) прежние ссылки этого блока (/detailskeyvm,
                         /sprsrv/vmperfkeytest, /detailsvmid, /sprsrv/vmperfperiodbysrvid) вели на
                         методы, не реализованные в controllers (маршруты закомментированы в
                         routers/routers.go) - убраны. Вместо них - два реально работающих
                         метода: SprSrvVMPerfDay (controllers/sprsrvvmperfday.go, график за текущий
                         день с 0:00 по "живым" данным гипервизора, grpcclient.GetVMByKeyWithPerf)
                         и SprSrvVMPerfPeriod (controllers/sprsrvvmperfperiod.go, график за
                         произвольный период по данным из БД - models.VMRecord/VMPerfRecord). -->
                    <a class="ctx-menu-item" href="/sprsrv/vmperfday/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}/{{$record.VM.Vm_Name}}" target="_blank">График загрузки виртуальной машины за текущий день с 0:00</a>
                    <a class="ctx-menu-item" href="/sprsrv/vmperfperiod/{{$record.VM.Vcenter}}/{{$record.VM.Datacenter}}/{{$record.VM.Cluster}}/{{$record.VM.Vm_Name}}" target="_blank">График загрузки виртуальной машины за период (по данным БД)</a>
                    {{else if eq $record.S.Type 2}}
                    <!-- Для гипервизора (Type 2) ссылки /sprsrv/hostperfbyserverid/... ведут на
                         метод HostPerfByServerId, не реализованный в controllers (маршрут
                         закомментирован в routers/routers.go) - убраны, вместо них поясняющая
                         надпись. -->
                    <span class="ctx-menu-item text-muted" title="Метод построения графика загрузки гипервизора пока не реализован">График загрузки гипервизора пока не реализован</span>
                    {{else if eq $record.S.Type 0}}
                    {{if le $record.S.ZabbixHostId 0}}
                    <span class="ctx-menu-item text-muted" title="Сервер не зарегистрирован в Zabbix (ZabbixHostId не заполнен) - данные о загрузке недоступны">Нет данных о загрузке (сервер не в Zabbix)</span>
                    {{end}}
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
