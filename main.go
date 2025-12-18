package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	errors := 0

	time.Sleep(30 * time.Second)

	for {
		resp, err := http.Get("http://srv.msk01.gigacorp.local/_stats")
		if err != nil {
			errors++
			if errors >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errors = 0
			}
			time.Sleep(30 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			errors++
			if errors >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errors = 0
			}
			time.Sleep(30 * time.Second)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			errors++
			if errors >= 3 {
				fmt.Println("Unable to fetch server statistic")
				errors = 0
			}
			time.Sleep(30 * time.Second)
			continue
		}

		errors = 0

		parts := strings.Split(strings.TrimSpace(string(data)), ",")
		if len(parts) != 6 {
			time.Sleep(30 * time.Second)
			continue
		}

		load, _ := strconv.ParseFloat(parts[0], 64)
		totalMem, _ := strconv.ParseUint(parts[1], 10, 64)
		usedMem, _ := strconv.ParseUint(parts[2], 10, 64)
		totalDisk, _ := strconv.ParseUint(parts[3], 10, 64)
		usedDisk, _ := strconv.ParseUint(parts[4], 10, 64)
		netUsage, _ := strconv.ParseUint(parts[5], 10, 64)

		// Проверки
		if load > 30 {
			fmt.Printf("Load Average is too high: %.2f\n", load)
		}

		if totalMem > 0 {
			if usedMem*10000/totalMem > 8000 { // > 80%
				p := float64(usedMem) / float64(totalMem) * 100
				fmt.Printf("Memory usage too high: %.1f%%\n", p)
			}
		}

		if totalDisk > 0 {
			if usedDisk*10000/totalDisk > 9000 { // > 90%
				free := float64(totalDisk-usedDisk) / 1048576
				fmt.Printf("Free disk space is too low: %.1f Mb left\n", free)
			}
		}

		const bw uint64 = 125000000
		if bw > 0 && netUsage*10000/bw > 9000 { // > 90%
			free := float64(bw-netUsage) * 8 / 1048576
			fmt.Printf("Network bandwidth usage high: %.1f Mbit/s available\n", free)
		}

		time.Sleep(30 * time.Second)
	}
}
