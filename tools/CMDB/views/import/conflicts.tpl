<nav aria-label="breadcrumb"><ol class="breadcrumb"><li class="breadcrumb-item"><a href="/import">Импорт</a></li><li class="breadcrumb-item active">Очередь конфликтов</li></ol></nav>

<h1 class="h4 mb-3">Очередь конфликтов</h1>

<form id="filter" class="filter-bar card card-body mb-3" hx-get="/import/conflicts" hx-target="#conflict-results" hx-swap="innerHTML"
      hx-push-url="true" hx-trigger="change, submit">
  <input type="hidden" name="run_id" value="{{.Filter.RunID}}">
  <div class="row g-2">
    <div class="col-md-3">
      <select name="status" class="form-select" aria-label="Статус">
        <option value="open" {{if eq .Filter.Status "open"}}selected{{end}}>Открытые</option>
        <option value="resolved" {{if eq .Filter.Status "resolved"}}selected{{end}}>Разрешённые</option>
        <option value="ignored" {{if eq .Filter.Status "ignored"}}selected{{end}}>Проигнорированные</option>
        <option value="" {{if eq .Filter.Status ""}}selected{{end}}>Все статусы</option>
      </select>
    </div>
    <div class="col-md-3">
      <select name="kind" class="form-select" aria-label="Вид">
        <option value="">Все виды</option>
        <option value="ambiguous_match" {{if eq .Filter.Kind "ambiguous_match"}}selected{{end}}>Неоднозначная идентификация</option>
        <option value="locked_attr" {{if eq .Filter.Kind "locked_attr"}}selected{{end}}>Заблокированный атрибут</option>
        <option value="apply_error" {{if eq .Filter.Kind "apply_error"}}selected{{end}}>Ошибка применения</option>
        <option value="not_discovered" {{if eq .Filter.Kind "not_discovered"}}selected{{end}}>Не найдена источником</option>
      </select>
    </div>
    {{if .Filter.RunID}}<div class="col-md-3 d-flex align-items-center text-body-secondary small">Только прогон №{{.Filter.RunID}} — <a href="/import/conflicts">сбросить</a></div>{{end}}
  </div>
</form>

<div id="conflict-results">{{template "import/_conflicts_results.tpl" .}}</div>
