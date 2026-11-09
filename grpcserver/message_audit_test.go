package grpcserver

import (
	"testing"

	"loadmon/internal/convaudit"
	"loadmon/models"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

// allowZero - поля результата конвертера, которые допустимо остаются нулевыми
// (нет аналога в источнике). Всё остальное нулевое считается потерянным при
// копировании полем.
type auditCase struct {
	name  string
	run   func() any // выполняет конвертацию и возвращает указатель на результат
	allow []string
}

func runAudit(t *testing.T, cases []auditCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := c.run()
			bad := convaudit.Unexpected(convaudit.ZeroFields(out), c.allow)
			if len(bad) > 0 {
				t.Error(convaudit.Report(c.name, bad))
			}
		})
	}
}

// clusterRecalcAllow - поля, обнуляемые самим vsphere.Cluster.CalcPerfomance (пересчёт из хостов в тесте без данных): минимумы, ID=индекс, CPUUtilization не считается.
var clusterRecalcAllow = []string{
	"CpuUsageMinimum", "CpuUsageMinimumTime", "MemUsageMinimum", "MemUsageMinimumTime",
	"DiskUsageMinimum", "DiskUsageMinimumTime", "DiskWriteMinimum", "DiskWriteMinimumTime",
	"DiskReadMinimum", "DiskReadMinimumTime", "NetUsageMinimum", "NetUsageMinimumTime",
	"DiskTotalReadLatencyMinimum", "DiskTotalReadLatencyMinimumTime", "DiskTotalWriteLatencyMinimum", "DiskTotalWriteLatencyMinimumTime",
}

