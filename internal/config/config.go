package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Config struct {
	Point   string
	System0 string
	System1 string
}

func LoadConf(configFile string) Config {
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
