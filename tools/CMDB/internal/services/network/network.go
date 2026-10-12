// Package network — сетевая часть модели: VLAN, подсети, интерфейсы КЕ и IP-адреса
// (cmdb.vlan, cmdb.subnet, cmdb.interface, cmdb.ip_address). Все изменения идут через changectx.
package network

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// Meta — общие для операций сведения о том, кто и зачем меняет данные.
type Meta struct {
	Reason string
	UserID int64
}

func (m Meta) ctxMeta() changectx.Meta {
	return changectx.Meta{Source: changectx.SourceManual, UserID: m.UserID, Reason: m.Reason}
}

// NormalizeWWN приводит WWPN/WWNN к виду 50:01:43:80:12:34:56:78 (нижний регистр, двоеточия).
// Принимает разделители ':', '-', '.', пробел и префикс 0x.
func NormalizeWWN(s string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(s))
	raw = strings.TrimPrefix(raw, "0x")
	raw = strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(raw)
	if len(raw) != 16 {
		return "", fmt.Errorf("%w: WWN %q должен содержать 16 шестнадцатеричных цифр", svcerr.ErrInvalidInput, s)
	}
	for _, c := range raw {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return "", fmt.Errorf("%w: WWN %q содержит недопустимый символ %q", svcerr.ErrInvalidInput, s, c)
		}
	}
	var b strings.Builder
	for i := 0; i < 16; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(raw[i : i+2])
	}
	return b.String(), nil
}

// NormalizeAddress проверяет IPv4/IPv6-адрес ("10.0.0.5" или "10.0.0.5/24") и возвращает его
// каноническую запись; адрес без маски получает полную (/32 или /128).
func NormalizeAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return "", fmt.Errorf("%w: адрес %q: %v", svcerr.ErrInvalidInput, s, err)
		}
		return p.String(), nil
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return "", fmt.Errorf("%w: адрес %q: %v", svcerr.ErrInvalidInput, s, err)
	}
	return netip.PrefixFrom(a, a.BitLen()).String(), nil
}

// VLANInput — создание VLAN.
type VLANInput struct {
	Number       int
	Name         string
	LocationCode string // "" — сквозной VLAN
	Description  string
	Meta
}

// CreateVLAN создаёт VLAN; номер уникален в пределах площадки (ErrDuplicate).
func CreateVLAN(ctx context.Context, in VLANInput) (id, changeSetID int64, err error) {
	changeSetID, err = changectx.Run(ctx, in.Meta.ctxMeta(), func(ctx context.Context, tx orm.TxOrmer) error {
		e := tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.vlan (vlan_number, name, location_code, description, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id
		`, in.Number, in.Name, nullableString(in.LocationCode), nullableString(in.Description), in.UserID, in.UserID).QueryRow(&id)
		if svcerr.ViolatesConstraint(e, "vlan_location_number_uniq") {
			return fmt.Errorf("%w: VLAN %d уже есть на этой площадке", svcerr.ErrDuplicate, in.Number)
		}
		if e != nil {
			return fmt.Errorf("network: вставка vlan: %w", e)
		}
		return nil
	})
	return id, changeSetID, err
}

// SubnetInput — создание подсети.
type SubnetInput struct {
	VRF          string // "" = default
	CIDR         string // сетевой адрес с маской, например 10.20.0.0/24
	Name         string
	VLANID       *int64
	LocationCode string
	Gateway      string
	Purpose      string
	Description  string
	Meta
}

// CreateSubnet создаёт подсеть; (vrf, cidr) уникальны (ErrDuplicate).
func CreateSubnet(ctx context.Context, in SubnetInput) (id, changeSetID int64, err error) {
	changeSetID, err = changectx.Run(ctx, in.Meta.ctxMeta(), func(ctx context.Context, tx orm.TxOrmer) error {
		e := tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.subnet (vrf, cidr, name, vlan_id, location_code, gateway, purpose, description, created_by, updated_by)
			VALUES (COALESCE(CAST(? AS text), 'default'), CAST(? AS cidr), ?, ?, ?, CAST(? AS inet), ?, ?, ?, ?)
			RETURNING id
		`, nullableString(in.VRF), in.CIDR, nullableString(in.Name), repo.NullPtr(in.VLANID), nullableString(in.LocationCode),
			nullableString(in.Gateway), nullableString(in.Purpose), nullableString(in.Description), in.UserID, in.UserID).QueryRow(&id)
		if svcerr.ViolatesConstraint(e, "subnet_vrf_cidr_uniq") {
			return fmt.Errorf("%w: подсеть %s уже есть", svcerr.ErrDuplicate, in.CIDR)
		}
		if e != nil {
			return fmt.Errorf("network: вставка subnet: %w", e)
		}
		return nil
	})
	return id, changeSetID, err
}

