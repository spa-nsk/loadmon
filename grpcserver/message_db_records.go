package grpcserver

import (
	"loadmon/models"

	pb "loadmon/protocol"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// ClusterRecordToMessage конвертирует models.ClusterRecord в gRPC-сообщение pb.ClusterRecord (все поля, включая метрики min/max/avg). При in == nil или out == nil ничего не делает.
func ClusterRecordToMessage(in *models.ClusterRecord, out *pb.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                                                                 //int64
	out.ClusterId = in.ClusterId                                                                   //int64
	out.Date = timestamppb.New(in.Date)                                                            //time.Time
	out.Vcenter = in.Vcenter                                                                       //string
	out.Datacenter = in.Datacenter                                                                 //string
	out.Name = in.Name                                                                             //string
	out.CpuMhzAvg = in.CpuMhzAvg                                                                   //int32
	out.AllNumCpuPkgs = int32(in.AllNumCpuPkgs)                                                    //int16
	out.AllNumCpuCores = int32(in.AllNumCpuCores)                                                  //int16
	out.AllNumCpuThreads = int32(in.AllNumCpuThreads)                                              //int16
	out.AllMemorySizeGB = in.AllMemorySizeGB                                                       //float64
	out.AllCapacityGb = in.AllCapacityGb                                                           //float64
	out.AllFreeSpaceGb = in.AllFreeSpaceGb                                                         //float64
	out.AllPercentUsed = in.AllPercentUsed                                                         //float64
	out.CpuUsageMaximum = in.Cpu_usage_maximum                                                     //float64
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time)                           //time.Time
	out.CpuUsageAverage = in.Cpu_usage_average                                                     //float64
	out.CpuUsageMinimum = in.Cpu_usage_minimum                                                     //float64
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time)                           ////time.Time
	out.MemUsageMaximum = in.Mem_usage_maximum                                                     //float64
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time)                           ////time.Time
	out.MemUsageAverage = in.Mem_usage_average                                                     //float64
	out.MemUsageMinimum = in.Mem_usage_minimum                                                     //float64
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time)                           ////time.Time
	out.DiskUsageMaximum = in.Disk_usage_maximum                                                   //float64
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time)                         ////time.Time
	out.DiskUsageAverage = in.Disk_usage_average                                                   //float64
	out.DiskUsageMinimum = in.Disk_usage_minimum                                                   //float64
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time)                         ////time.Time
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum                             //float64
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time)   ////time.Time
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average                             //float64
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum                             //float64
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time)   ////time.Time
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum                           //float64
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time) ////time.Time
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average                           //float64
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum                           //float64
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time) ////time.Time
	out.DiskWriteMaximum = in.Disk_write_maximum                                                   //float64
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time)                         ////time.Time
	out.DiskWriteAverage = in.Disk_write_average                                                   //float64
	out.DiskWriteMinimum = in.Disk_write_minimum                                                   //float64
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time)                         ////time.Time
	out.DiskReadMaximum = in.Disk_read_maximum                                                     //float64
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time)                           ////time.Time
	out.DiskReadAverage = in.Disk_read_average                                                     //float64
	out.DiskReadMinimum = in.Disk_read_minimum                                                     //float64
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time)                           ////time.Time
	out.NetUsageMaximum = in.Net_usage_maximum                                                     //float64
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time)                           ////time.Time
	out.NetUsageAverage = in.Net_usage_average                                                     //float64
	out.NetUsageMinimum = in.Net_usage_minimum                                                     //float64
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time)                           ////time.Time
	out.CpuBilling = in.CpuBilling                                                                 //float64
	out.DiskBilling = in.DiskBilling                                                               //float64
	out.MemBilling = in.MemBilling                                                                 //float64
	out.NetBilling = in.NetBilling                                                                 //float64
	out.HostCount = in.HostCount                                                                   //int64
	out.VMCount = in.VMCount                                                                       //int64
	out.StorageCount = in.StorageCount                                                             //int64
}

