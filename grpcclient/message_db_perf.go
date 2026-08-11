package grpcclient

import (
	"loadmon/models"
	pb "loadmon/protocol"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ClusterPerfToMessage(in *models.ClusterPerf, out *pb.ClusterPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.ClusterId = in.ClusterId                          //int64
	out.Start = timestamppb.New(in.Start)                 //time.Time
	out.CpuUsage = in.Cpu_usage                           //float64
	out.MemUsage = in.Mem_usage                           //float64
	out.DiskUsage = in.Disk_usage                         //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency   //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write                         //float64
	out.DiskRead = in.Disk_read                           //float64
	out.NetUsage = in.Net_usage                           //float64
	out.NetReceived = in.Net_received                     //float64
	out.NetTransmitted = in.Net_transmitted               //float64
}

// MessageToClusterPerf конвертирует gRPC-сообщение pb.ClusterPerf в доменную модель models.ClusterPerf. При in == nil или out == nil ничего не делает.
func MessageToClusterPerf(in *pb.ClusterPerf, out *models.ClusterPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.ClusterId = in.ClusterId                          //int64
	out.Start = in.Start.AsTime()                         //time.Time
	out.Cpu_usage = in.CpuUsage                           //float64
	out.Mem_usage = in.MemUsage                           //float64
	out.Disk_usage = in.DiskUsage                         //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency   //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite                         //float64
	out.Disk_read = in.DiskRead                           //float64
	out.Net_usage = in.NetUsage                           //float64
	out.Net_received = in.NetReceived                     //float64
	out.Net_transmitted = in.NetTransmitted               //float64

}

// HostPerfToMessage конвертирует доменную модель models.HostPerf в gRPC-сообщение pb.HostPerf. При in == nil или out == nil ничего не делает.
func HostPerfToMessage(in *models.HostPerf, out *pb.HostPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.HostId = in.HostId                                //int64
	out.Start = timestamppb.New(in.Start)                 //time.Time
	out.CpuUsage = in.Cpu_usage                           //float64
	out.MemUsage = in.Mem_usage                           //float64
	out.DiskUsage = in.Disk_usage                         //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency   //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write                         //float64
	out.DiskRead = in.Disk_read                           //float64
	out.NetUsage = in.Net_usage                           //float64
	out.NetReceived = in.Net_received                     //float64
	out.NetTransmitted = in.Net_transmitted               //float64

}

// MessageToHostPerf конвертирует gRPC-сообщение pb.HostPerf в доменную модель models.HostPerf. При in == nil или out == nil ничего не делает.
func MessageToHostPerf(in *pb.HostPerf, out *models.HostPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.HostId = in.HostId                                //int64
	out.Start = in.Start.AsTime()                         //time.Time
	out.Cpu_usage = in.CpuUsage                           //float64
	out.Mem_usage = in.MemUsage                           //float64
	out.Disk_usage = in.DiskUsage                         //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency   //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite                         //float64
	out.Disk_read = in.DiskRead                           //float64
	out.Net_usage = in.NetUsage                           //float64
	out.Net_received = in.NetReceived                     //float64
	out.Net_transmitted = in.NetTransmitted               //float64

}

// VMPerfRecordToMessage конвертирует доменную модель models.VMPerfRecord в gRPC-сообщение pb.VMPerfRecord. При in == nil или out == nil ничего не делает.
func VMPerfRecordToMessage(in *models.VMPerfRecord, out *pb.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.VMID = in.VMId                                    //int64
	out.Start = timestamppb.New(in.Start)                 //time.Time
	out.CpuUsage = in.Cpu_usage                           //float64
	out.MemUsage = in.Mem_usage                           //float64
	out.DiskUsage = in.Disk_usage                         //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency   //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write                         //float64
	out.DiskRead = in.Disk_read                           //float64
	out.NetUsage = in.Net_usage                           //float64
	out.NetReceived = in.Net_received                     //float64
	out.NetTransmitted = in.Net_transmitted               //float64

}

// MessageToVMPerfRecord конвертирует gRPC-сообщение pb.VMPerfRecord в доменную модель models.VMPerfRecord. При in == nil или out == nil ничего не делает.
func MessageToVMPerfRecord(in *pb.VMPerfRecord, out *models.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                        //int64
	out.VMId = in.VMID                                    //int64
	out.Start = in.Start.AsTime()                         //time.Time
	out.Cpu_usage = in.CpuUsage                           //float64
	out.Mem_usage = in.MemUsage                           //float64
	out.Disk_usage = in.DiskUsage                         //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency   //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite                         //float64
	out.Disk_read = in.DiskRead                           //float64
	out.Net_usage = in.NetUsage                           //float64
	out.Net_received = in.NetReceived                     //float64
	out.Net_transmitted = in.NetTransmitted               //float64

}
