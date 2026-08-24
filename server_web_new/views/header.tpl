<!doctype html>
<html lang="ru" data-bs-theme="light">

<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="Монитор утилизации ресурсов" />
  <meta name="author" content="Pavel Semenov" />
  <meta name="generator" content="spa.usvc.global.bcs" />
  <title>Монитор утилизации ресурсов</title>

  <!-- ?v={{.StaticVer}} - cache-busting: значение меняется при каждом
       перезапуске/выкладке приложения (см. staticAssetVersion в
       controllers/main_controller.go), чтобы браузер не продолжал
       использовать старую закэшированную версию файла после деплоя -
       именно это раньше приводило к пропаданию полос прокрутки у части
       пользователей, пока не чистили кэш вручную. -->
  <link href="/static/5.3.8/css/bootstrap.min.css?v={{.StaticVer}}" rel="stylesheet" />
  <link href="/static/css/sidebars.css?v={{.StaticVer}}" rel="stylesheet" />
  <link rel="stylesheet" href="/static/css/starter-template.css?v={{.StaticVer}}">
  <!-- общие стили форм /sprsrv (закрепляемая панель фильтра, "классическая"
       таблица, индикатор питания, панель пагинации, контекстное меню
       строки, карточка данных) - вынесены сюда из локальных <style> блоков
       шаблонов, образец - /sprsrv/view2, см. комментарий в самом файле -->
  <link rel="stylesheet" href="/static/css/loadmon.css?v={{.StaticVer}}">


  <meta name="theme-color" content="#712cf9" />

  <script src="/static/5.3.8/js/bootstrap.bundle.min.js?v={{.StaticVer}}" class="astro-vvvwv3sm"></script>
  <script src="/static/js/sidebars.js?v={{.StaticVer}}" class="astro-vvvwv3sm"></script>
  <script src="/static/js/color-modes.js?v={{.StaticVer}}"></script>
  <script src="/static/js2/jquery.min.js?v={{.StaticVer}}"></script>

  <style>
    html, body {
      height: 100%;
      margin: 0;
    }

    /* левая панель меню: фиксированная высота вьюпорта и собственная прокрутка,
       не зависящая от размеров окна и содержимого страницы */
    .sidebar-panel {
      width: 280px;
      max-width: 42vw;
      height: 100vh;
      overflow-y: auto;
      overflow-x: hidden;
      padding: 1rem 0.85rem;
      background-color: var(--bs-tertiary-bg);
      border-right: 1px solid var(--bs-border-color);
      scrollbar-width: thin;
    }

    .sidebar-panel .btn-toggle {
      width: 100%;
      text-align: left;
      font-size: 0.95rem;
      border-radius: 0.5rem;
    }

    .sidebar-panel .btn-toggle-nav a {
      display: block;
      padding: 0.35rem 0.6rem;
      border-radius: 0.4rem;
      transition: background-color .15s ease, color .15s ease;
    }

    .sidebar-panel .btn-toggle-nav a.link-menu:hover,
    .sidebar-panel .btn-toggle-nav a.link-menu:focus {
      background-color: var(--bs-primary) !important;
      color: #fff !important;
    }

    .sidebar-panel .brand-block {
      position: sticky;
      top: 0;
      background-color: var(--bs-tertiary-bg);
      z-index: 1;
    }

    .sidebar-panel .sidebar-footer {
      font-size: 0.8rem;
      border-top: 1px solid var(--bs-border-color);
      margin-top: 1rem;
      padding-top: 0.75rem;
    }

    .bd-placeholder-img {
      font-size: 1.125rem;
      text-anchor: middle;
      -webkit-user-select: none;
      -moz-user-select: none;
      user-select: none;
    }

    @media (min-width: 768px) {
      .bd-placeholder-img-lg {
        font-size: 3.5rem;
      }
    }

    .b-vertical-divider {
      width: 100%;
      height: 3rem;
      background-color: #0000001a;
      border: solid rgba(0, 0, 0, 0.15);
      border-width: 1px 0;
      box-shadow:
        inset 0 0.5em 1.5em #0000001a,
        inset 0 0.125em 0.5em #00000026;
    }

    .b-line-divider {
      width: 100%;
      height: 0.3rem;
      background-color: #0000001a;
      border: solid rgba(0, 0, 0, 0.15);
      border-width: 1px 0;
      box-shadow:
        inset 0 0.5em 1.5em #0000001a,
        inset 0 0.125em 0.5em #00000026;
    }

    .b-main-vr {
      flex-shrink: 0;
      width: 0.2rem;
      height: 100vh;
    }

    .bi {
      vertical-align: -0.125em;
      fill: currentColor;
    }

    .nav-scroller {
      position: relative;
      z-index: 2;
      height: 2.75rem;
      overflow-y: hidden;
    }

    .nav-scroller .nav {
      display: flex;
      flex-wrap: nowrap;
      padding-bottom: 1rem;
      margin-top: -1px;
      overflow-x: auto;
      text-align: center;
      white-space: nowrap;
      -webkit-overflow-scrolling: touch;
    }

    /* переключатель темы: жёлто-оранжевая луна на чёрном фоне (тёмная тема) /
       жёлтое солнце на белом фоне (светлая тема) - вместо прежней фиолетовой
       кнопки, выбивавшейся из общей цветовой гаммы интерфейса */
    .theme-toggle-wrap {
      display: flex;
      justify-content: center;
      margin-top: 0.85rem;
    }

    .theme-toggle-btn {
      position: relative;
      width: 3.38rem;
      height: 3.38rem;
      border-radius: 50%;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      border: 1px solid var(--bs-border-color);
      background-color: #bee7fb;
      color: #f5a623;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
      cursor: pointer;
      transition: background-color .2s ease, color .2s ease, border-color .2s ease, transform .15s ease;
    }

    .theme-toggle-btn:hover {
      transform: scale(1.08);
    }

    .theme-toggle-btn .theme-toggle-icon {
      width: 1.69em;
      height: 1.69em;
      transition: filter .2s ease;
    }

    /* эффект сияния у солнца (светлая тема) */
    html:not([data-bs-theme="dark"]) .theme-toggle-btn .theme-toggle-icon {
      filter: drop-shadow(0 0 3px rgba(245, 166, 35, 0.85)) drop-shadow(0 0 7px rgba(245, 166, 35, 0.55));
    }

    [data-bs-theme="dark"] .theme-toggle-btn {
      background-color: #14141a;
      color: #ffb454;
      border-color: #3a3a42;
    }

    /* подпись при наведении курсора мыши */
    .theme-toggle-btn::after {
      content: attr(data-tooltip);
      position: absolute;
      bottom: calc(100% + 0.5rem);
      left: 50%;
      transform: translateX(-50%) translateY(4px);
      white-space: nowrap;
      background-color: var(--bs-emphasis-color);
      color: var(--bs-body-bg);
      font-size: 0.75rem;
      padding: 0.3rem 0.6rem;
      border-radius: 0.35rem;
      opacity: 0;
      pointer-events: none;
      transition: opacity .15s ease, transform .15s ease;
      z-index: 10;
    }

    .theme-toggle-btn:hover::after,
    .theme-toggle-btn:focus-visible::after {
      opacity: 1;
      transform: translateX(-50%) translateY(0);
    }

    .link-menu:hover, .link-menu:focus {
      background-color: rgba(120, 120, 120, var(--bs-link-opacity, 1)) !important;
      color: RGBA(255, 255, 255, var(--bs-link-opacity, 1)) !important;
    }
  </style>

