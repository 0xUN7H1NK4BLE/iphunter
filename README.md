# Domain Resolver with Cloudflare IP Detection
This Go program reads a list of domain names from a file, resolves each domain to its IP addresses (both IPv4 and IPv6), and detects if any IP belongs to Cloudflare's network. It outputs the results in a formatted manner using ANSI color codes.

# Requirements
* Go (Golang) installed on your system.
* ip.conf file containing Cloudflare IP ranges in CIDR notation (one per line).

## Run the Program:
```
iphunter filename.txt
```

## Best uses 
```
iphunter filename.txt | tee domainl_ip
cat domain_ip | grep -v cloudflare | grep -oP '^\D*\[\K[^]]+' | tr ',' '\n'
```