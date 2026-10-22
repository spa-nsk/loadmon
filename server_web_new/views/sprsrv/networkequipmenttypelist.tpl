<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>Типы сетевого оборудования</h2>
        <div class="text-muted">Записей: {{.TypeCount}}</div>
    </div>
    <div class="d-flex gap-2">
        <a href="/sprsrv/networkequipmentlist/0" class="btn btn-outline-secondary rounded-pill px-3">К справочнику оборудования</a>
        <a href="/sprsrv/networkequipmenttypeedit/0" class="btn btn-success rounded-pill px-3">Добавить</a>
    </div>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote">{{.flash.notice}}</blockquote>
{{end}}

<table class="table table-bordered table-striped table-hover align-middle spr-classic-table">
    <thead>
        <tr>
            <th class="text-center">Id</th>
            <th>Наименование</th>
            <th>Описание</th>
            <th class="text-end">Устройств</th>
        </tr>
    </thead>
    <tbody>
        {{range $t := .TypeList}}
        <tr class="row-context-target" data-menu-id="netTypeRowMenu{{$t.Id}}">
            <td class="text-end">{{$t.Id}}</td>
            <td><a href="/sprsrv/networkequipmenttypeedit/{{$t.Id}}">{{$t.Name}}</a></td>
            <td>{{$t.Annotation}}</td>
            <td class="text-end">{{if gt $t.Count 0}}<a href="/sprsrv/networkequipmentlist/0">{{$t.Count}}</a>{{else}}0{{end}}</td>
            <td id="netTypeRowMenu{{$t.Id}}" class="d-none">
                <a class="ctx-menu-item" href="/sprsrv/networkequipmenttypeedit/{{$t.Id}}" target="_blank">Редактировать</a>
            </td>
        </tr>
        {{else}}
        <tr>
            <td colspan="4" class="text-center text-muted py-4">Типы пока не заведены - они появятся после импорта сетевого оборудования из Итиллиум или их можно добавить вручную</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div id="netTypeContextMenu" class="ctx-menu d-none"></div>

<script>
    // Контекстное меню по правой кнопке мыши на строке таблицы - приём как в /sprsrv/datacentersitelist.
    (function () {
        var menu = document.getElementById('netTypeContextMenu');
        if (!menu) return;
        function hideMenu() { menu.classList.add('d-none'); menu.innerHTML = ''; }
        document.querySelectorAll('.row-context-target').forEach(function (row) {
            row.addEventListener('contextmenu', function (evt) {
                var source = document.getElementById(row.getAttribute('data-menu-id'));
                if (!source) return;
                evt.preventDefault();
                menu.innerHTML = source.innerHTML;
                menu.classList.remove('d-none');
                var x = evt.clientX, y = evt.clientY;
                var maxX = window.innerWidth - menu.offsetWidth - 8, maxY = window.innerHeight - menu.offsetHeight - 8;
                if (x > maxX) x = Math.max(8, maxX);
                if (y > maxY) y = Math.max(8, maxY);
                menu.style.left = x + 'px';
                menu.style.top = y + 'px';
            });
        });
        document.addEventListener('click', function (evt) { if (!menu.contains(evt.target)) hideMenu(); });
        document.addEventListener('scroll', hideMenu, true);
        window.addEventListener('resize', hideMenu);
    })();
</script>
