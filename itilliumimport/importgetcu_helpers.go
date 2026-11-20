package itilliumimport

import (
	"strconv"
	"strings"

	"loadmon/itilium"
)

func cuTechFromRow(row itilium.CURow) string {
	if s := cuTrim(row.Code); s != "" {
		return s
	}
	return strings.TrimSpace(row.UID)
}

func aggregateCUImportOne(st *ImportCUStats, one cuImportOne) {
	switch {
	case one.err:
		st.Errors++
	case one.skipped:
		st.Skipped++
	case one.inserted:
		st.Inserted++
	default:
		st.Updated++
	}
	if one.inactive {
		st.Inactive++
	}
	if one.noLoc {
		st.NoLoc++
	}
	if one.unknownSt {
		st.UnknownSt++
	}
	recordImportOne(one)
}

// cuWhereScrap — «списан»/«продаж»/«утер» в тексте местоположения.
func cuWhereScrap(where string) bool {
	w := strings.ToLower(where)
	return strings.Contains(w, "списан") || strings.Contains(w, "продаж") || strings.Contains(w, "утер")
}

// cuSkipPlacementLikeNetwork — не резолвим площадку (как importnetworkequipment / tape getCU).
func cuSkipPlacementLikeNetwork(where string, itiliumStatus int) bool {
	if locationUndefinedForCU(where, itiliumStatus) {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(where), "!")
}

var cuBladeModelDescMarkers = []string{"HP", "Chieftec", "DELL", "Inflex", "DEPO", "Cisco"}

func cuBladeModel(row itilium.CURow) string {
	if m := cuTrim(row.ModelBlade); m != "" {
		return m
	}
	desc := cuTrim(row.Description)
	if desc == "" {
		return ""
	}
	upper := strings.ToUpper(desc)
	for _, marker := range cuBladeModelDescMarkers {
		if strings.Contains(upper, strings.ToUpper(marker)) {
			return desc
		}
	}
	return ""
}

// parseBladeBucketName — имя BladeSystems из поля bucket getCU (см. bladeRefRe).
func parseBladeBucketName(bucket string) string {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ""
	}
	if m := bladeRefRe.FindStringSubmatch(bucket); m != nil {
		if name := strings.TrimSpace(m[2]); name != "" {
			return name
		}
		if tech := strings.TrimSpace(m[1]); tech != "" {
			return tech
		}
		return ""
	}
	return bucket
}

func cuLinksUnit(row itilium.CURow) string {
	return strings.TrimSpace(row.LinksUnit.String())
}

func cuIntField(lv itilium.LabelValue) int {
	s := strings.TrimSpace(lv.ValueString())
	if s == "" {
		s = cuTrim(lv)
	}
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}