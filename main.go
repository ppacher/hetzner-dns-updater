package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	hetzner_dns "github.com/panta/go-hetzner-dns"
)

type record struct {
	name           string
	rrType         string
	value          string
	ttl            int
	deleteExisting bool
}

func (rec record) String() string {
	return fmt.Sprintf("%s %d %s %s", rec.name, rec.ttl, rec.rrType, rec.value)
}

const (
	recordEnvPrefix = "DNS_RECORD_"
)

func getRecordDefinitions() []record {
	lm := make(map[string]*record)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, recordEnvPrefix) {
			parts := strings.SplitN(env, "=", 2)
			envName, envValue := parts[0], parts[1]

			envNameParts := strings.Split(envName, "_")
			if len(envNameParts) != 4 {
				log.Printf("invalid number of parts in environment variable name %s\n", envName)
				continue
			}

			recordIndex := envNameParts[2]

			if _, ok := lm[recordIndex]; !ok {
				lm[recordIndex] = new(record)
			}

			switch envNameParts[3] {
			case "NAME":
				envValue = strings.TrimSuffix(envValue, ".")
				lm[recordIndex].name = envValue
			case "TYPE":
				lm[recordIndex].rrType = envValue
			case "VALUE":
				lm[recordIndex].value = envValue
			case "TTL":
				ttl, err := strconv.ParseInt(envValue, 0, 0)
				if err != nil {
					log.Printf("invalid value for record ttl: %q", envValue)
					continue
				}

				lm[recordIndex].ttl = int(ttl)
			case "OVERWRITE":
				val, err := strconv.ParseBool(envValue)
				if err != nil {
					log.Printf("invalid value for overwrite: %s", envValue)
				}

				lm[recordIndex].deleteExisting = val
			}
		}
	}

	result := make([]record, 0, len(lm))
	for _, rec := range lm {
		result = append(result, *rec)
	}

	return result
}

func main() {
	var (
		apiToken    = getRequiredEnv("HETZNER_DNS_API_TOKEN")
		dnsZoneName = getRequiredEnv("DNS_ZONE_NAME")
	)

	ctx := context.Background()

	records := getRecordDefinitions()
	if len(records) == 0 {
		log.Printf("no record definitions found")
		return
	}

	cli := &hetzner_dns.Client{ApiKey: apiToken}

	zones, err := cli.GetZones(ctx, dnsZoneName, "", 1, 100)
	if err != nil {
		log.Fatalf("failed to get zone by name: %s", err)
	}

	if len(zones.Zones) == 0 {
		allZones, err := cli.GetZones(ctx, "", "", 1, 100)
		if err != nil {
			log.Fatal("failed to get all DNS zones")
		}

		if len(allZones.Zones) == 0 {
			log.Printf("no dns zone available\n")
		}

		for _, z := range allZones.Zones {
			log.Printf("available zone: id=%s name=%s\n", z.ID, z.Name)
		}

		log.Fatalf("configured DNS zone %q does not exist", dnsZoneName)
	}

	log.Printf("%s: zone ID %s", zones.Zones[0].Name, zones.Zones[0].ID)

	existingRecords, err := cli.GetRecords(ctx, zones.Zones[0].ID, 0, 0)
	if err != nil {
		log.Fatalf("failed to get zone records: %s", err)
	}

	// build a simple lookup map indexed by "<NAME> <TYPE>"
	lm := make(map[string][]hetzner_dns.Record)
	for _, e := range existingRecords.Records {
		key := fmt.Sprintf("%s %s", e.Name, e.Type)
		lm[key] = append(lm[key], e)

		log.Printf("found existing records: %s %s %d %s %s\n", e.ID, e.Name, e.TTL, e.Type, e.Value)
	}

	for _, rec := range records {
		key := fmt.Sprintf("%s %s", rec.name, rec.rrType)
		existing := lm[key]

		if rec.deleteExisting {
			for _, e := range existing {
				log.Printf("deleting existing record %s: %s %s %s\n", e.ID, e.Name, e.Type, e.Value)
				if err := cli.DeleteRecord(ctx, e.ID); err != nil {
					log.Printf("failed to delete record %s", e.ID)
				}
			}

			// reset so we don't try to delete them again
			lm[key] = nil
		}

		log.Printf("creating record %s\n", rec.String())

		if _, err := cli.CreateOrUpdateRecord(ctx, hetzner_dns.RecordRequest{
			ZoneID: zones.Zones[0].ID,
			Type:   rec.rrType,
			Name:   rec.name,
			Value:  rec.value,
			TTL:    rec.ttl,
		}); err != nil {
			log.Printf("failed to create record: %s\n", err)
		}
	}
}

func getRequiredEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		log.Fatalf("%s is required", name)
	}
	return val
}
