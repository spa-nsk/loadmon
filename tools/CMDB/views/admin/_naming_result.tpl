{{if not .HasResult}}
<p class="text-body-secondary">Введите имя для разбора.</p>
{{else}}
<div class="card card-body">
  <p class="mb-2"><span class="badge {{namingClass (print .Result.Status)}} fs-6">{{.Result.Status}}</span></p>
  {{if .Result.Reason}}<p class="mb-2"><strong>Пояснение:</strong> {{.Result.Reason}}</p>{{end}}
  {{if .Result.Suggestion}}<p class="mb-0"><strong>Предложенное имя по профилю CMDB-NC-2026:</strong> <code>{{.Result.Suggestion}}</code></p>{{end}}
</div>
{{end}}
