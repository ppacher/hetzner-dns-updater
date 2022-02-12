# `hetzner-dns-updater`

A small utility tool to update a single record via Hetzner DNS API. This is mainly by our Nomad cluster to update the ingress DNS record to point to the cluster node that is running our ingress proxy Traefik.

Example usage:

```hcl2
job "traefik" {
    datacenters = ["dc1"]
    type = "service"

    group "traefik" {
        network {
            port "http" {
                host_network = "public"
            }
        }

        task "prepare" {
            lifecycle {
                hook = "prestart"
            }

            driver = "docker"
            config {
                image = "registry.service.consul:5000/hetzner-dns-updater:latest"
            }

            env {
                HETZNER_DNS_API_TOKEN = "your-api-token" # better use Hashicorp Vault for that
                DNS_ZONE_NAME = "example.com"
                DNS_RECORD_NAME = "ingress.cluster"  
                DNS_RECORD_TYPE = "A"
                DNS_RECORD_VALUE = "${NOMAD_IP_http}"
                DNS_RECORD_TTL = "60"
            }
        }

        task "traefik" {
            # ...
        }
    }
}
```

## TODO

    - support using existing environment variables:
      i.e. when running with network `mode = "host"` we can use "HOSTNAME"
    - support updating multiple records (i.e. A and AAAA).
