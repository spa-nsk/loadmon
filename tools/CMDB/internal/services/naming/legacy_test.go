package naming

import "testing"

// TestValidate_LegacyExamplesFromGuide — E1.7: литеральные примеры имён из самого документа
// LOI-CfM-UGUI-GI_Naming_Conventions Guide.docx (систематический разбор, не оценки по памяти),
// сгруппированные по разделу и распознаваемой схеме. Каждое имя должно определяться как legacy:
// импорт не переименовывает объект, но и не путает его с новым профилем CMDB-NC-2026 (E1.4/§5.7).
// Список включает все примеры, явно перечисленные в plan.md E1.7 (AMSDC1-S-01000 … HOUWTC-D-R001).
func TestValidate_LegacyExamplesFromGuide(t *testing.T) {
	reg := DefaultRegistry()

	cases := []struct {
		family string // для читаемости отчёта об ошибке; не проверяется буквально
		name   string
	}{
		// §2/§3.1: общее именование объектов — форма совпадает с профилем 2026, но площадка
		// встречается только в примерах Guide (legacyGuideSites).
		{"guide-object", "AMSDC1-S-01000"},
		{"guide-object", "AMSDC1-H-01000"},
		{"guide-object", "AMSDC1-V-01000"},
		{"guide-object", "AMSDC1-VD-01000"},
		{"guide-object", "BIN-S-50001"},
		{"guide-object", "AMSDC1-S-41589"}, // T-Systems Wintel DCS ESX VM, тот же шаблон

		// §3.1.3: резервные диапазоны test/temp/EDS (префикс A/R/T/G/E перед номером).
		{"guide-reserved-seq", "AMSDC1-S-A01"},
		{"guide-reserved-seq", "AMSDC1-S-A99"},
		{"guide-reserved-seq", "AMSDC1-S-R1234"},
		{"guide-reserved-seq", "CBJ-S-T1234"},
		{"guide-reserved-seq", "LONSC-L-G001"},
		{"guide-reserved-seq", "LONSC-L-T001"},
		{"guide-reserved-seq", "CBJ-D-E03001"},
		{"guide-reserved-seq", "CBJ-L-E05001"},
		{"guide-reserved-seq", "HOUWTC-D-R001"}, // §11.1 SRI: та же форма "R" перед номером

		// §3.2.2: имена кластеров и ресурсов кластера.
		{"guide-cluster-resource", "AMSDC1-S-12345C"},
		{"guide-cluster-resource", "AMSDC1-S-12345F"},
		{"guide-cluster-resource", "AMSDC1-S-12345X"},
		{"guide-cluster-resource", "AMSDC1-S-12345O"},
		{"guide-cluster-resource", "AMSDC1-S-12345Q"},
		{"guide-cluster-resource", "AMSDC1-S-1234F#"},
		{"guide-cluster-resource", "AMSDC1-S-1234Q2"},
		{"guide-cluster-resource", "AMSDC1-S-1234D1"},

		// §3.4-3.6: суффиксы поверх имени сервера (RIB/iSCSI/доп. NIC).
		{"guide-rib", "RIJPAT-S-003-R"},
		{"guide-rib", "CBJ-H-6000-R"},
		{"guide-iscsi", "AMSDC1-S-01000-iSCSI"},
		{"guide-nic", "CBJ-H-6000-a1"},
		{"guide-nic", "CBJ-H-6000-a2"},

		// §3.2.1 SAP; §3.2.2.1 DAG; §3.2.2.2 AlwaysOn AG Listener.
		{"guide-sap", "ICB0002"},
		{"guide-sap", "PAT0025"},
		{"guide-dag", "SEACMWDAG001"},
		{"guide-ag-listener", "CBJS6250CL01"},
		{"guide-ag-listener", "CBJS6250CL02"},
		{"guide-ag-listener", "MUNTSES09315CL1"},

		// §3.2.3 NLB; §3.2.6 Authentication Name.
		{"guide-nlb", "AMSDC1-S-12345C.europe.shell.com"},
		{"guide-auth", "SEACMW-ASA-001"},

		// §3.3: принтеры.
		{"guide-printer", "LONSC0A00010"},
		{"guide-printer", "LONSC0P00147"},
		{"guide-printer", "LONSC0A00010_TPA"},
		{"guide-printer", "LONSC0P00147_TPA"},
		{"guide-printer-dns", "CBJ0P005.ASIA-PAC.SHELL.COM"},

		// §3.7: сетевые устройства и их интерфейсы.
		{"guide-netdevice", "NLROTWEEMP007"},
		{"guide-netdevice", "USHOUICES002"},
		{"guide-netdevice", "USHOUICIB400"},
		{"guide-netdevice-iface", "nlpermp062.net-equip.shell.net"},
		{"guide-netdevice-iface", "nlrotrivmp004-isdn.net-equip.shell.net"},

		// §3.8: Firewall/ISA Proxy/VPN/NetCache — интерфейсы; §3.8.7 Web Filter/Content Gateway.
		{"guide-fw-proxy-vpn-iface", "cbj-f-10001-eth0"},
		{"guide-fw-proxy-vpn-iface", "cbj-fc-10001-eth3"},
		{"guide-fw-proxy-vpn-iface", "cbj-c-31001-int"},
		{"guide-fw-proxy-vpn-iface", "cbj-cc-31001-ext"},
		{"guide-fw-proxy-vpn-iface", "cbj-e-12001-eth0"},
		{"guide-fw-proxy-vpn-iface", "cbj-ec-12001-eth0"},
		{"guide-fw-proxy-vpn-iface", "cbj-c-101"},
		{"guide-fw-proxy-vpn-iface", "cbj-c-101-ext"},
		{"guide-webfilter-iface", "YUZDZE-WF-001-eth0"},
		{"guide-webfilter-iface", "YUZDZE-WCG-111-eth3"},

		// §3.10/§3.16: телефония и Special Voice.
		{"guide-telephony", "frberbx001-aau2.europe.shell.com"},
		{"guide-telephony", "MYCBJSE001.asia-pac.shell.com"},
		{"guide-telephony", "NLAMSSY002.europe.shell.com"},
		{"guide-telephony", "USHOUOTMET001.americas.shell.com"},

		// §3.11 RFID; §3.12 Legacy GI-S NetApp.
		{"guide-rfid", "TEXHLP-RFR-00001"},
		{"guide-rfid", "HOURDC-RFH-00003"},
		{"guide-netapp-legacy", "AMSDC2-NA-P0020"},
		{"guide-netapp-legacy", "AMSDC2-NA-P002M"},

		// §7.1: UNIX/Linux, включая запрещённые для новых систем mfu/taw формы.
		{"guide-unix", "rijvol-u-s011"},
		{"guide-unix", "hou-n-l123"},
		{"guide-unix", "houic-n-c01000"},
		{"guide-unix-legacy-mfu-taw", "amsdc2-u-s01mfu"},
		{"guide-unix-legacy-mfu-taw", "amsdc2-n-s02taw"},
		{"guide-unix-fs", "/nfs/home/username"},
		{"guide-unix-fs", "/nfs/projects/EP/FS_1"},

		// §8.1/§9: T-Systems Storage; HDS; Appcom; RSS; Citrix Silo.
		{"guide-tsystems-storage", "amsdc1sa001"},
		{"guide-tsystems-storage", "amsdc1is011"},
		{"guide-tsystems-storage", "amsdc1is010r"},
		{"guide-tsystems-storage", "amsdc1sw001"},
		{"guide-tsystems-storage", "amsdc1ts001"},
		{"guide-tsystems-hds", "C4NLAMSYH100"},
		{"guide-tsystems-appcom", "AA1LA169"},
		{"guide-tsystems-rss", "RSSSHLAMSZ009"},
		{"guide-tsystems-rss-unix", "AMSDC1-N-V10116"},
		{"guide-tsystems-citrix", "SHELL_CITRIXSILO-AMS_SBA_W2K8_BASE"},

		// §8.7: Exadata (rack/db/compute/storage/ILOM).
		{"guide-exadata-rack", "A2XA"},
		{"guide-exadata-rack", "M3XC"},
		{"guide-exadata-db", "XESH003T"},
		{"guide-exadata-compute", "A4XA101"},
		{"guide-exadata-compute", "M4XB204"},
		{"guide-exadata-storage", "A4XA1CEL01"},
		{"guide-exadata-ilom", "A4XA1CEL01-ilom"},
		{"guide-exadata-ilom", "M4XB2CEL04-ilom"},

		// §8.8-8.11: Oracle DB, XenDesktop, MS Cluster DR, Storage Cluster SHN.
		{"guide-tsystems-oracle", "OESH001P"},
		{"guide-tsystems-oracle", "OWSH001D"},
		{"guide-tsystems-xendesktop", "AMSDC1-XD-46500"},
		{"guide-tsystems-dr", "SHELL_DR_AMSDC1-S-1001"},
		{"guide-tsystems-shn", "C4CNBJ1SHN005"},

		// §9.1: NetApp 7-mode (физический filer, vfiler, том, qtree).
		{"guide-netapp7-filer", "c4nlamsyn16"},
		{"guide-netapp7-vfiler", "nlamvfsep809"},
		{"guide-netapp7-vfiler", "dkcovfsce001"},
		{"guide-netapp7-vfiler", "nlamvfsce012"},
		{"guide-netapp7-volume", "vt2_01125"},
		{"guide-netapp7-qtree", "qt_01126"},
		{"guide-netapp7-qtree", "qthome_01126"},
		{"guide-netapp7-qtree", "qtproj_01126"},
		{"guide-netapp7-share", `\\nlamvfsce012\qthome_01126`},

		// §10: AT&T (UCC Lync, uCPEFlexWare, Skype for Business).
		{"guide-att-lync", "M222477WLED1001"},
		{"guide-att-flexware", "USSHELLAUSTX0101UJZZ01"},
		{"guide-att-flexware", "USSHELLAUSTX0201UBRT01"},
		{"guide-att-skype", "muntse-s-sfesrv02.europe.shell.com"},
		{"guide-att-skype", "houcy1-s-sqlsrv01.americas.shell.com"},
		{"guide-att-skype", "cbjcen-swsint.asia-pac.shell.com"},

		// §11.1: NON-GI Asset Naming (Shell Retained Infrastructure, форма без дефиса).
		{"guide-sri", "HOUWTCOPR0001"},

		// Appendix C (SAN) / Appendix D (EVA).
		{"guide-san", "RIJPATsw001"},
		{"guide-san", "RIJPATRA01"},
		{"guide-san", "RIJPATTC01"},
		{"guide-san", "RIJPATtl03"},
		{"guide-san", "RIJPATWS001"},
		{"guide-san", "RIJPATDS011"},
		{"guide-san", "CBJRA01"},
		{"guide-san", "HOUICTC01"},
		{"guide-san-zone", "RIJPATTZra03za1"},
		{"guide-san-zone", "RIJPATTZra03va1"},
		{"guide-san-zone", "RIJPATTZTl03c5"},
		{"guide-san-zone", "HouicTZsma01"},
		{"guide-san-zone", "HouicTZSanBoot"},
		{"guide-san-adic", "RIJPATtl03D01"},
		{"guide-san-ds-iface", "DS011B1S2"},
		{"guide-eva-diskgroup", "DG01"},
		{"guide-eva-vdisk", "vr0001"},
		{"guide-eva-vdisk", "vm0002"},
		{"guide-eva-drgroup", "DR0001"},
		{"guide-eva-w2kvol", "vr0001-ra03"},
		{"guide-eva-bcjob", "S948_ss1"},
		{"guide-eva-cfg", "Amsdc1Amsdc2CFG"},

		// Appendix F: CommVault Galaxy / Tape Libraries / Storage Policies.
		{"guide-commvault-library", "HOUIC-CC1-L001-9"},
		{"guide-commvault-mdp", "HOUIC-CC1-L001-MDP1"},
		{"guide-commvault-dp", "HOUIC-S-100-DP1"},
		{"guide-commvault-dp", "HOUIC-S-101-DP1"},
		{"guide-commvault-tapedrive", "CBJ-TC50-D1200"},
		{"guide-commvault-tapedrive", "CBJ-TC50-D1205"},
		{"guide-commvault-policy", "HOUIC-S-A50-SP"},

		// Appendix G: EUC Devices.
		{"guide-euc-bladechassis", "BIN-BC-50001-R"},
		{"guide-euc-storage", "BIN-SDD-50001"},
		{"guide-euc-tape", "BINTL50001"},
		{"guide-euc-tape", "CBJTLV5001"},
		{"guide-euc-stms", "BIN-A-50001"},

		// Устаревшие коды классов (§5.6, registry.go) — сразу видны по коду, площадка не важна.
		{"deprecated-code", "OVBDC1-C-001"},
		{"deprecated-code", "OVBDC1-CC-001"},
		{"deprecated-code", "OVBDC1-F-001"},
		{"deprecated-code", "OVBDC1-FC-001"},
		{"deprecated-code", "OVBDC1-FM-001"},
		{"deprecated-code", "OVBDC1-E-00001"},
		{"deprecated-code", "OVBDC1-EC-00001"},
		{"deprecated-code", "OVBDC1-WF-001"},
		{"deprecated-code", "OVBDC1-N-00001"},
		{"deprecated-code", "OVBDC1-U-00001"},
		{"deprecated-code", "OVBDC1-NA-001"},
		{"deprecated-code", "OVBDC1-XD-00001"},
	}

	for _, c := range cases {
		res := Validate(reg, c.name)
		if res.Status != StatusLegacy {
			t.Errorf("[%s] Validate(%q) = %+v, want status=legacy", c.family, c.name, res)
		}
	}

	if len(cases) < 80 {
		t.Fatalf("ожидалось ≥80 примеров (plan.md E1.7), получено %d", len(cases))
	}
}

