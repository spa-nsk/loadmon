<form role="form" id="Update" method="POST" class="p-1">
  <div class="container-fluid">
    <div class="row">
      <div class="hero-text mb-3">
        <h2>Фильтры для отчета по загрузке серверов</h2>
      </div>
    </div>
    <div class="row g-3 align-items-end">
      <div class="col-sm-6 col-md-2">
        <label for="FromTime" class="form-label">Дата начала</label>
        <input name="FromTime" class="form-control" type="date" value="{{printf "%4d-%02d-%02d" .Filter.FromTime.Year .Filter.FromTime.Month .Filter.FromTime.Day }}" tabindex="1" />
      </div>
      <div class="col-sm-6 col-md-2">
        <label for="ToTime" class="form-label">Дата окончания</label>
        <input name="ToTime" class="form-control" type="date" value="{{printf "%4d-%02d-%02d" .Filter.ToTime.Year .Filter.ToTime.Month .Filter.ToTime.Day}}" tabindex="2" />
      </div>
      <div class="col-sm-12 col-md-8">
        <label for="Inventory" class="form-label">Инвентаризация</label>
        <select name="Inventory" size="1" class="form-select" value="{{.Filter.Inventory}}" tabindex="5">
          <option value="">Все варианты</option>
          <option value="нет рекомендаций по оптимизации" {{if eq .Filter.Inventory "нет рекомендаций по оптимизации"}}selected{{end}}>Нет рекомендаций по оптимизации</option>
          <option value="возможно нужно перераспределить ресурсы" {{if eq .Filter.Inventory "возможно нужно перераспределить ресурсы"}}selected{{end}}>Возможно нужно перераспределить ресурсы</option>
          <option value="недостаточно ресурсов по ряду метрик" {{if eq .Filter.Inventory "недостаточно ресурсов по ряду метрик"}}selected{{end}}>Недостаточно ресурсов по ряду метрик</option>
        </select>
      </div>
    </div>
    <div class="row g-3 align-items-end mt-1">
      <div class="col-sm-6 col-md-3">
        <label for="ServerName" class="form-label">Имя сервера</label>
        <input name="ServerName" class="form-control" type="text" value="{{.Filter.ServerName}}" tabindex="3" />
      </div>
      <div class="col-sm-6 col-md-3">
        <label for="Owner" class="form-label">Ответственный</label>
        <input name="Owner" class="form-control" type="text" value="{{.Filter.Owner}}" tabindex="4" />
      </div>
      <div class="col-sm-6 col-md-3">
        <label for="Allocation" class="form-label">Аллокация</label>
        <input name="Allocation" class="form-control" type="text" value="{{.Filter.Allocation}}" tabindex="6" />
      </div>
      <div class="col-sm-6 col-md-3">
        <label for="Service" class="form-label">Сервис</label>
        <input name="Service" class="form-control" type="text" value="{{.Filter.Service}}" tabindex="7" />
      </div>
    </div>
    <div class="row mt-3">
      <div class="col-12 d-flex gap-2">
        <input name="ok" type="submit" title="Применить фильтр" value="Применить" class="btn btn-primary rounded-pill px-4" tabindex="8" />
        <a name="cancel" href="/" title="Отменить изменения" class="btn btn-outline-secondary rounded-pill px-4">Отменить</a>
      </div>
    </div>
  </div>
</form>
