/*
 * pdf-export.js — общий механизм экспорта набора графиков ApexCharts в один
 * стильный, лаконичный PDF-документ (используется страницами
 * /sprsrv/vmperfday и /sprsrv/vmperfperiod, views/sprsrv/vmperfday.tpl и
 * vmperfperiod.tpl).
 *
 * Почему текст рисуется через canvas, а не через pdf.text() (jsPDF):
 * встроенные шрифты jsPDF (Helvetica/Times/Courier, стандарт PDF) не
 * содержат кириллицу — прямой вывод русского текста через pdf.text() даёт
 * "битые" символы. Подключать и встраивать TTF-шрифт с кириллицей в jsPDF
 * можно, но проще и надёжнее отрисовать весь текст страницы на обычном
 * HTML5 canvas (шрифт берётся из шрифтового стека браузера, кириллица
 * поддерживается без каких-либо дополнительных файлов), а затем вставить в
 * PDF уже готовое растровое изображение всей страницы (заголовок + график +
 * подвал) через pdf.addImage(). Так же гарантируется единый фирменный стиль
 * оформления страницы независимо от версии jsPDF.
 */

(function () {
    "use strict";

    // размер холста страницы в пикселях - под альбомную A4 (297x210 мм)
    // с запасом по разрешению для чёткой печати/просмотра
    var PAGE_W = 1683;
    var PAGE_H = 1190;

    var COLOR_ACCENT = "#2E5496";
    var COLOR_ACCENT_DARK = "#1F3864";
    var COLOR_TEXT = "#212529";
    var COLOR_MUTED = "#6c757d";
    var COLOR_LINE = "#dee2e6";
    var COLOR_BG = "#ffffff";

    var FONT_STACK = "'Segoe UI', Arial, 'Helvetica Neue', sans-serif";

    // Стиль осей/подписей/легенды, принудительно применяемый к графику
    // ПЕРЕД экспортом в изображение для PDF. На экране график может быть
    // отрисован в тёмной теме или со светлыми (полупрозрачными) подписями
    // осей, рассчитанными на цветной фон интерфейса - на белой странице PDF
    // это выглядит блёкло и плохо читается. Экспортный снимок всегда делается
    // в единой светлой, контрастной цветовой схеме независимо от темы,
    // выбранной пользователем на странице.
    var EXPORT_TEXT_COLOR = "#1c1f24";
    var EXPORT_GRID_COLOR = "#c7ccd3";
    var EXPORT_STYLE_OVERRIDES = {
        theme: { mode: "light" },
        chart: { foreColor: EXPORT_TEXT_COLOR },
        grid: { borderColor: EXPORT_GRID_COLOR },
        legend: { labels: { colors: EXPORT_TEXT_COLOR, useSeriesColors: false } },
        xaxis: { labels: { style: { colors: EXPORT_TEXT_COLOR, fontWeight: 600 } } },
        yaxis: { labels: { style: { colors: EXPORT_TEXT_COLOR, fontWeight: 600 } } },
        tooltip: { theme: "light" }
    };

    // Временно переключает график chart на контрастную "печатную" цветовую
    // схему, делает снимок (PNG dataURI) и возвращает исходное оформление
    // обратно - на экране пользователь по-прежнему видит график в своей
    // теме, в PDF попадает уже контрастный вариант. Возвращает Promise<string>
    // (data URI изображения).
    window.vmCaptureChartImage = function (chart, scale) {
        var originalConfig;
        try {
            originalConfig = JSON.parse(JSON.stringify(chart.w.config));
        } catch (e) {
            originalConfig = null;
        }

        return chart.updateOptions(EXPORT_STYLE_OVERRIDES, true, false).then(function () {
            return chart.dataURI({ scale: scale || 2 });
        }).then(function (res) {
            var restore = originalConfig ? chart.updateOptions(originalConfig, true, false) : Promise.resolve();
            return restore.then(function () { return res.imgURI; });
        }).catch(function (err) {
            // даже при ошибке экспорта возвращаем оформление на место
            if (originalConfig) chart.updateOptions(originalConfig, true, false);
            throw err;
        });
    };

    // Подбирает наибольший размер шрифта (не больше maxPx), при котором
    // строка text помещается в maxWidth; если и на минимальном размере не
    // помещается - обрезает строку многоточием.
    function fitText(ctx, text, maxWidth, maxPx, minPx, weight) {
        weight = weight || "normal";
        var px = maxPx;
        while (px > minPx) {
            ctx.font = weight + " " + px + "px " + FONT_STACK;
            if (ctx.measureText(text).width <= maxWidth) break;
            px -= 1;
        }
        ctx.font = weight + " " + px + "px " + FONT_STACK;
        var s = text;
        if (ctx.measureText(s).width > maxWidth) {
            while (s.length > 1 && ctx.measureText(s + "…").width > maxWidth) {
                s = s.slice(0, -1);
            }
            s = s + "…";
        }
        return { text: s, px: px };
    }

    function drawHeaderFooter(ctx, opts) {
        // фон
        ctx.fillStyle = COLOR_BG;
        ctx.fillRect(0, 0, PAGE_W, PAGE_H);

        // верхняя акцентная полоса - фирменный элемент документа
        ctx.fillStyle = COLOR_ACCENT_DARK;
        ctx.fillRect(0, 0, PAGE_W, 10);

        var marginX = 64;
        var y = 52;

        // название документа (мелко, серым) + правый край - дата/время формирования
        ctx.textBaseline = "alphabetic";
        ctx.fillStyle = COLOR_MUTED;
        ctx.font = "600 20px " + FONT_STACK;
        ctx.textAlign = "left";
        ctx.fillText("LOADMON · ОТЧЁТ О ЗАГРУЗКЕ ВИРТУАЛЬНОЙ МАШИНЫ", marginX, y);

        ctx.textAlign = "right";
        ctx.font = "20px " + FONT_STACK;
        ctx.fillText(opts.generatedAt, PAGE_W - marginX, y);
        ctx.textAlign = "left";

        // подзаголовок - имя сервера / путь ВМ (vCenter/датацентр/кластер/имя)
        y += 44;
        var fitted = fitText(ctx, opts.subTitle || "", PAGE_W - marginX * 2, 34, 18, "700");
        ctx.fillStyle = COLOR_TEXT;
        ctx.font = "700 " + fitted.px + "px " + FONT_STACK;
        ctx.fillText(fitted.text, marginX, y);

        // CPU/RAM/DISK (если передано) — та же справка, что в шапке страницы метрик
        if (opts.resourcesLabel) {
            y += 28;
            var fittedRes = fitText(ctx, opts.resourcesLabel, PAGE_W - marginX * 2, 22, 16, "400");
            ctx.fillStyle = COLOR_MUTED;
            ctx.font = "400 " + fittedRes.px + "px " + FONT_STACK;
            ctx.fillText(fittedRes.text, marginX, y);
        }

        // период/дата отчёта
        y += 34;
        ctx.fillStyle = COLOR_MUTED;
        ctx.font = "22px " + FONT_STACK;
        ctx.fillText(opts.periodLabel || "", marginX, y);

        // разделительная линия
        y += 22;
        ctx.strokeStyle = COLOR_LINE;
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(marginX, y);
        ctx.lineTo(PAGE_W - marginX, y);
        ctx.stroke();

        // название конкретного графика на странице
        var titleY = y + 48;
        var fittedTitle = fitText(ctx, opts.chartTitle, PAGE_W - marginX * 2, 30, 18, "700");
        ctx.fillStyle = COLOR_ACCENT;
        ctx.font = "700 " + fittedTitle.px + "px " + FONT_STACK;
        ctx.fillText(fittedTitle.text, marginX, titleY);

        // подвал: номер страницы слева, авторство/копирайт справа
        var footerY = PAGE_H - 34;
        ctx.strokeStyle = COLOR_LINE;
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(marginX, footerY - 22);
        ctx.lineTo(PAGE_W - marginX, footerY - 22);
        ctx.stroke();

        ctx.fillStyle = COLOR_MUTED;
        ctx.font = "18px " + FONT_STACK;
        ctx.textAlign = "left";
        ctx.fillText("Страница " + opts.pageNum + " из " + opts.totalPages, marginX, footerY);
        ctx.textAlign = "right";
        ctx.fillText("Сформировано автоматически системой Loadmon", PAGE_W - marginX, footerY);
        ctx.textAlign = "left";

        return { chartAreaTop: titleY + 26, chartAreaBottom: footerY - 44, marginX: marginX };
    }

    // Строит страницу PDF: рисует шапку/подзаголовок/название графика/подвал
    // на canvas, вписывает туда изображение графика (по центру, с сохранением
    // пропорций) и добавляет получившееся изображение страницы в pdf.
    // opts: { docTitle (не используется напрямую, оставлено для вызывающего
    // кода), subTitle, resourcesLabel (опц. CPU/RAM/DISK), periodLabel,
    // chartTitle, chartImgURI, pageNum, totalPages, isFirstPage }
    window.vmBuildPdfPage = function (pdf, opts) {
        return new Promise(function (resolve) {
            var canvas = document.createElement("canvas");
            canvas.width = PAGE_W;
            canvas.height = PAGE_H;
            var ctx = canvas.getContext("2d");

            var generatedAt = new Date().toLocaleString("ru-RU", {
                day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit"
            });

            var area = drawHeaderFooter(ctx, {
                subTitle: opts.subTitle,
                resourcesLabel: opts.resourcesLabel,
                periodLabel: opts.periodLabel,
                chartTitle: opts.chartTitle,
                pageNum: opts.pageNum,
                totalPages: opts.totalPages,
                generatedAt: generatedAt
            });

            function finish() {
                if (!opts.isFirstPage) pdf.addPage();
                var pageDataURI = canvas.toDataURL("image/jpeg", 0.95);
                var pdfW = pdf.internal.pageSize.getWidth();
                var pdfH = pdf.internal.pageSize.getHeight();
                pdf.addImage(pageDataURI, "JPEG", 0, 0, pdfW, pdfH);
                resolve();
            }

            if (!opts.chartImgURI) {
                finish();
                return;
            }

            var img = new Image();
            img.onload = function () {
                var maxW = PAGE_W - area.marginX * 2;
                var maxH = area.chartAreaBottom - area.chartAreaTop;
                var ratio = Math.min(maxW / img.width, maxH / img.height, 1);
                var w = img.width * ratio;
                var h = img.height * ratio;
                var x = (PAGE_W - w) / 2;
                var y = area.chartAreaTop + Math.max(0, (maxH - h) / 2);
                // лёгкая рамка-подложка под графиком - аккуратно отделяет его от текста
                ctx.fillStyle = "#fbfbfc";
                ctx.strokeStyle = COLOR_LINE;
                ctx.lineWidth = 1.5;
                ctx.fillRect(x - 12, y - 12, w + 24, h + 24);
                ctx.strokeRect(x - 12, y - 12, w + 24, h + 24);
                ctx.drawImage(img, x, y, w, h);
                finish();
            };
            img.onerror = finish;
            img.src = opts.chartImgURI;
        });
    };
})();
