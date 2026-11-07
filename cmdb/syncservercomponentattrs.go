package cmdb

import "strings"

// Назначение физического диска (models.ServerHardwareComponents.Role) -
// нормализованные значения DiskDriveUse из OneView localStorage/iLO SmartStorage.
const (
	DiskRoleData   = "Data"   // рабочий диск (входит в логический массив)
	DiskRoleSpare  = "Spare"  // горячий резерв (hot spare)
	DiskRoleRaw    = "Raw"    // не сконфигурирован в массив
	DiskRoleFailed = "Failed" // отказавший
	DiskRoleCache  = "Cache"  // кэш-диск (SmartCache)
)

// normalizeDiskRole приводит DiskDriveUse к одному из DiskRole*; неизвестное
// значение возвращается как есть (обрезанным), пустое - пустым (источник не
// отдал назначение; в 3D такой диск рисуется как обычный).
func normalizeDiskRole(use string) string {
	u := strings.ToLower(strings.TrimSpace(use))
	switch {
	case u == "":
		return ""
	case strings.Contains(u, "spare"):
		return DiskRoleSpare
	case strings.Contains(u, "fail"):
		return DiskRoleFailed
	case strings.Contains(u, "cache"):
		return DiskRoleCache
	case strings.Contains(u, "raw"), strings.Contains(u, "unassigned"), strings.Contains(u, "unconfigured"):
		return DiskRoleRaw
	case strings.Contains(u, "data"), strings.Contains(u, "volume"), strings.Contains(u, "member"):
		return DiskRoleData
	}
	return strings.TrimSpace(use)
}

// healthStatus собирает строку Status компонента из Redfish-пары
// {Health, State}: "OK", "Warning", "Critical"; состояние добавляется, только
// если оно отличается от штатного Enabled (например "OK / StandbySpare").
func healthStatus(health, state string) string {
	h := strings.TrimSpace(health)
	s := strings.TrimSpace(state)
	if strings.EqualFold(s, "Enabled") {
		s = ""
	}
	switch {
	case h != "" && s != "":
		return h + " / " + s
	case h != "":
		return h
	}
	return s
}
