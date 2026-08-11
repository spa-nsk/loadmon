package cmdb

import (
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"

	//"loadmon/itsystem"
	models "loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func SaveClusterRecord(cl *vsphere.Cluster, clId int64, currentTime time.Time) (int64, error) {
	//flMetrics := int32(0)
	//perf := make([]*vsphere.CLPerf, 0)
	//go func() {
	perf := grpcclient.LoadClusterMetrics(cl.Vcenter, cl.Datacenter, cl.Name)
	//atomic.StoreInt32(&flMetrics, int32(1))
	//}()

	o := orm.NewOrm()

	// формируем запись по кластеру и вставляем в базу данных (по одной записи на кластер в сутки с итоговыми данными за сутки)
	//clusterRecord := models.ClusterRecord{Name: cl.Name, ClusterId: clId, Date: time.Now().Local()}
	clusterRecord := models.ClusterRecord{Name: cl.Name, ClusterId: clId, Date: currentTime}
	importClusterRecord(&clusterRecord, cl)
	clusterId, err := o.Insert(&clusterRecord) // в переменной clusterId запоминаем id записи о кластере, далее этот номер будет использоваться для связи с хостами, харнилищами и виртуальными
	if err != nil {
		//fmt.Println("Ошибка вставки записи по кластеру", cl.Name, err)
		return 0, err
	}
	//ждем окончания запроса по метрикам
	//for i := int32(0); !atomic.CompareAndSwapInt32(&flMetrics, 1, 0); i++ { //ожидаем 1 в переменной flEnd
	//	if i >= atomic.LoadInt32(&vsphere.TimeOut) {
	//		fmt.Println(timeOutError, cl.Name)
	//		break
	//	}
	//	time.Sleep(time.Millisecond)
	//runtime.Gosched() //передаём управление планировщику
	// }

	//записываем метрики по кластеру
	for _, clPerf := range perf {
		clusterPerf := models.ClusterPerf{ClusterId: clusterId}
		ImportClusterPerf(&clusterPerf, clPerf)
		_, err := o.Insert(&clusterPerf)
		if err != nil {
			//fmt.Println("Ошибка вставки записи метрик по кластеру", cl.Name, err)
			return clusterId, err
		}
	}
	return clusterId, nil
}

// CheckEnclosureSprRecord проверяет наличие в справочнике блейд-корзин (BladeSystems)
// записи для корзины хоста и при необходимости создаёт новую. Возвращает Id корзины.
