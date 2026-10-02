<!-- Групповое редактирование площадок/ЦОД - по образцу /sprsrv/updateselectedbase
     (см. controllers/sprdatacenterbulk.go). -->
<div class="hero-text mb-3">
    <h2>Групповое редактирование площадок/ЦОД</h2>
    {{if .flash.error}}
    <blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
    {{end}}
    {{if .flash.notice}}
    <blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
    {{end}}
    <p class="text-muted">
        Заполните только те поля, которые нужно изменить. Пустые поля и значения "Не менять"
        не будут применены - у остальных площадок эти данные останутся прежними.
    </p>
</div>

<div class="card mb-3">
    <div class="card-header fw-semibold">Выбранные площадки ({{.SelectedCount}})</div>
    <div class="card-body">
        {{if gt .SelectedCount 0}}
        <div class="d-flex flex-wrap gap-2">
            {{range $rec := .SelectedRecords}}
            <span class="badge text-bg-secondary fs-6 fw-normal">{{$rec.Name}}</span>
            {{end}}
        </div>
        {{else}}
        <p class="text-muted mb-0">Список выбранных площадок пуст.</p>
        {{end}}
    </div>
</div>

<form role="form" id="UpdateSelectedDC" method="POST" class="pb-4" onsubmit="return confirmBulkSave();">

    <div class="card mb-3">
        <div class="card-header fw-semibold">Данные площадки</div>
        <div class="card-body">
            <div class="row g-3">
                <div class="col-sm-6 col-md-3">
                    <label for="IsDatacenter" class="form-label">Признак ЦОД</label>
                    <select name="IsDatacenter" id="IsDatacenter" class="form-select">
                        <option value="0">Не менять</option>
                        <option value="1">ЦОД</option>
                        <option value="2">Не ЦОД (площадка)</option>
                    </select>
                </div>
                <div class="col-sm-6 col-md-3">
                    <label for="CodeJira" class="form-label">Код Jira</label>
                    <input name="CodeJira" id="CodeJira" class="form-control" type="text" placeholder="оставить без изменений" />
                </div>
                <div class="col-12">
                    <label for="Annotation" class="form-label">Описание</label>
                    <textarea name="Annotation" id="Annotation" class="form-control" rows="3" placeholder="оставить без изменений"></textarea>
                </div>
            </div>
        </div>
    </div>

    <div class="d-flex gap-2">
        <input type="submit" title="Применить заполненные поля ко всем выбранным площадкам" value="Применить к выбранным" class="btn btn-primary rounded-pill px-4" />
        <a href="/sprsrv/datacentersitelist/0" title="Вернуться без сохранения" class="btn btn-outline-secondary rounded-pill px-4">Отмена</a>
    </div>
</form>

<div class="modal fade" id="bulkEmptyModal" tabindex="-1" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">Нечего применять</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
            </div>
            <div class="modal-body">
                Не заполнено ни одно поле - нечего применять к выбранным площадкам.
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
                Изменения будут применены к <strong id="bulkConfirmCount"></strong> выбранным площадкам. Продолжить?
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
        var form = document.getElementById('UpdateSelectedDC');

        if (!formHasFilledFields(form)) {
            new bootstrap.Modal(document.getElementById('bulkEmptyModal')).show();
            return false;
        }

        document.getElementById('bulkConfirmCount').textContent = count;
        var confirmModal = new bootstrap.Modal(document.getElementById('bulkConfirmModal'));
        var okBtn = document.getElementById('bulkConfirmOkBtn');
        var onConfirm = function () {
            okBtn.removeEventListener('click', onConfirm);
            confirmModal.hide();
            form.submit();
        };
        okBtn.addEventListener('click', onConfirm);
        confirmModal.show();
        return false;
    }
</script>
