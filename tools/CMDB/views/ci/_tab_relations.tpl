<script src="/static/vendor/cytoscape.min.js"></script>
<div id="rel-graph" class="border rounded mb-3" style="height: 320px"></div>
<script type="application/json" id="rel-graph-data">{{.RelGraph}}</script>
<script>
(function () {
  var container = document.getElementById('rel-graph');
  var elements = JSON.parse(document.getElementById('rel-graph-data').textContent || '[]');
  if (!container || !window.cytoscape) { return; }
  var dark = document.documentElement.getAttribute('data-bs-theme') === 'dark';
  var cy = window.cytoscape({
    container: container,
    elements: elements,
    style: [
      {selector: 'node', style: {label: 'data(label)', 'font-size': '11px', color: dark ? '#e9ecef' : '#212529',
        'background-color': '#0d6efd', width: 28, height: 28, 'text-valign': 'bottom', 'text-margin-y': 4}},
      {selector: 'node[kind = "self"]', style: {'background-color': '#dc3545', width: 36, height: 36}},
      {selector: 'edge', style: {label: 'data(label)', 'font-size': '9px', color: dark ? '#adb5bd' : '#495057',
        width: 2, 'line-color': '#adb5bd', 'target-arrow-color': '#adb5bd', 'target-arrow-shape': 'triangle',
        'curve-style': 'bezier'}}
    ],
    layout: {name: 'concentric', concentric: function (n) { return n.data('kind') === 'self' ? 2 : 1; }, minNodeSpacing: 40}
  });
  cy.on('tap', 'node', function (e) {
    var url = e.target.data('url');
    if (url) { window.location = url; }
  });
})();
</script>

<table class="table table-sm align-middle">
  <thead><tr><th>Связь</th><th>КЕ</th><th>Класс</th><th>Действует с (UTC)</th><th></th></tr></thead>
  <tbody>
    {{range .Relations}}
    <tr>
      <td>{{.Label}}</td>
      <td><a href="/cis/{{.PeerID}}">{{.PeerName}}</a></td>
      <td>{{.PeerClass}}</td>
      <td>{{.ValidFrom}}</td>
      <td>
        <form class="d-flex gap-1" hx-post="/relations/{{.ID}}/close" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
          <input type="hidden" name="ci" value="{{$.CIID}}">
          <input type="hidden" name="row_version" value="{{.Version}}">
          <input class="form-control form-control-sm" name="reason" placeholder="Причина *">
          <button class="btn btn-outline-danger btn-sm" type="submit">Закрыть</button>
        </form>
      </td>
    </tr>
    {{else}}
    <tr><td colspan="5" class="text-center text-body-secondary py-3">Действующих связей нет</td></tr>
    {{end}}
  </tbody>
</table>

<button type="button" class="btn btn-primary btn-sm" data-bs-toggle="modal" data-bs-target="#modal-add-relation">Добавить связь</button>

<div class="modal fade" id="modal-add-relation" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <form hx-post="/cis/{{.CIID}}/relations" hx-target="#card-tabs" hx-swap="outerHTML" autocomplete="off">
        <div class="modal-header">
          <h3 class="modal-title h6">Добавить связь</h3>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Закрыть"></button>
        </div>
        <div class="modal-body row g-2">
          <div class="col-md-6">
            <select name="direction" class="form-select" aria-label="Направление">
              <option value="out">эта КЕ → связанная</option>
              <option value="in">связанная → эта КЕ</option>
            </select>
          </div>
          <div class="col-md-6">
            <select name="rel_type" class="form-select" aria-label="Тип связи">
              {{range .RelTypes}}<option value="{{.Value}}">{{.Label}}</option>{{end}}
            </select>
          </div>
          <div class="col-12"><input class="form-control" name="peer_name" placeholder="Имя связанной КЕ *"></div>
          <div class="col-12"><input class="form-control" name="reason" placeholder="Причина *"></div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-outline-secondary" data-bs-dismiss="modal">Отмена</button>
          <button class="btn btn-primary" type="submit">Создать</button>
        </div>
      </form>
    </div>
  </div>
</div>
