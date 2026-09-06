<div class="hero-text mb-3">
    <h2>Импорт блейд-корзин из Итиллиум</h2>
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
        Загрузите xlsx-файл выгрузки Итиллиум по блейд-корзинам (шасси). Поиск существующей записи выполняется по имени шасси (например, C7000-01, C7000-02).
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
