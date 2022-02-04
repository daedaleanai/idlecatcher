/*
   stops google cloud instance, which is not using enough CPU

   output being something like:
   CPU usage: 0.07085953878406709
   CPU usage below threshold [count: 1]

*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// the CPU usage THRESHOLD under which a machine is considered idle
	THRESHOLD = flag.Float64("threshold", 0.1, "threshold under which the machine is considered idle")
	// defines how often the CPU stats are polled
	INTERVAL = flag.Int("interval", 30, "interval (in seconds) between polls")
	// defines number of idle intervals before the machine is shut down
	MAXIDLE = flag.Int("max_idle", 20, "number of idle intervals before the machine is shut down")
)

// getCPUSample returns the number of idle Ticks and the number of total Ticks,
// returns the idle Time and the total Time since the machine started.
func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			//ith the following columns:
			// - 'cpu'
			// - normal processes executing in the user mode
			// - niced processes executing in user mode
			// - process executing in kernel mode
			// - idle / not doing anythin
			// - waiting for I/O to complete
			// - servicing interrupts
			// - servicing softirqs
			for i, raw := range fields[1:] {
				val, err := strconv.ParseUint(raw, 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, raw, err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 3 {
					idle = val
				}
			}
			return
		}
	}
	return
}

func main() {
	flag.Parse()

	var count int

	idle0, total0 := getCPUSample()

	// one loop takes INTERVAL seconds
	for {
		// compute the current CPU usage
		time.Sleep(time.Duration(*INTERVAL) * time.Second)
		idle1, total1 := getCPUSample()

		idleDelta := idle1 - idle0
		totalDelta := total1 - total0
		cpuUsage := float64(totalDelta-idleDelta) / float64(totalDelta)

		idle0, total0 = idle1, total1

		fmt.Printf("CPU usage: %v\n", cpuUsage)

		// killing after maxCount times CPU usage below threshold
		if cpuUsage < *THRESHOLD {
			count++
			fmt.Printf("CPU usage below threshold [count: %v]\n", count)
			if *MAXIDLE <= count {
				syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
			}
		} else {
			count = 0
		}
	}
}
