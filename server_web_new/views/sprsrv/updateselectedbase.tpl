<div class="hero-text mb-3">
    <h2>Групповое редактирование серверов</h2>
    {{if .flash.error}}
    <blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
    {{end}}
    <p class="text-muted">
        Заполните только те поля, которые нужно изменить. Пустые поля и значения "не указано" / "не менять"
        не будут применены - у остальных серверов эти данные останутся прежними.
    </p>
</div>

<div class="card mb-3">
    <div class="card-header fw-semibold">Выбранные серверы ({{.SelectedCount}})</div>
    <div class="card-body">
        {{if gt .SelectedCount 0}}
        <div class="d-flex flex-wrap gap-2">
            {{range $rec := .SelectedRecords}}
            <span class="badge text-bg-secondary fs-6 fw-normal">{{$rec.Name}}</span>
            {{end}}
        </div>
        {{else}}
        <p class="text-muted mb-0">Список выбранных серверов пуст.</p>
        {{end}}
    </div>
</div>

<form role="form" id="UpdateSelectedBase" method="POST" class="pb-4" onsubmit="return confirmBulkSave();">

    <div class="card mb-3">
        <div class="card-header fw-semibold">Классификация</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-3">
                    <label for="TypeMicroservice" class="form-label">Роль в микросервисах</label>
                    <select name="TypeMicroservice" id="TypeMicroservice" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">Docker</option>
                        <option value="2">Kubernetes</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="RoleType" class="form-label">APP/DB сервер</label>
                    <select name="RoleType" id="RoleType" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">APP сервер</option>
                        <option value="2">DB сервер</option>
                        <option value="3">DB+APP сервер</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="TypeWork" class="form-label">Тип среды</label>
                    <select name="TypeWork" id="TypeWork" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">Продуктовая</option>
                        <option value="2">Внутренний тест</option>
                        <option value="3">Внешний тест</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="CriticalLevel" class="form-label">Уровень критичности</label>
                    <select name="CriticalLevel" id="CriticalLevel" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">Vital</option>
                        <option value="2">Critical</option>
                        <option value="3">Secondary</option>
                        <option value="4">Non Priority</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Active" class="form-label">Состояние</label>
                    <select name="Active" id="Active" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">Активен</option>
                        <option value="2">Выведен из эксплуатации</option>
                    </select>
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
                    <input name="Owner" id="Owner" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="BusinessOwner" class="form-label">Владелец от бизнеса</label>
                    <input name="BusinessOwner" id="BusinessOwner" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ITOwner" class="form-label">Владелец со стороны ИТ</label>
                    <input name="ITOwner" id="ITOwner" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="ProductOwner" class="form-label">Владелец продукта</label>
                    <input name="ProductOwner" id="ProductOwner" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Support" class="form-label">Ответственные за поддержку</label>
                    <input name="Support" id="Support" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="SupportTeam" class="form-label">Подразделение поддержки</label>
                    <input name="SupportTeam" id="SupportTeam" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Система и назначение</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-12">
                    <label for="ITSystemKey" class="form-label">ИТ система</label>
                    <select name="ITSystemKey" id="ITSystemKey" class="form-select">
                        <option value="">Не менять</option>
                        {{range $its := .ITSystemSpr}}
                        <option value="{{$its.ObjectKey}}">
                            {{$its.ObjectKey}} {{$its.Label}}: {{$its.FullName}} {{$its.Status}} {{$its.SystemType}} {{$its.BusinessColumn}} {{$its.CFO}}
                        </option>
                        {{end}}
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="OS" class="form-label">Операционная система</label>
                    <input name="OS" id="OS" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SystemType" class="form-label">Тип системы</label>
                    <input name="SystemType" id="SystemType" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="LicensingType" class="form-label">Тип лицензирования</label>
                    <input name="LicensingType" id="LicensingType" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="SolutionType" class="form-label">Вариант решения</label>
                    <input name="SolutionType" id="SolutionType" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Users" class="form-label">Пользователи системы</label>
                    <input name="Users" id="Users" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Allocation" class="form-label">Аллокация (подразделение)</label>
                    <input name="Allocation" id="Allocation" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-4">
                    <label for="Role" class="form-label">Роль сервера</label>
                    <input name="Role" id="Role" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-sm-6 col-md-6">
                    <label for="NetworkName" class="form-label">Наименование сети</label>
                    <input name="NetworkName" id="NetworkName" class="form-control" type="text" placeholder="оставить без изменений" title="Например: DMZ, Prom" />
                </div>
                <div class="col-12">
                    <label for="Anotation" class="form-label">Описание сервера</label>
                    <textarea name="Anotation" id="Anotation" class="form-control" rows="2" placeholder="оставить без изменений"></textarea>
                </div>
                <div class="col-12">
                    <label for="Details" class="form-label">Примечание</label>
                    <textarea name="Details" id="Details" class="form-control" rows="2" placeholder="оставить без изменений"></textarea>
                </div>
            </div>
        </div>
    </div>

    <div class="card mb-3">
        <div class="card-header fw-semibold">Расположение и обслуживание</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-4 col-md-6">
                    <label for="SZHistory" class="form-label">Номера СЗ</label>
                    <input name="SZHistory" id="SZHistory" class="form-control" type="text" placeholder="оставить без изменений" title="Через пробел, в хронологическом порядке" />
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Monitoring" class="form-label">Мониторинг</label>
                    <select name="Monitoring" id="Monitoring" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">включен</option>
                        <option value="2">отключен</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="Backuped" class="form-label">Резервное копирование</label>
                    <select name="Backuped" id="Backuped" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">включено</option>
                        <option value="2">отключено</option>
                    </select>
                </div>
                <div class="col-sm-12 col-md-6">
                    <label for="MonitoringSZ" class="form-label">Основание постановки на мониторинг</label>
                    <input name="MonitoringSZ" id="MonitoringSZ" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
            </div>
        </div>
    </div>

    <div class="d-flex gap-2">
        <input type="submit" title="Применить заполненные поля ко всем выбранным серверам" value="Применить к выбранным" class="btn btn-primary rounded-pill px-4" />
        <a href="/sprsrv/view2/all/0" title="Вернуться без сохранения" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>

