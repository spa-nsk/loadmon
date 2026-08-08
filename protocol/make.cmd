@echo off
set PATH=c:\gopath\bin;%PATH%
"%LOCALAPPDATA%\protoc\bin\protoc.exe" --proto_path=protocol --go_out=protocol --go_opt=paths=source_relative --go-grpc_out=protocol --go-grpc_opt=paths=source_relative protocol/vm.proto protocol/db.proto
