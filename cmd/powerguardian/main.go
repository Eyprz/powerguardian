package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"powerguardian/internal/config"
	"powerguardian/internal/exporter"
	"powerguardian/internal/measure"
)

const (
	configFile  = "pg.properties"
	metricsFile = "metrics.txt"
	listenAddr  = "0.0.0.0:8000"
)

func main() {
	// init process
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	config := config.LoadConf(configFile)
	measure.WriteMetricsfile(config.Point, config.System0, config.System1, 0, 0, metricsFile)

	// start measuring
	var wg sync.WaitGroup
	ctx, cancelMeasure := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelMeasure()
	wg.Add(1)
	go measure.MeasurePeriod(debug, config, ctx, &wg, metricsFile)

	// setup http server
	if *debug {
		log.Printf("Starting server on %s\n", listenAddr)
	}
	server := exporter.HttpServer(config, metricsFile, listenAddr)

	<-ctx.Done()
	log.Printf("Received signal. \nShutting down PowerGuardian...\n")

	// stop musurement
	cancelMeasure()
	if *debug {
		log.Println("Stopping measurement...")
	}
	wg.Wait()
	if *debug {
		log.Println("Measurement stopped.")
	}

	// shutdown http server
	httpCtx, cancelHttp := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelHttp()

	if err := server.Shutdown(httpCtx); err != nil {
		log.Fatalf("Failed to http shutdown server: %v", err)
	}
	log.Println("Http server shut down gracefully.")

	// wait for measurement to finish
	if *debug {
		log.Println("Waiting for measurement to finish...")
	}
}
