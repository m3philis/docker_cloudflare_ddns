# docker_cloudflare_ddns
Docker container to update cloudflare domains with current public IP of the container

## Usage

Build the docker container `docker buildx build . -t cf-ddns:latest`

Prerequisites:
* This container uses a YAML file to check for CloudFlare Zones and the subdomains in them.
* The syntax of the YAML file will be
```
---
cf_domains:
- cf_zone: $zone_name
  cf_api_token: $token
  cf_subdomains:
  - sub1
  - sub2
  - sub3
```
* The file needs to be mounted at /domains.yaml for the container to work

Command to run the container would be `docker run cf-ddns:latest -v /path/to/domains.yaml:/domains.yaml:ro`

Or better with docker compose:
```
  cf-updater:
    build: build/docker_cloudflare_ddns
    restart: unless-stopped
    volumes:
    - ./configs/cf_ddns/domains.yaml:/domains.yaml:ro
```

The container checks the public IP of the network it runs it against ifconfig.io. Then it checks the defined IPs in CloudFlare for the specified subdomains and if they differ from the public IP they will get updated. Afterwards the container checks if a DNS query will respond with the new IPs.

## If you run golang >= 1.21

Since golang 1.21 the slices package got added to the main library. Before that you need to include the experimental module.
In the go file, replace the `golang.org/x/exp/slices` import with `slices` and ignore the go.mod file. This is written for golang < 1.21 as alpine currently has golang version 1.20 in its repos.
