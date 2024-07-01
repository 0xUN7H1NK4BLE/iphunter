package main

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
)

var dnsServers = []string{
	"1.1.1.1",
	"8.8.8.8",
	"8.8.4.4",
	"9.9.9.9",
	"76.76.19.19",
}

var cloudflareIPs []string

func main() {
	// Load Cloudflare IPs from file
	if err := loadCloudflareIPs("ip.conf"); err != nil {
		fmt.Printf("Error loading Cloudflare IPs: %v\n", err)
		return
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <input_file>")
		return
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Could not open file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var wg sync.WaitGroup
	domainChannel := make(chan string, 1000)
	resultChannel := make(chan string, 1000)
	doneChannel := make(chan bool)

	// Worker pool for processing domains
	numWorkers := 1000 // Increase number of workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for input := range domainChannel {
				processInput(input, resultChannel)
			}
		}()
	}

	// Collect results and print them at once
	go func() {
		for result := range resultChannel {
			fmt.Println(result)
		}
		doneChannel <- true
	}()

	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	for scanner.Scan() {
		rawInput := strings.TrimSpace(scanner.Text())
		if rawInput != "" {
			domainChannel <- rawInput
		}
	}

	close(domainChannel)
	<-doneChannel

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}
}

func loadCloudflareIPs(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" {
			cloudflareIPs = append(cloudflareIPs, ip)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func processInput(input string, resultChannel chan<- string) {
	if net.ParseIP(input) != nil {
		ip := net.ParseIP(input)
		resultChannel <- formatResult(input, []net.IP{ip}, nil)
		return
	}

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		domain, err := extractDomain(input)
		if err == nil && domain != "" {
			resolveAndPrintDomain(domain, input, resultChannel)
		} else {
			resultChannel <- fmt.Sprintf("Invalid URL: %s", input)
		}
		return
	}

	resolveAndPrintDomain(input, input, resultChannel)
}

func extractDomain(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Hostname(), nil
}

var cache = struct {
	sync.Mutex
	m map[string][]net.IP
}{m: make(map[string][]net.IP)}

func resolveAndPrintDomain(domain, original string, resultChannel chan<- string) {
	cache.Lock()
	ips, ok := cache.m[domain]
	cache.Unlock()

	if !ok {
		ips, _ = net.LookupIP(domain)

		cache.Lock()
		cache.m[domain] = ips
		cache.Unlock()
	}

	ipv4s, ipv6s := splitIPs(ips)

	resultChannel <- formatResult(original, ipv4s, ipv6s)
}

func splitIPs(ips []net.IP) ([]net.IP, []net.IP) {
	var ipv4s, ipv6s []net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4s = append(ipv4s, ip)
		} else if ip.To16() != nil {
			ipv6s = append(ipv6s, ip)
		}
	}
	return ipv4s, ipv6s
}

func formatResult(original string, ipv4s, ipv6s []net.IP) string {
	var result strings.Builder

	// Append "url: " to the result
	result.WriteString(fmt.Sprintf("%s : ", original))

	// Append IPv4 addresses if they exist
	if len(ipv4s) > 0 {
		result.WriteString(fmt.Sprintf("[%s]", joinIPs(ipv4s)))
	}

	// Append IPv6 addresses if they exist
	if len(ipv6s) > 0 {
		result.WriteString(fmt.Sprintf(" [%s]", joinIPs(ipv6s)))
	}

	// Check if any IP is a Cloudflare IP and append
	if len(ipv4s) > 0 && isCloudflareIP(ipv4s[0]) {
		result.WriteString(" [cloudflare]")
	}

	// Convert the result to string and return
	return result.String()
}

func joinIPs(ips []net.IP) string {
	var ipStr strings.Builder
	for i, ip := range ips {
		if i > 0 {
			ipStr.WriteString(",")
		}
		ipStr.WriteString(ip.String())
	}
	return ipStr.String()
}

func isCloudflareIP(ip net.IP) bool {
	for _, cidr := range cloudflareIPs {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			fmt.Printf("Could not parse CIDR: %v\n", err)
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}