// TestServerConvertersCopyAllFields - серверные конвертеры домен -> gRPC не
// должны терять поля сообщений.
func TestServerConvertersCopyAllFields(t *testing.T) {
	runAudit(t, []auditCase{
		{name: "VMToMessageShort", run: func() any {
			in := vsphere.VMPoint{}
			convaudit.Fill(&in)
			out := &pb.VMPointShort{}
			VMToMessageShort(in, out)
			return out
		}},
		{name: "VMToMessageFull", run: func() any {
			in := vsphere.VMPoint{}
			convaudit.Fill(&in)
			out := &pb.VMPoint{}
			VMToMessageFull(in, out)
			return out
		}},
		{name: "HostToMessageShort", run: func() any {
			in := &vsphere.HostInfo{}
			convaudit.Fill(in)
			out := &pb.Host{}
			HostToMessageShort(in, out)
			return out
		}},
		{name: "HostToMessageFull", run: func() any {
			in := &vsphere.HostInfo{}
			convaudit.Fill(in)
			out := &pb.HostFull{}
			HostToMessageFull(in, out)
			return out
		}},
		{name: "ClusterStorageToMessage", run: func() any {
			in := &vsphere.Datastore{}
			convaudit.Fill(in)
			out := &pb.Datastore{}
			ClusterStorageToMessage(in, out)
			return out
		}},
		{name: "ClusterMetricToMessage", run: func() any {
			in := &vsphere.CLPerf{}
			convaudit.Fill(in)
			out := &pb.CLPerf{}
			ClusterMetricToMessage(in, out)
			return out
		}},
		{name: "ServerSprRecodToMessage", allow: []string{"DPCAddress", "DPCOffice"}, run: func() any {
			in := &models.ServerSprRecords{}
			convaudit.Fill(in)
			out := &pb.ServerSprRecords{}
			ServerSprRecodToMessage(in, out)
			return out
		}},
		{name: "VMSprRecodToMessage", run: func() any {
			in := &models.VMSprRecord{}
			convaudit.Fill(in)
			out := &pb.VMSprRecord{}
			VMSprRecodToMessage(in, out)
			return out
		}},
		{name: "ClusterSprRecordsToMessage", allow: []string{"Production"}, run: func() any {
			in := &models.ClusterSprRecords{}
			convaudit.Fill(in)
			out := &pb.ClusterSprRecord{}
			ClusterSprRecordsToMessage(in, out)
			return out
		}},
		{name: "ClusterRecordToMessage", allow: []string{"TypeWork"}, run: func() any {
			in := &models.ClusterRecord{}
			convaudit.Fill(in)
			out := &pb.ClusterRecord{}
			ClusterRecordToMessage(in, out)
			return out
		}},
		{name: "HostRecordToMessage", run: func() any {
			in := &models.HostRecord{}
			convaudit.Fill(in)
			out := &pb.HostRecord{}
			HostRecordToMessage(in, out)
			return out
		}},
		{name: "VMRecordToMessage", run: func() any {
			in := &models.VMRecord{}
			convaudit.Fill(in)
			out := &pb.VMRecord{}
			VMRecordToMessage(in, out)
			return out
		}},
		{name: "ClusterPerfToMessage", run: func() any {
			in := &models.ClusterPerf{}
			convaudit.Fill(in)
			out := &pb.ClusterPerf{}
			ClusterPerfToMessage(in, out)
			return out
		}},
		{name: "HostPerfToMessage", run: func() any {
			in := &models.HostPerf{}
			convaudit.Fill(in)
			out := &pb.HostPerf{}
			HostPerfToMessage(in, out)
			return out
		}},
		{name: "VMPerfRecordToMessage", run: func() any {
			in := &models.VMPerfRecord{}
			convaudit.Fill(in)
			out := &pb.VMPerfRecord{}
			VMPerfRecordToMessage(in, out)
			return out
		}},
		{name: "ClusterToMessage", allow: clusterRecalcAllow, run: func() any {
			in := vsphere.Cluster{}
			convaudit.Fill(&in)
			out := &pb.Cluster{}
			ClusterToMessage(in, out)
			return out
		}},
		{name: "ClusterToMessageMedium", allow: clusterRecalcAllow, run: func() any {
			in := vsphere.Cluster{}
			convaudit.Fill(&in)
			out := &pb.Cluster{}
			ClusterToMessageMedium(in, out)
			return out
		}},
		{name: "ClusterToMessageFull", allow: append([]string{"Perf[].Id", "Perf[].CpuUtilization"}, clusterRecalcAllow...), run: func() any {
			in := vsphere.Cluster{}
			convaudit.Fill(&in)
			out := &pb.ClusterFull{}
			ClusterToMessageFull(in, out)
			return out
		}},
		{name: "RecalcHostToMessageShort", run: func() any {
			in := &vsphere.HostInfo{}
			convaudit.Fill(in)
			out := &pb.Host{}
			RecalcHostToMessageShort(in, out)
			return out
		}},
		{name: "RecalcHostToMessageFull", run: func() any {
			in := &vsphere.HostInfo{}
			convaudit.Fill(in)
			out := &pb.HostFull{}
			RecalcHostToMessageFull(in, out)
			return out
		}},
		{name: "CopyHost", run: func() any {
			in := &pb.Host{}
			convaudit.Fill(in)
			return CopyHost(in)
		}},
		{name: "CopyPbHostFull", run: func() any {
			in := &pb.HostFull{}
			convaudit.Fill(in)
			return CopyPbHostFull(in)
		}},
		{name: "CopyClusterFull", run: func() any {
			in := &pb.ClusterFull{}
			convaudit.Fill(in)
			return CopyClusterFull(in)
		}},
		{name: "CopyClusterMedium", run: func() any {
			in := &pb.Cluster{}
			convaudit.Fill(in)
			return CopyClusterMedium(in)
		}},
		{name: "CopyClusterStorages", run: func() any {
			in := &pb.Datastore{}
			convaudit.Fill(in)
			return CopyClusterStorages(in)
		}},
		{name: "CopyDatastore", run: func() any {
			in := &pb.Datastore{}
			convaudit.Fill(in)
			return CopyDatastore(in)
		}},
		{name: "CopyCLPerf", run: func() any {
			in := &pb.CLPerf{}
			convaudit.Fill(in)
			return CopyCLPerf(in)
		}},
	})
}
