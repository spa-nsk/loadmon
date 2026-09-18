-- �������: server_hardware, cmdb_rack_cabinet, cmdb_datacenter_site.
-- ������� ����������� (snake_case �� ����� ����, ��� ��������������� column(...)
-- � ���� ��� FK/Name �����).
--
-- ���� ����� ���������� ������ � ������ AND sh.id = :id � WHERE.
-- LEFT JOIN �������� �� ������, ���� RackCabinetId ��������� �� ���� ���
-- ������������ DatacenterSiteId (���� � RackCabinet nullable) � �����
-- datacenter_name ����� NULL, �� ������ �� ����� �� ����� ��������.

SELECT
    sh.id            AS server_hardware_id,
    ds.name          AS datacenter_name,
    rc.name          AS rack_cabinet_name
FROM server_hardware sh
LEFT JOIN cmdb_rack_cabinet rc
    ON rc.id = sh.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds
    ON ds.id = rc.datacenter_site_id
WHERE sh.rack_cabinet_id IS NOT NULL
ORDER BY sh.id;