// TestValidate_LegacyServerSeqWidth — узкие места ширины номера сервера/ESX-хоста, задокументированные
// в Guide (§3.1.1: 3 цифры до 01.05.2004; §3.1.4/§3.1.1: 4 цифры для GI-H 6000-9999), проверенные на
// площадке профиля 2026 (не из legacyGuideSites), чтобы отличить эту причину от "площадка — Guide".
func TestValidate_LegacyServerSeqWidth(t *testing.T) {
	reg := DefaultRegistry()
	cases := []string{"OVBDC1-S-146", "OVBDC1-S-6001", "OVBDC1-H-6001"}
	for _, name := range cases {
		res := Validate(reg, name)
		if res.Status != StatusLegacy {
			t.Errorf("Validate(%q) = %+v, want status=legacy", name, res)
		}
		if res.Suggestion == "" {
			t.Errorf("Validate(%q): ожидалось предложение имени по профилю 2026", name)
		}
	}
}

// TestValidate_ArbitraryNameStillNoncompliant — legacy-правила не должны поглощать произвольные
// несоответствующие имена (AC-5: "webserver01" остаётся noncompliant, а не превращается в legacy).
func TestValidate_ArbitraryNameStillNoncompliant(t *testing.T) {
	reg := DefaultRegistry()
	cases := []string{"webserver01", "OVBDC1-CL-04", "totally-not-a-hostname-shape"}
	for _, name := range cases {
		res := Validate(reg, name)
		if res.Status != StatusNoncompliant {
			t.Errorf("Validate(%q) = %+v, want status=noncompliant", name, res)
		}
	}
}
