VERSION ?= latest
IMAGE ?= hetzner-dns-updater
REGISTRY ?= ghcr.io/ppacher

.PHONY: container push

container:
	docker build -t $(REGISTRY)/$(IMAGE):$(VERSION) .

push: container
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)