<div class="hero-text mb-3">
    <h2>Импорт справочника населённых пунктов</h2>
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
        Загрузите csv-файл открытых данных ОКТМО Росстата (раздел «Населённые пункты», например oktmo_data_20261001.csv; разделитель «;», кодировка UTF-8). Справочник перезагружается целиком; код населённого пункта (ОКТМО, 11 знаков) используется в справочнике площадок/ЦОД.
    </p>
    <form role="form" method="POST" enctype="multipart/form-data">
        <div class="row g-3 align-items-end">
            <div class="col-12 col-md-6">
                <label for="ImportFile" class="form-label">Файл (.csv)</label>
                <input type="file" id="ImportFile" name="ImportFile" class="form-control" accept=".csv" required>
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
     см. server_web_new/views/admin/importstorages.tpl и
     controllers/adminimportlocalities.go:AdminImportLocalitiesStatus. -->
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
            stats.innerHTML = 'Импорт завершён. Загружено населённых пунктов: <b>' + data.imported + '</b>.';
        }

        function renderRunning(data) {
            stats.textContent = 'Загружено ' + data.processed + ' из ' + data.file_rows;
        }

        function poll() {
            fetch('/admin/importlocalities/status?job=' + encodeURIComponent(jobId), { credentials: 'same-origin' })
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
