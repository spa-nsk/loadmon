-- =============================================================================
-- Список активных СХД (power=1): размещение и агрегированный datastore_name
-- (коды DatacenterSite.CodeJira по всем точкам размещения, через ";").
-- PostgreSQL (string_agg). Для одной СХД см. расшифровка местоположения СХД.sql
-- =============================================================================

SELECT ss.status,

    string_agg(concat(COALESCE(ds.name, '(ЦОД не указан)'::text), ' / ', COALESCE(rc.name, '(шкаф не указан)'::text),

        CASE

            WHEN ssp.start_unit_pos > 0 OR ssp.end_unit_pos > 0 THEN concat(' ', ssp.start_unit_pos, '-', ssp.end_unit_pos)

            ELSE ''::text

        END), '; '::text ORDER BY ds.name, rc.name, ssp.start_unit_pos) AS datacenter_room,

    string_agg(DISTINCT NULLIF(ds.code_jira, ''), ';'::text ORDER BY NULLIF(ds.code_jira, '')) AS datastore_name,

    ss.technical_number_i_t AS inventory_num,

    ss.ip_address,

    ss.model,

    ss.serial_num,

    ss.short_name,

    ss.size_unit,

    ss.start_unit_pos,

    ss.vendor,

    ss.u_i_d,

    ss.total_size_r_a_w,

    ss.used_percent,

    ss.type_work

   FROM storage_systems ss

     LEFT JOIN storage_system_placement ssp ON ssp.storage_systems_id = ss.id

     LEFT JOIN cmdb_rack_cabinet rc ON rc.id = ssp.rack_cabinet_id

     LEFT JOIN cmdb_datacenter_site ds ON ds.id = COALESCE(NULLIF(rc.datacenter_site_id, 0), ssp.datacenter_site_id)

  WHERE ss.power = 1

  GROUP BY ss.id, ss.status, ss.technical_number_i_t, ss.ip_address, ss.model, ss.serial_num, ss.short_name, ss.size_unit, ss.start_unit_pos, ss.vendor, ss.u_i_d, ss.total_size_r_a_w, ss.used_percent, ss.type_work

  ORDER BY ss.short_name;
