package vspheremetrics

import (
	"fmt"
	"log"
	"strings"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"net/url"

	"golang.org/x/net/context"
	"sort"
	"time"

	"errors"
	"sync/atomic"
)

var (
	infrastructureGlobal  *VMInfrastructure
	infrastructureMutex   sync.RWMutex
	vminfraNotInitialised string = "ошибка VMInfrastructure не инициализирован"
	nilPointerError       string = "Указатель nil"
	timeOutError          string = "Истекло время ожидания блокировки"
)

func (info *VMInfrastructure) LockGetMetric() error {
	if info == nil {
		return errors.New(nilPointerError)
	}
	for i := int32(0); !atomic.CompareAndSwapInt32(&info.PerfLock, 0, 1); i++ {
		if i >= TimeOut {
			return errors.New(timeOutError)
		}
		time.Sleep(time.Millisecond)
	}
	return nil
}

func (info *VMInfrastructure) UnLockGetMetric() error {
	if info == nil {
		return errors.New(nilPointerError)
	}
	atomic.StoreInt32(&info.PerfLock, 0)
	return nil
}

func (info *VMInfrastructure) IsLockGetMetric() bool {
	if info == nil {
		return false
	}
	if atomic.LoadInt32(&info.PerfLock) == 1 {
		return true
	} else {
		return false
	}
}

func (this *VMInfrastructure) appendScsiLun(vcenter string, datacenter string, cluster string, hi mo.HostSystem) {
	if this == nil {
		return
	}
	if hi.Config == nil {
		return
	}
	if hi.Config.StorageDevice != nil {

		for _, baseScsiDisk := range hi.Config.StorageDevice.ScsiLun { //types.HostScsiDisk
			scsiLun := baseScsiDisk.GetScsiLun() //*types.ScsiLun
			uid := ""
			sliceName := strings.Split(strings.ToUpper(scsiLun.CanonicalName), ".")

			if sliceName[0] == "NAA" {
				uid = VmfsStrLunToUID(sliceName[1])
			} else {
				//fmt.Println(vcenter+"/"+scsiLun.Uuid, "diskName:", scsiLun.CanonicalName)
			}
			lun, ok := this.LUNMapListUUID.Get(vcenter + "/" + scsiLun.Uuid)
			if lun == nil || !ok {
				this.LUNMapListUUID.Set(vcenter+"/"+scsiLun.Uuid, &ScsiLun{
					Vcenter:       vcenter,
					Datacenter:    datacenter,
					Cluster:       cluster,
					UID:           uid,
					CanonicalName: scsiLun.CanonicalName,
					UUID:          scsiLun.Uuid,
					LunType:       scsiLun.LunType,
					DisplayName:   scsiLun.DisplayName})
			}
		}
	}
}

func (this *VMInfrastructure) ClearUsersAndPasswords() {
	if this == nil {
		return
	}
	clearString := "********"
	for _, key := range this.VCenterMapList.Keys() {
		vc, ok := this.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		vc.Username = clearString
		vc.Password = clearString
	}
}

func (this *VMInfrastructure) importUsersAndPasswords(from *VMInfrastructure) {
	if this == nil {
		return
	}
	if from == nil {
		return
	}
	vcMapList := cmap.New[*VCenter]()
	for _, key := range this.VCenterMapList.Keys() {
		vc, ok := this.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		vcMapList.Set(key, &VCenter{Hostname: vc.Hostname, Username: vc.Username, Password: vc.Password, Wait: vc.Wait})
	}
	this.VCenterMapList = vcMapList
}

// SetGlobalVMInfrastructure установить значение infrastructureGlobal
func SetGlobalVMInfrastructure(init *VMInfrastructure) *VMInfrastructure {
	if init == nil {
		return nil
	}
	infrastructureMutex.RLock()
	infra := infrastructureGlobal
	infrastructureMutex.RUnlock()
	init.importUsersAndPasswords(infra)
	//заблокировать операции с infrastructureGlobal
	infrastructureMutex.Lock()
	infrastructureGlobal = init
	infrastructureMutex.Unlock()
	return infrastructureGlobal
}