// InterfaceInput — добавление интерфейса КЕ.
type InterfaceInput struct {
	CIID              int64
	Name              string
	IfType            string // ethernet (по умолчанию), fc, ib, bmc, virtual, lag, loopback
	MAC               string
	SpeedMbps         *int64
	MTU               *int
	WWPN              string
	WWNN              string
	ComponentID       *int64
	ParentInterfaceID *int64
	VLANID            *int64
	Description       string
	Meta
}

// AddInterface создаёт интерфейс; имя уникально в пределах КЕ, WWPN — глобально (ErrDuplicate).
func AddInterface(ctx context.Context, in InterfaceInput) (id, changeSetID int64, err error) {
	var wwpn, wwnn string
	if in.WWPN != "" {
		if wwpn, err = NormalizeWWN(in.WWPN); err != nil {
			return 0, 0, err
		}
	}
	if in.WWNN != "" {
		if wwnn, err = NormalizeWWN(in.WWNN); err != nil {
			return 0, 0, err
		}
	}

	changeSetID, err = changectx.Run(ctx, in.Meta.ctxMeta(), func(ctx context.Context, tx orm.TxOrmer) error {
		e := tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.interface
				(ci_id, component_id, parent_interface_id, name, if_type, mac, speed_mbps, mtu, wwpn, wwnn, vlan_id,
				 description, created_by, updated_by)
			VALUES (?, ?, ?, ?, COALESCE(CAST(? AS text), 'ethernet'), CAST(? AS macaddr), ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`, in.CIID, repo.NullPtr(in.ComponentID), repo.NullPtr(in.ParentInterfaceID), in.Name, nullableString(in.IfType),
			nullableString(in.MAC), repo.NullPtr(in.SpeedMbps), repo.NullPtr(in.MTU), nullableString(wwpn), nullableString(wwnn),
			repo.NullPtr(in.VLANID), nullableString(in.Description), in.UserID, in.UserID).QueryRow(&id)
		switch {
		case svcerr.ViolatesConstraint(e, "interface_ci_name_uniq"):
			return fmt.Errorf("%w: интерфейс %q уже есть у КЕ %d", svcerr.ErrDuplicate, in.Name, in.CIID)
		case svcerr.ViolatesConstraint(e, "interface_wwpn_uniq"):
			return fmt.Errorf("%w: WWPN %s уже назначен другому интерфейсу", svcerr.ErrDuplicate, wwpn)
		case e != nil:
			return fmt.Errorf("network: вставка interface: %w", e)
		}
		return nil
	})
	return id, changeSetID, err
}

// AssignIPInput — назначение IP-адреса КЕ (и, при необходимости, её интерфейсу).
type AssignIPInput struct {
	CIID        int64
	InterfaceID *int64 // интерфейс должен принадлежать той же КЕ
	Address     string // "10.0.0.5" или "10.0.0.5/24"
	VRF         string // "" = default
	Kind        string // primary (по умолчанию), secondary, vip, mgmt, bmc, service
	DNSName     string
	Status      string // active (по умолчанию), reserved, deprecated
	Description string
	Meta
}

// AssignIPResult — id адреса и подсеть, в которую он автоматически попал (0 — подсеть не найдена).
type AssignIPResult struct {
	ID          int64
	SubnetID    int64
	ChangeSetID int64
}

// AssignIP назначает адрес и определяет подсеть: самую узкую из cmdb.subnet, содержащую адрес
// в том же VRF. Адрес уникален в пределах VRF независимо от маски (ErrDuplicate).
func AssignIP(ctx context.Context, in AssignIPInput) (AssignIPResult, error) {
	var res AssignIPResult
	addr, err := NormalizeAddress(in.Address)
	if err != nil {
		return res, err
	}
	vrf := in.VRF
	if vrf == "" {
		vrf = "default"
	}

	csID, err := changectx.Run(ctx, in.Meta.ctxMeta(), func(ctx context.Context, tx orm.TxOrmer) error {
		var subnetID int64
		e := tx.RawWithCtx(ctx, `
			SELECT id FROM cmdb.subnet
			WHERE vrf = ? AND cidr >>= CAST(host(CAST(? AS inet)) AS inet)
			ORDER BY masklen(cidr) DESC LIMIT 1
		`, vrf, addr).QueryRow(&subnetID)
		if e != nil && e != orm.ErrNoRows {
			return fmt.Errorf("network: поиск подсети: %w", e)
		}

		e = tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.ip_address
				(vrf, address, subnet_id, ci_id, interface_id, kind, dns_name, status, description, created_by, updated_by)
			VALUES (?, CAST(? AS inet), ?, ?, ?, COALESCE(CAST(? AS text), 'primary'), ?, COALESCE(CAST(? AS text), 'active'), ?, ?, ?)
			RETURNING id
		`, vrf, addr, nullableInt(subnetID), in.CIID, repo.NullPtr(in.InterfaceID), nullableString(in.Kind),
			nullableString(in.DNSName), nullableString(in.Status), nullableString(in.Description), in.UserID, in.UserID).QueryRow(&res.ID)
		switch {
		case svcerr.ViolatesConstraint(e, "ip_address_vrf_host_uniq"):
			return fmt.Errorf("%w: адрес %s уже назначен в VRF %s", svcerr.ErrDuplicate, addr, vrf)
		case svcerr.ViolatesConstraint(e, "ip_address_interface_fkey"):
			return fmt.Errorf("%w: интерфейс не принадлежит КЕ %d", svcerr.ErrInvalidState, in.CIID)
		case e != nil:
			return fmt.Errorf("network: вставка ip_address: %w", e)
		}
		res.SubnetID = subnetID
		return nil
	})
	res.ChangeSetID = csID
	return res, err
}

