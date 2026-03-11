package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net" // give alias psnet to avoid naming conflict with "net" package (they have the same name "net")

	"net" // for IP address stuff
	"net/http"
	"os"
	"os/signal" // for graceful shutdown
	"syscall"
	"time"
)

// the JSON template (blueprint) (struct)
type SystemInfo struct {
	OS       string `json:"operating_system"` // allocate an OS field to hold a string. when sending as JSON, rename it to "operating_system" cause the frontend generally wants lowercase names. While go wants uppercase names
	Kernel   string `json:"kernel_version"`
	Hostname string `json:"hostname"`
	Uptime   uint64 `json:"uptime_seconds"`
	LocalIP  string `json:"local_ip"`

	CPUModel   string  `json:"cpu_model"`
	CPUPercent float32 `json:"cpu_usage_percent"`
	CPUTemp    float32 `json:"cpu_temperature_celsius"`

	RAMPercent  float32 `json:"ram_usage_percent"`
	RAMUsedByte uint64  `json:"ram_used_bytes"`

	DiskTotalByte uint64  `json:"disk_total_bytes"`
	DiskUsedByte  uint64  `json:"disk_used_bytes"`
	DiskPercent   float32 `json:"disk_usage_percent"`

	BatteryPercent float32 `json:"battery_percent"`
	IsCharging     bool    `json:"is_charging"`

	NetSentByte uint64 `json:"network_sent_bytes"`
	NetRecvByte uint64 `json:"network_received_bytes"`
}

// code to find the device's IP
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs() // if error comes in, assign to err, otherwise assign the response to addrs
	if err != nil {
		return "", err // return first variable (string) empty, return second variable (error) err [string, error]
	}

	for _, address := range addrs { // assign the given index to blank (_), assign the IPs to the address variable
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil // return first variable (string) the IP, since there is no error, return nil for error variable
			}
		}
	}

	return "IP not found", nil // if there is no error nor no IP adress, return string as "IP not found", and return err as nil cause no error
}

func statusHandler(w http.ResponseWriter, r *http.Request) {

	// only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Only method GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	// CORS permissions so the frontend can interact with API
	w.Header().Set("Access-Control-Allow-Origin", "*")             // since it runs locally, allow anyone to access
	w.Header().Set("Access-Control-Allow-Methods", "GET")          // only allow GET method
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // allows Content-Type header for CORS/JSON requests

	// host system information
	hInfo, err := host.Info()
	if err != nil {
		http.Error(w, "Host info could not be read", http.StatusInternalServerError)
		return
	}

	// RAM information
	v, err := mem.VirtualMemory()
	if err != nil {
		http.Error(w, "RAM info could not be read", http.StatusInternalServerError)
		return
	}

	// CPU usage information
	// 1st parameter: how long to measure core usage
	// 2nd parameter (false): gives average core usage across cores
	cPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		http.Error(w, "CPU usage info could not be read", http.StatusInternalServerError)
		return
	}
	// CPU model information
	cInfo, err := cpu.Info()
	if err != nil {
		http.Error(w, "CPU model info could not be read", http.StatusInternalServerError)
		return
	}
	// if one of the CPU data can't be read, throw error
	if len(cPercent) == 0 || len(cInfo) == 0 {
		http.Error(w, "CPU data could not be read", http.StatusInternalServerError)
		return
	}

	// get disk usage for the root filesystem (/)
	dInfo, err := disk.Usage("/")
	if err != nil {
		http.Error(w, "Disk info could not be read", http.StatusInternalServerError)
		return
	}

	// get IP or error from the IP finding function
	localIP, err := getLocalIP()
	if err != nil {
		http.Error(w, "Local IP info could not be read", http.StatusInternalServerError)
		return
	}

	// network traffic information (total upload/download bytes)
	netStats, err := psnet.IOCounters(false)
	if err != nil {
		http.Error(w, "Network info could not be read", http.StatusInternalServerError)
		return
	}
	// if network data can't be read, throw error
	if len(netStats) == 0 {
		http.Error(w, "Network data could not be read", http.StatusInternalServerError)
		return
	}

	// read temperature sensors and grab the highest temperature
	tempStats, err := host.SensorsTemperatures()
	if err != nil {
		http.Error(w, "Temperature info could not be read", http.StatusInternalServerError)
		return
	}
	// grab the highest temp and show it
	maxTemp := 0.0
	for _, temp := range tempStats {
		if temp.Temperature > maxTemp {
			maxTemp = temp.Temperature
		}
	}

	// battery info may not exist on non-laptop devices, so we handle it flexibly
	bats, err := battery.GetAll()
	batPercent := 0.0
	isCharging := false
	if err == nil && len(bats) > 0 && bats[0].Full > 0 {
		batPercent = (bats[0].Current / bats[0].Full) * 100
		isCharging = (bats[0].State.String() == "Charging")
	}

	// fill the template with system info
	info := SystemInfo{
		OS:          hInfo.OS, // write the value from hInfo.OS to the OS field of the template
		Kernel:      hInfo.KernelVersion,
		Hostname:    hInfo.Hostname,
		Uptime:      hInfo.Uptime,
		LocalIP:     localIP,
		CPUModel:    cInfo[0].ModelName,
		CPUPercent:  float32(cPercent[0]),
		CPUTemp:     float32(maxTemp),
		RAMPercent:  float32(v.UsedPercent),
		DiskPercent: float32(dInfo.UsedPercent),

		BatteryPercent: float32(batPercent),
		IsCharging:     isCharging,

		// total download/upload, frontend will calculate real time internet speed
		NetSentByte: netStats[0].BytesSent,
		NetRecvByte: netStats[0].BytesRecv,

		// we send values in bytes, frontend will convert to MB or GB when displaying
		RAMUsedByte:   v.Used,
		DiskTotalByte: dInfo.Total,
		DiskUsedByte:  dInfo.Used,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		fmt.Printf("JSON encode error: %v\n", err) // if JSON encoding fails, log the error and continue
	}
}

func main() {
	// starting the HTTP server
	http.HandleFunc("/api/status", statusHandler)

	srv := &http.Server{ // create an HTTP server with our custom config
		Addr:         ":8080",          // listen on port 8080
		ReadTimeout:  10 * time.Second, // max time to read request from client
		WriteTimeout: 10 * time.Second, // max time to write response to client
		IdleTimeout:  15 * time.Second, // max time to keep idle connection open
	}

	go func() {
		fmt.Println("Server started on port 8080. Available at http://localhost:8080/api/status")

		// error handler, if the http server couldnt start
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Println("ERROR:", err)
		}
	}()

	quit := make(chan os.Signal, 1) // create a channel to receive OS signals with buffer size 1

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // listen for Ctrl C or termination signals and send them to the quit channel

	<-quit // block and wait until we receive a signal on the quit channel

	fmt.Println("\nShutdown signal received. Server shutting down...") // inform that shutdown is happening

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // create a context that times out after 5 seconds for graceful shutdown

	defer cancel() // make sure we clean up the timeout context when we're done

	err := srv.Shutdown(ctx) // gracefully shut down the server within the 5 second timeout
	if err != nil {          // if shutdown fails print the error
		fmt.Println("ERROR:", err)
	}
}
