<style>
    /* поля, недоступные для редактирования при текущем типе сервера */
    .field-view-only {
        background-color: var(--bs-secondary-bg);
        color: var(--bs-secondary-color);
        cursor: not-allowed;
    }

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
    // Серийный номер редактируется только для физического сервера (0) и
    // гипервизора (2); vCenter/Датацентр/Кластер - только для виртуальной
    // машины (1) и гипервизора (2). Для остальных сочетаний поля переводятся
    // в режим "только для просмотра" (readonly), значение при этом видно и
    // может быть скопировано, но не редактируется.
    // Дублируется проверкой на сервере (controllers/sprsrvupdatebase.go,
    // controllers/sprsrvaddserver.go) - JS здесь только для удобства ввода.
    function applyServerTypeFieldRules() {
        var typeSelect = document.getElementById('Type');
        if (!typeSelect) return;
        var type = parseInt(typeSelect.value, 10);

        var allowSerial = (type === 0 || type === 2);
        var allowVCenterFields = (type === 1 || type === 2);

        setFieldViewOnly('SerialNumber', !allowSerial);
        setFieldViewOnly('Vcenter', !allowVCenterFields);
        setFieldViewOnly('Datacenter', !allowVCenterFields);
        setFieldViewOnly('Cluster', !allowVCenterFields);
    }

    function setFieldViewOnly(id, viewOnly) {
        var el = document.getElementById(id);
        if (!el) return;
        el.readOnly = viewOnly;
        el.classList.toggle('field-view-only', viewOnly);
        el.title = viewOnly ? 'Поле недоступно для редактирования при текущем типе сервера' : '';
    }

    document.addEventListener('DOMContentLoaded', function () {
        applyServerTypeFieldRules();
        var typeSelect = document.getElementById('Type');
        if (typeSelect) {
            typeSelect.addEventListener('change', applyServerTypeFieldRules);
        }
        initITSystemPicker();
    });

    // Поле "ИТ система" - два синхронизированных способа ввода:
    // 1) прямой ввод кода в ITSystemKey (значение поля отправляется формой как есть);
    // 2) поиск по справочнику (ITSystemSearch) с фильтрацией списка по вводимым
    //    символам (код/название/статус/тип/бизнес-направление/ЦФО) - выбор
    //    элемента списка подставляет его код в ITSystemKey.
    // Оба поля синхронизированы: выбор в справочнике обновляет код, а прямой
    // ввод/изменение кода обновляет подсказку с расшифровкой найденной записи.
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
                    applyITSystemCorrection(item.key);
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
        keyInput.addEventListener('blur', function () {
            updateHint();
            // прямой ввод кода вручную (без выбора из результатов поиска) -
            // применяем корректировку тоже, но только если код действительно
            // нашёлся в справочнике (иначе бессмысленно дёргать сервер)
            var match = findByKey(keyInput.value.trim());
            if (match) applyITSystemCorrection(match.key);
        });

        // первичная инициализация: если запись уже содержит код ИТ-системы,
        // сразу показать расшифровку и заполнить строку поиска для наглядности
        // (но НЕ применяем корректировку полей - это уже сохранённая запись,
        // трогать её значения только из-за того, что страница открылась, не
        // нужно; корректировка выполняется только при явной смене ИТ-системы
        // пользователем - см. applyITSystemCorrection выше/ниже)
        var initial = findByKey(keyInput.value.trim());
        if (initial) {
            searchInput.value = describe(initial);
        }
        updateHint();
    }

    // Подтягивает поля, которые пересчитывает controllers.correctServerSprRecord
    // (критичность, мониторинг, резервное копирование, ответственные, тип
    // системы, лицензирование, вариант решения, пользователи, аллокация) для
    // выбранной ИТ-системы - через JSON-предпросмотр /sprsrv/itsystemcorrectpreview
    // (controllers/sprsrvupdatebase.go: SprITSystemCorrectPreview), чтобы
    // изменения было видно в форме сразу при выборе ИТ-системы, до нажатия
    // "Сохранить" - та же самая логика в этот момент ещё не выполнялась бы
    // (correctServerSprRecord вызывается только при сохранении формы).
    function applyITSystemCorrection(key) {
        key = (key || '').trim();
        if (!key) return;

        var hint = document.getElementById('ITSystemKeyHint');
        var prevHintText = hint ? hint.textContent : '';
        if (hint) hint.textContent = prevHintText + ' (обновление полей…)';

        fetch('/sprsrv/itsystemcorrectpreview/' + encodeURIComponent(key))
            .then(function (r) { return r.json(); })
            .then(function (data) {
                if (hint) hint.textContent = prevHintText;
                if (!data || !data.found) return;

                setSelectValue('CriticalLevel', data.criticalLevel);
                setSelectValue('Monitoring', data.monitoring);
                setSelectValue('Backuped', data.backuped);

                setFieldValue('BusinessOwner', data.businessOwner);
                setFieldValue('ITOwner', data.itOwner);
                setFieldValue('Owner', data.owner);
                setFieldValue('ProductOwner', data.productOwner);
                setFieldValue('Support', data.support);
                setFieldValue('SupportTeam', data.supportTeam);
                setFieldValue('SystemType', data.systemType);
                setFieldValue('LicensingType', data.licensingType);
                setFieldValue('SolutionType', data.solutionType);
                setFieldValue('Users', data.users);
                setFieldValue('Allocation', data.allocation);
            })
            .catch(function () {
                // сеть/сервер недоступны - предпросмотр не критичен, поля формы
                // остаются как были, ошибку показывать не нужно (при
                // фактическом сохранении та же корректировка будет
                // предпринята заново на сервере)
                if (hint) hint.textContent = prevHintText;
            });
    }

    function setFieldValue(id, value) {
        var el = document.getElementById(id);
        if (!el || value === undefined || value === null) return;
        el.value = value;
    }

    function setSelectValue(id, value) {
        var el = document.getElementById(id);
        if (!el || value === undefined || value === null) return;
        el.value = String(value);
    }