// ReleaseIP освобождает адрес: строка удаляется из cmdb.ip_address (адрес снова доступен для
// назначения), а предыдущий владелец сохраняется в hist.ip_address и audit.change_log (op='D').
func ReleaseIP(ctx context.Context, id int64, expectedVersion int, m Meta) (int64, error) {
	return changectx.Run(ctx, m.ctxMeta(), func(ctx context.Context, tx orm.TxOrmer) error {
		var version int
		err := tx.RawWithCtx(ctx, `SELECT row_version FROM cmdb.ip_address WHERE id = ? FOR UPDATE`, id).QueryRow(&version)
		if err == orm.ErrNoRows {
			return svcerr.NotFound("ip_address", id)
		}
		if err != nil {
			return fmt.Errorf("network: чтение адреса: %w", err)
		}
		if version != expectedVersion {
			return svcerr.Conflict("ip_address", id, expectedVersion, version)
		}
		if _, err := tx.RawWithCtx(ctx, `DELETE FROM cmdb.ip_address WHERE id = ?`, id).Exec(); err != nil {
			return fmt.Errorf("network: удаление адреса: %w", err)
		}
		return nil
	})
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

// ListInterfaces возвращает интерфейсы КЕ.
func ListInterfaces(ctx context.Context, ciID int64) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT i.id::text AS id, i.row_version::text AS row_version, i.name, i.if_type,
		       COALESCE(i.mac::text, '') AS mac, COALESCE(i.wwpn, '') AS wwpn,
		       COALESCE(i.speed_mbps::text, '') AS speed_mbps, COALESCE(i.mtu::text, '') AS mtu,
		       COALESCE(v.vlan_number::text, '') AS vlan, i.enabled::text AS enabled,
		       COALESCE(i.description, '') AS description
		FROM cmdb.interface i LEFT JOIN cmdb.vlan v ON v.id = i.vlan_id
		WHERE i.ci_id = ? ORDER BY i.name`, ciID)
}

// ListIPs возвращает IP-адреса КЕ с именем интерфейса и подсетью.
func ListIPs(ctx context.Context, ciID int64) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT ip.id::text AS id, ip.row_version::text AS row_version, host(ip.address) AS address,
		       masklen(ip.address)::text AS masklen, ip.vrf, ip.kind, ip.status,
		       COALESCE(ip.dns_name, '') AS dns_name, COALESCE(i.name, '') AS iface, COALESCE(s.cidr::text, '') AS subnet
		FROM cmdb.ip_address ip
		LEFT JOIN cmdb.interface i ON i.id = ip.interface_id
		LEFT JOIN cmdb.subnet s ON s.id = ip.subnet_id
		WHERE ip.ci_id = ? ORDER BY ip.address`, ciID)
}
