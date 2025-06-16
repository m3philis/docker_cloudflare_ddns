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
	Zones []Zone `yaml:"cf_domains"`
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

func checkDNS(newIP string, subdomains []string, domain string) {

	var verified bool = true

	for _, subdomain := range subdomains {
		dnsResponse, err := exec.Command("/usr/bin/dig", subdomain, ".", domain, "+short", "@1.1.1.1").Output()
		if err != nil {
			log.Fatal(err)
		}

		dnsIP := strings.TrimSuffix(string(dnsResponse), "\n")
		if dnsIP != newIP {
			log.Printf("DNS for %s.%s was not updated correctly!", subdomain, domain)
			verified = false
		}

	}
	if verified {
		log.Println("All subdomains verified to return the new IP!")
	}
}

func createDNS(zone string, name string, newIP string, dnsType string) {

	//log.Printf("/usr/bin/flarectl dns create --zone %v --name %v --content %v --type %v\n", zone, name, newIP, dnsType)
	err := exec.Command("/usr/bin/flarectl", "dns", "create", "--zone", zone, "--name", name, "--content", newIP, "--type", dnsType).Run()
	if err != nil {
		log.Fatal(err)
	}
}

func deleteDNS(zone string, ID string) {
	err := exec.Command("/usr/bin/flarectl", "dns", "delete", "--zone", zone, "--id", ID).Run()
	if err != nil {
		log.Fatal(err)
	}
}

func updateDNS(zone string, ID string, newIP string) {

	err := exec.Command("/usr/bin/flarectl", "dns", "update", "--zone", zone, "--id", ID, "--content", newIP).Run()
	if err != nil {
		log.Fatal(err)
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

		log.Println("Getting public IP...")
		publicIP := getPublicIP()
		log.Printf("Public IP is %s\n", publicIP)

		for _, zone := range zones.Zones {

			subdomains := make(map[string]struct{})

			for _, subdomain := range zone.SubDomains {
					subdomains[subdomain] = struct{}{}
			}

			log.Printf("Getting current DNS info from CloudFlare for zone %s...\n", zone.Name)
			os.Setenv("CF_API_TOKEN", zone.ApiToken)
			currentDNS := getCurrentDNS(zone.Name)

			ipCheck := net.ParseIP(publicIP)

			var changedIP bool = false
			var dnsType string = "A"

			if ipCheck.To4() != nil {
				log.Println("Public IP is from type IPv4. Only updating A Records!")
				for _, dnsDomain := range currentDNS {
					if len(dnsDomain.Name) > len(zone.Name) {
						dnsDomain.Name = dnsDomain.Name[:len(dnsDomain.Name)-len(zone.Name)-1]
					}
					if dnsDomain.Type == "A" {
						if slices.Contains(zone.SubDomains, dnsDomain.Name) {
							delete(subdomains, dnsDomain.Name)
							if publicIP != dnsDomain.Content {
								log.Printf("IP for %s.%s is different to public IP! Updating CloudFlare DNS!\n", dnsDomain.Name, zone.Name)
								updateDNS(zone.Name, dnsDomain.ID, publicIP)
								changedIP = true
							} else {
								log.Printf("IP for %s.%s is already correct!\n", dnsDomain.Name, zone.Name)
							}
						} else {
							if dnsDomain.Type == "A" && dnsDomain.Name != zone.Name {
								log.Printf("DNS for %s.%s not needed anymore. Deleting entry!\n", dnsDomain.Name, zone.Name)
								deleteDNS(zone.Name, dnsDomain.ID)
							}
						}
					}
				}
			} else {
				dnsType = "AAAA"
				log.Println("Public IP is from type IPv6. Only updating AAAA Records!")
				for _, dnsDomain := range currentDNS {
					if dnsDomain.Type == "AAAA" {
						if slices.Contains(zone.SubDomains, dnsDomain.Name) {
							delete(subdomains, dnsDomain.Name)
							if publicIP != dnsDomain.Content {
								log.Printf("IP for %s.%s is different to public IP! Updating CloudFlare DNS!\n", dnsDomain.Name, zone.Name)
								updateDNS(publicIP, dnsDomain.ID, zone.Name)
								changedIP = true
							} else {
								log.Printf("IP for %s.%s is already correct!\n", dnsDomain.Name, zone.Name)
							}
						} else {
							if dnsDomain.Type == "AAAA" && dnsDomain.Name != zone.Name {
								log.Printf("DNS for %s.%s not needed anymore. Deleting entry!\n", dnsDomain.Name, zone.Name)
								deleteDNS(zone.Name, dnsDomain.ID)
							}
						}
					}
				}
			}

			for subdomain := range subdomains {
				log.Printf("%s.%s not found at CloudFlare, creating entry!\n", subdomain, zone.Name)
				createDNS(zone.Name, subdomain, publicIP, dnsType)
				changedIP = true
			}

			if changedIP {
				log.Println("Waiting a minute before checking DNS to let it propagate...")
				time.Sleep(1 * time.Minute)
				checkDNS(publicIP, zone.SubDomains, zone.Name)
			}
		}

		log.Println("Sleeping 5min...zzZ")
		time.Sleep(5 * time.Minute)
	}
}
