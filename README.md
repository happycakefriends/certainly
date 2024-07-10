# certainly

Certainly is a offensive security toolkit to capture large amounts of traffic in various network protocols in bitflip and typosquat scenarios. The tool was built to support research on these topics, originally [presented at BlackHat USA 2024](https://www.blackhat.com/us-24/briefings/schedule/index.html#flipping-bits-your-credentials-are-certainly-mine-40040). 

## How it works

**Built-in protocols**
 - DNS
 - HTTP(S)
 - IMAP(S)
 - SMTP(S)

### DNS
The core functionality of certainly revolves around the DNS server. It is designed to act as the authoritative name server for a number of apex domains, answering to any DNS questions around the zones with a response pointing to its own IP address while logging any and all requests coming its way. Certainly is trying to play nice and to not to break anything longer than necessary, because of this reason all the answers have TTL of one second.

### HTTP
When certainly receives a HTTP request, it first checks if the requested resource is something that it should apply an injection template on. If so, certainly will copy the request headers to a new proxied request towards the upstream (legit, non-bitflippped / non-typosquatted) domain, copy the response headers to the response sent to the victim while replacing the response body according to the template rules.

If injection template is not configured for the resource, certainly will instead respond with HTTP 307 (Temporary redirect) to the upstream target with the full request URI intact.

### HTTPS
HTTPS works similarly to HTTP, but in case certainly doesn't have a valid TLS certificate to present to the client, it will hold the TCP connection after the ClientHello in TLS handshake while fetching the certificate client requested in the TLS SNI. Certainly will try to priorize getting wildcard certificates for all subdomains in order to not to exhaust CA limits as well as to speed up the transaction when receiving a new connection. This behavior is very important as especially in the bitflips the targeted subdomains are far and between and not doing so will risk losing valuable data.

### IMAP(S)
Certainly will initiate the authentication sequence and log the user credentials as well as potential shared secret in case of CRAM-MD5, after which it will disconnect the user.

### SMTP(S)
Certainly will kindly accept all email sent towards it and proceed delivering it to the log files.

## Core features

### DNS
- Full authoritative DNS server support. Just point the nameserver addresses at your domain registrar of choice towards Certainly instance.
- CNAMEs to randomly generated UUID subdomains of the configured "main domain" in order to be able to track client behavior per-requester basis. This is omitted for CNAME requests against the "main domain" subdomains in order to prevent infinite loops. A record answers for these CNAMEs are also appended to the answer to lower the necessary network traffic.
- DNS based ACME challenge solver to support wildcard TLS certificate generation.
- Custom DNS records to present
- Configurable protocol(s) to listen; udp, tcp or both

### HTTPS
- Holding the TLS handshake in ClientHello phase while fetching the certificate to present in the background. This typically takes under 5 seconds.
- Optional upstream check for existence of a domain record before answering. If the upstream (sub)domain doesn't exist, certainly will proceed answering with NXDOMAIN as well.
- Injection templating based on request uri regexes. Templates have couple of keyword variables that will be replaced: CERTAINLY_UPSTREAM that will be replaced by the full response body of the upstream request, and CERTAINLY_HASH that will be replaced by a UUID generated for the orignal connection.
- Injection template filtering by a list of regexes. There's a lot of noise in the web today, and we saw a lot of random sweep scans hitting us with predetermined paths that we're better off by just ignoring.
- A custom route for `/callback/*` that will just simply answer with `204 No Content` instead of the default behavior of doing a temporary redirect. This is to catch and log potential callbacks from injected JavaScript resources without disturbing the intended behavior of the web application too much.

### Output
 - Default output format of JSONLines to feed in to your data analysis platform; ELK, Splunk, mad grep oneliners; whatever your prefer.
 - Extensible notification framework for sending automated notifications. Currently only supports Slack.

## Installation on Linux
For this documentation we're using `/path/to/install/certainly` as the example installation directory.

### 1) Create a user and installation path for Certainly
Create the application and the injection template directories
```
sudo mkdir -p /path/to/install/certainly/templates
```
Set up a system user and group to run Certainly
```
sudo useradd --system --user-group certainly
```
Set the ownership and permissions for the user to access the filesystem path.
```
sudo chown -R certainly:certainly /path/to/install/certainly
sudo chmod -R ug+w /path/to/install/certainly
```

### 2) Fetch the application and the example configuration
Certainly can be fetched by either `go install` or downloading and unarchiving a pre-built binary to your file system location of choice.

#### Using go install
```
go install github.com/happycakefriends/certainly@latest
```

Move the `certainly` binary to the installation path (not necessary, but streamlines this documentation). The installation directory for Go binaries can vary based on your configuration, but the default directory location is `$HOME/go/bin/`

When using this method, you will also need to download the latest `config.cfg` manually:

```
curl -L https://raw.githubusercontent.com/happycakefriends/certainly/main/config.cfg -o /path/to/install/certainly/config.cfg
```
#### Downloading a pre-built binary
Head over to [releases page](https://github.com/happycakefriends/certainly/releases/latest) to download the latest pre-built release for your platform of choice.

Unarchive the contents of the release archive to `/path/to/install/certainly`

### 3) Edit the certainly configuration file
Open `/path/to/install/certainly/config.cfg` in your favorite text editor and change the configuration values according to your needs.

### 4) Set capabilities to allow binding to privileged ports
Allow certainly to bind to necessary ports
```
sudo setcap 'cap_net_bind_service=+ep' /path/to/install/certainly/certainly
```

### 5) Create a systemd service
```
sudo touch /etc/systemd/system/certainly.service
sudo chmod 644 /etc/systemd/system/certainly.service
```

Open the newly created systemd service definition file with your favorite text editor and change the paths accordingly.
```
[Unit]
Description=Certainly
After=syslog.target network.target

[Service]
Type=simple
WorkingDirectory=/path/to/install/certainly
User=certainly
ExecStart=/path/to/install/certainly/certainly
TimeoutStartSec=30

[Install]
WantedBy=multi-user.target
```
Reload the service configurations
```
sudo systemctl daemon-reload
```

### 6) Start the systemd service
Start the service
```
sudo systemctl start certainly
```
(Optionally) set it up to automatically start when OS is booted
```
sudo systemctl enable certainly
```

### 7) Configure the domain
Point the nameservers of your domain to certainly instance at your registrars configuration panel. This varies based on registrar.