</head>

<body onload="setFocusList()" style="padding-top:0px; padding-bottom:0px; overflow:hidden">

<svg xmlns="http://www.w3.org/2000/svg" class="d-none">

<symbol id="database" viewBox="0 0 16 16">
<path d="M1.333 2.667C1.333 1.194 4.318 0 8 0s6.667 1.194 6.667 2.667V4c0 1.473-2.985 2.667-6.667 2.667S1.333 5.473 1.333 4z"/>
<path d="M1.333 6.334v3C1.333 10.805 4.318 12 8 12s6.667-1.194 6.667-2.667V6.334a6.5 6.5 0 0 1-1.458.79C11.81 7.684 9.967 8 8 8s-3.809-.317-5.208-.876a6.5 6.5 0 0 1-1.458-.79z"/>
<path d="M14.667 11.668a6.5 6.5 0 0 1-1.458.789c-1.4.56-3.242.876-5.21.876-1.966 0-3.809-.316-5.208-.876a6.5 6.5 0 0 1-1.458-.79v1.666C1.333 14.806 4.318 16 8 16s6.667-1.194 6.667-2.667z"/>
</symbol>

<symbol id="hdd" viewBox="0 0 16 16">
<path d="M12 2.5a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0m0 11a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0m-7.5.5a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1M5 2.5a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0M8 8a1 1 0 1 0 0-2 1 1 0 0 0 0 2"/>
<path d="M12 7a4 4 0 0 1-3.937 4c-.537.813-1.02 1.515-1.181 1.677a1.102 1.102 0 0 1-1.56-1.559c.1-.098.396-.314.795-.588A4 4 0 0 1 8 3a4 4 0 0 1 4 4m-1 0a3 3 0 1 0-3.891 2.865c.667-.44 1.396-.91 1.955-1.277.58-.38 1.008-.742 1.223-1.058A3 3 0 0 0 11 7"/>
<path d="M2 2a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2zm11 0a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1z"/>
</symbol>

