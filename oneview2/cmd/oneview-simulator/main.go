// Command oneview-simulator runs independent HPE OneView HTTP simulators for oneview2.
//
// Usage:
//
//	go run ./cmd/oneview-simulator -config simulator/simconfig.example.json
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spa-nsk/oneview2/simulator"
)

func main() {
	configPath := flag.String("config", "simulator/simconfig.example.json", "path to simulator JSON config")
	flag.Parse()

	cfg, err := simulator.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	var wg sync.WaitGroup
	for _, inst := range cfg.Instances {
		wg.Add(1)
		go func(inst simulator.InstanceConfig) {
			defer wg.Done()
			sim := simulator.New(inst)
			log.Printf("oneview-simulator: %s listening on http://%s (API %d, %d servers)",
				inst.Name, inst.Listen, inst.APIVersion, inst.ServerCount)
			if err := sim.ListenAndServe(); err != nil {
				log.Printf("oneview-simulator: %s stopped: %v", inst.Name, err)
			}
		}(inst)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("oneview-simulator: shutdown signal received")
}
