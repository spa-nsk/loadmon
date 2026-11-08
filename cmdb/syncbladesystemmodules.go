package cmdb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	models "loadmon/models"

	oneview "github.com/spa-nsk/oneview2"
)

// Разбор interconnect-модулей и модулей управления (OA/FLM) блейд-корзины из
// /rest/enclosures.
//
// Имена ключей сверены со swagger HPE OneView API 300 (oneview2/swagger 300.json,
// схема Enclosure и пример ответа):
//
//	interconnectBays[]: bayNumber, interconnectUri, logicalInterconnectUri,
//	                    interconnectModel, interconnectBayType, serialNumber
//	                    (у пустого отсека interconnectModel/serialNumber = null)
//	managerBays[]:      ipAddress, devicePresence, managerType ("OA"/"FLM")
//	                    (на новых версиях API добавляются bayNumber, role,
//	                    model, partNumber, serialNumber, fwVersion, status)
//
// Раньше код читал несуществующие ключи model/partNumber/status (interconnect) и
// model/role/mgmtIpAddress (manager), поэтому модель, статус и IP оставались
// пустыми, а пустые отсеки записывались как «модули». Теперь читаются все
// известные варианты ключей, пустые отсеки пропускаются, а модель/статус
// interconnect дополняются данными /rest/interconnects.

// collectInterconnects читает /rest/interconnects точки подключения и
// возвращает модули по URI. Ошибка не критична (nil): данные модулей тогда
// берутся только из interconnectBays корзины.
func collectInterconnects(ctx context.Context, c *oneview.Client) map[string]oneview.Interconnect {
	col, err := c.ListInterconnects(ctx, oneview.ListOptions{Count: -1})
	if err != nil || col == nil {
		fmt.Println("SyncBladeSystemsFromOneView: /rest/interconnects недоступен:", err)
		return nil
	}
	m := make(map[string]oneview.Interconnect, len(col.Members))
	for _, ic := range col.Members {
		if strings.TrimSpace(ic.URI) != "" {
			m[ic.URI] = ic
		}
	}
	return m
}

// buildInterconnectComponents строит компоненты типа 6 (interconnect) по
// занятым отсекам корзины. Slot: "Bay N, <статус>" (статус: /rest/interconnects
// → поле отсека → "Present"), его разбирает parseBladePSUBayAndStatus в
// controllers/sprbladesystems.go.
func buildInterconnectComponents(enc *oneview.Enclosure, ics map[string]oneview.Interconnect) []models.ServerHardwareComponents {
	var out []models.ServerHardwareComponents
	for idx, bay := range enc.InterconnectBays {
		bayNo := bayInt(bay, "bayNumber", idx+1)
		ic, hasIC := ics[bayString(bay, "interconnectUri")]

		model := firstNonEmpty(
			bayString(bay, "interconnectModel", "model", "productName"),
			ic.ProductName, ic.Model, ic.PartNumber,
			bayString(bay, "partNumber"),
		)
		serial := firstNonEmpty(bayString(bay, "serialNumber"), ic.SerialNumber)

		// Пустой отсек (модуль не установлен или не управляется OneView).
		if model == "" && serial == "" && !hasIC && bayString(bay, "interconnectUri") == "" {
			continue
		}
		if !bayPresent(bay) {
			continue
		}

		status := firstNonEmpty(ic.Status, bayString(bay, "status", "state"), "Present")
		if model == "" {
			fmt.Printf("syncBladeSystemComponents: корзина %s, interconnect bay %d: модель не определена, ключи отсека: %s\n",
				enc.Name, bayNo, bayKeys(bay))
		}
		out = append(out, models.ServerHardwareComponents{
			Type:      ServerHWComponentInterconnect,
			Vendor:    "HPE",
			Model:     model,
			SerialNum: firstNonEmpty(serial, "bay-ic-"+strconv.Itoa(idx)),
			// Slot оставляет bay+статус для совместимости с разбором visual;
			// Status дублируется в отдельном поле (форма и 3D читают его напрямую).
			Slot:   fmt.Sprintf("Bay %d, %s", bayNo, status),
			Status: status,
		})
	}
	return out
}

// buildManagerComponents строит компоненты типа 7 (OA для c7000, FLM для
// Synergy). Slot: "Bay N, IP x.x.x.x[, <роль>]" (разбор — parseBladePSUBayAndStatus).
func buildManagerComponents(enc *oneview.Enclosure) []models.ServerHardwareComponents {
	var out []models.ServerHardwareComponents
	for idx, bay := range enc.ManagerBays {
		if !bayPresent(bay) {
			continue
		}
		bayNo := bayInt(bay, "bayNumber", idx+1)
		ip := bayString(bay, "ipAddress", "mgmtIpAddress", "ip")
		role := bayString(bay, "role")
		model := firstNonEmpty(
			bayString(bay, "model"),
			managerTypeTitle(bayString(bay, "managerType", "deviceType", "type"), enc.EnclosureType),
		)

		status := firstNonEmpty(bayString(bay, "status", "state", "devicePresence"), "Present")
		slot := fmt.Sprintf("Bay %d", bayNo)
		switch {
		case ip != "" && role != "":
			slot += fmt.Sprintf(", IP %s, %s", ip, role)
		case ip != "":
			slot += ", IP " + ip
		case role != "":
			slot += ", " + role
		default:
			slot += ", " + status
		}
		out = append(out, models.ServerHardwareComponents{
			Type:      ServerHWComponentManagement,
			Vendor:    "HPE",
			Model:     model,
			SerialNum: firstNonEmpty(bayString(bay, "serialNumber"), "bay-mgmt-"+strconv.Itoa(idx)),
			Slot:      slot,
			Role:      role,   // Active / Standby / …
			Status:    status, // Present / OK / …
		})
	}
	return out
}

// managerTypeTitle раскрывает managerType отсека управления в название модуля.
// Пустой тип: для Synergy — FLM, иначе OA.
func managerTypeTitle(managerType, enclosureType string) string {
	switch strings.ToUpper(strings.TrimSpace(managerType)) {
	case "OA", "ONBOARDADMINISTRATOR":
		return "Onboard Administrator"
	case "FLM", "FRAMELINKMODULE":
		return "Synergy Frame Link Module"
	case "":
		if strings.EqualFold(strings.TrimSpace(enclosureType), "Synergy") ||
			strings.Contains(strings.ToLower(enclosureType), "synergy") {
			return "Synergy Frame Link Module"
		}
		return "Onboard Administrator"
	}
	return managerType
}

// bayPresent - отсек занят: devicePresence отсутствует (старые версии API) или
// равен Present/Subsumed. Absent/PresenceNoOp/PresenceUnknown — не занят.
func bayPresent(bay map[string]any) bool {
	p := strings.ToLower(strings.TrimSpace(bayString(bay, "devicePresence")))
	switch p {
	case "", "present", "subsumed":
		return true
	}
	return false
}

// bayKeys перечисляет ключи сырого отсека (для диагностики неопознанных полей).
func bayKeys(bay map[string]any) string {
	keys := make([]string, 0, len(bay))
	for k := range bay {
		keys = append(keys, k)
	}
	// стабильный порядок для читаемого лога
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return strings.Join(keys, ",")
}