<!-- модальные окна подтверждения/предупреждения вместо нативных alert()/confirm(),
     чтобы соблюдалась текущая тема оформления (data-bs-theme) и окно было
     по центру экрана, а не прижато к верхнему краю -->
<div class="modal fade" id="bulkEmptyModal" tabindex="-1" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">Нечего применять</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
            </div>
            <div class="modal-body">
                Не заполнено ни одно поле - нечего применять к выбранным серверам.
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-primary rounded-pill px-4" data-bs-dismiss="modal">Понятно</button>
            </div>
        </div>
    </div>
</div>

<div class="modal fade" id="bulkConfirmModal" tabindex="-1" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">Подтверждение группового изменения</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
            </div>
            <div class="modal-body">
                Изменения будут применены к <strong id="bulkConfirmCount"></strong> выбранным серверам. Продолжить?
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-outline-secondary rounded-pill px-4" data-bs-dismiss="modal">Отмена</button>
                <button type="button" id="bulkConfirmOkBtn" class="btn btn-primary rounded-pill px-4">Применить</button>
            </div>
        </div>
    </div>
</div>

<script>
    function formHasFilledFields(form) {
        var hasValue = false;
        Array.prototype.forEach.call(form.querySelectorAll('input[type=text], textarea'), function (el) {
            if (el.value.trim() !== '') hasValue = true;
        });
        Array.prototype.forEach.call(form.querySelectorAll('select'), function (el) {
            if (el.value !== '0' && el.value !== '') hasValue = true;
        });
        return hasValue;
    }

    function confirmBulkSave() {
        var count = {{.SelectedCount}};
        var form = document.getElementById('UpdateSelectedBase');

        if (!formHasFilledFields(form)) {
            var emptyModal = new bootstrap.Modal(document.getElementById('bulkEmptyModal'));
            emptyModal.show();
            return false;
        }

        document.getElementById('bulkConfirmCount').textContent = count;
        var confirmModalEl = document.getElementById('bulkConfirmModal');
        var confirmModal = new bootstrap.Modal(confirmModalEl);

        var okBtn = document.getElementById('bulkConfirmOkBtn');
        var onConfirm = function () {
            okBtn.removeEventListener('click', onConfirm);
            confirmModal.hide();
            form.submit();
        };
        okBtn.addEventListener('click', onConfirm);

        confirmModal.show();
        return false; // отправка формы произойдёт из onConfirm после подтверждения
    }
</script>
