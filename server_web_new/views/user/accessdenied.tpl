<div class="d-flex align-items-center justify-content-center" style="min-height: calc(100vh - 3rem);">
<div class="p-1" style="max-width:480px; width:100%;">
    <div class="row">
        <div class="hero-text mb-3">
            <h2>Доступ не предоставлен</h2>
            <p class="text-muted">
                Вы успешно прошли доменную аутентификацию, но ни одна из ваших
                групп Active Directory не найдена в списке групп, которым
                разрешён доступ к этой системе. Для получения доступа
                оформите заявку - после её обработки и добавления вашей
                группы в конфигурацию системы вход станет доступен.
            </p>
        </div>
    </div>
    <div class="d-grid gap-2 mb-2">
        {{if .AccessRequestURL}}
        <a href="{{.AccessRequestURL}}" target="_blank" rel="noopener" class="btn btn-primary rounded-pill">Запустить служебную записку для предоставления доступа</a>
        {{else}}
        <button type="button" class="btn btn-primary rounded-pill" disabled title="Адрес формы заявки не настроен в конфигурации системы">Запустить служебную записку для предоставления доступа</button>
        <div class="text-muted small text-center">Адрес формы заявки пока не настроен - обратитесь к администратору системы.</div>
        {{end}}
    </div>
</div>
</div>
