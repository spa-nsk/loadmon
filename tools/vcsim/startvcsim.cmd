@echo off
cd /d %~dp0..\..
go run ./tools/vcsim -config tools/vcsim/conf/vcsim.json %*
