@echo off
cd /d %~dp0..\..
go run ./tools/seeddata -config server_web_new/conf/app2.json %*
