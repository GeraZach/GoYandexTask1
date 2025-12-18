package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL      = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval   = 30 * time.Second
	errorThreshold = 3

	loadThreshold    = 30.0
	memoryThreshold  = 0.8
	diskThreshold    = 0.9
	networkThreshold = 0.9
)

// Конфигурация, которая может быть загружена из файла или переменных окружения
type Config struct {
	ServerURL        string
	PollInterval     time.Duration
	ErrorThreshold   int
	LoadThreshold    float64
	MemoryThreshold  float64
	DiskThreshold    float64
	NetworkThreshold float64
	NetworkBandwidth uint64 // Предполагаемая пропускная способность
}

type ServerStats struct {
	LoadAverage      float64
	TotalMemory      uint64
	UsedMemory       uint64
	TotalDisk        uint64
	UsedDisk         uint64
	NetworkBandwidth uint64
	NetworkUsage     uint64
}

func main() {
	config := Config{
		ServerURL:        serverURL,
		PollInterval:     pollInterval,
		ErrorThreshold:   errorThreshold,
		LoadThreshold:    loadThreshold,
		MemoryThreshold:  memoryThreshold,
		DiskThreshold:    diskThreshold,
		NetworkThreshold: networkThreshold,
		NetworkBandwidth: 125000000, // 1 Гбит/с в байтах/с
	}

	monitorServer(config)
}

func monitorServer(config Config) {
	var errorCount int
	var consecutiveErrors int

	for {
		stats, err := fetchServerStats(config.ServerURL)
		if err != nil {
			errorCount++
			consecutiveErrors++

			if consecutiveErrors >= config.ErrorThreshold {
				fmt.Println("Unable to fetch server statistic")
				// Можно отправить алерт или сделать паузу перед повторной попыткой
				time.Sleep(config.PollInterval * 2)
				consecutiveErrors = 0
			}

			time.Sleep(config.PollInterval)
			continue
		}

		// Сбрасываем счетчики ошибок
		consecutiveErrors = 0
		errorCount = 0

		// Устанавливаем предполагаемую пропускную способность из конфигурации
		stats.NetworkBandwidth = config.NetworkBandwidth

		checkThresholds(stats, config)

		time.Sleep(config.PollInterval)
	}
}

func fetchServerStats(url string) (*ServerStats, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status not OK: %s", resp.Status)
	}

	// Проверяем Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no data in response")
	}

	data := strings.TrimSpace(scanner.Text())
	values := strings.Split(data, ",")

	if len(values) != 6 {
		return nil, fmt.Errorf("invalid data format: expected 6 values, got %d", len(values))
	}

	stats := &ServerStats{}

	// Парсим все значения с проверкой ошибок
	parsedValues := make([]uint64, 5)
	var load float64

	// Load Average
	load, err = strconv.ParseFloat(values[0], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load average: %w", err)
	}
	stats.LoadAverage = load

	// Остальные значения как uint64
	for i := 1; i < 6; i++ {
		val, err := strconv.ParseUint(values[i], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %d: %w", i, err)
		}
		parsedValues[i-1] = val
	}

	stats.TotalMemory = parsedValues[0]
	stats.UsedMemory = parsedValues[1]
	stats.TotalDisk = parsedValues[2]
	stats.UsedDisk = parsedValues[3]
	stats.NetworkUsage = parsedValues[4]

	return stats, nil
}

func checkThresholds(stats *ServerStats, config Config) {
	// Load Average
	if stats.LoadAverage > config.LoadThreshold {
		fmt.Printf("Load Average is too high: %.2f\n", stats.LoadAverage)
	}

	// Memory
	if stats.TotalMemory > 0 {
		memoryUsage := float64(stats.UsedMemory) / float64(stats.TotalMemory)
		if memoryUsage > config.MemoryThreshold {
			fmt.Printf("Memory usage too high: %.1f%%\n", memoryUsage*100)
		}
	}

	// Disk
	if stats.TotalDisk > 0 {
		diskUsage := float64(stats.UsedDisk) / float64(stats.TotalDisk)
		if diskUsage > config.DiskThreshold {
			freeDiskMB := float64(stats.TotalDisk-stats.UsedDisk) / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %.1f Mb left\n", freeDiskMB)
		}
	}

	// Network
	if stats.NetworkBandwidth > 0 {
		networkUsage := float64(stats.NetworkUsage) / float64(stats.NetworkBandwidth)
		if networkUsage > config.NetworkThreshold {
			freeBandwidthBits := float64(stats.NetworkBandwidth-stats.NetworkUsage) * 8
			freeBandwidthMbit := freeBandwidthBits / (1024 * 1024)
			fmt.Printf("Network bandwidth usage high: %.1f Mbit/s available\n", freeBandwidthMbit)
		}
	}
}

// Функция для логирования в файл (опционально)
func logToFile(message string) {
	f, err := os.OpenFile("server_monitor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)

	if _, err := f.WriteString(logMessage); err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}
}
