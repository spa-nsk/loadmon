<h1 class="h4 mb-3">Справочники</h1>
<div class="row row-cols-1 row-cols-md-3 g-3">
  {{range .Specs}}
  <div class="col">
    <a href="/admin/ref/{{.Slug}}" class="card text-decoration-none h-100">
      <div class="card-body">
        <h2 class="h6 card-title mb-0">{{.Title}}</h2>
      </div>
    </a>
  </div>
  {{end}}
  <div class="col">
    <a href="/admin/baseline-configs" class="card text-decoration-none h-100">
      <div class="card-body">
        <h2 class="h6 card-title mb-0">Эталонные конфигурации</h2>
        <p class="card-text small text-muted mb-0">Требуемые атрибуты и ПО по классу, отчёт отклонений (E7.8)</p>
      </div>
    </a>
  </div>
</div>
