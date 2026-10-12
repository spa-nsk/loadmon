// Поведение layout: тёмная тема, тосты, подсветка ошибок полей. События приходят от сервера
// в заголовке HX-Trigger (internal/controllers/ui/base.go).
(function () {
  'use strict';

  function toast(kind, text) {
    var box = document.getElementById('toasts');
    if (!box || !window.bootstrap) { return; }
    var el = document.createElement('div');
    el.className = 'toast align-items-center border-0 text-bg-' + kind;
    el.setAttribute('role', 'alert');
    el.innerHTML = '<div class="d-flex"><div class="toast-body"></div>' +
      '<button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast" aria-label="Закрыть"></button></div>';
    el.querySelector('.toast-body').textContent = text;
    box.appendChild(el);
    el.addEventListener('hidden.bs.toast', function () { el.remove(); });
    new bootstrap.Toast(el, { delay: kind === 'danger' ? 8000 : 4000 }).show();
  }

  // «Показать на дату» (E6.1): момент хранится в cookie asof (путь /), читается сервером
  // (internal/filters.AsOf). Значение — UTC, формат RFC3339 без экранирования (двоеточия допустимы).
  function setAsOf(value) {
    if (value) {
      document.cookie = 'asof=' + value + ':00Z; path=/; SameSite=Lax';
    } else {
      document.cookie = 'asof=; path=/; max-age=0; SameSite=Lax';
    }
    var u = new URL(window.location.href);
    u.searchParams.delete('asof'); // явный ?asof= в адресе сильнее cookie — убрать, чтобы не «залипал»
    window.location.href = u.toString();
  }
  var asofApply = document.getElementById('asof-apply');
  if (asofApply) {
    asofApply.addEventListener('click', function () {
      var v = document.getElementById('asof-input').value; // 2026-09-01T10:00
      if (!v) { setAsOf(''); return; }
      setAsOf(v);
    });
  }
  var asofReset = document.getElementById('asof-reset');
  if (asofReset) { asofReset.addEventListener('click', function () { setAsOf(''); }); }

  document.body.addEventListener('toast', function (e) {
    toast(e.detail.kind || 'info', e.detail.text || '');
  });

  // Ошибки по полям: подсветить поля формы, из которой ушёл запрос.
  document.body.addEventListener('fieldErrors', function (e) {
    var form = e.target && e.target.closest ? e.target.closest('form') : null;
    if (!form) { return; }
    Object.keys(e.detail).forEach(function (name) {
      var field = form.querySelector('[name="' + name + '"]');
      if (field) { field.classList.add('is-invalid'); field.title = e.detail[name]; }
    });
  });
  document.body.addEventListener('htmx:beforeRequest', function (e) {
    var form = e.target && e.target.closest ? e.target.closest('form') : null;
    if (form) { form.querySelectorAll('.is-invalid').forEach(function (f) { f.classList.remove('is-invalid'); }); }
  });

  document.body.addEventListener('htmx:responseError', function (e) {
    toast('danger', 'Ошибка сервера (' + e.detail.xhr.status + '). Подробности в журнале cmdb-web.');
  });
  document.body.addEventListener('htmx:sendError', function () {
    toast('danger', 'Нет связи с сервером.');
  });

  // Массовые операции (список КЕ, E3.6): показать только выбранное в data-bulk-field поле
  // значения. Делегирование на document.body — тулбар переживает HTMX-подмену соседних узлов без
  // переустановки обработчика.
  document.body.addEventListener('change', function (e) {
    if (e.target.matches('[data-bulk-field]')) {
      var chosen = e.target.value;
      document.querySelectorAll('[data-bulk-value]').forEach(function (el) {
        el.hidden = el.getAttribute('data-bulk-value') !== chosen;
      });
    }
  });

  // Модальные формы (E3.1) внутри вкладок карточки КЕ: при успешной отправке весь #card-tabs
  // подменяется целиком (hx-swap="outerHTML"), включая открытую модалку — Bootstrap не успевает
  // вызвать свой обычный hide() и оставляет .modal-backdrop и класс modal-open на <body>. Без этой
  // подчистки страница осталась бы затемнённой и некликабельной после каждой успешной отправки.
  document.body.addEventListener('htmx:afterSwap', function (e) {
    if (e.target && e.target.id === 'card-tabs') {
      document.querySelectorAll('.modal-backdrop').forEach(function (el) { el.remove(); });
      document.body.classList.remove('modal-open');
      document.body.style.removeProperty('overflow');
      document.body.style.removeProperty('padding-right');
    }
  });

  var themeBtn = document.getElementById('theme-toggle');
  if (themeBtn) {
    themeBtn.addEventListener('click', function () {
      var root = document.documentElement;
      var next = root.getAttribute('data-bs-theme') === 'dark' ? 'light' : 'dark';
      root.setAttribute('data-bs-theme', next);
      try { localStorage.setItem('theme', next); } catch (err) { /* хранилище недоступно */ }
    });
  }
})();
