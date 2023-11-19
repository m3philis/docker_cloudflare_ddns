package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

type DNS struct {
	Name    string `json:"Name"`
	Content string `json:"Content"`
	ID      string `json:"ID"`
	Type    string `json:"Type"`
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

	response, err := exec.Command("/usr/bin/flarectl", "--json", "dns", "list", "--zone", os.Getenv("CF_ZONE")).Output()
	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(response, &dnsData)

	return dnsData
}

func updateDNS(newIP string, IDs []string, zone string) {

	for _, id := range IDs {
		exec.Command("/usr/bin/flarectl", "dns", "update", "--zone", zone, "--id", id, "--content", newIP)
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

	cfZone := os.Getenv("CF_ZONE")
	cfSubdomains := strings.Split(os.Getenv("CF_SUBDOMAINS"), ",")

	for true {
		log.Println("Getting public IP...")
		publicIP := getPublicIP()
		log.Printf("Public IP is %s\n", publicIP)
		log.Println("Getting current DNS info from CloudFlare...")
		currentDNS := getCurrentDNS(cfZone)

		if publicIP != currentDNS[0].Content {
			log.Println("Public IP is different to CloudFlare DNS! Updating!")
			var subdomainIDs []string

			for _, subdomain := range currentDNS {
				if subdomain.Type == "A" {
					if slices.Contains(cfSubdomains, subdomain.Name) {
						subdomainIDs = append(subdomainIDs, subdomain.ID)
					}
				}
			}

			updateDNS(publicIP, subdomainIDs, cfZone)
			checkDNS(publicIP, cfSubdomains)
		} else {
			log.Println("DNS is up2date. Doing nothing!")
		}

		log.Println("Sleeping 5min...zzZ")
		time.Sleep(5 * time.Minute)
	}
}
