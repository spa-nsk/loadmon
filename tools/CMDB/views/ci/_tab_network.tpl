<h2 class="h6">Интерфейсы</h2>
<table class="table table-sm align-middle">
  <thead><tr><th>Имя</th><th>Тип</th><th>MAC</th><th>WWPN</th><th>Скорость, Мбит/с</th><th>VLAN</th><th>Описание</th></tr></thead>
  <tbody>
    {{range .Interfaces}}
    <tr>
      <td>{{index . "name"}}</td><td>{{index . "if_type"}}</td><td>{{index . "mac"}}</td><td>{{index . "wwpn"}}</td>
      <td>{{index . "speed_mbps"}}</td><td>{{index . "vlan"}}</td><td>{{index . "description"}}</td>
    </tr>
    {{else}}
    <tr><td colspan="7" class="text-center text-body-secondary py-3">Интерфейсов нет</td></tr>
    {{end}}
  </tbody>
</table>

<button type="button" class="btn btn-primary btn-sm mb-4" data-bs-toggle="modal" data-bs-target="#modal-add-interface">Добавить интерфейс</button>

<div class="modal fade" id="modal-add-interface" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <form hx-post="/cis/{{.CIID}}/interfaces" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
        <div class="modal-header">
          <h3 class="modal-title h6">Добавить интерфейс</h3>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
        </div>
        <div class="modal-body row g-2">
          <div class="col-md-6"><input class="form-control" name="name" placeholder="Имя (eth0) *"></div>
          <div class="col-md-6">
            <select name="if_type" class="form-select" aria-label="Тип">
              {{range .IfTypes}}<option value="{{.}}">{{.}}</option>{{end}}
            </select>
          </div>
          <div class="col-md-6"><input class="form-control" name="mac" placeholder="MAC"></div>
          <div class="col-md-6"><input class="form-control" name="wwpn" placeholder="WWPN"></div>
          <div class="col-md-6"><input class="form-control" name="speed_mbps" placeholder="Мбит/с" inputmode="numeric"></div>
          <div class="col-md-6"><input class="form-control" name="reason" placeholder="Причина *"></div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Отмена</button>
          <button class="btn btn-primary" type="submit">Добавить</button>
        </div>
      </form>
    </div>
  </div>
</div>

<h2 class="h6">IP-адреса</h2>
<table class="table table-sm align-middle">
  <thead><tr><th>Адрес</th><th>Интерфейс</th><th>Вид</th><th>Статус</th><th>VRF</th><th>Подсеть</th><th>DNS</th><th></th></tr></thead>
  <tbody>
    {{range .IPs}}
    <tr>
      <td>{{index . "address"}}/{{index . "masklen"}}</td><td>{{index . "iface"}}</td><td>{{index . "kind"}}</td><td>{{index . "status"}}</td>
      <td>{{index . "vrf"}}</td><td>{{index . "subnet"}}</td><td>{{index . "dns_name"}}</td>
      <td>
        <form class="d-flex gap-1" hx-post="/ips/{{index . "id"}}/release" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
          <input type="hidden" name="ci" value="{{$.CIID}}">
          <input type="hidden" name="row_version" value="{{index . "row_version"}}">
          <input class="form-control form-control-sm" name="reason" placeholder="Причина *">
          <button class="btn btn-outline-danger btn-sm" type="submit">Освободить</button>
        </form>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="8" class="text-center text-body-secondary py-3">Адресов нет</td></tr>
    {{end}}
  </tbody>
</table>

<button type="button" class="btn btn-primary btn-sm" data-bs-toggle="modal" data-bs-target="#modal-assign-ip">Назначить адрес</button>

<div class="modal fade" id="modal-assign-ip" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <form hx-post="/cis/{{.CIID}}/ips" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
        <div class="modal-header">
          <h3 class="modal-title h6">Назначить адрес</h3>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
        </div>
        <div class="modal-body row g-2">
          <div class="col-md-6"><input class="form-control" name="address" placeholder="10.0.0.5 или 10.0.0.5/24 *"></div>
          <div class="col-md-6">
            <select name="interface_id" class="form-select" aria-label="Интерфейс">
              <option value="">без интерфейса</option>
              {{range .Interfaces}}<option value="{{index . "id"}}">{{index . "name"}}</option>{{end}}
            </select>
          </div>
          <div class="col-md-4">
            <select name="kind" class="form-select" aria-label="Вид">
              {{range .IPKinds}}<option value="{{.}}">{{.}}</option>{{end}}
            </select>
          </div>
          <div class="col-md-4"><input class="form-control" name="vrf" placeholder="VRF"></div>
          <div class="col-md-4"><input class="form-control" name="dns_name" placeholder="DNS-имя"></div>
          <div class="col-12"><input class="form-control" name="reason" placeholder="Причина *"></div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Отмена</button>
          <button class="btn btn-primary" type="submit">Назначить</button>
        </div>
      </form>
    </div>
  </div>
</div>
