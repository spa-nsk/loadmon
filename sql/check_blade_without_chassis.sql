-- Проверка блейд-серверов (server_hardware.form_factor = 2) на соответствие правилу:
-- включенный блейд может находиться только в блейд-корзине, отдельно допустимы
-- только списанные и находящиеся на складе (itilium_status 3 и 5).

-- 1. Блейды без корзины (blade_system_id пуст/0), сгруппированы по статусам.
SELECT sh.active, sh.itilium_status, COUNT(*) AS cnt
FROM server_hardware sh
WHERE sh.form_factor = 2 AND COALESCE(sh.blade_system_id, 0) = 0
GROUP BY sh.active, sh.itilium_status
ORDER BY sh.active, sh.itilium_status;

-- 2. Нарушения: блейд не выведен из эксплуатации (active <> 2), но корзины нет -
--    именно они видны в форме шкафа как "без позиции".
SELECT sh.id, sh.serial_num, sh.model, sh.active, sh.power_state, sh.itilium_status,
       rc.name AS rack_cabinet, ds.name AS datacenter_site
FROM server_hardware sh
LEFT JOIN cmdb_rack_cabinet rc ON rc.id = sh.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds ON ds.id = rc.datacenter_site_id
WHERE sh.form_factor = 2
  AND COALESCE(sh.blade_system_id, 0) = 0
  AND sh.active <> 2
ORDER BY ds.name, rc.name, sh.serial_num;

-- 3. Работающий (active = 1) блейд в корзине, но шкаф блейда не совпадает со шкафом корзины:
--    местоположение работающих блейдов определяется корзиной.
SELECT sh.id, sh.serial_num, bs.name AS chassis,
       sh.rack_cabinet_id AS blade_rack_id, bs.rack_cabinet_id AS chassis_rack_id
FROM server_hardware sh
JOIN blade_systems bs ON bs.id = sh.blade_system_id
WHERE sh.form_factor = 2 AND sh.active = 1
  AND COALESCE(sh.rack_cabinet_id, 0) <> COALESCE(bs.rack_cabinet_id, 0)
ORDER BY bs.name, sh.blade_num;

-- 4. Исправление пункта 3: работающий блейд размещается в шкафу своей корзины
--    (раскомментируйте после проверки результата запроса 3).
-- UPDATE server_hardware sh
-- SET rack_cabinet_id = bs.rack_cabinet_id
-- FROM blade_systems bs
-- WHERE bs.id = sh.blade_system_id
--   AND sh.form_factor = 2 AND sh.active = 1
--   AND COALESCE(bs.rack_cabinet_id, 0) <> 0
--   AND COALESCE(sh.rack_cabinet_id, 0) <> bs.rack_cabinet_id;
