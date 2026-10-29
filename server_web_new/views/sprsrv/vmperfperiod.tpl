<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Загрузка виртуальной машины за период</h2>
        <div class="text-muted">
            <b>{{.CurVcenter}}</b> / <b>{{.CurDatacenter}}</b> / <b>{{.CurCluster}}</b> / <b>{{.VMName}}</b>
            {{if ne .VMHostName ""}}({{.VMHostName}}){{end}}
        </div>
    </div>
    {{if ne .RecordCount 0}}
    <button type="button" id="vmPerfSavePdfBtn" class="btn btn-outline-primary rounded-pill px-3" onclick="vmSaveChartsToPdf()">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" class="me-1" aria-hidden="true"><path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V4.5L9.5 0H4Zm5 1.5L12.5 5H9.5V1.5ZM4.5 8h1.25a1.25 1.25 0 1 1 0 2.5H5v1.5H4V8h.5Zm.75 1.7a.45.45 0 0 0 0-.9H5v.9h.25Zm2.15-1.7H9v.9H8.15v.55H8.9v.9H8.15V12H7.4V8Zm3.35 0h1.1c.8 0 1.35.65 1.35 2s-.55 2-1.35 2h-1.1V8Zm.75 3.2h.3c.35 0 .6-.35.6-1.2s-.25-1.2-.6-1.2h-.3v2.4Z"/></svg>
        Сохранить графики в PDF
    </button>
    {{end}}
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<form id="vmPeriodFilterForm" class="vm-period-filter-bar mb-3 p-3 rounded border" method="get">
    <div class="row g-3 align-items-end">
        <div class="col-6 col-md-3">
            <label for="filterFrom" class="form-label">Дата начала</label>
            <input type="date" id="filterFrom" name="from" class="form-control" value="{{.FromDate}}">
        </div>
        <div class="col-6 col-md-3">
            <label for="filterTo" class="form-label">Дата окончания</label>
            <input type="date" id="filterTo" name="to" class="form-control" value="{{.ToDate}}">
        </div>
        <div class="col-6 col-md-3">
            <button type="submit" class="btn btn-primary rounded-pill px-3">Показать</button>
        </div>
    </div>
    <div class="row mt-2">
        <div class="col-12 text-muted small">Найдено записей за период: {{.RecordCount}}</div>
    </div>
</form>

{{if eq .RecordCount 0}}
<div class="alert alert-info" role="alert">
    За выбранный период данных по этой виртуальной машине не найдено.
</div>
{{else}}
<div id="vmPerfCharts" class="row g-3"></div>
{{end}}

<script src="/static/js/apexcharts.min.js?v={{.StaticVer}}"></script>
<script src="/static/js/jspdf.umd.min.js?v={{.StaticVer}}"></script>
<script src="/static/js/pdf-export.js?v={{.StaticVer}}"></script>

