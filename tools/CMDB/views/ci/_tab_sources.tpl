<p class="text-body-secondary">Откуда взято текущее значение атрибута и не заблокировано ли оно ручной правкой (intent §7.3). Заполняется при импорте (E4.7) — до первого прогона источников список пуст.</p>
<table class="table table-sm align-middle">
  <thead><tr><th>Объект</th><th>Атрибут</th><th>Источник</th><th>Значение источника</th><th>Замечено (UTC)</th><th>Заблокировано</th></tr></thead>
  <tbody>
    {{range .Sources}}
    <tr>
      <td>{{index . "entity"}}</td>
      <td>{{index . "attr_name"}}</td>
      <td>{{if index . "source_system_code"}}{{index . "source_system_code"}}{{else}}<span class="text-body-secondary">вручную</span>{{end}}</td>
      <td class="json-cell">{{index . "source_value"}}</td>
      <td>{{index . "observed_at"}}</td>
      <td>{{if eq (index . "is_locked") "true"}}<span class="badge text-bg-warning">заблокировано</span>{{end}}</td>
    </tr>
    {{else}}
    <tr><td colspan="6" class="text-center text-body-secondary py-3">Провенанс атрибутов ещё не зафиксирован</td></tr>
    {{end}}
  </tbody>
</table>
