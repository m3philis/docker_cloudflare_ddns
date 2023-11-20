# docker_cloudflare_ddns
Docker container to update cloudflare domains with current public IP of the container

## Usage

Build the docker container `docker buildx build . -t cf-ddns:latest`

Required environment variabes:
* CF_API_TOKEN: An API token with the permission to change DNS for a specific zone (at least) `CF_API_TOKEN=1234567890abcdefgh`
* CF_ZONE: The zone in cloudflare to be managed `CF_ZONE=example.com`
* CF_SUBDOMAINS: comma-separated list of subdomains to be updated. The root domain also works `CF_SUBDOMAINS=a.example.com,b.example.com,example.com`

Command to run the container would be `docker run cf-ddns:latest -e CF_API_TOKEN=abc -e CF_ZONE=example.com CF_SUBDOMAINS=a.example.com,b.example.com`

The container checks the public IP of the network it runs it against ifconfig.io. Then it checks the defined IPs in CloudFlare for the specified subdomains and if they differ from the public IP they will get updated. Afterwards the container checks if a DNS query will respond with the new IPs.

## If you run golang < 1.21

Since golang 1.21 the slices package got added to the main library. Before that you need to include the experimental module.
In the go file, replace the `slices` import with `golang.org/x/exp/slices`, run `go mod init cf-ddns` and `go get golang.org/x/exp`. This is done in the Dockerfile (except the replacement of the import in the file) as alpine currently has golang version 1.20 in its repos.
