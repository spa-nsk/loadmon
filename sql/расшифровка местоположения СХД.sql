-- =============================================================================
-- Размещение одной СХД по storage_system_placement (models.StorageSystemPlacement).
-- Логика площадки: если указан шкаф — ЦОД из RackCabinet.DatacenterSiteId,
-- иначе из StorageSystemPlacement.DatacenterSiteId (см. models_add.go).
--
-- PostgreSQL (string_agg). Для списка всех СХД см. storage-systems.sql.
-- Подставьте storage_systems.id вместо 355.
-- =============================================================================

SELECT
    ss.id AS storage_systems_id,
    ss.short_name,
    string_agg(
        concat(
            COALESCE(ds.name, '(ЦОД не указан)'),
            ' / ',
            COALESCE(rc.name, '(шкаф не указан)'),
            CASE
                WHEN ssp.start_unit_pos > 0 OR ssp.end_unit_pos > 0
                THEN concat(' ', ssp.start_unit_pos, '-', ssp.end_unit_pos)
                ELSE ''
            END
        ),
        '; ' ORDER BY ds.name, rc.name, ssp.start_unit_pos
    ) AS placement,
    string_agg(
        DISTINCT NULLIF(ds.code_jira, ''),
        ';' ORDER BY NULLIF(ds.code_jira, '')
    ) AS datastore_name
FROM storage_systems ss
JOIN storage_system_placement ssp
    ON ssp.storage_systems_id = ss.id
LEFT JOIN cmdb_rack_cabinet rc
    ON rc.id = ssp.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds
    ON ds.id = COALESCE(NULLIF(rc.datacenter_site_id, 0), ssp.datacenter_site_id)
WHERE ss.id = 355
GROUP BY ss.id, ss.short_name;
