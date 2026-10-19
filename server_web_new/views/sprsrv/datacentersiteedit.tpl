<div class="hero-text mb-3 d-flex flex-wrap justify-content-between align-items-start gap-2">
    <div>
        <h2>{{if .IsNew}}Новая площадка/ЦОД{{else}}Площадка/ЦОД{{end}}</h2>
        <div class="text-muted">{{if .IsNew}}Запись ещё не создана{{else}}Id записи: <b>{{.Id}}</b>{{end}}</div>
    </div>
    <a href="/sprsrv/datacentersitelist/0" class="btn btn-outline-secondary rounded-pill px-3">К списку</a>
</div>

{{if .flash.error}}
<blockquote class="blockquote text-danger">{{.flash.error}}</blockquote>
{{end}}
{{if .flash.notice}}
<blockquote class="blockquote text-success">{{.flash.notice}}</blockquote>
{{end}}

{{if .DC}}
<form role="form" method="POST">
    <div class="hw-card">
        <h6>Общие сведения</h6>
        <div class="row g-3">
            <div class="col-6">
                <label for="Name" class="form-label">Наименование</label>
                <input type="text" id="Name" name="Name" class="form-control" value="{{.DC.Name}}">
            </div>
            <div class="col-6">
                <label for="AltName" class="form-label">Альтернативное наименование</label>
                <input type="text" id="AltName" name="AltName" class="form-control" value="{{.DC.AltName}}">
            </div>
            <div class="col-12 col-md-8 position-relative">
                <label for="LocalitySearch" class="form-label">Населённый пункт</label>
                <input type="text" id="LocalitySearch" class="form-control" autocomplete="off"
                       placeholder="начните вводить название, например: Новосиб" value="{{.LocalityName}}">
                <div id="LocalityList" class="list-group position-absolute w-100 shadow d-none" style="z-index: 1050; max-height: 320px; overflow-y: auto;"></div>
                <div class="form-text" id="LocalityHint">{{if and .DC.LocalityCode (not .LocalityName)}}Код не найден в справочнике населённых пунктов{{else}}Выберите из списка - код подставится справа{{end}}</div>
            </div>
            <div class="col-12 col-md-4">
                <label for="LocalityCode" class="form-label">Код ОКТМО</label>
                <input type="text" id="LocalityCode" name="LocalityCode" class="form-control" maxlength="11" value="{{.DC.LocalityCode}}">
            </div>
            <div class="col-12 col-md-8">
                <label for="Address" class="form-label">Адрес</label>
                <input type="text" id="Address" name="Address" class="form-control" value="{{.DC.Address}}">
            </div>
            <div class="col-6 col-md-3">
                <label for="Floor" class="form-label">Этаж</label>
                <input type="text" id="Floor" name="Floor" class="form-control" value="{{.DC.Floor}}">
            </div>
            <div class="col-6 col-md-9">
                <label for="Room" class="form-label">Комната</label>
                <input type="text" id="Room" name="Room" class="form-control" value="{{.DC.Room}}">
            </div>
            <div class="col-12">
                <div class="form-check form-switch">
                    <input class="form-check-input" type="checkbox" id="IsDatacenter" name="IsDatacenter" value="1" {{if .DC.IsDatacenter}}checked{{end}}>
                    <label class="form-check-label" for="IsDatacenter">Признак "ЦОД"</label>
                </div>
                <div class="form-text">Отмечено - площадка является ЦОД, там размещается активное оборудование (которое может быть выключено). Снято - это просто площадка размещения без активного оборудования (склад, рабочая комната, площадка для списанного и т.п.).</div>
            </div>
            <div class="col-3">
                <label for="Width" class="form-label">Ширина</label>
                <input type="number" id="Width" name="Width" class="form-control" value="{{.DC.Width}}">
            </div>
            <div class="col-3">
                <label for="Height" class="form-label">Высота</label>
                <input type="number" id="Height" name="Height" class="form-control" value="{{.DC.Height}}">
            </div>
            <div class="col-6">
                <label for="CodeJira" class="form-label">Код Jira</label>
                <input type="text" id="CodeJira" name="CodeJira" class="form-control" value="{{.DC.CodeJira}}">
                <div class="form-text">Ключ проекта/записи в Jira для этого ЦОД.</div>
            </div>
            <div class="col-12">
                <label for="Annotation" class="form-label">Описание</label>
                <textarea id="Annotation" name="Annotation" class="form-control" rows="3">{{.DC.Annotation}}</textarea>
            </div>
        </div>
    </div>

    <div class="mt-3">
        <button type="submit" class="btn btn-primary rounded-pill px-4">Сохранить</button>
    </div>
</form>
{{end}}

<script>
    // Выбор населённого пункта: поиск по названию по мере набора
    // (/sprsrv/localitysearch), по выбору в поле "Код ОКТМО" подставляется код.
    (function () {
        var search = document.getElementById('LocalitySearch');
        var list = document.getElementById('LocalityList');
        var code = document.getElementById('LocalityCode');
        var hint = document.getElementById('LocalityHint');
        if (!search || !list || !code) return;

        var timer = null, seq = 0;

        function hide() { list.classList.add('d-none'); list.innerHTML = ''; }

        function pick(item) {
            code.value = item.oktmo;
            search.value = item.label;
            hint.textContent = 'Выбран код ' + item.oktmo;
            hide();
        }

        function render(items) {
            list.innerHTML = '';
            if (!items.length) {
                var empty = document.createElement('div');
                empty.className = 'list-group-item text-muted small';
                empty.textContent = 'Ничего не найдено';
                list.appendChild(empty);
            }
            items.forEach(function (it) {
                var a = document.createElement('button');
                a.type = 'button';
                a.className = 'list-group-item list-group-item-action small';
                a.textContent = it.label + '  [' + it.oktmo + ']';
                a.addEventListener('mousedown', function (e) { e.preventDefault(); pick(it); });
                list.appendChild(a);
            });
            list.classList.remove('d-none');
        }

        function query() {
            var q = search.value.trim();
            if (q.length < 2) { hide(); return; }
            var my = ++seq;
            fetch('/sprsrv/localitysearch?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
                .then(function (r) { return r.json(); })
                .then(function (items) { if (my === seq) render(items); })
                .catch(function () { hide(); });
        }

        search.addEventListener('input', function () {
            if (search.value.trim() === '') { code.value = ''; hint.textContent = 'Населённый пункт не выбран'; }
            clearTimeout(timer);
            timer = setTimeout(query, 200);
        });
        search.addEventListener('blur', function () { setTimeout(hide, 150); });
        search.addEventListener('keydown', function (e) { if (e.key === 'Escape') hide(); });
    })();
</script>
