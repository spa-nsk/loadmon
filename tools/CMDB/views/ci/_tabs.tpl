<div id="card-tabs">
  <ul class="nav nav-tabs mb-3">
    <li class="nav-item"><a class="nav-link {{if eq .Tab "general"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/general" hx-target="#card-tabs" hx-swap="outerHTML">Общие</a></li>
    {{if .HasClassTab}}<li class="nav-item"><a class="nav-link {{if eq .Tab "class"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/class" hx-target="#card-tabs" hx-swap="outerHTML">Атрибуты класса</a></li>{{end}}
    <li class="nav-item"><a class="nav-link {{if eq .Tab "components"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/components" hx-target="#card-tabs" hx-swap="outerHTML">Компоненты</a></li>
    <li class="nav-item"><a class="nav-link {{if eq .Tab "software"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/software" hx-target="#card-tabs" hx-swap="outerHTML">ПО</a></li>
    <li class="nav-item"><a class="nav-link {{if eq .Tab "network"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/network" hx-target="#card-tabs" hx-swap="outerHTML">Интерфейсы и IP</a></li>
    <li class="nav-item"><a class="nav-link {{if eq .Tab "relations"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/relations" hx-target="#card-tabs" hx-swap="outerHTML">Связи</a></li>
    <li class="nav-item"><a class="nav-link {{if eq .Tab "sources"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/sources" hx-target="#card-tabs" hx-swap="outerHTML">Источники</a></li>
    <li class="nav-item"><a class="nav-link {{if eq .Tab "history"}}active{{end}}" href="/cis/{{.CIID}}" hx-get="/cis/{{.CIID}}/tab/history" hx-target="#card-tabs" hx-swap="outerHTML">История</a></li>
  </ul>
  {{if eq .Tab "general"}}{{template "ci/_tab_general.tpl" .}}
  {{else if eq .Tab "class"}}{{template "ci/_tab_class.tpl" .}}
  {{else if eq .Tab "components"}}{{template "ci/_tab_components.tpl" .}}
  {{else if eq .Tab "software"}}{{template "ci/_tab_software.tpl" .}}
  {{else if and .AsOf (eq .Tab "network")}}<div class="alert alert-secondary">Интерфейсы и IP на прошлую дату не восстанавливаются (для них нет функций «на дату»): смотрите вкладку «История» или вернитесь к текущему состоянию.</div>
  {{else if eq .Tab "network"}}{{template "ci/_tab_network.tpl" .}}
  {{else if eq .Tab "relations"}}{{template "ci/_tab_relations.tpl" .}}
  {{else if and .AsOf (eq .Tab "sources")}}<div class="alert alert-secondary">Провенанс атрибутов не версионируется — на прошлую дату недоступен.</div>
  {{else if eq .Tab "sources"}}{{template "ci/_tab_sources.tpl" .}}
  {{else if eq .Tab "history"}}{{template "ci/_tab_history.tpl" .}}
  {{end}}
</div>
