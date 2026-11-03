<script>
// Смена дат периода сразу перезагружает страницу с новыми GET-параметрами -
// без отдельной кнопки "Применить" для полей, аналогично /sprsrv/clusterperioddb.
function zabbixPeriodNavigate() {
    var from = document.getElementById('fromDate').value;
    var to = document.getElementById('toDate').value;
    var enc = encodeURIComponent;
    window.location.href = '/sprsrv/zabbixperfperiod/{{.HostId}}?from=' + enc(from) + '&to=' + enc(to);
}

// Закрепление панели фильтра вверху экрана можно отключить чекбоксом -
// приём идентичен /sprsrv/clusterperioddb (сохраняется отдельным ключом в localStorage).
(function () {
    var STORAGE_KEY = 'zabbixperfperiod.filterSticky';

    function init() {
        var bar = document.querySelector('.zabbix-filter-bar');
        var toggle = document.getElementById('zabbixPeriodFilterStickyToggle');
        if (!bar || !toggle) return;

        function applySticky(sticky) {
            bar.classList.toggle('zabbix-filter-bar-unpinned', !sticky);
            toggle.checked = sticky;
        }

        var stored = null;
        try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
        applySticky(stored === null ? true : stored === '1');

        window.zabbixPeriodToggleFilterSticky = function (sticky) {
            applySticky(sticky);
            try { localStorage.setItem(STORAGE_KEY, sticky ? '1' : '0'); } catch (e) {}
        };
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
</script>

<!-- Стили закрепляемой панели фильтра, "классической" таблицы и т.д. вынесены
     в общий server_web_new/static/css/loadmon.css (подключён в header.tpl);
     оформление самих графиков (ApexCharts, PDF-экспорт) идентично
     /sprsrv/vmperfday, см. views/sprsrv/vmperfday.tpl - образец стиля. -->

<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Загрузка аппаратного сервера за период (по данным Zabbix){{if .ServerTitleName}}
            — <span class="fw-normal">{{.ServerTitleName}}</span>{{end}}
        </h2>
        {{if .ServerResourcesText}}
        <div class="text-muted small mt-1">{{.ServerResourcesText}}</div>
        {{end}}
        <div class="text-muted small mt-1">
            {{if and .HostName (ne .HostName .ServerTitleName)}}Zabbix: <b>{{.HostName}}</b>{{if ne .HostNameFQDN ""}} ({{.HostNameFQDN}}){{end}}{{end}}
            {{if gt .ServerId 0}}<a href="/sprsrv/updatebase/all/{{.ServerId}}" target="_blank" class="ms-2" title="Общие данные по серверу">Общие данные →</a>{{end}}
        </div>
    </div>
    {{if ne .PointCount 0}}
    <button type="button" id="zabbixPerfSavePdfBtn" class="btn btn-outline-primary rounded-pill px-3" onclick="zabbixSaveChartsToPdf()">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" class="me-1" aria-hidden="true"><path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V4.5L9.5 0H4Zm5 1.5L12.5 5H9.5V1.5ZM4.5 8h1.25a1.25 1.25 0 1 1 0 2.5H5v1.5H4V8h.5Zm.75 1.7a.45.45 0 0 0 0-.9H5v.9h.25Zm2.15-1.7H9v.9H8.15v.55H8.9v.9H8.15V12H7.4V8Zm3.35 0h1.1c.8 0 1.35.65 1.35 2s-.55 2-1.35 2h-1.1V8Zm.75 3.2h.3c.35 0 .6-.35.6-1.2s-.25-1.2-.6-1.2h-.3v2.4Z"/></svg>
        Сохранить графики в PDF
    </button>
    {{end}}
</div>

<div class="zabbix-filter-bar mb-3 p-3 rounded border">
    <div class="row g-3">
        <div class="col-6 col-md-3">
            <label for="fromDate" class="form-label">Период с</label>
            <input type="date" id="fromDate" class="form-control" value="{{.FromDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="toDate" class="form-label">по</label>
            <input type="date" id="toDate" class="form-control" value="{{.ToDate}}">
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 d-flex flex-wrap align-items-center justify-content-between gap-2">
            <button type="button" class="btn btn-primary rounded-pill px-3" onclick="zabbixPeriodNavigate()">Найти</button>
            <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" id="zabbixPeriodFilterStickyToggle" onchange="zabbixPeriodToggleFilterSticky(this.checked)">
                <label class="form-check-label small text-muted" for="zabbixPeriodFilterStickyToggle">Закреплять фильтр вверху экрана</label>
            </div>
        </div>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

{{if eq .PointCount 0}}
<div class="alert alert-info" role="alert">
    За выбранный период данных по метрикам Zabbix для этого сервера нет. Попробуйте расширить диапазон дат.
</div>
{{else}}
<div class="text-muted small mb-3">Точек за период: {{.PointCount}}. Каждая метрика показана тремя рядами - минимум/среднее/максимум по интервалам агрегации (интервал подобран автоматически по длине периода).</div>
<div id="zabbixPerfCharts" class="row g-3"></div>
{{end}}

<script src="/static/js/apexcharts.min.js?v={{.StaticVer}}"></script>
<script src="/static/js/jspdf.umd.min.js?v={{.StaticVer}}"></script>
<script src="/static/js/pdf-export.js?v={{.StaticVer}}"></script>

<style>
    .vm-perf-card {
        background-color: var(--bs-tertiary-bg);
        border: 1px solid var(--bs-border-color);
        border-radius: 0.6rem;
        padding: 1rem;
        height: 100%;
    }

    .vm-perf-card h6 {
        color: var(--bs-emphasis-color);
        font-weight: 600;
        margin-bottom: 0.75rem;
    }

    .vm-perf-card .vm-perf-chart-wrap {
        position: relative;
    }

    /* убираем белую подложку тулбара ApexCharts на тёмной теме */
    .apexcharts-toolbar {
        z-index: 5;
    }

    .apexcharts-menu {
        background: var(--bs-tertiary-bg) !important;
        border-color: var(--bs-border-color) !important;
        color: var(--bs-emphasis-color) !important;
    }

    .apexcharts-menu-item:hover {
        background: var(--bs-secondary-bg) !important;
    }
</style>

<script>
    (function () {
        var chartsData = {{.ChartsJSON}};
        if (!chartsData || !chartsData.length) return;

        var container = document.getElementById('zabbixPerfCharts');
        if (!container) return;

        // цвета серии - минимум/среднее/максимум одного экземпляра метрики
        // рисуются оттенками одного цвета (макс - самый насыщенный), разные
        // экземпляры (диски/интерфейсы) - разными цветами палитры
        var palette = ['#2fb344', '#4dabf7', '#f59f00', '#e64980', '#7950f2', '#15aabf'];

        var c = (window.LoadmonApexTheme && LoadmonApexTheme.colors()) || { mode: 'light', foreColor: '#334155', grid: 'rgba(15,23,42,0.12)' };
        var apexTheme = c.mode;
        var gridColor = c.grid;

        // экземпляры отрисованных графиков (для последующей выгрузки в PDF -
        // см. zabbixSaveChartsToPdf ниже)
        var renderedCharts = [];

        chartsData.forEach(function (chart, idx) {
            var seriesNames = Object.keys(chart.series || {});
            if (!seriesNames.length) return;

            var col = document.createElement('div');
            col.className = 'col-12 col-xl-6';

            var card = document.createElement('div');
            card.className = 'vm-perf-card';

            var title = document.createElement('h6');
            title.textContent = chart.title + (chart.unit ? ' (' + chart.unit + ')' : '');
            card.appendChild(title);

            var wrap = document.createElement('div');
            wrap.className = 'vm-perf-chart-wrap';
            wrap.id = 'zabbixPerfChart' + idx;
            card.appendChild(wrap);

            col.appendChild(card);
            container.appendChild(col);

            // группируем ряды по экземпляру (текст до последней запятой в имени
            // ряда - см. controllers/sprsrvzabbixperfperiod.go: "<label>, мин/сред/макс"),
            // чтобы назначить одному экземпляру один цвет для всех трёх рядов
            var instanceNames = [];
            var instanceOf = {};
            seriesNames.forEach(function (name) {
                var inst = name.replace(/, (мин|сред|макс)$/, '');
                if (instanceNames.indexOf(inst) === -1) instanceNames.push(inst);
                instanceOf[name] = inst;
            });

            var series = seriesNames.map(function (name) {
                var points = (chart.series[name] || []).map(function (p) {
                    return [p.t, p.v];
                });
                return { name: name, data: points };
            });

            var colors = seriesNames.map(function (name) {
                var instIdx = instanceNames.indexOf(instanceOf[name]);
                return palette[instIdx % palette.length];
            });
            // толщина линии по роли ряда - "макс"/"мин" тоньше и полупрозрачнее среднего,
            // чтобы не перегружать график при нескольких экземплярах метрики сразу
            var widths = seriesNames.map(function (name) {
                return / сред$/.test(name) ? 2.5 : 1;
            });
            var dashes = seriesNames.map(function (name) {
                return / мин$/.test(name) ? 3 : (/ макс$/.test(name) ? 3 : 0);
            });

            var options = {
                chart: {
                    type: 'line',
                    height: 320,
                    background: 'transparent',
                    foreColor: c.foreColor,
                    toolbar: { show: true, tools: { download: false, zoom: true, pan: true, reset: true } },
                    zoom: { enabled: true },
                    animations: { enabled: false },
                    dropShadow: { enabled: false }
                },
                theme: { mode: apexTheme },
                series: series,
                colors: colors,
                stroke: { curve: 'smooth', width: widths, dashArray: dashes },
                dataLabels: { enabled: false },
                markers: { size: 0 },
                legend: { show: true, labels: { colors: c.foreColor } },
                xaxis: {
                    type: 'datetime',
                    labels: {
                        datetimeUTC: false,
                        datetimeFormatter: { year: 'yyyy', month: "MMM 'yy", day: 'dd MMM', hour: 'HH:mm' }
                    },
                    tooltip: { enabled: false }
                },
                yaxis: {
                    min: 0,
                    max: chart.unit === '%' ? 100 : undefined,
                    labels: {
                        // ГБ/МБ-с-значения обычно дробные и меньше 1 при малой
                        // загрузке - вторым знаком после запятой их не видно
                        // на "%"/мс-графиках, поэтому точность подбирается по
                        // единице измерения графика
                        formatter: function (v) {
                            if (v === undefined || v === null) return '';
                            var digits = (chart.unit === 'МБ/с' || chart.unit === 'ГБ') ? 2 : 1;
                            return v.toFixed(digits);
                        }
                    }
                },
                tooltip: {
                    theme: apexTheme,
                    shared: true,
                    x: { format: 'dd.MM HH:mm' },
                    y: { formatter: function (v) { return (v === undefined || v === null) ? '' : v.toFixed(2) + (chart.unit ? ' ' + chart.unit : ''); } }
                },
                grid: { borderColor: gridColor }
            };

            var apexChart = new ApexCharts(wrap, options);
            apexChart.render();
            if (window.LoadmonApexTheme) LoadmonApexTheme.register(apexChart);
            renderedCharts.push({ title: chart.title + (chart.unit ? ' (' + chart.unit + ')' : ''), chart: apexChart });
        });

        // Сохранение всего набора графиков в один PDF-файл - логика идентична
        // /sprsrv/vmperfday (см. подробный комментарий там и в pdf-export.js).
        window.zabbixSaveChartsToPdf = function () {
            var btn = document.getElementById('zabbixPerfSavePdfBtn');
            if (!renderedCharts.length || !window.jspdf || !window.vmBuildPdfPage) return;
            if (btn) { btn.disabled = true; btn.textContent = 'Формирование PDF...'; }

            var jsPDF = window.jspdf.jsPDF;
            var pdf = new jsPDF({ orientation: 'landscape', unit: 'pt', format: 'a4' });

            var meta = {
                docTitle: 'Loadmon — загрузка аппаратного сервера за период (Zabbix)',
                subTitle: '{{if .ServerTitleName}}{{.ServerTitleName}}{{else}}{{.HostName}}{{if ne .HostNameFQDN ""}} ({{.HostNameFQDN}}){{end}}{{end}}',
                resourcesLabel: '{{.ServerResourcesText}}',
                periodLabel: 'Период: {{.FromDate}} — {{.ToDate}}'
            };

            var chain = Promise.resolve();
            renderedCharts.forEach(function (item, i) {
                chain = chain.then(function () {
                    return vmCaptureChartImage(item.chart, 2).then(function (imgURI) {
                        return vmBuildPdfPage(pdf, {
                            docTitle: meta.docTitle,
                            subTitle: meta.subTitle,
                            resourcesLabel: meta.resourcesLabel,
                            periodLabel: meta.periodLabel,
                            chartTitle: item.title,
                            chartImgURI: imgURI,
                            pageNum: i + 1,
                            totalPages: renderedCharts.length,
                            isFirstPage: i === 0
                        });
                    });
                });
            });

            chain.then(function () {
                var fname = 'zabbix_perf_period_{{.HostName}}_{{.FromDate}}_{{.ToDate}}.pdf';
                pdf.save(fname.replace(/[^A-Za-z0-9_.\-]/g, '_'));
            }).finally(function () {
                if (btn) { btn.disabled = false; btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" class="me-1" aria-hidden="true"><path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V4.5L9.5 0H4Zm5 1.5L12.5 5H9.5V1.5ZM4.5 8h1.25a1.25 1.25 0 1 1 0 2.5H5v1.5H4V8h.5Zm.75 1.7a.45.45 0 0 0 0-.9H5v.9h.25Zm2.15-1.7H9v.9H8.15v.55H8.9v.9H8.15V12H7.4V8Zm3.35 0h1.1c.8 0 1.35.65 1.35 2s-.55 2-1.35 2h-1.1V8Zm.75 3.2h.3c.35 0 .6-.35.6-1.2s-.25-1.2-.6-1.2h-.3v2.4Z"/></svg> Сохранить графики в PDF'; }
            });
        };
    })();
</script>