</script>

<div class="hero-text mb-3">
    <h2>{{if .IsNew}}Новый сервер{{else}}Общие данные сервера{{end}}</h2>
    {{if .flash.error}}
    <blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
    {{end}}
</div>

{{if not .IsNew}}
<div class="text-muted small mb-3">
    Дата создания: {{if .record.TimeCreate.IsZero}}&mdash;{{else}}{{.record.TimeCreate.Format "2006-01-02 15:04:05"}}{{end}}{{if .CreatedByName}} ({{.CreatedByName}}){{end}}
    &middot;
    Дата изменения: {{if .record.Time_Mfy.IsZero}}&mdash;{{else}}{{.record.Time_Mfy.Format "2006-01-02 15:04:05"}}{{end}}
    &middot;
    Автор последних изменений: {{if .ModifiedByName}}{{.ModifiedByName}}{{else}}&mdash;{{end}}
    &middot;
    <a href="/admin/serverspreventlog?server={{.record.Id}}" target="_blank">История изменений карточки</a>
</div>
{{end}}

<form role="form" id="UpdateBase" method="POST" class="pb-4">

    <div class="card mb-3">
        <div class="card-header fw-semibold">Общие данные</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-6">
                    <label for="Type" class="form-label">Тип сервера</label>
                    <select name="Type" id="Type" class="form-select">
                        <option value="0" {{if eq .record.Type 0}}selected{{end}}>Физический сервер</option>
                        <option value="1" {{if eq .record.Type 1}}selected{{end}}>Виртуальная машина</option>
                        <option value="2" {{if eq .record.Type 2}}selected{{end}}>Гипервизор VMware</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-6">
                    <label for="SerialNumber" class="form-label">Серийный номер</label>
                    <input name="SerialNumber" id="SerialNumber" class="form-control" type="text" value="{{.record.SerialNumber}}" title="Заполняется для физического сервера и гипервизора" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Vcenter" class="form-label">vCenter</label>
                    <input name="Vcenter" id="Vcenter" class="form-control" type="text" value="{{.record.Vcenter}}" title="Заполняется для виртуальной машины и гипервизора" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Datacenter" class="form-label">Датацентр</label>
                    <input name="Datacenter" id="Datacenter" class="form-control" type="text" value="{{.record.Datacenter}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Cluster" class="form-label">Кластер</label>
                    <input name="Cluster" id="Cluster" class="form-control" type="text" value="{{.record.Cluster}}" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Классификация</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-3">
                    <label for="TypeMicroservice" class="form-label">Роль в микросервисах</label>
                    <select name="TypeMicroservice" id="TypeMicroservice" class="form-select">
                        <option value="0" {{if eq .record.TypeMicroservice 0}}selected{{end}}>не микросервис</option>
                        <option value="1" {{if eq .record.TypeMicroservice 1}}selected{{end}}>Docker</option>
                        <option value="2" {{if eq .record.TypeMicroservice 2}}selected{{end}}>Kubernetes</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="RoleType" class="form-label">APP/DB сервер</label>
                    <select name="RoleType" id="RoleType" class="form-select">
                        <option value="0" {{if eq .record.RoleType 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.RoleType 1}}selected{{end}}>APP сервер</option>
                        <option value="2" {{if eq .record.RoleType 2}}selected{{end}}>DB сервер</option>
                        <option value="3" {{if eq .record.RoleType 3}}selected{{end}}>DB+APP сервер</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="TypeWork" class="form-label">Тип среды</label>
                    <select name="TypeWork" id="TypeWork" class="form-select">
                        <option value="0" {{if eq .record.TypeWork 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.TypeWork 1}}selected{{end}}>Продуктовая</option>
                        <option value="2" {{if eq .record.TypeWork 2}}selected{{end}}>Внутренний тест</option>
                        <option value="3" {{if eq .record.TypeWork 3}}selected{{end}}>Внешний тест</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="CriticalLevel" class="form-label">Уровень критичности</label>
                    <select name="CriticalLevel" id="CriticalLevel" class="form-select">
                        <option value="0" {{if eq .record.CriticalLevel 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.CriticalLevel 1}}selected{{end}}>Vital</option>
                        <option value="2" {{if eq .record.CriticalLevel 2}}selected{{end}}>Critical</option>
                        <option value="3" {{if eq .record.CriticalLevel 3}}selected{{end}}>Secondary</option>
                        <option value="4" {{if eq .record.CriticalLevel 4}}selected{{end}}>Non Priority</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Active" class="form-label">Состояние</label>
                    <select name="Active" id="Active" class="form-select">
                        <option value="1" {{if eq .record.Active 1}}selected{{end}}>Активен</option>
                        <option value="2" {{if eq .record.Active 2}}selected{{end}}>Выведен из эксплуатации</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Monitoring" class="form-label">Мониторинг</label>
                    <select name="Monitoring" id="Monitoring" class="form-select">
                        <option value="0" {{if eq .record.Monitoring 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.Monitoring 1}}selected{{end}}>включен</option>
                        <option value="2" {{if eq .record.Monitoring 2}}selected{{end}}>отключен</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="MonitoringSZ" class="form-label">Основание постановки на мониторинг</label>
                    <input name="MonitoringSZ" id="MonitoringSZ" class="form-control" type="text" value="{{.record.MonitoringSZ}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Backuped" class="form-label">Резервное копирование</label>
                    <select name="Backuped" id="Backuped" class="form-select">
                        <option value="0" {{if eq .record.Backuped 0}}selected{{end}}>не указано</option>
                        <option value="1" {{if eq .record.Backuped 1}}selected{{end}}>включено</option>
                        <option value="2" {{if eq .record.Backuped 2}}selected{{end}}>отключено</option>
                    </select>
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Идентификация и сеть</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-4">
                    <label for="Name" class="form-label">Имя сервера</label>
                    <input name="Name" id="Name" class="form-control" type="text" value="{{.record.Name}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="NameFQDN" class="form-label">Полное имя (FQDN)</label>
                    <input name="NameFQDN" id="NameFQDN" class="form-control" type="text" value="{{.record.NameFQDN}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="NameInfo" class="form-label">Альтернативное имя</label>
                    <input name="NameInfo" id="NameInfo" class="form-control" type="text" value="{{.record.NameInfo}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="IPAddress" class="form-label">IP адрес(а)</label>
                    <input name="IPAddress" id="IPAddress" class="form-control" type="text" value="{{.record.IPAddress}}" title="Список через разделитель ;" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="MacAddress" class="form-label">MAC адрес(а)</label>
                    <input name="MacAddress" id="MacAddress" class="form-control" type="text" value="{{.record.MacAddress}}" title="Список через разделитель ;" />
                </div>
                <div class="col-sm-6 col-md-2">
                    <label for="Network" class="form-label">Сеть</label>
                    <input name="Network" id="Network" class="form-control" type="text" value="{{.record.Network}}" title="Например: vlan144" />
                </div>
                <div class="col-sm-6 col-md-2">
                    <label for="NetworkName" class="form-label">Наименование сети</label>
                    <input name="NetworkName" id="NetworkName" class="form-control" type="text" value="{{.record.NetworkName}}" title="Например: DMZ, Prom" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Ответственные</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-4">
                    <label for="Owner" class="form-label">Ответственный</label>
                    <input name="Owner" id="Owner" class="form-control" type="text" value="{{.record.Owner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="BusinessOwner" class="form-label">Владелец от бизнеса</label>
                    <input name="BusinessOwner" id="BusinessOwner" class="form-control" type="text" value="{{.record.BusinessOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ITOwner" class="form-label">Владелец со стороны ИТ</label>
                    <input name="ITOwner" id="ITOwner" class="form-control" type="text" value="{{.record.ITOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ProductOwner" class="form-label">Владелец продукта</label>
                    <input name="ProductOwner" id="ProductOwner" class="form-control" type="text" value="{{.record.ProductOwner}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Support" class="form-label">Ответственные за поддержку</label>
                    <input name="Support" id="Support" class="form-control" type="text" value="{{.record.Support}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="SupportTeam" class="form-label">Подразделение поддержки</label>
                    <input name="SupportTeam" id="SupportTeam" class="form-control" type="text" value="{{.record.SupportTeam}}" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Система и назначение</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-12">
                    <label class="form-label">ИТ система</label>
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
                    <!-- источник данных справочника ИТ-систем для поиска на клиенте - сами
                         элементы скрыты, JS читает их data-атрибуты один раз при загрузке
                         страницы (см. скрипт ниже); html/template экранирует значения, что
                         избавляет от ручного экранирования при встраивании в JSON/JS -->
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
                <div class="col-sm-6 col-md-3">
                    <label for="OS" class="form-label">Операционная система</label>
                    <input name="OS" id="OS" class="form-control" type="text" value="{{.record.OS}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SystemType" class="form-label">Тип системы</label>
                    <input name="SystemType" id="SystemType" class="form-control" type="text" value="{{.record.SystemType}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="LicensingType" class="form-label">Тип лицензирования</label>
                    <input name="LicensingType" id="LicensingType" class="form-control" type="text" value="{{.record.LicensingType}}" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SolutionType" class="form-label">Вариант решения</label>
                    <input name="SolutionType" id="SolutionType" class="form-control" type="text" value="{{.record.SolutionType}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Users" class="form-label">Пользователи системы</label>
                    <input name="Users" id="Users" class="form-control" type="text" value="{{.record.Users}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Allocation" class="form-label">Аллокация (подразделение)</label>
                    <input name="Allocation" id="Allocation" class="form-control" type="text" value="{{.record.Allocation}}" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Role" class="form-label">Роль сервера</label>
                    <input name="Role" id="Role" class="form-control" type="text" value="{{.record.Role}}" title="Например: сервер баз данных, сервер приложений, хост" />
                </div>
                <div class="col-12">
                    <label for="Anotation" class="form-label">Описание сервера</label>
                    <textarea name="Anotation" id="Anotation" class="form-control" rows="2">{{.record.Anotation}}</textarea>
                </div>
                <div class="col-12">
                    <label for="Details" class="form-label">Примечание</label>
                    <textarea name="Details" id="Details" class="form-control" rows="2">{{.record.Details}}</textarea>
                </div>
                <div class="col-12">
                    <label class="form-label">Источник последнего обновления</label>
                    <div>
                        <span class="badge {{if eq .record.SourceUpdateType 1}}text-bg-secondary{{else if eq .record.SourceUpdateType 2}}text-bg-info{{else if eq .record.SourceUpdateType 3}}text-bg-primary{{else if eq .record.SourceUpdateType 4}}text-bg-warning{{else}}text-bg-light{{end}}">
                            {{if eq .record.SourceUpdateType 1}}Вручную{{else if eq .record.SourceUpdateType 2}}VMware{{else if eq .record.SourceUpdateType 3}}HPE OneView{{else if eq .record.SourceUpdateType 4}}Zabbix{{else}}Не указано{{end}}
                        </span>
                        <span class="small text-muted ms-1">проставляется автоматически при синхронизации/сохранении, вручную не редактируется</span>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Расположение и обслуживание</div>
        <div class="card-body">
            <div class="row g-3">
                {{if .PlacementText}}
                <div class="col-12">
                    <label class="form-label">Физическое размещение (ЦОД / Шкаф)</label>
                    <div class="form-control-plaintext">
                        {{.PlacementText}}
                        {{if .PlacementHWSprId}}
                        <a href="/sprsrv/hwconfedit/{{.PlacementHWSprId}}" class="ms-2 small" title="Изменить размещение в карточке оборудования">изменить в карточке оборудования</a>
                        {{end}}
                    </div>
                </div>
                {{end}}
                <div class="col-sm-8 col-md-6">
                    <label for="DPCOffice" class="form-label">ЦОД: адрес и площадка</label>
                    <select name="DPCOffice" id="DPCOffice" class="form-select">
                        <option value="0" {{if eq .record.DPCOffice 0}}selected{{end}}>не выбрано</option>
                        <option value="101" {{if eq .record.DPCOffice 101}}selected{{end}}>623к, г. Новосибирск, ул. Советская 37</option>
                        <option value="102" {{if eq .record.DPCOffice 102}}selected{{end}}>624к, г. Новосибирск, ул. Советская 37</option>
                        <option value="103" {{if eq .record.DPCOffice 103}}selected{{end}}>316к, г. Новосибирск, ул. Советская 37</option>
                        <option value="121" {{if eq .record.DPCOffice 121}}selected{{end}}>Мегафон, г. Новосибирск, ул. Толмачевская 19а к3</option>
                        <option value="131" {{if eq .record.DPCOffice 131}}selected{{end}}>Ростелеком, г. Новосибирск, ул. Менделеева 1</option>
                        <option value="201" {{if eq .record.DPCOffice 201}}selected{{end}}>3-02 г. Москва, ул. Мира 69</option>
                        <option value="202" {{if eq .record.DPCOffice 202}}selected{{end}}>4-19 г. Москва, ул. Мира 69</option>
                        <option value="281" {{if eq .record.DPCOffice 281}}selected{{end}}>DSP г. Москва, ул. Шарикоподшипниковская улица 11/9</option>
                        <option value="291" {{if eq .record.DPCOffice 291}}selected{{end}}>М1 г. Москва, ул. Варшавское шоссе 125</option>
                    </select>
                </div>
                <div class="col-sm-4 col-md-6">
                    <label for="SZHistory" class="form-label">Номера СЗ</label>
                    <input name="SZHistory" id="SZHistory" class="form-control" type="text" value="{{.record.SZHistory}}" title="Через пробел, в хронологическом порядке" />
                </div>
            </div>
        </div>
    </div>

    <div class="d-flex gap-2">
        <input type="submit" title="{{if .IsNew}}Создать сервер{{else}}Сохранить изменения{{end}}" value="{{if .IsNew}}Создать{{else}}Сохранить{{end}}" class="btn btn-primary rounded-pill px-4" />
        <a href="/sprsrv/view2/{{.backmode}}/0" title="Вернуться без сохранения" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>