func GlobalInitVMInfrastructure() *VMInfrastructure {

	infrastructureMutex.Lock()
	defer infrastructureMutex.Unlock()
	infrastructureGlobal = &VMInfrastructure{}
	infrastructureGlobal.Init()
	return infrastructureGlobal
}

func GetGlobalVMInfrastructure() *VMInfrastructure {

	if infrastructureGlobal == nil {
		infrastructureGlobal = &VMInfrastructure{}
		infrastructureGlobal.Init()
	}
	return infrastructureGlobal
}

func DestroyGlobalVMInfrastructureObj() {
	infrastructureMutex.Lock()
	defer infrastructureMutex.Unlock()
	if infrastructureGlobal != nil {
		infrastructureGlobal.Destroy()
	}
}

func (infra *VMInfrastructure) GetHostList() ([]HostInfo, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var hostList []HostInfo
	hostList = make([]HostInfo, 0)

	for _, key := range infra.HostMapList.Keys() {
		hi, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		hiN := *hi
		hiN.Perf = make([]*HSPerf, len(hi.Perf))
		for i, perfI := range hi.Perf {
			perf := HSPerf{}
			perf = *perfI
			hiN.Perf[i] = &perf
		}
		//copy(hiN.Perf, hi.Perf)
		hostList = append(hostList, hiN)
	}

	sort.Sort(HSListID(hostList))
	return hostList, nil
}

func (infra *VMInfrastructure) GetHostListFilter(vcenter, datacenter, cluster, hostname string) ([]HostInfo, error) {

	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var hostList []HostInfo
	hostList = make([]HostInfo, 0)
	for _, key := range infra.HostMapList.Keys() {
		hi, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		if vcenter != "all" && hi.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && hi.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && hi.Cluster != cluster {
			continue
		}
		if hostname != "all" && hi.Name != hostname {
			continue
		}
		hiN := *hi
		hiN.Perf = make([]*HSPerf, len(hi.Perf))
		copy(hiN.Perf, hi.Perf)
		hostList = append(hostList, hiN)
	}

	sort.Sort(HSListID(hostList))

	return hostList, nil
}

func (infra *VMInfrastructure) GetVMList() ([]VMPoint, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var vmList []VMPoint
	vmList = make([]VMPoint, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		vmN := *vm
		vmN.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmN.Perf, vm.Perf)
		vmList = append(vmList, vmN)
	}

	sort.Sort(VMListID(vmList))
	return vmList, nil
}

func (infra *VMInfrastructure) GetVMListFilter(vcenter, datacenter, cluster string, vmname string) ([]VMPoint, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var vmList []VMPoint
	vmList = make([]VMPoint, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && vm.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && datacenter != "" && vm.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && cluster != "" && vm.Cluster != cluster {
			continue
		}
		if vmname != "all" && vmname != "" && !strings.Contains(vm.Name, vmname) {
			continue
		}

		vmN := *vm
		vmN.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmN.Perf, vm.Perf)
		vmList = append(vmList, vmN)
	}
	sort.Sort(VMListID(vmList))
	return vmList, nil
}

func (infra *VMInfrastructure) GetVM(vcenter, datacenter, cluster string, vmname string) (VMPoint, error) {
	if infra == nil {
		return VMPoint{}, errors.New(vminfraNotInitialised)
	}
	//блокируем  и снимаем при выходе из функции

	key := vcenter + "/" + datacenter + "/" + cluster + "/" + vmname
	vm, ok := infra.VMMapList.Get(key)

	if ok {
		return *vm, nil
	}
	return VMPoint{}, errors.New("VM не найдена")
}

func (infra *VMInfrastructure) GetVMKey(key string) (VMPoint, error) {
	if infra == nil {
		return VMPoint{}, errors.New(vminfraNotInitialised)
	}
	vm, ok := infra.VMMapList.Get(key)

	if ok {
		return *vm, nil
	}
	return VMPoint{}, errors.New("VM не найдена")
}