// MessageToClusterRecord конвертирует gRPC-сообщение pb.ClusterRecord в доменную модель models.ClusterRecord. При in == nil или out == nil ничего не делает.
func MessageToClusterRecord(in *pb.ClusterRecord, out *models.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                                                         //int64
	out.ClusterId = in.ClusterId                                                           //int64
	out.Date = in.Date.AsTime()                                                            //time.Time
	out.Vcenter = in.Vcenter                                                               //string
	out.Datacenter = in.Datacenter                                                         //string
	out.Name = in.Name                                                                     //string
	out.CpuMhzAvg = in.CpuMhzAvg                                                           //int32
	out.AllNumCpuPkgs = int16(in.AllNumCpuPkgs)                                            //int16
	out.AllNumCpuCores = int16(in.AllNumCpuCores)                                          //int16
	out.AllNumCpuThreads = int16(in.AllNumCpuThreads)                                      //int16
	out.AllMemorySizeGB = in.AllMemorySizeGB                                               //float64
	out.AllCapacityGb = in.AllCapacityGb                                                   //float64
	out.AllFreeSpaceGb = in.AllFreeSpaceGb                                                 //float64
	out.AllPercentUsed = in.AllPercentUsed                                                 //float64
	out.Cpu_usage_maximum = in.CpuUsageMaximum                                             //float64
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime()                           //time.Time
	out.Cpu_usage_average = in.CpuUsageAverage                                             //float64
	out.Cpu_usage_minimum = in.CpuUsageMinimum                                             //float64
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime()                           //time.Time
	out.Mem_usage_maximum = in.MemUsageMaximum                                             //float64
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime()                           //time.Time
	out.Mem_usage_average = in.MemUsageAverage                                             //float64
	out.Mem_usage_minimum = in.MemUsageMinimum                                             //float64
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime()                           //time.Time
	out.Disk_usage_maximum = in.DiskUsageMaximum                                           //float64
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime()                         //time.Time
	out.Disk_usage_average = in.DiskUsageAverage                                           //float64
	out.Disk_usage_minimum = in.DiskUsageMinimum                                           //float64
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime()                         //time.Time
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum                     //float64
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime()   //time.Time
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage                     //float64
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum                     //float64
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime()   //time.Time
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum                   //float64
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage                   //float64
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum                   //float64
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime() //time.Time
	out.Disk_write_maximum = in.DiskWriteMaximum                                           //float64
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime()                         //time.Time
	out.Disk_write_average = in.DiskWriteAverage                                           //float64
	out.Disk_write_minimum = in.DiskWriteMinimum                                           //float64
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime()                         //time.Time
	out.Disk_read_maximum = in.DiskReadMaximum                                             //float64
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime()                           //time.Time
	out.Disk_read_average = in.DiskReadAverage                                             //float64
	out.Disk_read_minimum = in.DiskReadMinimum                                             //float64
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime()                           //time.Time
	out.Net_usage_maximum = in.NetUsageMaximum                                             //float64
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime()                           //time.Time
	out.Net_usage_average = in.NetUsageAverage                                             //float64
	out.Net_usage_minimum = in.NetUsageMinimum                                             //float64
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime()                           //time.Time
	out.CpuBilling = in.CpuBilling                                                         //float64
	out.DiskBilling = in.DiskBilling                                                       //float64
	out.MemBilling = in.MemBilling                                                         //float64
	out.NetBilling = in.NetBilling                                                         //float64
	out.HostCount = in.HostCount                                                           //int64
	out.VMCount = in.VMCount                                                               //int64
	out.StorageCount = in.StorageCount                                                     //int64
}

