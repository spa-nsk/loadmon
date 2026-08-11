package grpcclient

import (
	"loadmon/models"
	pb "loadmon/protocol"

	"google.golang.org/protobuf/types/known/timestamppb"
)

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
}

// VMRecordToMessage конвертирует доменную модель models.VMRecord в gRPC-сообщение pb.VMRecord. При in == nil или out == nil ничего не делает.
func VMRecordToMessage(in *models.VMRecord, out *pb.VMRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMSprId = in.VMsprId
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
}

// MessageToVMRecord конвертирует gRPC-сообщение pb.VMRecord в доменную модель models.VMRecord. При in == nil или out == nil ничего не делает.
func MessageToVMRecord(in *pb.VMRecord, out *models.VMRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMsprId = in.VMSprId
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
}

// ClusterPerfToMessage конвертирует доменную модель models.ClusterPerf (метрика кластера для БД) в gRPC-сообщение pb.ClusterPerf. При in == nil или out == nil ничего не делает.