func (infra *VMInfrastructure) GetVMPerfomance(id int64) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vm.ID == id {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

func (infra *VMInfrastructure) GetVMPerfomanceName(name string) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if strings.Contains(vm.HostName, name) {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

func (infra *VMInfrastructure) GetVMPerfomanceKey(vcenter string, datacenter string, cluster string, vmname string) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)

	key := vcenter + "/" + datacenter + "/" + cluster + "/" + vmname
	vm, ok := infra.VMMapList.Get(key)
	if ok {
		vmp = *vm
		vmp.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmp.Perf, vm.Perf)
		return &vmp
	}
	return &vmp
}

func (infra *VMInfrastructure) GetVMPerfomanceID(id int64) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		if vm.ID == id {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

func (vcenter *VCenter) Connect() (*govmomi.Client, error) {

	if vcenter == nil {
		return nil, errors.New("ошибка VCenter не инициализирован")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	username := url.QueryEscape(vcenter.Username)
	password := url.QueryEscape(vcenter.Password)
	u, err := url.Parse("https://" + username + ":" + password + "@" + vcenter.Hostname + "/sdk")
	if err != nil {
		log.Printf("vcenter %s: ошибка разбора строки кодключения vcenter - %s\n", vcenter.Hostname, err)
		return nil, err
	}
	client, err := govmomi.NewClient(ctx, u, true) //insecure false
	if err != nil {
		log.Printf("vcenter %s: невозможно подключиться к vcenter - %s username=%s\n", vcenter.Hostname, err, vcenter.Username)
		return nil, err
	}
	return client, nil
}

func (infra *VMInfrastructure) Init() {
	if infra == nil {
		fmt.Println("func (infra *VMInfrastructure) Init()", errors.New(vminfraNotInitialised))
		return
	}
	infra.VCenterMapList = cmap.New[*VCenter]()
	infra.ClusterMapList = cmap.New[*Cluster]()
	infra.VMMapList = cmap.New[*VMPoint]()
	infra.VMIDMapList = cmap.New[*VMPoint]()
	infra.VMUUIDMapList = cmap.New[*VMPoint]()
	infra.HostMapList = cmap.New[*HostInfo]()
	infra.HostBySNMapList = cmap.New[*HostInfo]()
	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

func (infra *VMInfrastructure) Destroy() {

	if infra == nil {
		return
	}
	for _, key := range infra.VCenterMapList.Keys() {
		infra.VCenterMapList.Remove(key)
	}

	for _, key := range infra.ClusterMapList.Keys() {
		infra.ClusterMapList.Remove(key)
	}

	for _, key := range infra.VMMapList.Keys() {
		infra.VMMapList.Remove(key)
	}

	for _, key := range infra.VMIDMapList.Keys() {
		infra.VMIDMapList.Remove(key)
	}

	for _, key := range infra.VMUUIDMapList.Keys() {
		infra.VMUUIDMapList.Remove(key)
	}

	for _, key := range infra.HostMapList.Keys() {
		infra.HostMapList.Remove(key)
	}

	infra.HostBySNMapList.Clear()

	for _, key := range infra.DataStoreMapList.Keys() {
		infra.DataStoreMapList.Remove(key)
	}

	for _, key := range infra.LUNMapListUUID.Keys() {
		infra.LUNMapListUUID.Remove(key)
	}
	for _, key := range infra.TagsVM.Keys() {
		infra.TagsVM.Remove(key)
	}
	for _, key := range infra.TagsDC.Keys() {
		infra.TagsDC.Remove(key)
	}

	infra.ClusterMapList = cmap.New[*Cluster]()

	infra.VMMapList = cmap.New[*VMPoint]()
	infra.VMIDMapList = cmap.New[*VMPoint]()
	infra.VMUUIDMapList = cmap.New[*VMPoint]()

	infra.HostMapList = cmap.New[*HostInfo]()
	infra.HostBySNMapList = cmap.New[*HostInfo]()
	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

func (infra *VMInfrastructure) DestroyData() {

	if infra == nil {
		return
	}

	for _, key := range infra.ClusterMapList.Keys() {

		infra.ClusterMapList.Remove(key)
	}

	for _, key := range infra.VMMapList.Keys() {
		infra.VMMapList.Remove(key)
	}

	for _, key := range infra.VMIDMapList.Keys() {
		infra.VMIDMapList.Remove(key)
	}

	for _, key := range infra.VMUUIDMapList.Keys() {
		infra.VMUUIDMapList.Remove(key)
	}

	for _, key := range infra.HostMapList.Keys() {
		infra.HostMapList.Remove(key)
	}

	infra.HostBySNMapList.Clear()

	for _, key := range infra.DataStoreMapList.Keys() {
		infra.DataStoreMapList.Remove(key)
	}

	for _, key := range infra.LUNMapListUUID.Keys() {
		infra.LUNMapListUUID.Remove(key)
	}
	for _, key := range infra.TagsVM.Keys() {
		infra.TagsVM.Remove(key)
	}
	for _, key := range infra.TagsDC.Keys() {
		infra.TagsDC.Remove(key)
	}
	infra.ClusterMapList = cmap.New[*Cluster]()

	infra.HostMapList = cmap.New[*HostInfo]()

	infra.HostBySNMapList = cmap.New[*HostInfo]()

	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

func (infra *VMInfrastructure) DestroyPerfomance() {

	if infra == nil {
		return
	}
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		vm.VMClearPerfomance()
		vm.Perf = make([]*VMPerf, 0)
	}

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		hs.HSClearPerfomance()
		hs.Perf = make([]*HSPerf, 0)
	}

	currentTime := time.Now().Local()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		cl.CLClearPerfomance()
		cl.Date = currentTime //инициализируем Дату и время в структуре кластера
	}
}

func (infra *VMInfrastructure) AddVCenter(hostname string, username string, password string, wait int) error {

	if infra == nil {
		return errors.New(vminfraNotInitialised)
	}

	vc := VCenter{Hostname: hostname, Username: username, Password: password, Wait: wait, DoneFl: 0}

	infra.VCenterMapList.Set(hostname, &vc)
	return nil
}

func (infra *VMInfrastructure) StartVcenterTimer() {
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		vc.Ticker = time.NewTicker(time.Second * time.Duration(vc.Wait))
	}
	infra.Ticker = time.NewTicker(time.Hour * time.Duration(1))
	infra.Done = make(chan bool)
}

// функция пока не используется
func (infra *VMInfrastructure) CalcMetricPerHour() {
	for {
		select {
		case <-infra.Done:
			return
		case <-infra.Ticker.C:
			atomic.StoreUint64(&infra.MetricsPerHour, atomic.LoadUint64(&infra.CountMetrics))
			atomic.StoreUint64(&infra.CountMetrics, 0)
		}
	}
}

func (infra *VMInfrastructure) StopVcenterTimer() {
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		if atomic.LoadInt64(&vc.CountThread) > 0 {
			vc.Ticker.Stop()
		}
	}
	if atomic.LoadInt64(&infra.CountThread) > 0 {
		infra.Ticker.Stop()
	}
}

