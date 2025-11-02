package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// --- Configuration ---
const (
	DefaultScanPort  = 3000
	DefaultAPIPort   = 7370
	ScanTimeout      = 1 * time.Second
	MaxAPIRequests   = 3
	APIServerTimeout = 5 * time.Second
	ShutdownDelay    = 100 * time.Millisecond
	ShutdownTimeout  = 1 * time.Second
)

// Global storage for discovered IPs
var discoveredIPs []string

func main() {
	scanPort := DefaultScanPort
	apiPort := DefaultAPIPort
	serveAPI := false

	// --- parse discover:// URI if provided ---
	if len(os.Args) > 1 {
		arg := strings.Trim(os.Args[1], `"'`)
		if strings.HasPrefix(arg, "discover://") {
			u, err := url.Parse(arg)
			if err == nil {
				if p := u.Query().Get("port"); p != "" {
					if val, err := strconv.Atoi(p); err == nil {
						scanPort = val
					}
				}
				if s := u.Query().Get("serve"); s == "true" {
					serveAPI = true
				}
				if ap := u.Query().Get("apiPort"); ap != "" {
					if val, err := strconv.Atoi(ap); err == nil {
						apiPort = val
					}
				}
			}
		}
	}

	// Run discovery scan
	discoveredIPs = runDiscoveryScan(scanPort)

	// Start API server if requested
	if serveAPI {
		startAPIServer(apiPort)
	}
}

func runDiscoveryScan(port int) []string {
	ip, ipnet, err := localSubnet()
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	fmt.Printf("Scanning subnet: %s for port %d...\n", ipnet.String(), port)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var foundIPs []string

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()

			addr := fmt.Sprintf("%s:%d", ip, port)
			conn, err := net.DialTimeout("tcp", addr, ScanTimeout)
			if err != nil {
				return
			}
			conn.Close()

			mu.Lock()
			foundIPs = append(foundIPs, addr)
			fmt.Printf("✅ Active server found: %s\n", addr)
			mu.Unlock()
		}(ipCopy)
	}

	wg.Wait()

	if len(foundIPs) == 0 {
		fmt.Println("\n❌ No active servers found.")
	} else {
		fmt.Printf("\n✅ Total servers found: %d\n", len(foundIPs))
	}

	return foundIPs
}

func startAPIServer(port int) {
	var requestCount int32

	// Create server with timeout
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	// Context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), APIServerTimeout)
	defer cancel()

	// CORS middleware
	corsHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// Handle /getIP endpoint
	http.HandleFunc("/getIP", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		fmt.Printf("Request %d/%d received from %s\n", count, MaxAPIRequests, r.RemoteAddr)

		w.Header().Set("Content-Type", "application/json")

		// Return as JSON array
		fmt.Fprint(w, "[")
		for i, ip := range discoveredIPs {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `"%s"`, ip)
		}
		fmt.Fprint(w, "]")

		if count >= MaxAPIRequests {
			fmt.Println("Max requests reached. Shutting down...")
			go func() {
				time.Sleep(ShutdownDelay)
				cancel()
			}()
		}
	}))

	fmt.Printf("\nAPI server started on port %d\n", port)
	if len(discoveredIPs) > 0 {
		fmt.Printf("Discovered IPs: %v\n", discoveredIPs)
	} else {
		fmt.Println("No IPs discovered - will return empty array")
	}
	fmt.Printf("Waiting for up to %d requests or %v timeout...\n", MaxAPIRequests, APIServerTimeout)

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Server error:", err)
		}
	}()

	// Wait for context cancellation (either timeout or max requests)
	<-ctx.Done()

	// Shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)

	fmt.Println("Server stopped.")
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func localSubnet() (net.IP, *net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip, ipnet, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil {
				return ip, ipnet, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no active IPv4 interface found")
}
