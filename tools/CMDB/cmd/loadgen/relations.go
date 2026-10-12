package main

import (
	"context"
	"math/rand"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
)

type relPair struct {
	relType  string
	src, dst int64
}

// generateRelations — vm→host (runs_on), vm→cluster (member_of, если хост в кластере),
// vm→datastore (uses_storage), host→cluster (member_of), array→volume (contains),
// datastore→volume (uses_storage, привязка датастора к тому под ним) — тот же набор типов связей,
// что реально используют существующие коннекторы (vcenter.go, storage.Build), не выдуманный
// отдельно для нагрузочного теста.
func generateRelations(ctx context.Context, ids map[string][]int64) int {
	hosts := ids["hypervisor_host"]
	clusters := ids["cluster"]
	vms := ids["vm"]
	datastores := ids["datastore"]
	arrays := ids["storage_array"]
	volumes := ids["volume"]

	var pairs []relPair
	hostCluster := map[int64]int64{}
	for i, h := range hosts {
		if len(clusters) == 0 {
			break
		}
		c := clusters[i%len(clusters)]
		hostCluster[h] = c
		pairs = append(pairs, relPair{"member_of", h, c})
	}
	for _, vm := range vms {
		if len(hosts) == 0 {
			break
		}
		h := hosts[rand.Intn(len(hosts))]
		pairs = append(pairs, relPair{"runs_on", vm, h})
		if c, ok := hostCluster[h]; ok {
			pairs = append(pairs, relPair{"member_of", vm, c})
		}
		if len(datastores) > 0 {
			pairs = append(pairs, relPair{"uses_storage", vm, datastores[rand.Intn(len(datastores))]})
		}
	}
	for i, v := range volumes {
		if len(arrays) == 0 {
			break
		}
		pairs = append(pairs, relPair{"contains", arrays[i%len(arrays)], v})
	}
	for i, d := range datastores {
		if len(volumes) == 0 {
			break
		}
		pairs = append(pairs, relPair{"uses_storage", d, volumes[i%len(volumes)]})
	}

	total := 0
	for start := 0; start < len(pairs); start += batchSize {
		end := min(start+batchSize, len(pairs))
		batch := pairs[start:end]
		_, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceSystem, UserID: 0}, func(ctx context.Context, tx orm.TxOrmer) error {
			var sb strings.Builder
			sb.WriteString(`INSERT INTO cmdb.ci_rel (rel_type, src_ci_id, dst_ci_id) VALUES `)
			args := make([]any, 0, len(batch)*3)
			for i, p := range batch {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString("(?,?,?)")
				args = append(args, p.relType, p.src, p.dst)
			}
			if _, err := tx.RawWithCtx(ctx, sb.String(), args...).Exec(); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			fatalf("generateRelations: %v", err)
		}
		total += len(batch)
	}
	return total
}
