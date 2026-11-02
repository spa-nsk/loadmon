<!doctype html>
<html lang="ru" data-bs-theme="light">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Авторизация — Монитор утилизации ресурсов</title>
  <script src="/static/js/color-modes.js?v={{.StaticVer}}"></script>
  <style>
    html, body {
      margin: 0;
      height: 100%;
      overflow: hidden;
      background: #0a0a0a;
    }
    html:not([data-bs-theme="dark"]) body {
      background: #f4f4f2;
    }
    #matrixRain {
      display: block;
      width: 100vw;
      height: 100vh;
    }
    .auth-caption {
      position: fixed;
      left: 50%;
      bottom: 8%;
      transform: translateX(-50%);
      font-family: "Consolas", "Courier New", monospace;
      font-size: 0.95rem;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      pointer-events: none;
      z-index: 2;
      color: #111;
      opacity: 0.75;
    }
    [data-bs-theme="dark"] .auth-caption {
      color: #ffb000;
      text-shadow: 0 0 8px rgba(255, 176, 0, 0.85), 0 0 24px rgba(255, 140, 0, 0.45);
      opacity: 0.9;
    }
  </style>
</head>
<body data-auth-target="{{.AuthGateTarget}}" data-auth-min-ms="{{.AuthGateMinMs}}">
  <canvas id="matrixRain" aria-hidden="true"></canvas>
  <div class="auth-caption">Авторизация…</div>
  <script>
(function () {
  var body = document.body;
  var target = body.getAttribute('data-auth-target') || '/sprsrv/view2/all/0';
  // только относительный путь приложения — защита от битого/внешнего next
  if (target.charAt(0) !== '/' || target.indexOf('//') === 0) {
    target = '/sprsrv/view2/all/0';
  }
  var minMs = parseInt(body.getAttribute('data-auth-min-ms'), 10);
  if (!minMs || minMs < 0) {
    minMs = 15000;
  }
  var started = Date.now();

  function go() {
    var left = minMs - (Date.now() - started);
    if (left > 0) {
      setTimeout(function () { window.location.replace(target); }, left);
    } else {
      window.location.replace(target);
    }
  }
  // переход только после минимальной паузы (авторизация уже завершена на сервере)
  go();

  var canvas = document.getElementById('matrixRain');
  var ctx = canvas.getContext('2d');
  var cols = [];
  var fontSize = 16;
  var glyphs = 'ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ0123456789ABCDEF<>*+-=';

  function isDark() {
    return document.documentElement.getAttribute('data-bs-theme') === 'dark';
  }

  function resize() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    var n = Math.ceil(canvas.width / fontSize);
    cols = new Array(n);
    for (var i = 0; i < n; i++) {
      cols[i] = Math.random() * -40;
    }
  }

  function rainColor() {
    return isDark() ? '#ffb000' : '#111111';
  }

  function fadeFill() {
    if (isDark()) {
      ctx.fillStyle = 'rgba(8, 6, 0, 0.12)';
    } else {
      ctx.fillStyle = 'rgba(244, 244, 242, 0.14)';
    }
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }

  function frame() {
    fadeFill();
    var color = rainColor();
    ctx.font = fontSize + 'px Consolas, "Courier New", monospace';
    if (isDark()) {
      ctx.shadowColor = 'rgba(255, 176, 0, 0.9)';
      ctx.shadowBlur = 12;
    } else {
      ctx.shadowColor = 'transparent';
      ctx.shadowBlur = 0;
    }
    ctx.fillStyle = color;
    for (var i = 0; i < cols.length; i++) {
      var ch = glyphs.charAt((Math.random() * glyphs.length) | 0);
      var x = i * fontSize;
      var y = cols[i] * fontSize;
      ctx.globalAlpha = isDark() ? (0.35 + Math.random() * 0.65) : (0.45 + Math.random() * 0.55);
      ctx.fillText(ch, x, y);
      if (y > canvas.height && Math.random() > 0.975) {
        cols[i] = 0;
      } else {
        cols[i]++;
      }
    }
    ctx.globalAlpha = 1;
    ctx.shadowBlur = 0;
    requestAnimationFrame(frame);
  }

  window.addEventListener('resize', resize);
  resize();
  ctx.fillStyle = isDark() ? '#080600' : '#f4f4f2';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  requestAnimationFrame(frame);
})();
  </script>
</body>
</html>
