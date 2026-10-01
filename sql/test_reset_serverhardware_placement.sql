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
-- ВАЖНО: у ServerHardware размещение описывается ТОЛЬКО RackCabinetId (FK
-- на справочник models.RackCabinet) и числовыми StartUnitPos/SizeUnit.
-- Текстовые DatacenterRoom/DatacenterName/RackName/RackNum/StartUnit УДАЛЕНЫ
-- из models.ServerHardware (2026-10-01, атавизм после внедрения
-- RackCabinetId) - ниже зачищаются только оставшиеся поля размещения.
--
-- Совместим с обоими драйверами БД проекта (postgres и sqlite3).
-- =============================================================================

-- Зачистка размещения ВСЕХ серверов/гипервизоров:
UPDATE server_hardware
SET rack_cabinet_id = 0,
    start_unit_pos  = 0,
    size_unit       = 0
WHERE rack_cabinet_id > 0
   OR start_unit_pos > 0
   OR size_unit > 0;

-- Вариант с фильтром по конкретным серверам (раскомментировать и подставить
-- нужные серийные номера вместо примера, вместо UPDATE выше):
-- UPDATE server_hardware
-- SET rack_cabinet_id = 0, start_unit_pos = 0, size_unit = 0
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
-- 6) DatacenterName/RackName (поля HP OneView), DatacenterRoom (поле
--    Итиллиум) и текстовые RackNum/StartUnit в модели больше нет - опрос
--    HP OneView их не восстановит.
