package grpcclient

import (
	"testing"

	"loadmon/grpcserver"
	"loadmon/internal/convaudit"
	"loadmon/models"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

// rtCase - round-trip "домен -> gRPC (серверный конвертер) -> домен
// (клиентский конвертер)". Отличия между исходным и полученным объектами - поля,
// потерянные одним из конвертеров или не имеющие места в сообщении.
type rtCase struct {
	name  string
	run   func() (orig, back any)
	allow []string // допустимые отличия (поля без аналога в контракте)
}

func runRoundTrip(t *testing.T, cases []rtCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			orig, back := c.run()
			bad := convaudit.Unexpected(convaudit.Diff(orig, back), c.allow)
			if len(bad) > 0 {
				t.Error(convaudit.Report(c.name+" (потеряно при round-trip)", bad))
			}
		})
	}
}

func TestRoundTripDBRecords(t *testing.T) {
	runRoundTrip(t, []rtCase{
		{name: "ServerSprRecords", run: func() (any, any) {
			in := &models.ServerSprRecords{}
			convaudit.Fill(in)
			msg := &pb.ServerSprRecords{}
			grpcserver.ServerSprRecodToMessage(in, msg)
			out := &models.ServerSprRecords{}
			MessageToServerSprRecod(msg, out)
			return in, out
		}},
		{name: "VMSprRecord", run: func() (any, any) {
			in := &models.VMSprRecord{}
			convaudit.Fill(in)
			msg := &pb.VMSprRecord{}
			grpcserver.VMSprRecodToMessage(in, msg)
			out := &models.VMSprRecord{}
			MessageToVMSprRecod(msg, out)
			return in, out
		}},
		{name: "ClusterSprRecords", run: func() (any, any) {
			in := &models.ClusterSprRecords{}
			convaudit.Fill(in)
			msg := &pb.ClusterSprRecord{}
			grpcserver.ClusterSprRecordsToMessage(in, msg)
			out := &models.ClusterSprRecords{}
			MessageToClusterSprRecords(msg, out)
			return in, out
		}},
		{name: "ClusterRecord(server->client)", run: func() (any, any) {
			in := &models.ClusterRecord{}
			convaudit.Fill(in)
			msg := &pb.ClusterRecord{}
			grpcserver.ClusterRecordToMessage(in, msg)
			out := &models.ClusterRecord{}
			MessageToClusterRecord(msg, out)
			return in, out
		}},
		{name: "ClusterRecord(client->server)", run: func() (any, any) {
			in := &models.ClusterRecord{}
			convaudit.Fill(in)
			msg := &pb.ClusterRecord{}
			ClusterRecordToMessage(in, msg)
			out := &models.ClusterRecord{}
			grpcserver.MessageToClusterRecord(msg, out)
			return in, out
		}},
		{name: "HostRecord(server->client)", run: func() (any, any) {
			in := &models.HostRecord{}
			convaudit.Fill(in)
			msg := &pb.HostRecord{}
			grpcserver.HostRecordToMessage(in, msg)
			out := &models.HostRecord{}
			MessageToHostRecord(msg, out)
			return in, out
		}},
		{name: "HostRecord(client->server)", run: func() (any, any) {
			in := &models.HostRecord{}
			convaudit.Fill(in)
			msg := &pb.HostRecord{}
			HostRecordToMessage(in, msg)
			out := &models.HostRecord{}
			grpcserver.MessageToHostRecord(msg, out)
			return in, out
		}},
		{name: "VMRecord(server->client)", run: func() (any, any) {
			in := &models.VMRecord{}
			convaudit.Fill(in)
			msg := &pb.VMRecord{}
			grpcserver.VMRecordToMessage(in, msg)
			out := &models.VMRecord{}
			MessageToVMRecord(msg, out)
			return in, out
		}},
		{name: "VMRecord(client->server)", run: func() (any, any) {
			in := &models.VMRecord{}
			convaudit.Fill(in)
			msg := &pb.VMRecord{}
			VMRecordToMessage(in, msg)
			out := &models.VMRecord{}
			grpcserver.MessageToVMRecord(msg, out)
			return in, out
		}},
		{name: "ClusterPerf", run: func() (any, any) {
			in := &models.ClusterPerf{}
			convaudit.Fill(in)
			msg := &pb.ClusterPerf{}
			grpcserver.ClusterPerfToMessage(in, msg)
			out := &models.ClusterPerf{}
			MessageToClusterPerf(msg, out)
			return in, out
		}},
		{name: "HostPerf", run: func() (any, any) {
			in := &models.HostPerf{}
			convaudit.Fill(in)
			msg := &pb.HostPerf{}
			grpcserver.HostPerfToMessage(in, msg)
			out := &models.HostPerf{}
			MessageToHostPerf(msg, out)
			return in, out
		}},
		{name: "VMPerfRecord", run: func() (any, any) {
			in := &models.VMPerfRecord{}
			convaudit.Fill(in)
			msg := &pb.VMPerfRecord{}
			grpcserver.VMPerfRecordToMessage(in, msg)
			out := &models.VMPerfRecord{}
			MessageToVMPerfRecord(msg, out)
			return in, out
		}},
	})
}

