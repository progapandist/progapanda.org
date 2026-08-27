KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

IMAGE := progapandist/progapanda-org
VISITOR_IMAGE := progapandist/hello2
# Built for this machine, for `make dev`. A separate tag on purpose: a native
# build must never end up on the tag the cluster pulls.
VISITOR_DEV_IMAGE := progapandist/hello2:dev
PLATFORM := linux/amd64
DEPLOYMENT := deployment/progapanda-org
SELECTOR := app.kubernetes.io/name=progapanda-org

.PHONY: test frontend visitor-image dev run-tui build deploy deploy-tui clean

test:
	go test ./...

frontend:
	rm -rf dist
	docker build -f Dockerfile.frontend --output type=local,dest=dist .

# Native architecture, so it runs on this machine. The deploy builds the same
# Dockerfile for the cluster's architecture and pushes it.
visitor-image:
	docker build -f Dockerfile.visitor -t $(VISITOR_DEV_IMAGE) .

dev: frontend visitor-image ## Serve the whole site on :4567
	VISITOR_IMAGE=$(VISITOR_DEV_IMAGE) go run ./cmd/webterm

run-tui: ## Run the TUI on its own, in this terminal
	go run ./cmd/hello2

build: test frontend
	GOOS=linux GOARCH=amd64 go build -o webterm ./cmd/webterm
	docker build --platform $(PLATFORM) -t $(IMAGE) .
	docker build --platform $(PLATFORM) -f Dockerfile.visitor -t $(VISITOR_IMAGE) .

# Both images go to the registry. Rolling the pods recreates the
# Docker-in-Docker sidecars with empty image caches, and each one pulls the
# visitor image back for itself — see the readiness probe in k8s/deployment.yaml,
# which is what stops a pod taking traffic before it has one.
deploy: build
	@# An arm64 image on this tag fails as "exec format error" inside the
	@# cluster, long after the push looked fine.
	@docker image inspect $(VISITOR_IMAGE) --format '{{.Architecture}}' | grep -qx amd64 \
		|| { echo "$(VISITOR_IMAGE) is not amd64 — refusing to push" >&2; \
		     docker image inspect $(VISITOR_IMAGE) --format '  is: {{.Architecture}}' >&2; exit 1; }
	docker push $(IMAGE)
	docker push $(VISITOR_IMAGE)
	kubectl apply -f k8s
	kubectl rollout restart $(DEPLOYMENT)
	kubectl rollout status $(DEPLOYMENT) --timeout=300s

clean:
	rm -rf build webterm