// HostRecordToMessage конвертирует доменную модель models.HostRecord в gRPC-сообщение pb.HostRecord. При in == nil или out == nil ничего не делает.
func HostRecordToMessage(in *models.HostRecord, out *pb.HostRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                                                                 //int64
	out.ServerId = in.ServerId                                                                     //int64
	out.ClusterId = in.ClusterId                                                                   //int64
	out.Date = timestamppb.New(in.Date)                                                            //time.Time
	out.Vcenter = in.Vcenter                                                                       //string
	out.Datacenter = in.Datacenter                                                                 //string
	out.Cluster = in.Cluster                                                                       //string
	out.Name = in.Name                                                                             //string
	out.Vendor = in.Vendor                                                                         //string
	out.Model = in.Model                                                                           //string
	out.CpuModel = in.CpuModel                                                                     //string
	out.CpuMhz = in.CpuMhz                                                                         //int32
	out.NumCpuPkgs = int32(in.NumCpuPkgs)                                                          //int16
	out.NumCpuCores = int32(in.NumCpuCores)                                                        //int16
	out.NumCpuThreads = int32(in.NumCpuThreads)                                                    //int16
	out.NumNics = in.NumNics                                                                       //int32
	out.NumHBAs = in.NumHBAs                                                                       //int32
	out.MemorySizeGB = in.MemorySizeGB                                                             //float64
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity                                 //float64
	out.CpuUsageMaximum = in.Cpu_usage_maximum                                                     //float64
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time)                           //time.Time
	out.CpuUsageAverage = in.Cpu_usage_average                                                     //float64
	out.CpuUsageMinimum = in.Cpu_usage_minimum                                                     //float64
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time)                           //time.Time
	out.MemUsageMaximum = in.Mem_usage_maximum                                                     //float64
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time)                           //time.Time
	out.MemUsageAverage = in.Mem_usage_average                                                     //float64
	out.MemUsageMinimum = in.Mem_usage_minimum                                                     //float64
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time)                           //time.Time
	out.DiskUsageMaximum = in.Disk_usage_maximum                                                   //float64
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time)                         //time.Time
	out.DiskUsageAverage = in.Disk_usage_average                                                   //float64
	out.DiskUsageMinimum = in.Disk_usage_minimum                                                   //float64
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time)                         //time.Time
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum                             //float64
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time)   //time.Time
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average                             //float64
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum                             //float64
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time)   //time.Time
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum                           //float64
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time) //time.Time
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average                           //float64
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum                           //float64
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time) //time.Time
	out.DiskWriteMaximum = in.Disk_write_maximum                                                   //float64
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time)                         //time.Time
	out.DiskWriteAverage = in.Disk_write_average                                                   //float64
	out.DiskWriteMinimum = in.Disk_write_minimum                                                   //float64
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time)                         //time.Time
	out.DiskReadMaximum = in.Disk_read_maximum                                                     //float64
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time)                           //time.Time
	out.DiskReadAverage = in.Disk_read_average                                                     //float64
	out.DiskReadMinimum = in.Disk_read_minimum                                                     //float64
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time)                           //time.Time
	out.NetUsageMaximum = in.Net_usage_maximum                                                     //float64
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time)                           //time.Time
	out.NetUsageAverage = in.Net_usage_average                                                     //float64
	out.NetUsageMinimum = in.Net_usage_minimum                                                     //float64
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time)                           //time.Time
	out.CpuBilling = in.CpuBilling                                                                 //float64
	out.MemBilling = in.MemBilling                                                                 //float64
	out.DiskBilling = in.DiskBilling                                                               //float64
	out.NetBilling = in.NetBilling                                                                 //float64
	out.IpAddress = in.IpAddress
}

