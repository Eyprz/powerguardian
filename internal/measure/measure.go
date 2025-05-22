package measure

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/ads1x15"
	"periph.io/x/host/v3"
	"powerguardian/internal/config"
)

func MeasurePeriod(debug *bool, config config.Config, ctx context.Context, wg *sync.WaitGroup, metricsFile string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
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

			err = WriteMetricsfile(config.Point, config.System0, config.System1, amps0, amps1, metricsFile)
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

func WriteMetricsfile(point string, system0 string, system1 string, amps0 float64, amps1 float64, metricsFile string) error {
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

func getAmps(channel ads1x15.Channel) (float64, error) {
	const (
		numSmaples     = 500
		gainVoltageMax = physic.ElectricPotential(4096) * physic.MilliVolt
		freq           = physic.Frequency(860 * physic.Hertz)
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
		if err != nil {
			log.Fatalf("Failed to read from pin: %v", err)
		}
		voltage := float64(v.V)
		voltage /= float64(physic.MilliVolt)
		sumI += voltage
		sqI += voltage * voltage
	}
	rmsI := math.Sqrt(sqI / float64(numSmaples))
	eva := math.Round(rmsI*2.0*1) / 100
	return eva, nil
}
