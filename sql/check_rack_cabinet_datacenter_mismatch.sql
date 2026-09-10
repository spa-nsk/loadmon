-- Диагностика "странных" шкафов в фильтре ЦОД/Шкаф на /sprsrv/view2
-- (2026-09-10, по запросу пользователя: "наблюдаю странные комбинации ЦОД
-- и шкафов. Иногда попадают шкафы из других ЦОД").
--
-- Гипотеза: JS-фильтр на форме честно сравнивает RackCabinet.DatacenterSiteId
-- (атрибут data-dc опции "Шкаф") с выбранным ЦОД - если в выпадающем списке
-- под одним ЦОД показываются шкафы, явно принадлежащие другому (например,
-- "6.2-B-02" под ЦОД "6.1 (624)"), это значит, что у соответствующей записи
-- cmdb_rack_cabinet реально ХРАНИТСЯ чужой DatacenterSiteId - вероятно,
-- наследие СТАРОГО бага импорта (до фикса уникальности пары (Name,
-- DatacenterSiteId), см. CHANGELOG.md запись про задачу #155): раньше шкаф
-- искался ПРИ ИМПОРТЕ строго по имени, и при совпадении имени на разных
-- площадках шкаф "перепривязывался" к площадке из последней импортированной
-- строки. Фикс #155 не является ретроактивным - он только не даёт этому
-- происходить ВПРЕДЬ, старые уже испорченные записи остаются как есть.
--
-- Запрос 1: все шкафы, чьё имя выглядит как "X.Y-..." (характерный паттерн
-- для площадок вида "Новосибирск, КЦ ... 6.1"/"6.2"), вместе с реальным ЦОД,
-- к которому они сейчас привязаны в БД - смотрим глазами, совпадает ли
-- префикс имени шкафа ("6.1"/"6.2"/...) с именем площадки.
SELECT
    rc.id            AS rack_id,
    rc.name          AS rack_name,
    rc.datacenter_site_id,
    ds.name          AS datacenter_site_name,
    ds.is_datacenter
FROM cmdb_rack_cabinet rc
LEFT JOIN cmdb_datacenter_site ds ON ds.id = rc.datacenter_site_id
ORDER BY rc.name;

-- Запрос 2: конкретно шкаф "6.2-B-02" (из скриншота) - к какому ЦОД он
-- реально привязан сейчас.
SELECT
    rc.id, rc.name, rc.datacenter_site_id, ds.name AS datacenter_site_name
FROM cmdb_rack_cabinet rc
LEFT JOIN cmdb_datacenter_site ds ON ds.id = rc.datacenter_site_id
WHERE rc.name = '6.2-B-02';

-- Запрос 3: площадки с похожими именами (потенциальные дубли одной и той
-- же физической площадки, заведённые под разными Id из-за небольших
-- различий в строке "Местоположение" при разных импортах - см. anotation
-- у DatacenterSite, обычно содержит адрес/этаж/комнату).
SELECT id, name, is_datacenter, annotation
FROM cmdb_datacenter_site
WHERE name LIKE '%6.1%' OR name LIKE '%6.2%' OR name LIKE '%Советской%'
ORDER BY name;

-- Запрос 4 (Москва, по первому скриншоту): шкафы вида "3.02-..." против
-- шкафов "ЦОД М1 / R15-*" - последние явно НЕ должны попадать в тот же ЦОД,
-- что и "3.02-...". Проверяем их реальную привязку.
SELECT
    rc.id, rc.name, rc.datacenter_site_id, ds.name AS datacenter_site_name
FROM cmdb_rack_cabinet rc
LEFT JOIN cmdb_datacenter_site ds ON ds.id = rc.datacenter_site_id
WHERE rc.name LIKE '3.02-%' OR rc.name LIKE 'R15-%' OR rc.name LIKE '%R15%'
ORDER BY rc.name;
