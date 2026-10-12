<h1 class="h4 mb-3">Отчёты</h1>

<div class="row g-3">
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Инвентарь по локации/стойке</h2>
      <p class="text-muted small mb-2">Действующие КЕ, сгруппированные по площадке и стойке (свободное поле, заполняется не всегда).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/inventory">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Ёмкость СХД</h2>
      <p class="text-muted small mb-2">Raw/usable по массивам и пулам, фактически размещённое (сумма томов).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/storage">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">ВМ по кластерам</h2>
      <p class="text-muted small mb-2">Виртуальные машины, сгруппированные по кластеру виртуализации (через хост).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/vm-clusters">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">КЕ без владельца</h2>
      <p class="text-muted small mb-2">Действующие КЕ без персонального владельца (owner_person_id).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/no-owner">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Отклонения именования</h2>
      <p class="text-muted small mb-2">КЕ, не соответствующие профилю именования (legacy/noncompliant/exception).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/naming">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Запрещённое/неразрешённое ПО</h2>
      <p class="text-muted small mb-2">Установки ПО с политикой forbidden/review (<a href="/admin/ref/software">справочник ПО</a>), на дату (?asof=).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/software">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Отклонения от эталонных конфигураций</h2>
      <p class="text-muted small mb-2">Требуемые атрибуты и ПО (<a href="/admin/baseline-configs">эталонные конфигурации</a>), на дату или против среза.</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/baseline-config">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Изменения за период</h2>
      <p class="text-muted small mb-2">Журнал изменений с диапазоном дат и выгрузкой CSV.</p>
      <a class="btn btn-sm btn-outline-secondary" href="/changes">Открыть</a>
    </div></div>
  </div>
</div>

<h2 class="h5 mt-4 mb-3">Отчёты соответствия</h2>
<div class="row g-3">
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Свежесть инвентаризации</h2>
      <p class="text-muted small mb-2">КЕ без подтверждения источником более 30 дней (KPI из intent §9).</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/stale-inventory">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Объекты КИИ и их состав</h2>
      <p class="text-muted small mb-2">КЕ с признаком объекта критической информационной инфраструктуры и связанные с ними КЕ.</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/kii">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Доля отечественного оборудования/ПО</h2>
      <p class="text-muted small mb-2">Признак отечественного происхождения по моделям (<a href="/admin/ref/model">справочник</a>) и ПО.</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/domestic">Открыть</a>
    </div></div>
  </div>
  <div class="col-md-6 col-lg-4">
    <div class="card h-100"><div class="card-body">
      <h2 class="h6">Legacy-имена</h2>
      <p class="text-muted small mb-2">КЕ по прежнему профилю именования (Guide 2019) с предложением имени по CMDB-NC-2026.</p>
      <a class="btn btn-sm btn-outline-secondary" href="/reports/legacy-names">Открыть</a>
    </div></div>
  </div>
</div>

<p class="text-muted small mt-3">Дашборд показателей качества данных (полнота атрибутов, устаревание, дубликаты, открытые конфликты) — <a href="/quality">отдельная страница</a>.</p>
