<div class="d-flex justify-content-between align-items-center mb-3">
  <h1 class="h4 mb-0">ВМ по кластерам</h1>
  <a id="export-csv" class="btn btn-outline-secondary" href="/reports/vm-clusters/export.csv">Выгрузить CSV</a>
</div>

<form id="filter" class="card card-body mb-3" method="get" action="/reports/vm-clusters">
  <div class="row g-2">
    <div class="col-md-3">
      <label class="form-label" for="cluster">Кластер</label>
      <select id="cluster" name="cluster" class="form-select" onchange="this.form.submit()">
        <option value="">Все кластеры</option>
        {{range .Clusters}}<option value="{{.Value}}" {{if eq .Value (printf "%d" $.ClusterID)}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
    </div>
  </div>
</form>

{{if .Truncated}}<div class="alert alert-warning">Показаны первые {{len .Rows}} из {{.Total}} — полный список в выгрузке CSV.</div>{{end}}

<div class="card">
  <div class="card-body p-0">
    <table class="table table-sm mb-0">
      <thead><tr><th>ВМ</th><th>Кластер</th><th>Хост</th><th class="text-end">vCPU</th><th class="text-end">ОЗУ, ГБ</th><th>Гостевая ОС</th><th>Стадия ЖЦ</th></tr></thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td><a href="/cis/{{.id}}">{{.name}}</a></td>
          <td>{{if .cluster_name}}{{.cluster_name}}{{else}}<span class="text-muted">без кластера</span>{{end}}</td>
          <td>{{if .host_name}}{{.host_name}}{{else}}<span class="text-muted">без хоста</span>{{end}}</td>
          <td class="text-end">{{.vcpu}}</td><td class="text-end">{{.ram_gb}}</td><td>{{.guest_os}}</td><td>{{.lifecycle_stage}}</td>
        </tr>
        {{else}}<tr><td colspan="7" class="text-muted p-3">Нет действующих ВМ по выбранному фильтру.</td></tr>{{end}}
      </tbody>
    </table>
  </div>
</div>

<script>
  document.getElementById('export-csv').addEventListener('click', function (e) {
    var q = new URLSearchParams(new FormData(document.getElementById('filter')));
    e.currentTarget.href = '/reports/vm-clusters/export.csv?' + q.toString();
  });
</script>
