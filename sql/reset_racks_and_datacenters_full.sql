-- =============================================================================
-- ПОЛНАЯ зачистка справочников "Площадка/ЦОД" (cmdb_datacenter_site) и
-- "Шкаф" (cmdb_rack_cabinet) + всех ссылок на них - для чистой переимпорта
-- новыми методами/концепцией (2026-09-10, по запросу пользователя: "данные
-- отправить не смогу, запрет безопасности. Проще очистить базу и попробовать
-- загрузить новыми методами и концепцией" - в ответ на подозрение, что часть
-- шкафов сейчас привязана к ЧУЖОМУ ЦОД из-за старого импорт-бага, см.
-- CHANGELOG.md запись про задачу #155: до фикса уникальности пары (Name,
-- DatacenterSiteId) шкаф при импорте искался строго по имени и при совпадении
-- имён на разных площадках "перепривязывался" к площадке из последней
-- импортированной строки).
--
-- ⚠️ ЭТО ПОЛНАЯ, а не точечная зачистка - удаляются ВСЕ записи справочников
-- площадок и шкафов и ВСЕ ссылки на них (StorageSystemPlacement целиком,
-- RackCabinetId у BladeSystems/ServerHardware). Делайте бэкап БД перед
-- запуском. После зачистки справочники наполнятся заново САМИМ импортом
-- (resolveRackCabinet/датацентр по "Местоположение" заводят записи "на
-- лету", см. itilliumimport/*.go) - вручную заводить площадки/шкафы не
-- нужно, если весь набор данных приходит из Итиллиум.
--
-- Порядок важен из-за внешних ключей: сначала обнуляем/удаляем ссылки на
-- RackCabinet/DatacenterSite в других таблицах, потом сами справочники.
-- Совместимо с postgres и sqlite3 (без CASCADE/TRUNCATE, только
-- DELETE/UPDATE - тот же приём, что в остальных sql/test_reset_*.sql).
-- =============================================================================

-- 1) Размещение СХД (models.StorageSystemPlacement) - таблица-связка
--    целиком про размещение, безопасно удалить ВСЕ строки (и шкафные, и
--    "безшкафные" site-only, см. CHANGELOG запись про задачу #160).
DELETE FROM storage_system_placement;

-- 1а) "Осиротевшие" legacy-колонки storage_systems (если ещё физически
--     существуют в БД - см. подробный комментарий в
--     test_reset_storage_placement.sql про config.migrateStorageSystemsPlacement,
--     которая при старте приложения молча ВОССТАНОВИТ старые размещения из
--     этих колонок поверх очищенной выше storage_system_placement, если их
--     не обезвредить). Если колонок уже нет - UPDATE упадёт с "column does
--     not exist"/"no such column", это ожидаемо и безопасно пропустить.
UPDATE storage_systems SET rack_cabinet_id = 0, start_unit_pos = 0, end_unit_pos = 0, size_unit = 0
WHERE rack_cabinet_id IS NOT NULL AND rack_cabinet_id > 0;

-- 2) Размещение блейд-корзин (RackCabinetId - поле прямо в таблице
--    blade_systems, БЕЗ префикса cmdb_ - см. test_reset_blade_placement.sql).
UPDATE blade_systems
SET rack_cabinet_id = 0,
    start_unit_pos  = 0,
    end_unit_pos    = 0,
    size_unit       = 0
WHERE rack_cabinet_id > 0
   OR start_unit_pos > 0
   OR end_unit_pos > 0
   OR size_unit > 0;

-- 3) Размещение аппаратных серверов/гипервизоров - обе группы полей (новая
--    RackCabinetId + старые текстовые из Итиллиум/HP OneView), см. подробный
--    комментарий в test_reset_serverhardware_placement.sql.
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

-- 4) Сами справочники - сначала шкафы (ссылаются на площадки), потом
--    площадки.
DELETE FROM cmdb_rack_cabinet;
DELETE FROM cmdb_datacenter_site;

-- =============================================================================
-- Проверка результата (ожидается 0 везде):
-- =============================================================================
-- SELECT count(*) FROM cmdb_rack_cabinet;
-- SELECT count(*) FROM cmdb_datacenter_site;
-- SELECT count(*) FROM storage_system_placement;
-- SELECT count(*) FROM blade_systems WHERE rack_cabinet_id > 0;
-- SELECT count(*) FROM server_hardware WHERE rack_cabinet_id > 0;

-- =============================================================================
-- После выполнения:
-- =============================================================================
-- 1) перезапустите (или убедитесь, что запущен) свежий бинарник loadmon со
--    всеми правками этой сессии (см. CHANGELOG.md записи (55)-(63));
-- 2) заново запустите импорт из Итиллиум В ЭТОМ ПОРЯДКЕ: сначала блейд-
--    корзины/rack-серверы (создают базовые площадки/шкафы), затем СХД -
--    resolveRackCabinet (itilliumimport/resolvecabinet.go) при повторных
--    запусках ищет шкаф строго по паре (Name, DatacenterSiteId), поэтому
--    коллизии имён на разных площадках больше не будут "перепривязывать"
--    чужой шкаф - каждая площадка получит СВОИ отдельные записи шкафов;
-- 3) проверьте /sprsrv/view2 - каскадный фильтр "Площадка/ЦОД -> Шкаф" -
--    под каждым ЦОД должны быть только шкафы этого же ЦОД;
-- 4) проверьте /sprsrv/hwconflist, /sprsrv/storagesystemslist,
--    /sprsrv/bladesystemslist - колонка "Размещение".