<symbol id="ssd" viewBox="0 0 16 16">
<path d="M4.75 4a.75.75 0 0 0-.75.75v3.5c0 .414.336.75.75.75h6.5a.75.75 0 0 0 .75-.75v-3.5a.75.75 0 0 0-.75-.75zM5 8V5h6v3zm0-5.5a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0m7 0a.5.5 0 1 1-1 0 .5.5 0 0 1 1 0M4.5 11a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1m7 0a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1"/>
<path d="M2 2a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2zm11 12V2a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1z"/>
</symbol>

<symbol id="display" viewBox="0 0 16 16">
<path d="M0 4s0-2 2-2h12s2 0 2 2v6s0 2-2 2h-4q0 1 .25 1.5H11a.5.5 0 0 1 0 1H5a.5.5 0 0 1 0-1h.75Q6 13 6 12H2s-2 0-2-2zm1.398-.855a.757.757 0 0 0-.254.302A1.46 1.46 0 0 0 1 4.01V10c0 .325.078.502.145.602.173.137.351.207.562.207.25 0 .418-.126.493-.311.075-.185.133-.437.133-.752V4.01a1.498 1.498 0 0 0-1.354-1.355zM14 4.01V10c0 .325-.078.502-.145.602-.173.137-.351.207-.562.207-.25 0-.418-.126-.493-.311A1.47 1.47 0 0 1 13 9.01V4.01a1.498 1.498 0 0 1 1.354-1.355A1.46 1.46 0 0 1 14 4.01z"/>
</symbol>

<symbol id="floppy" viewBox="0 0 16 16">
<path d="M11 2H9v3h2z"/>
<path d="M1.5 0h11.586a1.5 1.5 0 0 1 1.06.44l1.415 1.414A1.5 1.5 0 0 1 16 2.914V14.5a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 0 14.5v-13A1.5 1.5 0 0 1 1.5 0M1 1.5v13a.5.5 0 0 0 .5.5H2v-4.5A1.5 1.5 0 0 1 3.5 9h9a1.5 1.5 0 0 1 1.5 1.5V15h1.5a.5.5 0 0 0 .5-.5V2.914a.5.5 0 0 0-.146-.353l-1.415-1.414A.5.5 0 0 0 13.086 1H13v4.5A1.5 1.5 0 0 1 11.5 7h-7A1.5 1.5 0 0 1 3 5.5V1H1.5a.5.5 0 0 0-.5.5z"/>
</symbol>

<symbol id="delete" viewBox="0 0 16 16">
<path d="M0 2a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v2a1 1 0 0 1-1 1v7.5a2.5 2.5 0 0 1-2.5 2.5h-9A2.5 2.5 0 0 1 1 12.5V5a1 1 0 0 1-1-1zm2 3v7.5A1.5 1.5 0 0 0 3.5 14h9a1.5 1.5 0 0 0 1.5-1.5V5zm13-3H1v2h14zM5 7.5a.5.5 0 0 1 .5-.5h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5m0 2a.5.5 0 0 1 .5-.5h2a.5.5 0 0 1 0 1h-2a.5.5 0 0 1-.5-.5"/>
</symbol>

<symbol id="power" viewBox="0 0 16 16">
<path d="M7.5 1v7h1V1z"/>
<path d="M3 8.812a5 5 0 0 1 2.578-4.375l-.485-.874A6 6 0 1 0 11 3.616l-.501.865A5 5 0 1 1 3 8.812z"/>
</symbol>

