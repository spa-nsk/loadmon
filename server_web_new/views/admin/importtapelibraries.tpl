<div class="hero-text mb-3">
    <h2>Импорт ленточных библиотек из Итиллиум</h2>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

<div class="hw-card">
    <h6>Загрузка файла</h6>
    <p class="text-muted">
        Загрузите xlsx-файл выгрузки Итиллиум по ленточным библиотекам. Запись ищется по серийному номеру (S/N). Адрес (площадка, шкаф, начальный юнит) берётся из поля "Местоположение", размер - из колонки units, сервер подключения - по "Имя ПК" из справочника серверов.
    </p>
    <form role="form" method="POST" enctype="multipart/form-data">
        <div class="row g-3 align-items-end">
            <div class="col-12 col-md-6">
                <label for="ImportFile" class="form-label">Файл (.xlsx)</label>
                <input type="file" id="ImportFile" name="ImportFile" class="form-control" accept=".xlsx" required>
            </div>
            <div class="col-12 col-md-auto">
                <button type="submit" class="btn btn-primary rounded-pill px-4">Загрузить и импортировать</button>
            </div>
        </div>
    </form>
</div>

{{if .JobId}}
<!-- Индикатор прогресса и итоговая статистика последнего запущенного импорта
     - тот же приём (AJAX-опрос раз в секунду), что на /admin/importblades,
     см. server_web_new/views/admin/importblades.tpl и
     controllers/adminimporttapelibraries.go:AdminImportTapeLibrariesStatus. -->
<div class="hw-card mt-3" id="importProgressCard">
    <h6>Импорт файла: {{.JobId}}</h6>
    <div class="progress" style="height: 26px;">
        <div id="importProgressBar" class="progress-bar progress-bar-striped progress-bar-animated bg-primary"
             role="progressbar" style="width: 0%;" aria-valuenow="0" aria-valuemin="0" aria-valuemax="100">0%</div>
    </div>
    <div id="importProgressStats" class="mt-3 small text-muted">Опрос статуса...</div>
</div>

<script>
    (function () {
        var jobId = "{{.JobId}}";
        if (!jobId) return;

        var bar = document.getElementById('importProgressBar');
        var stats = document.getElementById('importProgressStats');
        var timer = null;

        function renderDone(data) {
            bar.classList.remove('progress-bar-animated');
            if (data.err) {
                bar.classList.remove('bg-primary');
                bar.classList.add('bg-danger');
                stats.innerHTML = '<span class="text-danger">Импорт завершился с ошибкой: ' + data.err + '</span>';
                return;
            }
            bar.classList.remove('bg-primary');
            bar.classList.add('bg-success');
            stats.innerHTML =
                'Импорт завершён. Всего импортировано: <b>' + data.imported + '</b>' +
                ' &mdash; новых: <b>' + data.new + '</b>' +
                ', обновлено: <b>' + data.updated + '</b>' +
                ', с признаком списания/вывода: <b>' + data.inactive + '</b>.';
        }

        function renderRunning(data) {
            stats.textContent = 'Обработано ' + data.processed + ' из ' + data.file_rows +
                ' (импортировано: ' + data.imported + ', новых: ' + data.new +
                ', обновлено: ' + data.updated + ', с признаком списания/вывода: ' + data.inactive + ')';
        }

        function poll() {
            fetch('/admin/importtapelibraries/status?job=' + encodeURIComponent(jobId), { credentials: 'same-origin' })
                .then(function (r) { return r.json(); })
                .then(function (data) {
                    if (data.error) {
                        stats.textContent = data.error;
                        return;
                    }
                    var pct = data.percent || 0;
                    bar.style.width = pct + '%';
                    bar.setAttribute('aria-valuenow', pct);
                    bar.textContent = pct + '%';

                    if (data.done) {
                        renderDone(data);
                        if (timer) { clearInterval(timer); timer = null; }
                    } else {
                        renderRunning(data);
                    }
                })
                .catch(function () {
                    // Сеть/сервер временно недоступны - следующий тик опроса попробует снова.
                });
        }

        poll();
        timer = setInterval(poll, 1000);
    })();
</script>
{{end}}