<style>
    .vm-period-filter-bar {
        background-color: var(--bs-body-bg);
        border-color: var(--bs-border-color) !important;
    }

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

        var container = document.getElementById('vmPerfCharts');
        if (!container) return;

        var c = (window.LoadmonApexTheme && LoadmonApexTheme.colors()) || { mode: 'light', foreColor: '#334155', grid: 'rgba(15,23,42,0.12)' };
        var apexTheme = c.mode;
        var gridColor = c.grid;

        // цвета: минимум/среднее/максимум - холодная-нейтральная-тёплая
        // гамма, "текущее значение" (raw, из VMPerfRecord, только за
        // последние 3 суток - см. vmPerfRetention в
        // cmdb/savemetric_orchestration.go) - отдельным контрастным цветом,
        // сплошной линией поверх агрегатов (тот же общий график/ось времени,
        // просто более частые точки на своём хвосте периода)
        var colorMin = '#4dabf7';
        var colorAvg = '#7950f2';
        var colorMax = '#e64980';
        var colorRaw = '#f59f00';

        // экземпляры отрисованных графиков (для последующей выгрузки в PDF -
        // см. vmSaveChartsToPdf ниже)
        var renderedCharts = [];

        chartsData.forEach(function (chart, idx) {
            var hasAgg = (chart.min && chart.min.length) || (chart.avg && chart.avg.length) || (chart.max && chart.max.length);
            var hasRaw = chart.raw && chart.raw.length;
            if (!hasAgg && !hasRaw) return;

            var col = document.createElement('div');
            col.className = 'col-12 col-xl-6';

            var card = document.createElement('div');
            card.className = 'vm-perf-card';

            var title = document.createElement('h6');
            title.textContent = chart.title + (chart.unit ? ' (' + chart.unit + ')' : '');
            card.appendChild(title);

            var wrap = document.createElement('div');
            wrap.className = 'vm-perf-chart-wrap';
            wrap.id = 'vmPerfChart' + idx;
            card.appendChild(wrap);

            col.appendChild(card);
            container.appendChild(col);

            function toXY(points) {
                return (points || []).map(function (p) { return [p.t, p.v]; });
            }

            var series = [];
            var colors = [];
            var strokeCurves = [];
            var strokeWidths = [];

            if (chart.min && chart.min.length) {
                series.push({ name: 'Минимум', type: 'area', data: toXY(chart.min) });
                colors.push(colorMin); strokeCurves.push('smooth'); strokeWidths.push(1.5);
            }
            if (chart.avg && chart.avg.length) {
                series.push({ name: 'Среднее', type: 'area', data: toXY(chart.avg) });
                colors.push(colorAvg); strokeCurves.push('smooth'); strokeWidths.push(1.5);
            }
            if (chart.max && chart.max.length) {
                series.push({ name: 'Максимум', type: 'area', data: toXY(chart.max) });
                colors.push(colorMax); strokeCurves.push('smooth'); strokeWidths.push(1.5);
            }
            if (hasRaw) {
                // Раньше здесь был type: 'scatter' + strokeWidths: 0 - точки
                // без соединяющей линии, из-за чего на плотных данных
                // VMPerf (частые замеры за последние 3 суток) визуально
                // получался "набор жирных точек", а не график. Меняем на
                // обычную линию (как у Min/Avg/Max), чтобы этот "хвост"
                // периода смотрелся продолжением общего графика, а не
                // отдельным облаком точек.
                series.push({ name: 'Текущее значение', type: 'line', data: toXY(chart.raw) });
                colors.push(colorRaw); strokeCurves.push('straight'); strokeWidths.push(2);
            }

            var options = {
                chart: {
                    height: 340,
                    background: 'transparent',
                    foreColor: c.foreColor,
                    toolbar: { show: true, tools: { download: false, zoom: true, pan: true, reset: true } },
                    zoom: { enabled: true },
                    animations: { enabled: false }
                },
                theme: { mode: apexTheme },
                series: series,
                colors: colors,
                stroke: { curve: strokeCurves, width: strokeWidths },
                // градиентная заливка под линиями минимума/среднего/максимума -
                // облегчает восприятие тренда; для линии "текущее значение"
                // (line) заливка не применяется - у неё и так свой цвет,
                // заливка только мешала бы читать пересечение с агрегатами
                fill: {
                    type: series.map(function (s) { return s.type === 'area' ? 'gradient' : 'solid'; }),
                    gradient: {
                        shadeIntensity: 1,
                        opacityFrom: 0.28,
                        opacityTo: 0.02,
                        stops: [0, 90, 100]
                    }
                },
                markers: { size: 0, hover: { sizeOffset: 3 } },
                dataLabels: { enabled: false },
                legend: { show: true, labels: { colors: c.foreColor } },
                xaxis: {
                    type: 'datetime',
                    labels: { datetimeUTC: false, format: 'dd.MM HH:mm' },
                    tooltip: { enabled: false }
                },
                yaxis: {
                    min: 0,
                    max: chart.unit === '%' ? 100 : undefined,
                    labels: {
                        formatter: function (v) { return (v === undefined || v === null) ? '' : v.toFixed(1); }
                    }
                },
                tooltip: {
                    theme: apexTheme,
                    shared: true,
                    x: { format: 'dd.MM.yyyy HH:mm:ss' },
                    y: { formatter: function (v) { return (v === undefined || v === null) ? '' : v.toFixed(2) + (chart.unit ? ' ' + chart.unit : ''); } }
                },
                grid: { borderColor: gridColor }
            };

            var apexChart = new ApexCharts(wrap, options);
            apexChart.render();
            if (window.LoadmonApexTheme) LoadmonApexTheme.register(apexChart);
            renderedCharts.push({ title: chart.title + (chart.unit ? ' (' + chart.unit + ')' : ''), chart: apexChart });
        });

        // Сохранение всего набора графиков в один стильный PDF-документ - см.
        // /static/js/pdf-export.js (vmBuildPdfPage): весь текст страницы
        // рисуется на canvas, чтобы корректно поддерживалась кириллица
        // (встроенные шрифты jsPDF её не содержат).
        window.vmSaveChartsToPdf = function () {
            var btn = document.getElementById('vmPerfSavePdfBtn');
            if (!renderedCharts.length || !window.jspdf || !window.vmBuildPdfPage) return;
            if (btn) { btn.disabled = true; btn.textContent = 'Формирование PDF...'; }

            var jsPDF = window.jspdf.jsPDF;
            var pdf = new jsPDF({ orientation: 'landscape', unit: 'pt', format: 'a4' });

            var meta = {
                subTitle: '{{.CurVcenter}} / {{.CurDatacenter}} / {{.CurCluster}} / {{.VMName}}{{if ne .VMHostName ""}} ({{.VMHostName}}){{end}}',
                periodLabel: 'Период: с {{.FromDate}} по {{.ToDate}} (записей: {{.RecordCount}})'
            };

            var chain = Promise.resolve();
            renderedCharts.forEach(function (item, i) {
                chain = chain.then(function () {
                    return vmCaptureChartImage(item.chart, 2).then(function (imgURI) {
                        return vmBuildPdfPage(pdf, {
                            subTitle: meta.subTitle,
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
                var fname = 'vm_perf_period_{{.CurVM}}_{{.FromDate}}_{{.ToDate}}.pdf';
                pdf.save(fname.replace(/[^A-Za-z0-9_.\-]/g, '_'));
            }).finally(function () {
                if (btn) { btn.disabled = false; btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" class="me-1" aria-hidden="true"><path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V4.5L9.5 0H4Zm5 1.5L12.5 5H9.5V1.5ZM4.5 8h1.25a1.25 1.25 0 1 1 0 2.5H5v1.5H4V8h.5Zm.75 1.7a.45.45 0 0 0 0-.9H5v.9h.25Zm2.15-1.7H9v.9H8.15v.55H8.9v.9H8.15V12H7.4V8Zm3.35 0h1.1c.8 0 1.35.65 1.35 2s-.55 2-1.35 2h-1.1V8Zm.75 3.2h.3c.35 0 .6-.35.6-1.2s-.25-1.2-.6-1.2h-.3v2.4Z"/></svg> Сохранить графики в PDF'; }
            });
        };
    })();
</script>
