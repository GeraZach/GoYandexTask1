package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	monitor()
}

func monitor() {
	const url = "http://srv.msk01.gigacorp.local/_stats"
	const interval = 30 * time.Second
	errorCount := 0

	for {
		time.Sleep(interval)

		// Получение данных
		resp, err := http.Get(url)
		if err != nil {
			errorCount++
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}
			continue
		}

		if resp.StatusCode != 200 {
			errorCount++
			resp.Body.Close()
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}
			continue
		}

		// Чтение данных
		scanner := bufio.NewScanner(resp.Body)
		if !scanner.Scan() {
			errorCount++
			resp.Body.Close()
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}
			continue
		}

		data := scanner.Text()
		resp.Body.Close()

		// Парсинг данных
		parts := strings.Split(strings.TrimSpace(data), ",")
		if len(parts) != 6 {
			errorCount++
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}
			continue
		}

		// Сброс счетчика ошибок
		errorCount = 0

		// Парсинг значений
		load, err1 := strconv.ParseFloat(parts[0], 64)
		totalMem, err2 := strconv.ParseUint(parts[1], 10, 64)
		usedMem, err3 := strconv.ParseUint(parts[2], 10, 64)
		totalDisk, err4 := strconv.ParseUint(parts[3], 10, 64)
		usedDisk, err5 := strconv.ParseUint(parts[4], 10, 64)
		netUsage, err6 := strconv.ParseUint(parts[5], 10, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
			continue
		}

		// Проверка порогов

		// 1. Load Average
		if load > 30 {
			fmt.Printf("Load Average is too high: %.2f\n", load)
		}

		// 2. Memory usage
		if totalMem > 0 && float64(usedMem)/float64(totalMem) > 0.8 {
			usagePercent := float64(usedMem) / float64(totalMem) * 100
			fmt.Printf("Memory usage too high: %.1f%%\n", usagePercent)
		}

		// 3. Disk space
		if totalDisk > 0 && float64(usedDisk)/float64(totalDisk) > 0.9 {
			freeMB := float64(totalDisk-usedDisk) / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %.1f Mb left\n", freeMB)
		}

		// 4. Network bandwidth (предполагаем 1 Гбит/с = 125000000 байт/с)
		const netBandwidth uint64 = 125000000
		if netBandwidth > 0 && float64(netUsage)/float64(netBandwidth) > 0.9 {
			freeMbits := float64(netBandwidth-netUsage) * 8 / (1024 * 1024)
			fmt.Printf("Network bandwidth usage high: %.1f Mbit/s available\n", freeMbits)
		}
	}
}
