package main

import (
	"log"
	"os"
	"strconv"

	"github.com/timohirt/terraform-provider-hetznerdns/hetznerdns/api"
)

func main() {
	var (
		apiToken    = getRequiredEnv("HETZNER_DNS_API_TOKEN")
		dnsZoneName = getRequiredEnv("DNS_ZONE_NAME")
		dnsRecord   = getRequiredEnv("DNS_RECORD_NAME")
		dnsType     = getRequiredEnv("DNS_RECORD_TYPE")
		dnsValue    = getRequiredEnv("DNS_RECORD_VALUE")
		dnsTTL      = getRequiredEnv("DNS_RECORD_TTL")
	)

	ttl64, err := strconv.ParseInt(dnsTTL, 0, 0)
	if err != nil {
		log.Fatalf("DNS_RECORD_TTL: invalid value. expected a number")
	}
	ttl := int(ttl64)

	cli, err := api.NewClient(apiToken)
	if err != nil {
		log.Fatalf("failed to create API client: %s", err)
	}

	zone, err := cli.GetZoneByName(dnsZoneName)
	if err != nil {
		log.Fatalf("failed to get zone by name: %s", err)
	}

	log.Printf("%s: zone ID %s", zone.Name, zone.ID)

	existingRecord, err := cli.GetRecordByName(zone.ID, dnsRecord)
	if err != nil {
		// TODO(ppacher): create PR for timohirt/terraform-provider-hetznerdns so
		// we get the status code out of it?
		log.Printf("failed to get record by name: %s", err)

		_, err := cli.CreateRecord(api.CreateRecordOpts{
			ZoneID: zone.ID,
			Type:   dnsType,
			Name:   dnsRecord,
			Value:  dnsValue,
			TTL:    &ttl,
		})
		if err != nil {
			log.Fatalf("failed to create record: %s", err)
		}
		return
	}

	log.Printf("record ID for %s is %s, updating...", existingRecord.Name, existingRecord.ID)
	if existingRecord.Type == dnsType {
		existingRecord.TTL = &ttl
		existingRecord.Value = dnsValue

		if _, err := cli.UpdateRecord(*existingRecord); err != nil {
			log.Fatalf("failed to update record: %s", err)
		}
		return
	}

	// changing the DNS type is not supported so we need to destroy
	// and re-create the record
	if err := cli.DeleteRecord(existingRecord.ID); err != nil {
		log.Fatalf("failed to delete existing record with ID %s: %s", existingRecord.ID, err)
	}
	if _, err := cli.CreateRecord(api.CreateRecordOpts{
		ZoneID: zone.ID,
		Type:   dnsType,
		Name:   dnsRecord,
		Value:  dnsValue,
		TTL:    &ttl,
	}); err != nil {
		log.Fatalf("failed to create record: %s", err)
	}
}

func getRequiredEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		log.Fatalf("%s is required", name)
	}
	return val
}
