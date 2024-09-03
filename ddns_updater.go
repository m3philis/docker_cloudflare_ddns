package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type DNS struct {
	Name    string `json:"Name"`
	Content string `json:"Content"`
	ID      string `json:"ID"`
	Type    string `json:"Type"`
}

type Zone struct {
	Name       string   `yaml:"cf_zone"`
	SubDomains []string `yaml:"cf_subdomains"`
	ApiToken   string   `yaml:"cf_api_token"`
}

type Domains struct {
	Zones    []Zone
	ApiToken string `yaml:"cf_api_token"`
}

func getPublicIP() string {
	publicIP, err := exec.Command("curl", "ifconfig.io").Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSuffix(string(publicIP), "\n")
}

func getCurrentDNS(zone string) []DNS {

	var dnsData []DNS

	response, err := exec.Command("/usr/bin/flarectl", "--json", "dns", "list", "--zone", zone).Output()
	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(response, &dnsData)

	return dnsData
}

func updateDNS(newIP string, ID string, zone string) {

	err := exec.Command("/usr/bin/flarectl", "dns", "update", "--zone", zone, "--id", ID, "--content", newIP).Run()
	if err != nil {
		log.Fatal(err)
	}
}

func checkDNS(newIP string, subdomains []string) {

	var verified bool = true

	for _, subdomain := range subdomains {
		dnsResponse, err := exec.Command("/usr/bin/dig", subdomain, "+short", "@1.1.1.1").Output()
		if err != nil {
			log.Fatal(err)
		}

		dnsIP := strings.TrimSuffix(string(dnsResponse), "\n")
		if dnsIP != newIP {
			log.Printf("DNS for %s was not updated correctly!", subdomain)
			verified = false
		}

	}
	if verified {
		log.Println("All subdomains verified to return the new IP!")
	}
}

func main() {

	var zones Domains

	for {
		file, err := os.ReadFile("/domains.yaml")
		if err != nil {
			log.Fatal(err)
		}

		if err := yaml.Unmarshal(file, &zones); err != nil {
			log.Fatal(err)
		}

		os.Setenv("CF_API_TOKEN", zones.ApiToken)

		log.Println("Getting public IP...")
		publicIP := getPublicIP()
		log.Printf("Public IP is %s\n", publicIP)

		log.Println("Getting current DNS info from CloudFlare...")
		for _, zone := range zones.Zones {

			currentDNS := getCurrentDNS(zone.Name)

			ipCheck := net.ParseIP(publicIP)
			var changedIP bool = false
			if ipCheck.To4() != nil {
				log.Println("Public IP is from type IPv4. Only updating A Records!")
				for _, subdomain := range currentDNS {
					if subdomain.Type == "A" {
						if slices.Contains(zone.SubDomains, subdomain.Name) {
							if publicIP != subdomain.Content {
								log.Printf("IP for %s is different to public IP! Updating CloudFlare DNS!\n", subdomain.Name)
								updateDNS(publicIP, subdomain.ID, zone.Name)
								changedIP = true
							} else {
								log.Printf("IP for %s is already correct!\n", subdomain.Name)
							}
						}
					}
				}
			} else {
				log.Println("Public IP is from type IPv6. Only updating AAAA Records!")
				for _, subdomain := range currentDNS {
					if subdomain.Type == "AAAA" {
						if slices.Contains(zone.SubDomains, subdomain.Name) {
							if publicIP != subdomain.Content {
								log.Printf("IP for %s is different to public IP! Updating CloudFlare DNS!\n", subdomain.Name)
								changedIP = true
								updateDNS(publicIP, subdomain.ID, zone.Name)
							} else {
								log.Printf("IP for %s is already correct!\n", subdomain.Name)
							}
						}
					}
				}
			}

			if changedIP {
				log.Println("Waiting a minute before checking DNS to let it propagate...")
				time.Sleep(1 * time.Minute)
				checkDNS(publicIP, zone.SubDomains)
			}
		}

		log.Println("Sleeping 5min...zzZ")
		time.Sleep(5 * time.Minute)
	}
}