func TestRoundTripVsphere(t *testing.T) {
	runRoundTrip(t, []rtCase{
		{name: "HSPerf", run: func() (any, any) {
			in := &vsphere.HSPerf{}
			convaudit.Fill(in)
			out := &vsphere.HSPerf{}
			MessageToHostMetric(grpcserver.HSPerfToMessage(in), out)
			return in, out
		}},
		{name: "CLPerf", run: func() (any, any) {
			in := &vsphere.CLPerf{}
			convaudit.Fill(in)
			msg := &pb.CLPerf{}
			grpcserver.ClusterMetricToMessage(in, msg)
			out := &vsphere.CLPerf{}
			MessageToClusterMetric(msg, out)
			return in, out
		}},
	})
}

// pbCase - round-trip "gRPC-сообщение -> домен (клиентский конвертер) ->
// gRPC-сообщение (серверный конвертер)". Отличия - поля сообщения, потерянные
// одним из двух конвертеров.
type pbCase struct {
	name  string
	run   func() (orig, back any)
	allow []string
}

func runPbRoundTrip(t *testing.T, cases []pbCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			orig, back := c.run()
			bad := convaudit.Unexpected(convaudit.Diff(orig, back), c.allow)
			if len(bad) > 0 {
				t.Error(convaudit.Report(c.name+" (потеряно при round-trip)", bad))
			}
		})
	}
}

// Известные и осознанные расхождения round-trip.
//
// hostAvgAllow - серверный HostToMessage* делит накопленную сумму на PerfCount
// (hs.CPUUsageAverage - сумма), а клиент кладёт полученное среднее в то же поле
// как есть, поэтому "среднее" не совпадает с исходным (семантика поля).
//
// vmAggAllow - агрегаты ВМ (min/max/avg и времена) клиент пересчитывает из Perf
// (CopyAndReCalcVMPerformance), а не берёт из сообщения; Perf в тесте синтетический.
var hostAvgAllow = []string{"CpuUsageAverage", "MemUsageAverage", "DiskUsageAverage", "DiskReadAverage", "DiskWriteAverage", "NetUsageAverage", "DiskTotalReadLatencyAverage", "DiskTotalWriteLatencyAverage"}

func vmAggAllow() []string {
	res := []string{"PerfCount"}
	for _, g := range []string{"CpuUsage", "MemUsage", "DiskUsage", "DiskRead", "DiskWrite", "NetUsage", "DiskTotalReadLatency", "DiskTotalWriteLatency"} {
		for _, s := range []string{"Maximum", "Minimum", "Average", "MaximumTime", "MinimumTime"} {
			res = append(res, g+s)
		}
	}
	return res
}

func TestRoundTripPbMessages(t *testing.T) {
	runPbRoundTrip(t, []pbCase{
		{name: "Datastore", allow: []string{"VmKeys.Count"}, run: func() (any, any) {
			in := &pb.Datastore{}
			convaudit.Fill(in)
			dom := &vsphere.Datastore{}
			MessageToClusterStorages(in, dom)
			out := &pb.Datastore{}
			grpcserver.ClusterStorageToMessage(dom, out)
			return in, out
		}},
		{name: "Host", allow: hostAvgAllow, run: func() (any, any) {
			in := &pb.Host{}
			convaudit.Fill(in)
			dom := &vsphere.HostInfo{}
			MessageToHostInfo(in, dom)
			out := &pb.Host{}
			grpcserver.HostToMessageShort(dom, out)
			return in, out
		}},
		{name: "HostFull", allow: hostAvgAllow, run: func() (any, any) {
			in := &pb.HostFull{}
			convaudit.Fill(in)
			dom := &vsphere.HostInfo{}
			MessageToHostInfoFull(in, dom)
			out := &pb.HostFull{}
			grpcserver.HostToMessageFull(dom, out)
			return in, out
		}},
		{name: "VMPoint", allow: vmAggAllow(), run: func() (any, any) {
			in := &pb.VMPoint{}
			convaudit.Fill(in)
			dom := GetVMPointRPC(in)
			CopyAndReCalcVMPerformance(&dom, in)
			out := &pb.VMPoint{}
			grpcserver.VMToMessageFull(dom, out)
			return in, out
		}},
		{name: "Cluster(Medium)", run: func() (any, any) {
			in := &pb.Cluster{}
			convaudit.Fill(in)
			dom := &vsphere.Cluster{}
			MessageToClusterMedium(in, dom)
			out := &pb.Cluster{}
			grpcserver.ClusterToMessageMedium(*dom, out)
			return in, out
		}},
		{name: "ClusterFull", allow: []string{"Storages[].VmKeys.Count"}, run: func() (any, any) {
			in := &pb.ClusterFull{}
			convaudit.Fill(in)
			dom := &vsphere.Cluster{}
			MessageToClusterFull(in, dom)
			out := &pb.ClusterFull{}
			grpcserver.ClusterToMessageFull(*dom, out)
			return in, out
		}},
	})
}
