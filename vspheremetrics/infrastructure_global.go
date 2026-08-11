package vspheremetrics

import (
	"strings"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/vmware/govmomi/vim25/mo"

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

// LockGetMetric устанавливает блокировку сбора метрик (spin-wait до TimeOut),
// используется для приостановки опроса на время изменения конфигурации.
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

// UnLockGetMetric снимает блокировку сбора метрик, установленную LockGetMetric.
func (info *VMInfrastructure) UnLockGetMetric() error {
	if info == nil {
		return errors.New(nilPointerError)
	}
	atomic.StoreInt32(&info.PerfLock, 0)
	return nil
}

// IsLockGetMetric возвращает true, если сбор метрик в данный момент заблокирован.
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

// ClearUsersAndPasswords маскирует логины и пароли всех vCenter в инфраструктуре
// (заменяет их на "********"), используется перед выводом/экспортом конфигурации.
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

// GlobalInitVMInfrastructure создаёт новый пустой экземпляр VMInfrastructure,
// делает его глобальным (infrastructureGlobal) и возвращает указатель на него.
func GlobalInitVMInfrastructure() *VMInfrastructure {

	infrastructureMutex.Lock()
	defer infrastructureMutex.Unlock()
	infrastructureGlobal = &VMInfrastructure{}
	infrastructureGlobal.Init()
	return infrastructureGlobal
}

// GetGlobalVMInfrastructure возвращает глобальный экземпляр VMInfrastructure,
// инициализируя его при первом обращении, если он ещё не создан.
func GetGlobalVMInfrastructure() *VMInfrastructure {

	if infrastructureGlobal == nil {
		infrastructureGlobal = &VMInfrastructure{}
		infrastructureGlobal.Init()
	}
	return infrastructureGlobal
}

// DestroyGlobalVMInfrastructureObj освобождает ресурсы, удерживаемые глобальным экземпляром VMInfrastructure.
func DestroyGlobalVMInfrastructureObj() {
	infrastructureMutex.Lock()
	defer infrastructureMutex.Unlock()
	if infrastructureGlobal != nil {
		infrastructureGlobal.Destroy()
	}
}

// GetHostList возвращает список всех хостов ESXi из глобальной инфраструктуры.
