<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/import">Импорт</a></li><li class="breadcrumb-item active">Прогон №{{.Run.ID}}</li></ol></nav>

<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">
    Прогон №{{.Run.ID}} — {{.Run.ConnectorName}}
    {{if .Run.DryRun}}<span class="badge text-bg-secondary">предпросмотр, не записан</span>{{else}}<span class="badge text-bg-primary">применён</span>{{end}}
    {{if eq .Run.Status "succeeded"}}<span class="badge text-bg-success">выполнен</span>
    {{else if eq .Run.Status "failed"}}<span class="badge text-bg-danger">сбой</span>
    {{else}}<span class="badge text-bg-warning">выполняется</span>{{end}}
  </h1>
  <a class="btn btn-outline-secondary" href="/import">К списку прогонов</a>
</div>

{{if .Run.ErrorMessage}}<div class="alert alert-danger">{{.Run.ErrorMessage}}</div>{{end}}

<div class="row row-cols-2 row-cols-md-4 g-3 mb-3">
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Total}}</div><div class="text-body-secondary small">Всего строк</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Created}}</div><div class="text-body-secondary small">Создано</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Updated}}</div><div class="text-body-secondary small">Изменено</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Unchanged}}</div><div class="text-body-secondary small">Без изменений</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Conflicts}}</div><div class="text-body-secondary small">Неоднозначные</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Errors}}</div><div class="text-body-secondary small">Ошибки строк</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.Skipped}}</div><div class="text-body-secondary small">Пропущено</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.NotDiscovered}}</div><div class="text-body-secondary small">Не найдено источником</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.RelCreated}}</div><div class="text-body-secondary small">Связей создано</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{.Run.RelClosed}}</div><div class="text-body-secondary small">Связей закрыто</div></div></div>
  <div class="col"><div class="card card-body text-center"><div class="fs-4">{{if .Run.ChangeSetID}}{{.Run.ChangeSetID}}{{else}}—{{end}}</div><div class="text-body-secondary small">Набор изменений</div></div></div>
</div>

<a class="btn btn-outline-secondary" href="/import/conflicts?status=&amp;run_id={{.Run.ID}}">Конфликты этого прогона</a>
