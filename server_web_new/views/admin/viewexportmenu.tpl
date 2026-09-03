<!-- Контроллер: controllers/adminviewexport.go (AdminViewExportMenu).
     Список режимов "Выгрузка по View" - каждый режим соответствует файлу
     <viewsqldir>/<name>.sql (см. viewExportModes в контроллере). -->

<div class="hero-text mb-3">
    <h2>Выгрузка по View</h2>
    <p class="text-muted">
        Каждый пункт ниже выполняет SQL-запрос из одноимённого файла
        <code>&lt;name&gt;.sql</code> в каталоге запросов (ключ <code>viewsqldir</code>
        в app.conf, по умолчанию <code>./conf/viewsql</code>) и показывает результат
        таблицей с постраничным просмотром. Режим - только просмотр: изменить набор
        или порядок колонок можно только правкой самого файла запроса. Доступны
        выгрузки результата в CSV и XLSX (по всей выборке, не только по текущей
        странице).
    </p>
</div>

<div class="list-group">
    {{range $m := .Modes}}
    <a href="/admin/viewexport/{{$m.Name}}" class="list-group-item list-group-item-action">
        {{$m.Title}}
        <span class="text-muted small">({{$m.Name}}.sql)</span>
    </a>
    {{end}}
</div>
