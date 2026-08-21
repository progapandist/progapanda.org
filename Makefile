BINARY  := hello2
IMAGE   := progapandist/hello
PLATFORM:= linux/amd64
PODS    := $(shell kubectl get pods -l app.kubernetes.io/name=progapanda-org -o name)

.PHONY: run build test clean deploy

run: ## Run the TUI locally
	go run .

test:
	go test ./...

build: test ## Cross-compile for the container (linux/amd64)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

clean:
	rm -f $(BINARY)

# Inject the binary into the Docker-in-Docker sidecar of every web pod and
# rebuild the image tag the daemon serves to visitors. No registry involved:
# the base image is already cached inside each DinD daemon.
# ponytail: emptyDir storage, so this is lost on pod restart. Push the image
# to Docker Hub instead if it needs to survive.
deploy: build
	@for pod in $(PODS); do \
		echo "==> $$pod"; \
		kubectl cp $(BINARY) $${pod#pod/}:/tmp/$(BINARY) -c dind-daemon; \
		kubectl exec $${pod#pod/} -c dind-daemon -- sh -c '\
			cd /tmp && printf "FROM $(IMAGE)\nCOPY $(BINARY) /app/$(BINARY)\n" > Dockerfile.$(BINARY) && \
			docker build -q -f Dockerfile.$(BINARY) -t $(IMAGE) . '; \
	done
	@echo "Deployed. New sessions can run ./$(BINARY)"
