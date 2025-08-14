# Service Detection

This repository contains a application written in Go that detects if a user is using UK DNS Privacy Project resolvers. The application handles both DNS requests and HTTP requests, storing records of IP addresses that interact with it and returns the results as json.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:
    ```bash
    git clone https://github.com/UK-DNS-Privacy-Project/service-detection.git
    ```

2. Navigate to the project directory:
    ```bash
    cd service-detection
    ```

3. Build the application:
    ```bash
    go build -o service-detection
    ```

### Running the Application

1. Start the DNS and HTTP servers:
    ```bash
    ./service-detection
    ```

2. The DNS server will start on port 53 and the HTTP server on port 8080.

### Environment Variables

- `ACME_CHALLENGE_DOMAIN`: The domain for ACME challenge DNS queries (i.e. `_acme-challenge.lookup.dnsprivacy.co.uk.`).
- `ACME_CHALLENGE_DNS_1`: The first DNS server to forward ACME challenge queries to.
- `ACME_CHALLENGE_DNS_2`: The second DNS server to forward ACME challenge queries to.
- `DOMAIN`: The primary domain for DNS records (i.e. `lookup.dnsprivacy.co.uk.`).
- `SOA_ADMIN`: The admin email for SOA records (i.e. `dnsadmin.dnsadmin.org.uk`).
- `TARGET_NS`: The target nameserver for NS records (should point to this application).
- `TARGET_IPV4`: The target IPv4 address for A records (should point to this application).
- `TARGET_IPV6`: The target IPv6 address for AAAA records (should point to this application).

### API Endpoints

- `/json`: Returns stored IP addresses for the requested host in JSON format.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.