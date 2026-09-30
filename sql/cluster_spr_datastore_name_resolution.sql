-- =============================================================================
-- Проверка ClusterSprRecords.DatacenterName: объединение code_jira всех ESXi
-- (Type=2) кластера через ";" без дублей (cmdb/clusterdatacentername.go).
-- PostgreSQL.
--
-- ПЕРЕИМЕНОВАНО (2026-09-30) из DatastoreName/datastore_name - опечатка.
--
-- Подставьте cluster_spr_records.id или пару vcenter + cluster_id.
-- Порядок кодов в Go — sort.Strings; здесь string_agg ORDER BY code_jira
-- (тот же набор, стабильный порядок).
-- =============================================================================

SELECT
    csr.id              AS cluster_spr_id,
    csr.vcenter,
    csr.cluster_id,
    csr.cluster,
    csr.datacenter_name AS stored_datacenter_name,
    string_agg(
        DISTINCT NULLIF(ds.code_jira, ''),
        ';' ORDER BY NULLIF(ds.code_jira, '')
    )                   AS resolved_datacenter_name
FROM cluster_spr_records csr
JOIN server_spr_records h
    ON h.type = 2
   AND h.vcenter = csr.vcenter
   AND h.cluster_id = csr.cluster_id
   AND h.hw_spr_id > 0
LEFT JOIN server_hardware sh
    ON sh.id = h.hw_spr_id
LEFT JOIN cmdb_rack_cabinet rc
    ON rc.id = sh.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds
    ON ds.id = rc.datacenter_site_id
WHERE csr.id = 1   -- TODO: Id записи cluster_spr_records
GROUP BY csr.id, csr.vcenter, csr.cluster_id, csr.cluster, csr.datacenter_name;
