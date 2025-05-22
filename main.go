package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
	"context"
	"os/signal"
	"syscall"
	"sync"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/ads1x15"
	"periph.io/x/host/v3"
)

type Config struct {
	Point string
	System0 string
	System1 string
}

const (
	configFile = "pg.properties"
	metricsFile = "metrics.txt"
	listenAddr = "0.0.0.0:8000"
)

func main(){
	// init process
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	config := loadConf()
	writeMetricsfile(config.Point, config.System0, config.System1, 0, 0)

	// start measuring
	var wg sync.WaitGroup
	ctx, cancelMeasure := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelMeasure()
	wg.Add(1)
	go measurePeriod(debug, config, ctx, &wg)

	// setup http server
	http.HandleFunc("/metrics", exporter)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  20 * time.Second,
	}
	if *debug {
		log.Printf("Starting server on %s\n", listenAddr)
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
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

func exporter(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(metricsFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", metricsFile, err)
		http.Error(w, "Failed to read metrics file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err = w.Write(content)
	if err != nil {
		log.Fatalf("Failed to write response: %v", err)
		return
	}
}

func measurePeriod(debug *bool, config Config, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <- ctx.Done():
			if *debug {
				log.Println("Received interrupt signal, stopping measurement.")
			}
			return
		default:
			t_start := time.Now()
			const (
				Differential0_1 ads1x15.Channel = 0
				Differential2_3 ads1x15.Channel = 3
			)
			amps0, err := getAmps(Differential0_1)
			if err != nil {
				log.Fatalf("Failed to get amps for system0")
			}
			amps1, err := getAmps(Differential2_3)
			if err != nil {
				log.Fatalf("Failed to get amps for system1")
			}

			err = writeMetricsfile(config.Point, config.System0, config.System1, amps0, amps1)
			if err != nil {
				log.Fatalf("Failed to write metrics file: %v", err)
			}
			if *debug {
				log.Println("Metrics file written successfully.")
			}

			t_end := time.Now()
			if *debug {
				fmt.Println("------------DEBUG------------")
				fmt.Println()
				fmt.Printf("System-%s\t: %f A\nSystem-%s\t: %f A\n", config.System0, amps0, config.System1, amps1)
				fmt.Printf("Time taken\t: %v\n", t_end.Sub(t_start))
				fmt.Printf("Point\t\t: %s\n", config.Point)
				fmt.Println()
				fmt.Println("-----------------------------")
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func loadConf() Config {
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		log.Println(configFile, "file not found, creating a new one.")
		file, err := os.Create(configFile)
		if err != nil {
			log.Fatalf("Failed to create %s : %v", configFile, err)
		}
		file.WriteString("point=point\n")
		file.WriteString("system0=0\n")
		file.WriteString("system1=1\n")
		file.Close()
	}
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open %s: %v", configFile, err)
	}
	defer file.Close()

	var config Config
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "point") {
			config.Point = strings.TrimSpace(strings.Split(line, "=")[1])
		} else if strings.HasPrefix(line, "system0") {
			config.System0 = strings.TrimSpace(strings.Split(line, "=")[1])
		} else if strings.HasPrefix(line, "system1") {
			config.System1 = strings.TrimSpace(strings.Split(line, "=")[1])
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read %s: %v", configFile, err)
	}
	if config.Point == "" || config.System0 == "" || config.System1 == "" {
		log.Fatalf("Missing required properties in %s", configFile)
	}
	return config
}

func getAmps(channel ads1x15.Channel) (float64, error) {
	const (
		numSmaples = 500
		gainVoltageMax = physic.ElectricPotential(4096) * physic.MilliVolt
		freq = physic.Frequency(860 * physic.Hertz)
	)

	if _, err := host.Init(); err != nil {
		log.Fatalf("Failed to init periph: %v", err)
		return 0, err
	}

	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("Failed to open I2C bus: %v", err)
		return 0, err
	}
	defer bus.Close()

	adc, err := ads1x15.NewADS1115(bus, &ads1x15.DefaultOpts)
	if err != nil {
		log.Fatalf("Failed to create ADS1115: %v", err)
		return 0, err
	}
	pin, err := adc.PinForChannel(channel, gainVoltageMax, freq, 1)
	if err != nil {
		log.Fatalf("Failed to get pin for channel: %v", err)
		return 0, err
	}

	var sumI, sqI float64
	for i := 0; i < numSmaples; i++ {
		v, err := pin.Read()
		if err != nil{
			log.Fatalf("Failed to read from pin: %v", err)
		}
		voltage := float64(v.V)
		voltage /= float64(physic.MilliVolt)
		sumI += voltage
		sqI += voltage * voltage
	}
	rmsI := math.Sqrt(sqI / float64(numSmaples))
	eva := math.Round(rmsI * 2.0 * 1) / 100
	return eva, nil
}

func writeMetricsfile(point string, system0 string, system1 string, amps0 float64, amps1 float64) error {
	file, err := os.OpenFile(metricsFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open %s: %v", metricsFile, err)
		return err
	}
	defer file.Close()

	rawMessage := 
`# HELP system_value Fixed system value
# TYPE ampere_value gauge
ampere_value{system="%s", point="%s"} %.2f
ampere_value{system="%s", point="%s"} %.2f
`
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(fmt.Sprintf(rawMessage, system0, point, amps0, system1, point, amps1))
	if err != nil {
		log.Fatalf("Failed to write to %s: %v", metricsFile, err)
		return err
	}
	err = writer.Flush()
	if err != nil {
		log.Fatalf("Failed to flush writer: %v", err)
		return err
	}
	return nil
}

