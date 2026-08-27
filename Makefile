KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

IMAGE := progapandist/progapanda-org
VISITOR_IMAGE := progapandist/hello
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
	docker build -f Dockerfile.visitor -t $(VISITOR_IMAGE) .

dev: frontend visitor-image ## Serve the whole site on :4567
	go run ./cmd/webterm

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
	docker push $(IMAGE)
	docker push $(VISITOR_IMAGE)
	kubectl apply -f k8s
	kubectl rollout restart $(DEPLOYMENT)
	kubectl rollout status $(DEPLOYMENT) --timeout=300s

clean:
	rm -rf build webterm
