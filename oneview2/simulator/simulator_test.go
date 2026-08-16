package simulator_test

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	oneview "github.com/spa-nsk/oneview2"
	"github.com/spa-nsk/oneview2/simulator"
)

func TestSimulatorsCollect600Servers(t *testing.T) {
	gdPort := freePort(t)
	ovPort := freePort(t)

	gd := simulator.New(simulator.InstanceConfig{
		Name: "gd", Listen: "127.0.0.1:" + gdPort, APIVersion: 300, ServerCount: 300,
		Username: "Administrator", Password: "admin",
	})
	ov := simulator.New(simulator.InstanceConfig{
		Name: "ov", Listen: "127.0.0.1:" + ovPort, APIVersion: 7400, ServerCount: 300,
		Username: "Administrator", Password: "admin",
	})

	go func() { _ = gd.ListenAndServe() }()
	go func() { _ = ov.ListenAndServe() }()
	waitHTTP(t, "http://127.0.0.1:"+gdPort+"/rest/version")
	waitHTTP(t, "http://127.0.0.1:"+ovPort+"/rest/version")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	servers, err := oneview.CollectServers(ctx, []oneview.Config{
		{Endpoint: "http://127.0.0.1:" + gdPort, Username: "Administrator", Password: "admin", APIVersion: 300},
		{Endpoint: "http://127.0.0.1:" + ovPort, Username: "Administrator", Password: "admin", APIVersion: 7400, InsecureTLS: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 600 {
		t.Fatalf("want 600 servers, got %d", len(servers))
	}

	withDC, withRack, withMem, withDisk := 0, 0, 0, 0
	for i := range servers {
		s := &servers[i]
		if s.Datacenter.Name != "" {
			withDC++
		}
		if s.Rack.Name != "" {
			withRack++
		}
		if len(s.Memory) > 0 {
			withMem++
		}
		if len(s.Controllers) > 0 && len(s.Controllers[0].Drives) > 0 {
			withDisk++
		}
	}
	if withDC < 500 || withRack < 500 {
		t.Fatalf("location coverage too low: dc=%d rack=%d", withDC, withRack)
	}
	if withMem < 500 || withDisk < 500 {
		t.Fatalf("inventory coverage too low: mem=%d disk=%d", withMem, withDisk)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
}

func waitHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", url)
}

func TestGenerateInventoryCount(t *testing.T) {
	inv := simulator.GenerateInventory(simulator.InstanceConfig{APIVersion: 7400, ServerCount: 300})
	if len(inv.ServerOrder) != 300 {
		t.Fatalf("servers: %d", len(inv.ServerOrder))
	}
	if len(inv.Datacenters) != 5 {
		t.Fatalf("datacenters: %d", len(inv.Datacenters))
	}
}
