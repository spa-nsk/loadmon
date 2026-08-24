<div class="hero-text mb-3">
    <h2>Выгрузка метрик утилизации</h2>
    <p class="text-muted">
        Единообразная выгрузка метрик CPU / память / диск / сеть по виртуальным машинам и
        аппаратным серверам, зарегистрированным в Zabbix, за произвольный период.
        Период до {{.DetailedPeriodLimitDays}} суток включительно выгружается подробно
        (минутная сетка, значение "продлевается" от последнего известного замера), более
        длинный период - по дням (среднее за календарные сутки). Детальные метрики
        виртуальных машин хранятся только {{.DetailedPeriodLimitDays}} суток и не пишутся
        для рабочих станций Windows - подробности в комментарии к
        <code>AdminMetricsUnifiedExport</code> (controllers/adminmetricsexport.go) и в
        kontext/03-data-model.md.
    </p>
    {{if .Detailed}}
    <p class="small text-success mb-0">Выбранный период ({{.FromDate}} — {{.ToDate}}) уложится в подробный (минутный) режим выгрузки.</p>
    {{else}}
    <p class="small text-warning mb-0">Выбранный период ({{.FromDate}} — {{.ToDate}}) длиннее {{.DetailedPeriodLimitDays}} суток - выгрузка будет по дням (среднее за сутки), не по минутам.</p>
    {{end}}
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form role="form" id="MetricsExportForm" class="metricsexport-filter-bar mb-3 p-3 rounded border" method="GET">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="filterFrom" class="form-label">Период с</label>
            <input type="date" id="filterFrom" name="from" class="form-control" value="{{.FromDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterTo" class="form-label">Период по</label>
            <input type="date" id="filterTo" name="to" class="form-control" value="{{.ToDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterName" class="form-label">Имя сервера/ВМ содержит</label>
            <input type="text" id="filterName" name="name" class="form-control"
                placeholder="необязательно" value="{{.CurName}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterITSystem" class="form-label">Код ИТ-системы</label>
            <input type="text" id="filterITSystem" name="itsystem" class="form-control"
                placeholder="необязательно, точное совпадение" value="{{.CurITSystem}}">
        </div>
    </div>
    <div class="row g-3 align-items-center mt-1">
        <div class="col-auto">
            <div class="form-check">
                <input class="form-check-input" type="checkbox" id="includeVM" name="includevm"
                    value="on" {{if .IncludeVM}}checked{{end}}>
                <label class="form-check-label" for="includeVM">Виртуальные машины (по данным vm_record)</label>
            </div>
        </div>
        <div class="col-auto">
            <div class="form-check">
                <input class="form-check-input" type="checkbox" id="includeZabbix" name="includezabbix"
                    value="on" {{if .IncludeZabbix}}checked{{end}}>
                <label class="form-check-label" for="includeZabbix">Аппаратные серверы в Zabbix (по данным zabbix_item)</label>
            </div>
        </div>
    </div>
    <div class="row mt-3">
        <div class="col-12 d-flex flex-wrap gap-2">
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="metricsExportSubmit('xlsx')"
                title="Сформировать и скачать XLSX за указанный период и фильтр">
                Сформировать выгрузку (XLSX)
            </button>
            <a href="/admin/metricsexport" class="btn btn-outline-secondary rounded-pill px-3">Сбросить фильтр</a>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12">
            <span class="text-muted small">
                Совет: чем длиннее период и шире фильтр (или без фильтра - выгрузка по всем
                серверам сразу), тем больше файл - минутная сетка за месяц по одному объекту
                и метрике уже даёт около 43 000 строк. Для больших выгрузок сужайте период
                и/или указывайте имя сервера/ИТ-системы.
            </span>
        </div>
    </div>
</form>

<!-- Скрытые "невидимые" поля checkbox для чекбоксов, снятых пользователем - HTML не
     отправляет невыставленный чекбокс вообще, а не значение "off"; контроллер
     (AdminMetricsUnifiedExport) трактует отсутствие параметра includevm/includezabbix как
     "включено" (this.GetString(...) != "off"), поэтому нужно явно передать "off" при снятой
     галочке - иначе снятие чекбокса в форме визуально не будет иметь эффекта. -->
<script>
    function metricsExportSubmit(exportFormat) {
        var form = document.getElementById('MetricsExportForm');

        function ensureHiddenFlag(checkboxId, hiddenName) {
            var cb = document.getElementById(checkboxId);
            var existing = document.getElementById(hiddenName + 'Hidden');
            if (existing) existing.remove();
            if (cb && !cb.checked) {
                var hidden = document.createElement('input');
                hidden.type = 'hidden';
                hidden.id = hiddenName + 'Hidden';
                hidden.name = hiddenName;
                hidden.value = 'off';
                form.appendChild(hidden);
            }
        }
        ensureHiddenFlag('includeVM', 'includevm');
        ensureHiddenFlag('includeZabbix', 'includezabbix');

        var exportField = document.getElementById('exportFieldHidden');
        if (exportField) exportField.remove();
        var hidden = document.createElement('input');
        hidden.type = 'hidden';
        hidden.id = 'exportFieldHidden';
        hidden.name = 'export';
        hidden.value = exportFormat;
        form.appendChild(hidden);

        form.submit();
    }
</script>
