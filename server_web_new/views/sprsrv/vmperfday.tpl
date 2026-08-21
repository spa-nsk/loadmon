<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Загрузка виртуальной машины за текущий день</h2>
        <div class="text-muted">
            <b>{{.CurVcenter}}</b> / <b>{{.CurDatacenter}}</b> / <b>{{.CurCluster}}</b> / <b>{{.VMName}}</b>
            {{if ne .VMHostName ""}}({{.VMHostName}}){{end}}
        </div>
    </div>
    {{if ne .PerfCount 0}}
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

{{if eq .PerfCount 0}}
<div class="alert alert-info" role="alert">
    За текущий день данных по метрикам этой виртуальной машины пока нет.
</div>
{{else}}
<div class="text-muted small mb-3">Точек за сегодня: {{.PerfCount}}</div>
<div id="vmPerfCharts" class="row g-3"></div>
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

        var container = document.getElementById('vmPerfCharts');
        if (!container) return;

        // цвета серии - палитра, читаемая и на светлой, и на тёмной теме
        var palette = ['#2fb344', '#4dabf7', '#f59f00', '#e64980', '#7950f2', '#15aabf'];

        var isDark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
        var apexTheme = isDark ? 'dark' : 'light';
        var gridColor = isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)';

        // экземпляры отрисованных графиков (для последующей выгрузки в PDF -
        // см. vmSaveChartsToPdf ниже)
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
            wrap.id = 'vmPerfChart' + idx;
            card.appendChild(wrap);

            col.appendChild(card);
            container.appendChild(col);

            var series = seriesNames.map(function (name) {
                var points = (chart.series[name] || []).map(function (p) {
                    return [p.t, p.v];
                });
                return { name: name, data: points };
            });

            var colors = seriesNames.map(function (name, i) { return palette[i % palette.length]; });

            var options = {
                chart: {
                    type: 'area',
                    height: 320,
                    background: 'transparent',
                    toolbar: { show: true, tools: { download: false, zoom: true, pan: true, reset: true } },
                    zoom: { enabled: true },
                    animations: { enabled: false },
                    dropShadow: { enabled: false }
                },
                theme: { mode: apexTheme },
                series: series,
                colors: colors,
                stroke: { curve: 'smooth', width: 1.5 },
                // градиентная заливка под линией к оси X - облегчает восприятие
                // формы графика (тренд/пики), особенно на нескольких сериях сразу
                fill: {
                    type: 'gradient',
                    gradient: {
                        shadeIntensity: 1,
                        opacityFrom: 0.35,
                        opacityTo: 0.02,
                        stops: [0, 90, 100]
                    }
                },
                dataLabels: { enabled: false },
                markers: { size: 0 },
                legend: { show: seriesNames.length > 1, labels: { colors: 'var(--bs-emphasis-color)' } },
                xaxis: {
                    type: 'datetime',
                    labels: {
                        datetimeUTC: false,
                        format: 'HH:mm',
                        datetimeFormatter: { hour: 'HH:mm', minute: 'HH:mm' }
                    },
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
                    x: { format: 'HH:mm:ss' },
                    y: { formatter: function (v) { return (v === undefined || v === null) ? '' : v.toFixed(2) + (chart.unit ? ' ' + chart.unit : ''); } }
                },
                grid: { borderColor: gridColor }
            };

            var apexChart = new ApexCharts(wrap, options);
            apexChart.render();
            renderedCharts.push({ title: chart.title + (chart.unit ? ' (' + chart.unit + ')' : ''), chart: apexChart });
        });

        // Сохранение всего набора графиков в один PDF-файл. jsPDF со
        // встроенными (не встроенными в файл) шрифтами не поддерживает
        // кириллицу, поэтому весь текст страницы (шапка, подзаголовок,
        // название графика, подвал) рисуется на HTML5-canvas (шрифт браузера,
        // кириллица работает "из коробки"), туда же вставляется сам график
        // (PNG из ApexCharts) - в PDF попадает уже готовое изображение
        // страницы целиком (см. vmBuildPdfPage в pdf-export.js).
        window.vmSaveChartsToPdf = function () {
            var btn = document.getElementById('vmPerfSavePdfBtn');
            if (!renderedCharts.length || !window.jspdf || !window.vmBuildPdfPage) return;
            if (btn) { btn.disabled = true; btn.textContent = 'Формирование PDF...'; }

            var jsPDF = window.jspdf.jsPDF;
            var pdf = new jsPDF({ orientation: 'landscape', unit: 'pt', format: 'a4' });

            var meta = {
                docTitle: 'Loadmon — загрузка виртуальной машины за текущий день',
                subTitle: '{{.CurVcenter}} / {{.CurDatacenter}} / {{.CurCluster}} / {{.VMName}}{{if ne .VMHostName ""}} ({{.VMHostName}}){{end}}',
                periodLabel: 'Дата: ' + new Date().toLocaleDateString('ru-RU')
            };

            var chain = Promise.resolve();
            renderedCharts.forEach(function (item, i) {
                chain = chain.then(function () {
                    return vmCaptureChartImage(item.chart, 2).then(function (imgURI) {
                        return vmBuildPdfPage(pdf, {
                            docTitle: meta.docTitle,
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
                var fname = 'vm_perf_day_{{.CurVM}}_' + new Date().toISOString().slice(0, 10) + '.pdf';
                pdf.save(fname.replace(/[^A-Za-z0-9_.\-]/g, '_'));
            }).finally(function () {
                if (btn) { btn.disabled = false; btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" class="me-1" aria-hidden="true"><path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V4.5L9.5 0H4Zm5 1.5L12.5 5H9.5V1.5ZM4.5 8h1.25a1.25 1.25 0 1 1 0 2.5H5v1.5H4V8h.5Zm.75 1.7a.45.45 0 0 0 0-.9H5v.9h.25Zm2.15-1.7H9v.9H8.15v.55H8.9v.9H8.15V12H7.4V8Zm3.35 0h1.1c.8 0 1.35.65 1.35 2s-.55 2-1.35 2h-1.1V8Zm.75 3.2h.3c.35 0 .6-.35.6-1.2s-.25-1.2-.6-1.2h-.3v2.4Z"/></svg> Сохранить графики в PDF'; }
            });
        };
    })();
</script>
