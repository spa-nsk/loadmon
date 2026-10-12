<div class="alert alert-warning">
  <strong>Токен выпущен — скопируйте его сейчас, второй раз он не показывается</strong>
  (в БД хранится только его SHA-256, как у сессий).
  <div class="input-group mt-2">
    <input class="form-control font-monospace" value="{{.IssuedToken}}" readonly onclick="this.select()">
    <button class="btn btn-outline-secondary" type="button" onclick="navigator.clipboard.writeText('{{.IssuedToken}}')">Копировать</button>
  </div>
</div>
