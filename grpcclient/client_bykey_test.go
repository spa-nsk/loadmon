package grpcclient

import (
	"testing"

	"loadmon/grpcserver"
	"loadmon/internal/convaudit"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

// TestVMPointShortRoundTrip проверяет, что серверный VMToMessageShort и
// клиентский GetVMPointShortRPC не теряют поля, входящие в укороченное
// сообщение (PerfCount=1, чтобы деление среднего на PerfCount было тождеством).
// Allow-list - поля VMPoint, которых в VMPointShort нет по контракту (ряд Perf и списки).
func TestVMPointShortRoundTrip(t *testing.T) {
	var orig vsphere.VMPoint
	convaudit.Fill(&orig)
	orig.PerfCount = 1
	var msg pb.VMPointShort
	grpcserver.VMToMessageShort(orig, &msg)
	back := GetVMPointShortRPC(&msg)

	// По контракту в VMPointShort нет только временного ряда и списков файлов/дисков/хранилищ.
	allow := []string{"Perf", "Files", "Disk", "UsageDatastore"}
	bad := convaudit.Unexpected(convaudit.Diff(orig, back), allow)
	if len(bad) > 0 {
		t.Error(convaudit.Report("VMPoint short (потеряно при round-trip)", bad))
	}
	if v := GetVMPointShortRPC(nil); v.Name != "" || v.ID != 0 {
		t.Errorf("nil должен давать нулевую ВМ, получено %+v", v)
	}
}

// TestRPCHelpersSkipNil проверяет, что конвертеры списков пропускают nil и
// корректно обрабатывают пустой ввод.
func TestRPCHelpersSkipNil(t *testing.T) {
	if n := len(VMFilesRPC([]*pb.VMFile{nil, {Name: "a.vmx", Size: 1}})); n != 1 {
		t.Errorf("VMFilesRPC: ожидался 1 элемент, получено %d", n)
	}
	if n := len(VMDisksRPC([]*pb.GuestDiskInfo{nil, {DiskPath: "C:\\"}})); n != 1 {
		t.Errorf("VMDisksRPC: ожидался 1 элемент, получено %d", n)
	}
	if n := len(VMUsageOnDatastoreRPC([]*pb.VMUsageOnDatastore{nil, {Datastore: "ds"}})); n != 1 {
		t.Errorf("VMUsageOnDatastoreRPC: ожидался 1 элемент, получено %d", n)
	}
	if n := len(HSPerfRPC([]*pb.HSPerf{nil, {}})); n != 1 {
		t.Errorf("HSPerfRPC: ожидался 1 элемент, получено %d", n)
	}
	if n := len(CLPerfRPC([]*pb.CLPerf{nil, {}})); n != 1 {
		t.Errorf("CLPerfRPC: ожидался 1 элемент, получено %d", n)
	}
	if VMFilesRPC(nil) == nil || HSPerfRPC(nil) == nil {
		t.Error("пустой ввод должен давать пустой (не nil) срез")
	}
}
