-- =============================================================================
-- Зачистка данных размещения справочника аппаратных серверов/гипервизоров
-- (models.ServerHardware, таблица server_hardware) для проверки корректной
-- загрузки размещения из файла Итиллиум и проверки отображения
-- (hwconflist.tpl/hwconfedit.tpl, фильтр /sprsrv/view2, карточка
-- /sprsrv/updatebase, канвасы /admin/hwinfra и /admin/infrastructure).
--
-- Как и у блейд-корзин (см. test_reset_blade_placement.sql) - у ServerHardware
-- нет отдельной таблицы-связки размещения, все поля хранятся прямо в таблице
-- server_hardware как обычные (не осиротевшие) поля текущей модели.
--
-- ВАЖНО: у ServerHardware, в отличие от BladeSystems/StorageSystems, ЕСТЬ два
-- параллельных набора полей размещения (см. models_add.go):
--   1) новая схема - RackCabinetId (FK на справочник models.RackCabinet);
--   2) старые текстовые поля из разных источников -
--      DatacenterRoom/RackNum (текст, приходит из Итиллиум) и
--      DatacenterName/RackName (текст, приходит из HP OneView).
-- Обе группы полей заполняются НЕЗАВИСИМО (Итиллиум пишет группу 1+2а, HP
-- OneView синхронизация пишет только группу 2б - см. cmdb/updatehwsprhp.go).
-- Ниже зачищается ВСЯ информация о размещении - если вам нужно проверить
-- ТОЛЬКО импорт Итиллиум и не трогать данные, которые может восстановить
-- следующий опрос HP OneView, закомментируйте строки с DatacenterName/RackName
-- (группа 2б) в UPDATE ниже - тогда HP OneView не перезапишет их обратно, но
-- они и не будут зачищены для проверки самого Итиллиум-импорта.
--
-- Совместим с обоими драйверами БД проекта (postgres и sqlite3).
-- =============================================================================

-- Зачистка размещения ВСЕХ серверов/гипервизоров (обе группы полей):
UPDATE server_hardware
SET rack_cabinet_id = 0,
    start_unit_pos  = 0,
    size_unit       = 0,
    start_unit      = '',
    datacenter_room = '',
    rack_num        = '',
    datacenter_name = '',
    rack_name       = ''
WHERE rack_cabinet_id > 0
   OR start_unit_pos > 0
   OR size_unit > 0
   OR coalesce(start_unit, '') <> ''
   OR coalesce(datacenter_room, '') <> ''
   OR coalesce(rack_num, '') <> ''
   OR coalesce(datacenter_name, '') <> ''
   OR coalesce(rack_name, '') <> '';

-- Вариант с фильтром по конкретным серверам (раскомментировать и подставить
-- нужные серийные номера вместо примера, вместо UPDATE выше):
-- UPDATE server_hardware
-- SET rack_cabinet_id = 0, start_unit_pos = 0, size_unit = 0, start_unit = '',
--     datacenter_room = '', rack_num = '', datacenter_name = '', rack_name = ''
-- WHERE serial_num IN ('SERIAL1', 'SERIAL2');

-- =============================================================================
-- Проверка результата:
-- =============================================================================
-- SELECT count(*) AS placed_left FROM server_hardware WHERE rack_cabinet_id > 0;  -- ожидается 0 (или не 0, если использовался вариант с фильтром)

-- Справочники площадок и шкафов (cmdb_datacenter_site / cmdb_rack_cabinet)
-- этим скриптом НЕ трогаются намеренно - общие данные, разделяемые с
-- размещением блейд-корзин/СХД, а не артефакт именно импорта серверов.

-- После выполнения:
-- 1) запустите импорт rack-серверов/блейд-серверов из Итиллиум как обычно;
-- 2) проверьте /sprsrv/hwconflist - колонка "Размещение" должна заполниться
--    заново из файла;
-- 3) проверьте карточку /sprsrv/hwconfedit/:id конкретного сервера;
-- 4) проверьте фильтр "Площадка/ЦОД" на /sprsrv/view2 и карточку
--    /sprsrv/updatebase/all/:id (блок "Физическое размещение");
-- 5) при необходимости проверьте канвасы /admin/hwinfra и /admin/infrastructure.
-- 6) учтите: если сервер также опрашивается HP OneView, следующий его опрос
--    сам восстановит DatacenterName/RackName (см. cmdb/updatehwsprhp.go) - это
--    ожидаемо, эти два поля не относятся к импорту Итиллиум.