func (infra *VMInfrastructure) SetDoneVcenter() {
	log.Println("Остановка таймеров обработчиков метрик")
	infra.StopVcenterTimer()
	log.Println("Таймеры обработчиков метрик остановлены")
	count := infra.VCenterMapList.Count()
	for i, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			log.Println("Ошибка поиска vcenter:", key)
			continue
		}
		if atomic.LoadInt64(&vc.CountThread) > 0 {
			log.Println("Передача сигнала остановки обработчика: ", vc.Hostname, i, "из", count)
			atomic.StoreInt32(&vc.DoneFl, 1)
			log.Println("Сигнал остановки обработчика:", vc.Hostname, "передан")
		}
	}
}

func (infra *VMInfrastructure) GetVCenter(hostname string) (VCenter, error) {

	if infra == nil {
		return VCenter{}, errors.New(vminfraNotInitialised)
	}
	vc, ok := infra.VCenterMapList.Get(hostname)
	if ok {
		return *vc, nil
	}
	return VCenter{}, errors.New(hostname + " не найден.")
}

// возвращает максимальное значение из среза
// сейчас сделана корректировка если длинна среза равна 2, то возвращает 1 значение
func maxValue(value []int64) int64 {
	start := true
	max := int64(0)
	for i := range value {
		if start {
			start = false
			max = value[i]
			continue
		}
		if max < value[i] {
			max = value[i]
		}
	}
	return max
}