// MessageToHostRecord конвертирует gRPC-сообщение pb.HostRecord в доменную модель models.HostRecord. При in == nil или out == nil ничего не делает.
func MessageToHostRecord(in *pb.HostRecord, out *models.HostRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                 //int64
	out.ServerId = in.ServerId     //int64
	out.ClusterId = in.ClusterId   //int64
	out.Date = in.Date.AsTime()    //time.Time
	out.Vcenter = in.Vcenter       //string
	out.Datacenter = in.Datacenter //string
	out.Cluster = in.Cluster       //string
	out.Name = in.Name             //string
	out.Vendor = in.Vendor         //string
	out.Model = in.Model
	out.CpuModel = in.CpuModel                                                             //string
	out.CpuMhz = in.CpuMhz                                                                 //int32
	out.NumCpuPkgs = int16(in.NumCpuPkgs)                                                  //int16
	out.NumCpuCores = int16(in.NumCpuCores)                                                //int16
	out.NumCpuThreads = int16(in.NumCpuThreads)                                            //int16
	out.NumNics = in.NumNics                                                               //int32
	out.NumHBAs = in.NumHBAs                                                               //int32
	out.MemorySizeGB = in.MemorySizeGB                                                     //float64
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity                         //float64
	out.Cpu_usage_maximum = in.CpuUsageMaximum                                             //float64
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime()                           //time.Time
	out.Cpu_usage_average = in.CpuUsageAverage                                             //float64
	out.Cpu_usage_minimum = in.CpuUsageMinimum                                             //float64
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime()                           //time.Time
	out.Mem_usage_maximum = in.MemUsageMaximum                                             //float64
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime()                           //time.Time
	out.Mem_usage_average = in.MemUsageAverage                                             //float64
	out.Mem_usage_minimum = in.MemUsageMinimum                                             //float64
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime()                           //time.Time
	out.Disk_usage_maximum = in.DiskUsageMaximum                                           //float64
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime()                         //time.Time
	out.Disk_usage_average = in.DiskUsageAverage                                           //float64
	out.Disk_usage_minimum = in.DiskUsageMinimum                                           //float64
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime()                         //time.Time
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum                     //float64
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime()   //time.Time
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage                     //float64
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum                     //float64
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime()   //time.Time
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum                   //float64
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage                   //float64
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum                   //float64
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime() //time.Time
	out.Disk_write_maximum = in.DiskWriteMaximum                                           //float64
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime()                         //time.Time
	out.Disk_write_average = in.DiskWriteAverage                                           //float64
	out.Disk_write_minimum = in.DiskWriteMinimum                                           //float64
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime()                         //time.Time
	out.Disk_read_maximum = in.DiskReadMaximum                                             //float64
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime()                           //time.Time
	out.Disk_read_average = in.DiskReadAverage                                             //float64
	out.Disk_read_minimum = in.DiskReadMinimum                                             //float64
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime()                           //time.Time
	out.Net_usage_maximum = in.NetUsageMaximum                                             //float64
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime()                           //time.Time
	out.Net_usage_average = in.NetUsageAverage                                             //float64
	out.Net_usage_minimum = in.NetUsageMinimum                                             //float64
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime()                           //time.Time
	out.CpuBilling = in.CpuBilling                                                         //float64
	out.MemBilling = in.MemBilling                                                         //float64
	out.DiskBilling = in.DiskBilling                                                       //float64
	out.NetBilling = in.NetBilling                                                         //float64
	out.IpAddress = in.IpAddress
}

// VMRecordToMessage конвертирует models.VMRecord в gRPC-сообщение pb.VMRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMRecordToMessage(in *models.VMRecord, out *pb.VMRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMSprId = in.VMSprId
	out.ClusterId = in.ClusterId
	out.Date = timestamppb.New(in.Date)
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.CpuUsageMaximum = in.Cpu_usage_maximum
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time)
	out.CpuUsageAverage = in.Cpu_usage_average
	out.CpuUsageMinimum = in.Cpu_usage_minimum
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time)
	out.MemUsageMaximum = in.Mem_usage_maximum
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time)
	out.MemUsageAverage = in.Mem_usage_average
	out.MemUsageMinimum = in.Mem_usage_minimum
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time)
	out.DiskUsageMaximum = in.Disk_usage_maximum
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time)
	out.DiskUsageAverage = in.Disk_usage_average
	out.DiskUsageMinimum = in.Disk_usage_minimum
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time)
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time)
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time)
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time)
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time)
	out.DiskWriteMaximum = in.Disk_write_maximum
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time)
	out.DiskWriteAverage = in.Disk_write_average
	out.DiskWriteMinimum = in.Disk_write_minimum
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time)
	out.DiskReadMaximum = in.Disk_read_maximum
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time)
	out.DiskReadAverage = in.Disk_read_average
	out.DiskReadMinimum = in.Disk_read_minimum
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time)
	out.NetUsageMaximum = in.Net_usage_maximum
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time)
	out.NetUsageAverage = in.Net_usage_average
	out.NetUsageMinimum = in.Net_usage_minimum
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time)
	out.CpuBilling = in.CpuBilling
	out.MemBilling = in.MemBilling
	out.DiskBilling = in.DiskBilling
	out.NetBilling = in.NetBilling
	out.Vm_Network = in.Vm_Network
}

