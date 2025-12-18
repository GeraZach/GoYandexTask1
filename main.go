package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL      = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval   = 30 * time.Second
	errorThreshold = 3
)

func main() {
	monitor()
}

func monitor() {
	errorCount := 0
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		time.Sleep(pollInterval)

		// Получаем статистику
		resp, err := client.Get(serverURL)
		if err != nil {
			handleError(&errorCount)
			continue
		}

		// Проверяем статус
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			handleError(&errorCount)
			continue
		}

		// Читаем данные
		scanner := bufio.NewScanner(resp.Body)
		if !scanner.Scan() {
			resp.Body.Close()
			handleError(&errorCount)
			continue
		}

		data := strings.TrimSpace(scanner.Text())
		resp.Body.Close()

		// Парсим данные
		parts := strings.Split(data, ",")
		if len(parts) != 6 {
			handleError(&errorCount)
			continue
		}

		// Сбрасываем счетчик ошибок
		errorCount = 0

		// Парсим все значения
		load, err1 := strconv.ParseFloat(parts[0], 64)
		totalMem, err2 := strconv.ParseUint(parts[1], 10, 64)
		usedMem, err3 := strconv.ParseUint(parts[2], 10, 64)
		totalDisk, err4 := strconv.ParseUint(parts[3], 10, 64)
		usedDisk, err5 := strconv.ParseUint(parts[4], 10, 64)
		netUsage, err6 := strconv.ParseUint(parts[5], 10, 64)

		// Если ошибка парсинга, пропускаем
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
			continue
		}

		// Проверяем пороги
		checkThresholds(load, totalMem, usedMem, totalDisk, usedDisk, netUsage)
	}
}

func handleError(errorCount *int) {
	*errorCount++
	if *errorCount >= errorThreshold {
		fmt.Println("Unable to fetch server statistic")
		*errorCount = 0 // Сбрасываем после вывода
	}
}

func checkThresholds(load float64, totalMem, usedMem, totalDisk, usedDisk, netUsage uint64) {
	// 1. Load Average (> 30)
	if load > 30 {
		fmt.Printf("Load Average is too high: %.2f\n", load)
	}

	// 2. Memory usage (> 80%)
	if totalMem > 0 {
		memoryUsage := float64(usedMem) / float64(totalMem)
		if memoryUsage > 0.8 {
			fmt.Printf("Memory usage too high: %.1f%%\n", memoryUsage*100)
		}
	}

	// 3. Disk space (> 90%)
	if totalDisk > 0 {
		diskUsage := float64(usedDisk) / float64(totalDisk)
		if diskUsage > 0.9 {
			freeMB := float64(totalDisk-usedDisk) / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %.1f Mb left\n", freeMB)
		}
	}

	// 4. Network bandwidth (> 90%)
	// Предполагаем 1 Гбит/с = 125000000 байт/с
	const networkBandwidth uint64 = 125000000
	if networkBandwidth > 0 {
		networkUsage := float64(netUsage) / float64(networkBandwidth)
		if networkUsage > 0.9 {
			freeMbits := float64(networkBandwidth-netUsage) * 8 / (1024 * 1024)
			fmt.Printf("Network bandwidth usage high: %.1f Mbit/s available\n", freeMbits)
		}
	}
}
