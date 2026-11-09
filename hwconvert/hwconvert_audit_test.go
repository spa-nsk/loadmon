package hwconvert

import (
	"testing"

	"loadmon/internal/convaudit"
	pb "loadmon/protocol"

	oneview "github.com/spa-nsk/oneview2"
)

// TestServerHardwareRoundTripAudit - round-trip oneview.Server -> pb.ServerHardware
// -> oneview.Server: отличия показывают поля, которые контракт
// (protocol/vm.proto, ServerHardware) не передаёт либо конвертер теряет.
// Исключений нет: любое новое поле oneview.Server, не переданное по gRPC,
// считается ошибкой (нужно добавить поле в ServerHardware и в конвертеры).
func TestServerHardwareRoundTripAudit(t *testing.T) {
	in := &oneview.Server{}
	convaudit.Fill(in)
	msg := &pb.ServerHardware{}
	ServerHardwareToMessage(in, msg)
	out := &oneview.Server{}
	MessageToServerHardware(msg, out)
	bad := convaudit.Diff(in, out)
	if len(bad) > 0 {
		t.Error(convaudit.Report("ServerHardware round-trip", bad))
	}
}
