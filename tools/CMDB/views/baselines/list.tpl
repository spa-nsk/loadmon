<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">Срезы (baseline)</h1>
  <a class="btn btn-outline-secondary" href="/baselines/compare?a=now&amp;b=now">Сравнить состояния</a>
</div>

<form class="card card-body mb-4" hx-post="/baselines" hx-swap="none">
  <h2 class="h6">Создать срез</h2>
  <p class="text-body-secondary small mb-2">Срез фиксирует ссылки на версии строк на выбранный момент (UTC) по фильтру области. Черновик можно удалить; после заморозки срез неизменяем и получает контрольную сумму SHA-256.</p>
  <div class="row g-2">
    <div class="col-md-4"><label class="form-label" for="bl-name">Название</label><input id="bl-name" name="name" class="form-control" required placeholder="Срез на 01.10"></div>
    <div class="col-md-3"><label class="form-label" for="bl-asof">На момент (UTC)</label><input id="bl-asof" type="datetime-local" name="as_of" class="form-control" value="{{.NowInput}}"></div>
    <div class="col-md-5"><label class="form-label" for="bl-desc">Описание</label><input id="bl-desc" name="description" class="form-control"></div>
    <div class="col-md-3"><label class="form-label" for="bl-classes">Классы КЕ (через запятую)</label><input id="bl-classes" name="classes" class="form-control" placeholder="server, vm"></div>
    <div class="col-md-3"><label class="form-label" for="bl-loc">Площадка</label><input id="bl-loc" name="location_code" class="form-control" placeholder="OVB-DC1"></div>
    <div class="col-md-2"><label class="form-label" for="bl-env">Окружение</label><input id="bl-env" name="environment_code" class="form-control" placeholder="PROD"></div>
    <div class="col-md-2"><label class="form-label" for="bl-comp">Владелец</label><input id="bl-comp" name="company_code" class="form-control"></div>
    <div class="col-md-2"><label class="form-label" for="bl-like">Имя содержит</label><input id="bl-like" name="name_like" class="form-control"></div>
  </div>
  <div class="mt-3"><button type="submit" class="btn btn-primary">Создать срез</button></div>
</form>

<table class="table table-sm table-hover align-middle">
  <thead><tr><th>№</th><th>Название</th><th>На момент (UTC)</th><th>Статус</th><th>Позиций</th><th>Контрольная сумма</th><th></th></tr></thead>
  <tbody>
    {{range .Baselines}}
    <tr>
      <td>{{.ID}}</td>
      <td><a href="/baselines/{{.ID}}">{{.Name}}</a></td>
      <td>{{.AsOf}}</td>
      <td>{{template "baselines/_status.tpl" .Status}}</td>
      <td>{{.ItemCount}}</td>
      <td><code class="small">{{if .Checksum}}{{slice .Checksum 0 12}}…{{end}}</code></td>
      <td class="text-end"><a class="btn btn-sm btn-outline-secondary" href="/baselines/{{.ID}}">Открыть</a></td>
    </tr>
    {{else}}
    <tr><td colspan="7" class="text-center text-body-secondary py-4">Срезов ещё нет</td></tr>
    {{end}}
  </tbody>
</table>
