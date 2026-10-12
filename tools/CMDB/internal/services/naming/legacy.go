// Package naming: распознавание имён по устаревшим правилам LOI-CfM-UGUI-GI Naming Conventions
// Guide (2019, далее Guide) — профиль GUIDE-2019 (intent §5.6, план E1.4/E1.7). В отличие от
// профиля CMDB-NC-2026 эти правила больше не выдают новых имён (см. BuildHostName), но обязаны
// распознавать уже существующие: импорт не переименовывает объект, только помечает
// naming_status = legacy и, где это однозначно возможно, предлагает имя по профилю 2026.
//
// Формы имён здесь взяты из систематического разбора самого документа (не по памяти) — раздел
// "Equipment Naming Conventions" (§3), UNIX/Linux (§7), T-Systems (§8-9), AT&T (§10), Non-GI
// Asset Naming (§11) и приложения C/D/F/G (SAN/EVA/CommVault/EUC).
package naming

import (
	"regexp"
	"strings"
)

// legacyGuideSites — компрессированные коды площадок, встречающиеся в иллюстративных примерах
// самого Guide (вымышленные площадки старого Shell-профиля, не входящие в профиль CMDB-NC-2026).
// Нужны только для разбора "Общего именования объектов" (§2 ниже): его шаблон
// <Loc>-<Designation>-<Seq> для активных кодов (S, H, V, VD, D, L, T) буквально совпадает по
// форме с profильом CMDB-NC-2026 — единственный различающий признак это принадлежность площадки
// профилю GUIDE-2019, а не CMDB-NC-2026 (ср. E1.4: диапазоны Guide заносятся как занятые резервы
// под guide2019-площадками, а не превращаются в новые площадки профиля 2026).
var legacyGuideSites = map[string]bool{
	"AMSDC1": true, "AMSDC2": true, "BIN": true, "CBJ": true, "LONSC": true,
	"RIJPAT": true, "RIJVOL": true, "HOUIC": true, "DOHCOR": true, "LUT2": true,
	"TEXHLP": true, "HOURDC": true, "MUNTSE": true, "MUNTSD": true, "SEACMW": true,
	"HOUWTC": true, "HOUCY1": true, "CBJCEN": true, "PEJJBT": true, "LDM": true,
	"BAN": true, "YUZDZE": true, "HOUTSW": true, "HOU": true, "ASS": true,
}

// isLegacyGuideSite сообщает, встречается ли код площадки в примерах Guide.
func isLegacyGuideSite(site string) bool {
	return legacyGuideSites[strings.ToUpper(site)]
}

// LegacyMatch — распознанная схема Guide 2019: к какому семейству относится имя и почему оно
// больше не выдаётся новым КЕ.
type LegacyMatch struct {
	Family string // машиночитаемый идентификатор схемы, например "guide-sap", "guide-netapp7-volume"
	Note   string // человекочитаемое пояснение (RU)
}

type legacyRule struct {
	family string
	note   string
	re     *regexp.Regexp
}