<symbol id="activity" viewBox="0 0 16 16">
<path fill-rule="evenodd" d="M6 2a.5.5 0 0 1 .47.33L10 12.036l1.53-4.208A.5.5 0 0 1 12 7.5h3.5a.5.5 0 0 1 0 1h-3.15l-1.88 5.17a.5.5 0 0 1-.94 0L6 3.964 4.47 8.171A.5.5 0 0 1 4 8.5H.5a.5.5 0 0 1 0-1h3.15l1.88-5.17A.5.5 0 0 1 6 2z"/>
</symbol>

<symbol id="arrow-left" viewBox="0 0 16 16">
  <path fill-rule="evenodd" d="M15 8a.5.5 0 0 0-.5-.5H2.707l3.147-3.146a.5.5 0 1 0-.708-.708l-4 4a.5.5 0 0 0 0 .708l4 4a.5.5 0 0 0 .708-.708L2.707 8.5H14.5A.5.5 0 0 0 15 8z"/>
</symbol>

<symbol id="bi-trash3" viewBox="0 0 16 16">
  <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5M11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5a.5.5 0 0 1 .471-.532zm3.326 0a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5zm3.297.47.5 8.5a.5.5 0 1 1-.998.06l-.5-8.5a.5.5 0 0 1 .528-.47z"/>
</symbol>

<symbol id="graph-up" viewBox="0 0 16 16">
  <path fill-rule="evenodd" d="M0 0h1v15h15v1H0V0zm14.817 3.113a.5.5 0 0 1 .07.704l-4.5 5.5a.5.5 0 0 1-.74.037L7.06 6.767l-3.656 5.027a.5.5 0 0 1-.808-.588l4-5.5a.5.5 0 0 1 .758-.06l2.609 2.61 4.15-5.073a.5.5 0 0 1 .704-.072z"/>
</symbol>

<symbol id="graph-up2" viewBox="0 0 16 16">
  <path fill-rule="evenodd" d="M0 0h1v15h15v1H0V0zm14.817 3.113a.5.5 0 0 1 .07.704l-4.5 5.5a.5.5 0 0 1-.74.037L7.06 6.767l-3.656 5.027a.5.5 0 0 1-.808-.588l4-5.5a.5.5 0 0 1 .758-.06l2.609 2.61 4.15-5.073a.5.5 0 0 1 .704-.072z"/>
</symbol>

<symbol id="house-fill" viewBox="0 0 16 16">
  <path d="M8.707 1.5a1 1 0 0 0-1.414 0L.646 8.146a.5.5 0 0 0 .708.708L8 2.207l6.646 6.647a.5.5 0 0 0 .708-.708L13 5.793V2.5a.5.5 0 0 0-.5-.5h-1a.5.5 0 0 0-.5.5v1.293L8.707 1.5z"/>
  <path d="m8 3.293 6 6V13.5a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 2 13.5V9.293l6-6z"/>
</symbol>

<symbol id="check2" viewBox="0 0 16 16">
  <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
</symbol>

<symbol id="circle-half" viewBox="0 0 16 16">
  <path d="M8 15A7 7 0 1 0 8 1v14zm0 1A8 8 0 1 1 8 0a8 8 0 0 1 0 16z"/>
</symbol>

<symbol id="moon-stars-fill" viewBox="0 0 16 16">
  <path d="M6 .278a.768.768 0 0 1 .08.858 7.208 7.208 0 0 0-.878 3.46c0 4.021 3.278 7.277 7.318 7.277.527 0 1.04-.055 1.533-.16a.787.787 0 0 1 .81.316.733.733 0 0 1-.031.893A8.349 8.349 0 0 1 8.344 16C3.734 16 0 12.286 0 7.71 0 4.266 2.114 1.312 5.124.06A.752.752 0 0 1 6 .278z"/>
  <path d="M10.794 3.148a.217.217 0 0 1 .412 0l.387 1.162c.173.518.579.924 1.097 1.097l1.162.387a.217.217 0 0 1 0 .412l-1.162.387a1.734 1.734 0 0 0-1.097 1.097l-.387 1.162a.217.217 0 0 1-.412 0l-.387-1.162A1.734 1.734 0 0 0 9.31 6.593l-1.162-.387a.217.217 0 0 1 0-.412l1.162-.387a1.734 1.734 0 0 0 1.097-1.097l.387-1.162zM13.863.099a.145.145 0 0 1 .274 0l.258.774c.115.346.386.617.732.732l.774.258a.145.145 0 0 1 0 .274l-.774.258a1.156 1.156 0 0 0-.732.732l-.258.774a.145.145 0 0 1-.274 0l-.258-.774a1.156 1.156 0 0 0-.732-.732l-.774-.258a.145.145 0 0 1 0-.274l.774-.258c.346-.115.617-.386.732-.732L13.863.099z"/>
