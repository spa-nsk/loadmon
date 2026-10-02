-- =============================================================================
-- Проверка цепочки Vm_EsxHost -> DatacenterSite.CodeJira для vm_spr_record
-- (логика cmdb/vmdatacentername.go, PostgreSQL).
--
-- ПЕРЕИМЕНОВАНО (2026-09-30) из datastore_name - опечатка.
--
-- Подставьте vcenter и cluster_id (MOID кластера, как в cluster_spr_records).
-- Имена столбцов — snake_case Beego ORM; при расхождении со схемой: \d vm_spr_record.
-- =============================================================================

SELECT
    vm.id                   AS vm_spr_id,
    vm.vm_name,
    vm.vm_esx_host,
    vm.datacenter_name      AS stored_datacenter_name,
    ds.code_jira            AS resolved_code_jira,
    CASE
        WHEN COALESCE(vm.datacenter_name, '') = COALESCE(ds.code_jira, '')
        THEN 'ok'
        ELSE 'mismatch'
    END                     AS check_status
FROM vm_spr_record vm
LEFT JOIN server_spr_records h
    ON h.type = 2
   AND h.stat <> 666 -- 666 - запись помечена к уничтожению
   AND h.vcenter = vm.vcenter
   AND (
           lower(trim(h.name)) = lower(trim(vm.vm_esx_host))
        OR lower(trim(h.name_fqdn)) = lower(trim(vm.vm_esx_host))
       )
LEFT JOIN server_hardware sh
    ON sh.id = h.hw_spr_id
   AND h.hw_spr_id > 0
LEFT JOIN cmdb_rack_cabinet rc
    ON rc.id = sh.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds
    ON ds.id = rc.datacenter_site_id
WHERE vm.vcenter = 's-msk-vc-01.example:443'   -- TODO: ваш vCenter
  AND vm.cluster_id = 'domain-c2638'           -- TODO: MOID кластера
ORDER BY vm.vm_name;