// re — сокращение для regexp.MustCompile с флагом регистронезависимости (документ Guide по
// умолчанию регистронезависим, см. §1.4 "Typographical Conventions").
func lre(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)^` + pattern + `$`)
}

var legacyRules = []legacyRule{
	// --- §3.1.3 Резервные диапазоны (test/temp/EDS): префикс A/R/T/G/E перед номером ---
	{"guide-reserved-seq", "резервный диапазон Guide (test/temp/EDS, §3.1.3): префикс A/R/T/G/E перед номером",
		lre(`[A-Z0-9]{2,10}-[A-Z]{1,4}-[ARTGE]\d{1,5}`)},

	// --- §3.2.2 Имена кластеров и ресурсов: буква/буква+цифра/буква+# приклеены к номеру узла ---
	{"guide-cluster-resource", "имя кластера или ресурса кластера Guide (§3.2.2): суффикс приклеен к номеру узла без разделителя",
		lre(`[A-Z0-9]{2,10}-[A-Z]{1,4}-\d{3,6}[A-Z](\d{1,2}|#)?`)},

	// --- §3.4 RIB, §3.5 iSCSI, §3.6 доп. NIC — суффикс через дефис поверх основного имени ---
	{"guide-rib", "Remote Insight Board (§3.4): суффикс -R поверх имени сервера",
		lre(`[A-Z0-9-]+-\d{2,6}-R`)},
	{"guide-iscsi", "интерфейс iSCSI сервера (§3.5): суффикс -iSCSI поверх имени сервера",
		lre(`[A-Z0-9-]+-\d{2,6}-ISCSI`)},
	{"guide-nic", "дополнительный NIC/IP сервера (§3.6): суффикс <порт><индекс> поверх имени",
		lre(`[A-Z0-9-]+-\d{2,6}-[A-Z]\d`)},

	// --- §3.2.1 SAP-сервер: <SiteCode 3><Seq 4> ---
	{"guide-sap", "имя SAP-сервера Guide (§3.2.1): <SiteCode><4 цифры>, ограничение SAP в 8 символов",
		lre(`[A-Z]{3}\d{4}`)},

	// --- §3.2.2.1 Exchange 2010 DAG ---
	{"guide-dag", "Exchange 2010 Database Availability Group (§3.2.2.1)",
		lre(`[A-Z0-9]{2,10}DAG\d{3}`)},
	// --- §3.2.2.2 SQL AlwaysOn AG Listener ---
	{"guide-ag-listener", "SQL Server AlwaysOn AG Listener (§3.2.2.2): <сжатое имя WSFC>L<seq>",
		lre(`[A-Z0-9]{5,15}L\d{1,2}`)},
	// --- §3.2.3 NLB ---
	{"guide-nlb", "Network Load Balancing (§3.2.3): <первый узел>C.<domain>.com",
		lre(`[A-Z0-9-]+C\.[A-Z-]+\.SHELL\.COM`)},
	// --- §3.2.6 Authentication Name ---
	{"guide-auth", "объект аутентификации Kerberos для Exchange 2010 (§3.2.6): <Loc>-ASA-<seq>",
		lre(`[A-Z0-9]{2,10}-ASA-\d{3}`)},

	// --- §3.3 Принтеры ---
	{"guide-printer", "имя принтера Guide (§3.3): <Loc>0[PA]<5 цифр>[_суффикс]",
		lre(`[A-Z0-9]{2,10}0[PA]\d{5}(_[A-Z]+)?`)},
	{"guide-printer-dns", "DNS-имя очереди печати Guide (§3.3)",
		lre(`[A-Z0-9]{2,10}0P\d{3,5}\.[A-Z-]+\.SHELL\.COM`)},

	// --- §3.7 Сетевые устройства и их интерфейсы ---
	{"guide-netdevice", "сетевое устройство Guide (§3.7): <CountryCode><Loc><Designation><seq>",
		lre(`[A-Z]{2}[A-Z0-9]{2,9}[A-Z]{2}\d{3,4}`)},
	{"guide-netdevice-iface", "интерфейс сетевого устройства Guide (§3.7.1): *.net-equip.shell.net",
		lre(`[A-Z0-9]+(-[A-Z]+)?\.NET-EQUIP\.SHELL\.NET`)},

	// --- §3.8 Firewall/ISA Proxy/VPN/NetCache и их интерфейсы (нижний регистр в примерах) ---
	{"guide-fw-proxy-vpn-iface", "интерфейс firewall/ISA proxy/VPN endpoint Guide (§3.8.1-3.8.6)",
		lre(`[A-Z0-9]{2,10}-(F|FC|C|CC|E|EC)-\d{3,5}(-[A-Z0-9]+)?`)},
	{"guide-webfilter-iface", "интерфейс Web Filter / Content Gateway Guide (§3.8.7)",
		lre(`[A-Z0-9]{2,10}-(WF|WCG)-\d{3}-[A-Z0-9]+`)},

	// --- §3.10 Телефония, §3.16 Special Voice ---
	{"guide-telephony", "оборудование телефонии Guide (§3.10/§3.16): <CC><Loc><Designation><seq>[-интерфейс].<region>.shell.com",
		lre(`[A-Z]{2}[A-Z0-9]{2,8}(BX|CM|VM|CC|SE|SY|UM|CX|CR|ET|TU|VL)\d{3}(-(AAU|NIU)\d)?\.[A-Z-]+\.SHELL\.COM`)},

	// --- §3.11 RFID ---
	{"guide-rfid", "устройство RFID Guide (§3.11): <Loc>-RF[ACHR]-<5 цифр>",
		lre(`[A-Z0-9]{2,10}-RF[ACHR]-\d{5}`)},

	// --- §3.12 Legacy GI-S NetApp ---
	{"guide-netapp-legacy", `явно помеченная "Legacy" схема GI-S NetApp (§3.12): <Loc>-NA-<Designation><seq><iface>`,
		lre(`[A-Z0-9]{2,10}-NA-[A-Z]?\d{3,4}[A-Z0-9]?`)},

	// --- §7.1 UNIX/Linux (включая запрещённые для новых систем mfu/taw формы) ---
	{"guide-unix-legacy-mfu-taw", "запрещённая для новых систем форма UNIX/Linux Guide (§7.1): mfu/taw",
		lre(`[A-Z0-9]{2,10}-[UN]-[A-Z]\d{2}(MFU|TAW)`)},
	{"guide-unix", "имя UNIX/Linux Guide (§7.1): <Loc>-<U|N>-<роль><seq>, старая схема не мигрировалась",
		lre(`[A-Z0-9]{2,10}-[UN]-[A-Z]\d{3,5}`)},
	{"guide-unix-fs", "путь файловой системы UNIX/Linux Guide (§7.3)",
		regexp.MustCompile(`^/nfs/[a-zA-Z0-9/_.-]+$`)},

	// --- §8.1/§9 T-Systems Storage (нижний регистр, без разделителей) ---
	{"guide-tsystems-storage", "устройство хранения T-Systems Guide (§8.1/§9): <loc><sa|is|sw|ts><3 цифры>[interface]",
		lre(`[A-Z0-9]{2,10}(SA|IS|SW|TS)\d{3}[A-Z]{0,2}`)},
	{"guide-tsystems-hds", "дисковый массив HDS/Hitachi T-Systems Guide (§9): C4<CC><Loc>SYH<seq>",
		lre(`C4[A-Z]{2}[A-Z]{2,4}SYH\d{3}`)},

	// --- §8.2 T-Systems Appcom ---
	{"guide-tsystems-appcom", "имя AppCom T-Systems Guide (§8.2)",
		lre(`(A|BC|F|S|U|N|P|V)[A-Z][1-4](L|H)[A-Z]\d{3}`)},

	// --- §8.4/§8.5 T-Systems RSS ---
	{"guide-tsystems-rss", "Remote Site Services VM T-Systems Guide (§8.4): RSSSHL<Loc>Z<seq>",
		lre(`RSSSHL[A-Z]{2,4}Z\d{3}`)},
	{"guide-tsystems-rss-unix", "Linux/Unix RSS VM T-Systems Guide (§8.5)",
		lre(`[A-Z0-9]{2,10}-[NU]-V\d{5}`)},

	// --- §8.6 T-Systems Citrix Silo ---
	{"guide-tsystems-citrix", "Citrix Silo T-Systems Guide (§8.6)",
		lre(`SHELL_CITRIXSILO-[A-Z]+_[A-Z]+_[A-Z0-9]+_[A-Z]+`)},

	// --- §8.7 T-Systems Exadata ---
	{"guide-exadata-ilom", "плата ILOM Exadata T-Systems Guide (§8.7.5)",
		lre(`[A-Z0-9]{2,4}X[A-C]\dCEL\d{2}-ILOM`)},
	{"guide-exadata-storage", "storage-сервер Exadata T-Systems Guide (§8.7.4)",
		lre(`[A-Z0-9]{2,4}X[A-C]\dCEL\d{2}`)},
	{"guide-exadata-compute", "compute-сервер Exadata T-Systems Guide (§8.7.3)",
		lre(`[A-Z0-9]{2,4}X[A-C]\d{3}`)},
	{"guide-exadata-db", "база данных Exadata T-Systems Guide (§8.7.2): X<region><customer><seq><purpose>",
		lre(`X[A-Z]{3}\d{3}[A-Z]`)},
	{"guide-exadata-rack", "стойка Exadata T-Systems Guide (§8.7.1)",
		lre(`[A-Z0-9]{2,4}X[A-C]`)},

	// --- §8.8 T-Systems Oracle DB (вне Exadata) ---
	{"guide-tsystems-oracle", "база данных Oracle T-Systems Guide (§8.8): O<region><customer><seq><purpose>",
		lre(`O[A-Z]{3}\d{3}[A-Z]`)},

	// --- §8.9 T-Systems XenDesktop, §8.10 MS Cluster DR, §8.11 Storage Cluster ---
	{"guide-tsystems-xendesktop", "XenDesktop T-Systems Guide (§8.9): <Loc>-XD-<seq>",
		lre(`[A-Z0-9]{2,10}-XD-\d{5}`)},
	{"guide-tsystems-dr", "MS Cluster DR T-Systems Guide (§8.10): SHELL_DR_<основной сервер>",
		lre(`SHELL_DR_[A-Z0-9-]+`)},
	{"guide-tsystems-shn", `Storage Cluster "SHN" T-Systems Guide (§8.11)`,
		lre(`C4CN[A-Z]{2}\dSHN\d{3}`)},

	// --- §9.1 NetApp 7-mode (физические appliance, vfiler, том, qtree, share) ---
	{"guide-netapp7-filer", "физический NetApp filer T-Systems Guide (§9.1.1): c4<cc><loc>sy<n><seq>",
		lre(`C4[A-Z]{2}[A-Z]{2}SY[A-Z]\d{2}`)},
	{"guide-netapp7-vfiler", "virtual filer (vfiler) NetApp 7-mode Guide (§9.1.2)",
		lre(`[A-Z]{2}[A-Z]{2}VF[A-Z]{2}\d{3}`)},
	{"guide-netapp7-volume", "имя тома NetApp 7-mode Guide (§9.1.3): vt<tier>_<seq>",
		lre(`VT\d_\d{5}`)},
	{"guide-netapp7-qtree", "имя qtree NetApp 7-mode Guide (§9.1.4)",
		lre(`QT[A-Z]*_\d{5}`)},
	{"guide-netapp7-share", `CIFS share NetApp 7-mode Guide (§9.1.5): \\vfiler\qtree`,
		regexp.MustCompile(`(?i)^\\\\[a-z0-9]+\\[a-z0-9_]+$`)},

	// --- §10.1-10.3 AT&T ---
	{"guide-att-lync", "AT&T UCC Lync Service (§10.1)",
		lre(`[A-Z]\d{6}[A-Z]{2,6}\d{3,4}`)},
	{"guide-att-flexware", "AT&T uCPEFlexWare Service (§10.2)",
		lre(`[A-Z]{2}SHELL[A-Z0-9]{6,14}U[A-Z]{3}\d{2}`)},
	{"guide-att-skype", "AT&T UCC Skype for Business Service (§10.3)",
		lre(`[A-Z0-9]{5,7}-((S|V)-)?[A-Z]{2,6}\d{0,2}\.(EUROPE|AMERICAS|ASIA-PAC)\.SHELL\.COM`)},

	// --- §11.1 NON-GI Asset Naming (Shell Retained Infrastructure) ---
	{"guide-sri", `актив Shell Retained Infrastructure (§11.1): "R" перед номером без разделителя`,
		lre(`[A-Z0-9]{4,10}OPR\d{4}`)},

	// --- Appendix C: SAN, Appendix D: EVA ---
	{"guide-san", "устройство SAN (Appendix C) или EVA (Appendix D): FC switch/RAID array/tape/WS/DS",
		lre(`[A-Z0-9]{3,10}(SW|RA|TC|TL|WS|DS)\d{2,3}`)},
	{"guide-san-zone", "зона SAN/EVA (Appendix C/D): TZ<...>",
		lre(`[A-Z0-9]{3,10}TZ[A-Z0-9]+`)},
	{"guide-san-adic", "привод ADIC Scalar i500 (Appendix C)",
		lre(`[A-Z0-9]{3,10}TL\d{2}D\d{2}`)},
	{"guide-san-ds-iface", "интерфейс Disk Storage (Appendix C): <DSalias>B<bay>S<port>",
		lre(`[A-Z0-9]{0,10}DS\d{3}B\dS\d`)},
	{"guide-eva-diskgroup", "Disk Group EVA (Appendix D)",
		lre(`DG\d{2}`)},
	{"guide-eva-vdisk", "виртуальный/снимочный диск EVA (Appendix D): vr/vm<seq>",
		lre(`V[MR]\d{4}`)},
	{"guide-eva-drgroup", "Data Replication Group EVA (Appendix D)",
		lre(`DR\d{4}`)},
	{"guide-eva-w2kvol", "W2K volume на EVA (Appendix D): <VirtualDisk>-<EVAAlias>",
		lre(`V[MR]\d{4}-RA\d{2}`)},
	{"guide-eva-bcjob", "Business Copy Job/Schedule EVA (Appendix D)",
		lre(`[A-Z0-9]+_(SS|SC)\d`)},
	{"guide-eva-cfg", "файл конфигурации зоны Continuous Access EVA (Appendix D)",
		lre(`[A-Z0-9]{4,20}CFG`)},

	// --- Appendix F: CommVault Galaxy / Tape Libraries / Storage Policies ---
	{"guide-commvault-mdp", "Master Drive Pool CommVault Galaxy (Appendix F)",
		lre(`[A-Z0-9]{3,10}-CC\d-L\d{3}-MDP\d`)},
	{"guide-commvault-library", "библиотека CommVault Galaxy (Appendix F)",
		lre(`[A-Z0-9]{3,10}-CC\d-L\d{3}-\d+`)},
	{"guide-commvault-dp", "Drive Pool CommVault Galaxy (Appendix F)",
		lre(`[A-Z0-9-]+-DP\d`)},
	{"guide-commvault-tapedrive", "привод ленточной библиотеки CommVault Galaxy (Appendix F)",
		lre(`[A-Z0-9]{2,10}-TC\d{2}-D\d{4}`)},
	{"guide-commvault-policy", "Storage Policy CommVault Galaxy (Appendix F)",
		lre(`[A-Z0-9]+-S-[A-Z0-9]+-SP`)},

	// --- Appendix G: EUC Devices ---
	{"guide-euc-bladechassis", "blade-шасси EUC (Appendix G): 5-значный номер вместо 3-значного профиля 2026",
		lre(`[A-Z0-9]{2,10}-BC-\d{5}-R`)},
	{"guide-euc-storage", "устройство хранения EUC (Appendix G): <Loc>-SD[DNS]-<seq>",
		lre(`[A-Z0-9]{2,10}-SD[DNS]-\d{5}`)},
	{"guide-euc-tape", "ленточная библиотека / D2D backup EUC (Appendix G)",
		lre(`[A-Z0-9]{2,10}TLV?\d{4,5}`)},
	{"guide-euc-stms", "Storage Management Server EUC (Appendix G): <Loc>-A-<seq>",
		lre(`[A-Z0-9]{2,10}-A-\d{5}`)},
}

// matchLegacy перебирает правила Guide 2019 и возвращает первое совпадение. Порядок важен: более
// специфичные формы (суффиксы поверх основного имени, кластерные ресурсы) идут раньше общих.
func matchLegacy(name string) (LegacyMatch, bool) {
	for _, r := range legacyRules {
		if r.re.MatchString(name) {
			return LegacyMatch{Family: r.family, Note: r.note}, true
		}
	}
	return LegacyMatch{}, false
}
