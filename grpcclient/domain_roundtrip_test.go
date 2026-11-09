package grpcclient

import (
	"testing"

	"loadmon/grpcserver"
	"loadmon/internal/convaudit"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

// TestRoundTripDomainHost - round-trip "vsphere.HostInfo -> pb.Host/HostFull
// (сервер) -> vsphere.HostInfo (клиент)". Ловит поля домена, для которых нет
// места в сообщении (пробел контракта) или которые теряет конвертер. PerfCount=1,
// чтобы деление среднего на PerfCount на сервере было тождеством. Perf в
// краткое сообщение (pb.Host) по контракту не входит.
func TestRoundTripDomainHost(t *testing.T) {
	in := &vsphere.HostInfo{}
	convaudit.Fill(in)
	in.PerfCount = 1

	full := &pb.HostFull{}
	grpcserver.HostToMessageFull(in, full)
	outFull := &vsphere.HostInfo{}
	MessageToHostInfoFull(full, outFull)
	if bad := convaudit.Diff(in, outFull); len(bad) > 0 {
		t.Error(convaudit.Report("HostInfo -> HostFull -> HostInfo", bad))
	}

	short := &pb.Host{}
	grpcserver.HostToMessageShort(in, short)
	outShort := &vsphere.HostInfo{}
	MessageToHostInfo(short, outShort)
	if bad := convaudit.Unexpected(convaudit.Diff(in, outShort), []string{"Perf"}); len(bad) > 0 {
		t.Error(convaudit.Report("HostInfo -> Host -> HostInfo", bad))
	}
}

// TestRoundTripDomainVMPoint - то же для vsphere.VMPoint -> pb.VMPoint ->
// vsphere.VMPoint: проверяются поля, не зависящие от пересчёта агрегатов из
// Perf (агрегаты клиент считает сам, CopyAndReCalcVMPerformance).
func TestRoundTripDomainVMPoint(t *testing.T) {
	in := vsphere.VMPoint{}
	convaudit.Fill(&in)
	in.PerfCount = 1
	msg := &pb.VMPoint{}
	grpcserver.VMToMessageFull(in, msg)
	out := GetVMPointRPC(msg)
	CopyAndReCalcVMPerformance(&out, msg)

	allow := []string{"Perf"}
	for _, g := range []string{"CPUUsage", "MemUsage", "DiskUsage", "DiskRead", "DiskWrite", "NetUsage", "DiskTotalReadLatency", "DiskTotalWriteLatency"} {
		for _, s := range []string{"Maximum", "Minimum", "Average", "MaximumTime", "MinimumTime"} {
			allow = append(allow, g+s)
		}
	}
	if bad := convaudit.Unexpected(convaudit.Diff(&in, &out), allow); len(bad) > 0 {
		t.Error(convaudit.Report("VMPoint -> pb.VMPoint -> VMPoint", bad))
	}
}

// TestVMLatencyAggregatesRecalc - клиент пересчитывает по Perf агрегаты
// задержек диска (min/max/avg и время экстремумов) так же, как остальные.
func TestVMLatencyAggregatesRecalc(t *testing.T) {
	in := vsphere.VMPoint{}
	in.Perf = []*vsphere.VMPerf{
		{ID: 1, DiskTotalReadLatency: 4, DiskTotalWriteLatency: 10},
		{ID: 2, DiskTotalReadLatency: 8, DiskTotalWriteLatency: 2},
	}
	in.PerfCount = 2
	msg := &pb.VMPoint{}
	grpcserver.VMToMessageFull(in, msg)
	out := GetVMPointRPC(msg)
	CopyAndReCalcVMPerformance(&out, msg)
	if out.DiskTotalReadLatencyMaximum != 8 || out.DiskTotalReadLatencyMinimum != 4 || out.DiskTotalReadLatencyAverage != 6 {
		t.Errorf("read latency: max=%v min=%v avg=%v", out.DiskTotalReadLatencyMaximum, out.DiskTotalReadLatencyMinimum, out.DiskTotalReadLatencyAverage)
	}
	if out.DiskTotalWriteLatencyMaximum != 10 || out.DiskTotalWriteLatencyMinimum != 2 || out.DiskTotalWriteLatencyAverage != 6 {
		t.Errorf("write latency: max=%v min=%v avg=%v", out.DiskTotalWriteLatencyMaximum, out.DiskTotalWriteLatencyMinimum, out.DiskTotalWriteLatencyAverage)
	}
}
