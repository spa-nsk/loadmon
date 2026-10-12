<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/admin/ref">Справочники</a></li><li class="breadcrumb-item active">{{.Spec.Title}}</li></ol></nav>

<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">{{.Spec.Title}}</h1>
  <div class="d-flex gap-2">
    <a class="btn btn-outline-secondary" href="/admin/ref/{{.Spec.Slug}}/export">Экспорт CSV</a>
    <form class="d-inline" hx-post="/admin/ref/{{.Spec.Slug}}/import" hx-encoding="multipart/form-data" hx-swap="none">
      <input type="file" name="file" accept=".csv" required class="form-control form-control-sm d-inline-block w-auto align-middle">
      <button type="submit" class="btn btn-outline-secondary">Импорт CSV</button>
    </form>
    <a class="btn btn-primary" href="/admin/ref/{{.Spec.Slug}}/new">Создать</a>
  </div>
</div>

<form class="card card-body mb-3" hx-get="/admin/ref/{{.Spec.Slug}}" hx-target="#ref-results" hx-push-url="true"
      hx-trigger="submit, keyup changed delay:400ms from:#q">
  <input id="q" name="q" class="form-control" placeholder="Поиск" value="{{.Q}}">
</form>

<div id="ref-results">{{template "admin/_ref_results.tpl" .}}</div>