// MessageToVMRecord выполняет обратную конвертацию pb.VMRecord в models.VMRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMRecord(in *pb.VMRecord, out *models.VMRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMSprId = in.VMSprId
	out.ClusterId = in.ClusterId
	out.Date = in.Date.AsTime()
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.Cpu_usage_maximum = in.CpuUsageMaximum
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime()
	out.Cpu_usage_average = in.CpuUsageAverage
	out.Cpu_usage_minimum = in.CpuUsageMinimum
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime()
	out.Mem_usage_maximum = in.MemUsageMaximum
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime()
	out.Mem_usage_average = in.MemUsageAverage
	out.Mem_usage_minimum = in.MemUsageMinimum
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime()
	out.Disk_usage_maximum = in.DiskUsageMaximum
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime()
	out.Disk_usage_average = in.DiskUsageAverage
	out.Disk_usage_minimum = in.DiskUsageMinimum
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime()
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime()
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime()
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime()
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime()
	out.Disk_write_maximum = in.DiskWriteMaximum
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime()
	out.Disk_write_average = in.DiskWriteAverage
	out.Disk_write_minimum = in.DiskWriteMinimum
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime()
	out.Disk_read_maximum = in.DiskReadMaximum
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime()
	out.Disk_read_average = in.DiskReadAverage
	out.Disk_read_minimum = in.DiskReadMinimum
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime()
	out.Net_usage_maximum = in.NetUsageMaximum
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime()
	out.Net_usage_average = in.NetUsageAverage
	out.Net_usage_minimum = in.NetUsageMinimum
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime()
	out.CpuBilling = in.CpuBilling
	out.MemBilling = in.MemBilling
	out.DiskBilling = in.DiskBilling
	out.NetBilling = in.NetBilling
	out.Vm_Network = in.Vm_Network
}

