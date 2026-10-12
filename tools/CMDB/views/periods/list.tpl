<h1 class="h4 mb-3">Учётные периоды</h1>
<p class="text-body-secondary">Закрытый месяц (UTC) защищён: изменения с датой учёта в нём отклоняются, кроме правок администратора учёта с обязательной причиной. Закрыть можно только закончившийся месяц. Роль «Администратор учёта» появится с E7 — сейчас действия доступны единственному пользователю, но фиксируются в журнале действий.</p>

<div class="row g-3 mb-4">
  <form class="col-md-6" hx-post="/periods/close" hx-swap="none">
    <div class="card card-body">
      <h2 class="h6">Закрыть период</h2>
      <div class="row g-2">
        <div class="col-4"><label class="form-label" for="pc-month">Месяц</label><input id="pc-month" name="month" class="form-control" value="{{.LastMonth}}" placeholder="ГГГГ-ММ" required></div>
        <div class="col-8"><label class="form-label" for="pc-reason">Комментарий</label><input id="pc-reason" name="reason" class="form-control"></div>
      </div>
      <div class="mt-2"><button type="submit" class="btn btn-primary">Закрыть</button></div>
    </div>
  </form>
  <form class="col-md-6" hx-post="/periods/reopen" hx-swap="none">
    <div class="card card-body">
      <h2 class="h6">Переоткрыть период</h2>
      <div class="row g-2">
        <div class="col-4"><label class="form-label" for="po-month">Месяц</label><input id="po-month" name="month" class="form-control" placeholder="ГГГГ-ММ" required></div>
        <div class="col-8"><label class="form-label" for="po-reason">Причина (обязательна)</label><input id="po-reason" name="reason" class="form-control" required></div>
      </div>
      <div class="mt-2"><button type="submit" class="btn btn-outline-danger">Переоткрыть</button></div>
    </div>
  </form>
</div>

<table class="table table-sm align-middle mb-4">
  <thead><tr><th>Период</th><th>Статус</th><th>Закрыт (UTC)</th><th>Кем</th><th>Комментарий</th><th>Переоткрыт</th><th>Причина</th></tr></thead>
  <tbody>
    {{range .Periods}}
    <tr>
      <td>{{.Month}}</td>
      <td>{{if eq .Status "closed"}}<span class="badge text-bg-dark">закрыт</span>{{else}}<span class="badge text-bg-warning">открыт</span>{{end}}</td>
      <td>{{.ClosedAt}}</td><td>{{.ClosedBy}}</td><td>{{.CloseReason}}</td><td>{{.ReopenedAt}}</td><td>{{.ReopenReason}}</td>
    </tr>
    {{else}}<tr><td colspan="7" class="text-center text-body-secondary py-3">Закрытых периодов ещё не было</td></tr>{{end}}
  </tbody>
</table>

<h2 class="h5">Правки задним числом</h2>
<form class="row g-2 mb-2" method="get" action="/periods">
  <div class="col-auto"><label class="visually-hidden" for="bd-from">С</label><input id="bd-from" type="date" name="from" class="form-control form-control-sm" value="{{.From}}"></div>
  <div class="col-auto"><label class="visually-hidden" for="bd-to">По</label><input id="bd-to" type="date" name="to" class="form-control form-control-sm" value="{{.To}}"></div>
  <div class="col-auto"><button class="btn btn-sm btn-outline-secondary" type="submit">Показать</button></div>
</form>
<table class="table table-sm align-middle">
  <thead><tr><th>Набор</th><th>Дата учёта (UTC)</th><th>Записано (UTC)</th><th>Пользователь</th><th>Источник</th><th>Изменений</th><th>Причина</th><th></th></tr></thead>
  <tbody>
    {{range .Report}}
    <tr>
      <td><a href="/changes?q={{.Reason}}">{{.ChangeSetID}}</a></td><td>{{.EffectiveAt}}</td><td>{{.StartedAt}}</td><td>{{.UserID}}</td><td>{{.Source}}</td><td>{{.Entries}}</td><td>{{.Reason}}</td>
      <td>{{if .ClosedPeriod}}<span class="badge text-bg-danger">закрытый период</span>{{end}}</td>
    </tr>
    {{else}}<tr><td colspan="8" class="text-center text-body-secondary py-3">Правок задним числом за период нет</td></tr>{{end}}
  </tbody>
</table>
