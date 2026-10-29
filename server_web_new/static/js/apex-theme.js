/**
 * Палитра ApexCharts под data-bs-theme (светлая/тёмная).
 * Явный foreColor нужен: при переключении темы без перезагрузки и когда
 * theme.mode один даёт слабый контраст подписей/иконок на карточках.
 */
(function (global) {
    'use strict';

    function isDark() {
        return document.documentElement.getAttribute('data-bs-theme') === 'dark';
    }

    function colors() {
        var dark = isDark();
        return {
            mode: dark ? 'dark' : 'light',
            foreColor: dark ? '#e2e8f0' : '#334155',
            grid: dark ? 'rgba(255,255,255,0.14)' : 'rgba(15,23,42,0.14)',
            muted: dark ? '#94a3b8' : '#64748b'
        };
    }

    /** Опции для chart/theme/grid/axes/legend/tooltip — для new ApexCharts и updateOptions. */
    function chartOpts() {
        var c = colors();
        return {
            chart: { foreColor: c.foreColor, background: 'transparent' },
            theme: { mode: c.mode },
            grid: { borderColor: c.grid },
            legend: { labels: { colors: c.foreColor } },
            tooltip: { theme: c.mode },
            xaxis: {
                labels: { style: { colors: c.foreColor } },
                axisBorder: { color: c.grid },
                axisTicks: { color: c.grid }
            },
            yaxis: { labels: { style: { colors: c.foreColor } } }
        };
    }

    var charts = [];

    function register(chart) {
        if (chart && charts.indexOf(chart) < 0) charts.push(chart);
    }

    function applyAll() {
        if (!charts.length) return;
        var opts = chartOpts();
        charts.forEach(function (ch) {
            try { ch.updateOptions(opts, false, false); } catch (e) { /* ignore destroyed */ }
        });
    }

    function watch() {
        if (window.MutationObserver) {
            new MutationObserver(function (muts) {
                for (var i = 0; i < muts.length; i++) {
                    if (muts[i].attributeName === 'data-bs-theme') { applyAll(); return; }
                }
            }).observe(document.documentElement, { attributes: true, attributeFilter: ['data-bs-theme'] });
        }
    }

    global.LoadmonApexTheme = {
        isDark: isDark,
        colors: colors,
        chartOpts: chartOpts,
        register: register,
        applyAll: applyAll,
        watch: watch
    };
    watch();
})(window);