// ClusterPerfToMessage конвертирует models.ClusterPerf в gRPC-сообщение pb.ClusterPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterPerfToMessage(in *models.ClusterPerf, out *pb.ClusterPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ClusterId = in.ClusterId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToClusterPerf выполняет обратную конвертацию pb.ClusterPerf в models.ClusterPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToClusterPerf(in *pb.ClusterPerf, out *models.ClusterPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ClusterId = in.ClusterId
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// HostPerfToMessage конвертирует models.HostPerf в gRPC-сообщение pb.HostPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func HostPerfToMessage(in *models.HostPerf, out *pb.HostPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.HostId = in.HostId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToHostPerf выполняет обратную конвертацию pb.HostPerf в models.HostPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToHostPerf(in *pb.HostPerf, out *models.HostPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.HostId = in.HostId
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// VMPerfRecordToMessage конвертирует models.VMPerfRecord в gRPC-сообщение pb.VMPerfRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMPerfRecordToMessage(in *models.VMPerfRecord, out *pb.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.VMID = in.VMId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToVMPerfRecord выполняет обратную конвертацию pb.VMPerfRecord в models.VMPerfRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMPerfRecord(in *pb.VMPerfRecord, out *models.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.VMId = in.VMID
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// MessageToServerSprRecord выполняет обратную конвертацию pb.ServerSprRecords в models.ServerSprRecords.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToServerSprRecord(in *pb.ServerSprRecords, out *models.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.OldId = in.OldId
	out.Type = int(in.Type)
	out.TypeMicroservice = int(in.TypeMicroservice)
	out.RoleType = int(in.RoleType)
	out.TypeWork = int(in.TypeWork)
	out.CriticalLevel = int(in.CriticalLevel)
	out.VMSprId = in.VMSprId
	out.Stat = int(in.Stat)
	out.PowerState = int(in.PowerState)
	out.Active = int(in.Active)
	if in.TimeActiveState != nil {
		out.TimeActiveState = in.TimeActiveState.AsTime()
	}
	out.ActivatorUser = in.ActivatorUser
	if in.TimeDeActiveState != nil {
		out.TimeDeActiveState = in.TimeDeActiveState.AsTime()
	}
	out.DeActivatorUser = in.DeActivatorUser
	if in.TimeCreate != nil {
		out.TimeCreate = in.TimeCreate.AsTime()
	}
	out.CreatorUser = in.CreatorUser
	out.BusinesPrivilage = in.BusinesPrivilage
	out.Name = in.Name
	out.NameFQDN = in.NameFQDN
	out.NameInfo = in.NameInfo
	out.IPAddress = in.IPAddress
	out.IPSharedVIP = in.IPSharedVIP
	out.Network = in.Network
	out.NetworkName = in.NetworkName
	out.CPUCount = in.CPUCount
	out.CPULogical = in.CPULogical
	out.MemSize = in.MemSize
	out.DiskSize = in.DiskSize
	out.Service = in.Service
	out.ServiceId = in.ServiceId
	out.ITSystemKey = in.ITSystemKey
	out.OS = in.OS
	out.Component = in.Component
	out.Owner = in.Owner
	out.BusinessOwner = in.BusinessOwner
	out.ITOwner = in.ITOwner
	out.ProductOwner = in.ProductOwner
	out.Support = in.Support
	out.SupportTeam = in.SupportTeam
	out.SystemType = in.SystemType
	out.LicensingType = in.LicensingType
	out.SolutionType = in.SolutionType
	out.Users = in.Users
	out.Allocation = in.Allocation
	out.Anotation = in.Anotation
	out.Role = in.Role
	out.Details = in.Details
	out.SZHistory = in.SZhistory
	out.Warning = in.Warning
	out.Level = in.Level
	out.Monitoring = int(in.Monitoring)
	out.Backuped = int(in.Backuped)
	if in.TimeActiveMonitoring != nil {
		out.TimeActiveMonitoring = in.TimeActiveMonitoring.AsTime()
	}
	out.MonitoringSZ = in.MonitoringSZ
	out.Id_User = in.Id_User
	if in.Time_Mfy != nil {
		out.Time_Mfy = in.Time_Mfy.AsTime()
	}
	out.Privilage = int(in.Privilage)
	out.ClusterId = in.ClusterId
	out.Cluster = in.Cluster
	out.Datacenter = in.Datacenter
	out.Vcenter = in.Vcenter
	out.MacAddress = in.MacAddress
	out.HWSprId = in.HWSprId
	out.ZabbixHostId = in.ZabbixHostId
	out.ScomHostId = in.ScomHostId
	out.SerialNumber = in.SerialNumber
	out.SourceUpdateType = int(in.SourceUpdateType)
	out.TagName = in.TagName
	out.TagDescription = in.TagDescription
}

// MessageToVMSprRecord выполняет обратную конвертацию pb.VMSprRecord в models.VMSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMSprRecord(in *pb.VMSprRecord, out *models.VMSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.ClusterId = in.ClusterId
	out.Vm_Name = in.Vm_Name
	out.Vm_GuestFullName = in.Vm_GuestFullName
	out.Vm_HostName = in.Vm_HostName
	out.Vm_PathName = in.Vm_PathName
	out.Vm_PowerState = in.Vm_PowerState
	out.KeyName = in.KeyName
	out.Vm_Uuid = in.Vm_Uuid
	out.Vm_GuestId = in.Vm_GuestId
	out.Vm_Annotation = in.Vm_Annotation
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.Cpu_usage_maximum = in.CpuUsageMaximum
	out.Mem_usage_maximum = in.MemUsageMaximum
	out.Disk_usage_maximum = in.DiskUsageMaximum
	out.Net_usage_maximum = in.NetUsageMaximum
	out.Vm_MacAddress = in.Vm_MacAddress
	out.Vm_GuestDetailedData = in.GuestDetailedData
	out.Vm_ToolsStatus = in.ToolsStatus
	out.Vm_DomainName = in.DomainName
	out.DatacenterName = in.DatacenterName
	out.Production = int(in.Production)
	out.TypeWork = int(in.TypeWork)
	out.Vm_EsxHost = in.Vm_EsxHost
	out.Vm_Network = in.Vm_Network
	out.WorkStation = in.WorkStation
}

// TODO: clusterAssignMetrics helper consolidates duplicated metric assignments not shown in screenshots