</symbol>

<symbol id="sun-fill" viewBox="0 0 16 16">
  <path d="M8 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM8 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 0zm0 13a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2a.5.5 0 0 1 .5-.5zM4.464 4.464a.5.5 0 0 1 .707 0l1.414 1.414a.5.5 0 1 1-.707.707L4.464 5.171a.5.5 0 0 1 0-.707zm7.072 7.072a.5.5 0 0 1 .707 0l1.414 1.414a.5.5 0 0 1-.707.707l-1.414-1.414a.5.5 0 0 1 0-.707zM16 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5zM3 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5zm7.072-7.072a.5.5 0 0 1 0 .707l-1.414 1.414a.5.5 0 1 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0zm-7.072 7.072a.5.5 0 0 1 0 .707L1.586 9.414a.5.5 0 1 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0z"/>
</symbol>
</svg>

<main class="d-flex flex-nowrap" style="overflow:hidden;">
  <h1 class="visually-hidden">Мониторинг утилизации ресурсов</h1>
  <div class="b-vertical-divider b-main-vr"></div>
  <div class="flex-shrink-0 sidebar-panel">
    <div class="brand-block">
      <a href="/" class="d-flex align-items-center pb-3 mb-2 link-body-emphasis text-decoration-none border-bottom">
        <span class="fs-5 fw-semibold">Мониторинг ресурсов</span>
      </a>
    </div>
    <ul class="list-unstyled ps-0">
      <li class="mb-1">
        <button class="btn btn-toggle d-inline-flex align-items-center rounded border-0 collapsed"
          data-bs-toggle="collapse" data-bs-target="#home-collapse" aria-expanded="true">
          Общие
        </button>
        <div class="collapse show" id="home-collapse">
          <ul class="btn-toggle-nav list-unstyled fw-normal pb-1 small">
            {{if .InSession}}
            <li>
              <a href="/user/logout" class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Выйти</a>
            </li>
            {{else}}
            <li>
              <a href="/user/login/sprsrv_view2_all_0" 
                class="link-body-emphasis d-inline-flex text-decoration-none rounded">Авторизоваться</a>
            </li>
            {{end}}
          </ul>
        </div>
      </li>
{{if .InSession}}
      <li class="mb-1">
        <button class="btn btn-toggle d-inline-flex align-items-center rounded border-0 collapsed"
          data-bs-toggle="collapse" data-bs-target="#dashboard-collapse" aria-expanded="false">
          Данные из Vmware
        </button>
        <div class="collapse hide" id="dashboard-collapse">
          <ul class="btn-toggle-nav list-unstyled fw-normal pb-1 small">
            <li>
              <a href="/sprsrv/vm/all/all/all/all" 
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Загрузка виртуальных машин с
                00:00</a>
            </li>
            <li>
              <a href="/sprsrv/listclusterview"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Список Кластеров</a>
            </li>
            <li>
              <a href="/sprsrv/cluster/all/all/all"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Загрузка кластеров и хостов с
                00:00</a>
            </li>
            <li>
              <a href="/sprsrv/clusterperioddb"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Загрузка кластеров и хостов за
                период (по данным БД)</a>
            </li>
          </ul>
        </div>
      </li>
      <li class="mb-1">
        <button class="btn btn-toggle d-inline-flex align-items-center rounded border-0 collapsed"
          data-bs-toggle="collapse" data-bs-target="#orders-collapse" aria-expanded="false">
          Данные из HP One View
        </button>
        <div class="collapse hide" id="orders-collapse">
          <ul class="btn-toggle-nav list-unstyled fw-normal pb-1 small">
            <li>
              <a href="/sprsrv/hwviewlist"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Список серверов из HP One
                View</a>
            </li>
            <li>
              <a href="/sprsrv/srvhwviewbysn/000000"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Конфигурация сервера по
                серийному номеру</a>
            </li>
          </ul>
        </div>
      </li>
      <li class="mb-1">
        <button class="btn btn-toggle d-inline-flex align-items-center rounded border-0 collapsed"
          data-bs-toggle="collapse" data-bs-target="#spr-collapse" aria-expanded="false">
          Справочники
        </button>
        <div class="collapse hide" id="spr-collapse">
          <ul class="btn-toggle-nav list-unstyled fw-normal pb-1 small">
            <li>
              <a href="/sprsrv/view2/all/0"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Справочник серверов</a>
            </li>
            <li>
              <a href="/sprsrv/hwconflist/0"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Конфигурации серверов</a>
            </li>
            <li>
              <a href="/sprsrv/storagesystemslist/0"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Справочник СХД</a>
            </li>
            <li>
              <a href="/sprsrv/bladesystemslist/0"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Блейд-корзины</a>
            </li>
            <li>
              <a href="/sprsrv/clustersprlist/0"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Справочник кластеров</a>
            </li>
          </ul>
        </div>
      </li>
{{if eq .Privilage 3}}
      <li class="mb-1">
        <button class="btn btn-toggle d-inline-flex align-items-center rounded border-0 collapsed"
          data-bs-toggle="collapse" data-bs-target="#admin-collapse" aria-expanded="false">
          Администрирование
        </button>
        <div class="collapse hide" id="admin-collapse">
          <ul class="btn-toggle-nav list-unstyled fw-normal pb-1 small">
            <li>
              <a href="/admin/userlog"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Журнал действий
                пользователей</a>
            </li>
            <li>
              <a href="/admin/metricsexport"
                class="link-body-emphasis d-inline-flex text-decoration-none rounded link-menu">Выгрузка метрик
                утилизации</a>
            </li>
          </ul>
        </div>
      </li>
{{end}}
{{end}}
    </ul>
    <div class="sidebar-footer text-center">
    {{if .InSession}}
    <p class="text-muted mb-1">
      Добро пожаловать:<br>{{.First}} {{.Last}}<br>{{.Username}}
    </p>
    {{end}}

    <p class="text-muted mb-1">
      &copy; 2019
    </p>
    <p class="text-muted mb-1">
      <a href="http://www.bcs.ru">BCS</a> <a href="https://confluence.bcs.ru/pages/viewpage.action?pageId=670280406" target="_blank">ДАТИ ФГ БКС</a>
    </p>
    <p class="text-muted mb-0">
      <a href="https://confluence.bcs.ru/display/~SemenovPA" target="_blank">Семенов Павел</a>
    </p>

    <div class="theme-toggle-wrap">
      <button
        type="button"
        id="themeToggleBtn"
        class="theme-toggle-btn"
        data-tooltip-light="Светлая тема (нажмите для тёмной)"
        data-tooltip-dark="Тёмная тема (нажмите для светлой)"
        aria-label="Переключить тему оформления"
      >
        <svg class="bi theme-toggle-icon" aria-hidden="true">
          <use href="#sun-fill"></use>
        </svg>
      </button>
    </div>
    <script>
      (function () {
        var btn = document.getElementById('themeToggleBtn');
        if (!btn) return;
        var icon = btn.querySelector('.theme-toggle-icon use');

        function applyState(theme) {
          var isDark = theme === 'dark';
          icon.setAttribute('href', isDark ? '#moon-stars-fill' : '#sun-fill');
          btn.setAttribute('data-tooltip', isDark ? btn.getAttribute('data-tooltip-dark') : btn.getAttribute('data-tooltip-light'));
          btn.setAttribute('aria-pressed', String(isDark));
        }

        applyState(document.documentElement.getAttribute('data-bs-theme'));

        btn.addEventListener('click', function () {
          var current = document.documentElement.getAttribute('data-bs-theme');
          var next = current === 'dark' ? 'light' : 'dark';
          document.documentElement.setAttribute('data-bs-theme', next);
          localStorage.setItem('theme', next);
          applyState(next);
        });
      })();
    </script>
    </div>
  </div>

  <div class="b-vertical-divider b-main-vr"></div>

  <div class="flex-grow-1 p-3 p-md-4 content-panel" style="height:100vh;">
