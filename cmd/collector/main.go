package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/observiq/observiq-collector/internal/version"
	"github.com/observiq/observiq-collector/pkg/collector"
	"github.com/spf13/pflag"
)

func main() {
	fmt.Println(version.Date(), version.Version(), version.GitHash())

	var configPath = pflag.String("config", "./config.yaml", "the collector config path")
	pflag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	collector := collector.New(*configPath, version.Version(), nil)
	if err := collector.Run(ctx); err != nil {
		log.Panicf("Collector failed to start: %s", err)
	}
	defer collector.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case status := <-collector.Status():
			if !status.Running {
				log.Panicf("Collector stopped running: %s", status.Err)
			}
		}
	}
}
