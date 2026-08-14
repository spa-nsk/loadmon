package grpcserver

import (
	"loadmon/hwconvert"
	pb "loadmon/protocol"

	oneview "github.com/spa-nsk/oneview2"
)

// ServerHardwareToMessage конвертирует oneview2.Server в pb.ServerHardware.
func ServerHardwareToMessage(in *oneview.Server, out *pb.ServerHardware) {
	hwconvert.ServerHardwareToMessage(in, out)
}

// MessageToServerHardware выполняет обратную конвертацию pb.ServerHardware в oneview2.Server.
func MessageToServerHardware(in *pb.ServerHardware, out *oneview.Server) {
	hwconvert.MessageToServerHardware(in, out)
}
