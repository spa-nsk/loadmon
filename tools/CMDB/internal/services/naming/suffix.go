// Производные имена — суффиксы к имени родителя (intent §5.4). В отличие от основного
// шаблона хоста они лимитированы как DNS-метки (≤63 символа, RFC 1123), а не 15 символами.
package naming

import (
	"fmt"
	"regexp"
	"strings"
)

const maxDNSLabelLen = 63

var rfc1123LabelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func checkDNSLabel(name string) error {
	if len(name) == 0 || len(name) > maxDNSLabelLen {
		return fmt.Errorf("naming: %q должно быть 1-%d символов", name, maxDNSLabelLen)
	}
	if !rfc1123LabelRe.MatchString(name) {
		return fmt.Errorf("naming: %q не соответствует RFC 1123 (a-z0-9-, без _, не начинается/не заканчивается на -)", name)
	}
	return nil
}

// BuildBMCName строит имя BMC/out-of-band интерфейса: <host>-bmc (замена suffix "-R" из Guide).
func BuildBMCName(hostName string) (string, error) {
	name := DNSName(hostName) + "-bmc"
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// BuildStorageIfaceName строит имя интерфейса хранения (iSCSI/NVMe-oF/NFS): <host>-st<n>.
func BuildStorageIfaceName(hostName string, n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("naming: номер интерфейса %d не может быть отрицательным", n)
	}
	name := fmt.Sprintf("%s-st%d", DNSName(hostName), n)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// ClusterResourceKind — тип ресурса/listener кластера (intent §5.4).
type ClusterResourceKind byte

const (
	ClusterResourceDB       ClusterResourceKind = 'q' // СУБД
	ClusterResourceFile     ClusterResourceKind = 'f' // файловый
	ClusterResourceApp      ClusterResourceKind = 'a' // приложение
	ClusterResourceListener ClusterResourceKind = 'l' // listener
	ClusterResourceVIP      ClusterResourceKind = 'v' // VIP
)

// BuildClusterResourceName строит имя ресурса/listener кластера: <cluster>-<R><n>.
func BuildClusterResourceName(clusterName string, kind ClusterResourceKind, n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("naming: номер ресурса %d не может быть отрицательным", n)
	}
	name := fmt.Sprintf("%s-%c%d", DNSName(clusterName), kind, n)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// BuildPortName строит имя порта коммутатора/СХД: <device>-p<nn>.
func BuildPortName(deviceName string, port int) (string, error) {
	if port < 0 {
		return "", fmt.Errorf("naming: номер порта %d не может быть отрицательным", port)
	}
	name := fmt.Sprintf("%s-p%d", DNSName(deviceName), port)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// BuildHBAPortName строит имя порта HBA: <host>-hba<n>p<m>.
func BuildHBAPortName(hostName string, hba, port int) (string, error) {
	if hba < 0 || port < 0 {
		return "", fmt.Errorf("naming: номера HBA/порта не могут быть отрицательными")
	}
	name := fmt.Sprintf("%s-hba%dp%d", DNSName(hostName), hba, port)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// BuildSANZoneName строит имя SAN-зоны (single-initiator zoning): z_<host>_hba<n>_<array>.
func BuildSANZoneName(hostName string, hba int, arrayName string) (string, error) {
	if hba < 0 {
		return "", fmt.Errorf("naming: номер HBA %d не может быть отрицательным", hba)
	}
	name := fmt.Sprintf("z_%s_hba%d_%s", DNSName(hostName), hba, DNSName(arrayName))
	// SAN-зоны не являются DNS-метками (допускают "_"), поэтому длину проверяем отдельно
	// и не прогоняем через checkDNSLabel/RFC1123.
	if len(name) == 0 || len(name) > 128 {
		return "", fmt.Errorf("naming: %q должно быть 1-128 символов", name)
	}
	return name, nil
}

// BuildVolumeName строит имя тома/LUN: <cluster-or-host>_<purpose>_<nnn>.
func BuildVolumeName(parentName, purpose string, n int) (string, error) {
	if n < 0 || n > 999 {
		return "", fmt.Errorf("naming: номер тома %d должен быть в диапазоне 0-999", n)
	}
	purpose = strings.ToLower(purpose)
	name := fmt.Sprintf("%s_%s_%03d", DNSName(parentName), purpose, n)
	if len(name) == 0 || len(name) > 128 {
		return "", fmt.Errorf("naming: %q должно быть 1-128 символов", name)
	}
	return name, nil
}

// BuildSVMName строит имя SVM/файлового сервера хранения: <SITE>-svm-<nnn>.
func BuildSVMName(site string, n int) (string, error) {
	site = strings.ToUpper(site)
	if !siteRe.MatchString(site) {
		return "", fmt.Errorf("naming: недопустимый SITE %q: ожидается 1-6 символов A-Z/0-9", site)
	}
	if n < 0 || n > 999 {
		return "", fmt.Errorf("naming: номер SVM %d должен быть в диапазоне 0-999", n)
	}
	name := fmt.Sprintf("%s-svm-%03d", strings.ToLower(site), n)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}

// BuildK8sClusterName строит имя кластера Kubernetes/OpenShift/Deckhouse: k8s-<site>-<env>-<nn> (intent §5.5).
func BuildK8sClusterName(site, env string, n int) (string, error) {
	site = strings.ToUpper(site)
	if !siteRe.MatchString(site) {
		return "", fmt.Errorf("naming: недопустимый SITE %q: ожидается 1-6 символов A-Z/0-9", site)
	}
	if n < 0 || n > 99 {
		return "", fmt.Errorf("naming: номер кластера %d должен быть в диапазоне 0-99", n)
	}
	env = strings.ToLower(env)
	name := fmt.Sprintf("k8s-%s-%s-%02d", strings.ToLower(site), env, n)
	if err := checkDNSLabel(name); err != nil {
		return "", err
	}
	return name, nil
}
