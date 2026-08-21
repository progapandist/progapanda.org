KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

BINARY  := hello2
# Cross-compiled for the container, so it lives in dist/ and not in the repo
# root — it is a linux/amd64 ELF and will not run on your Mac. Use `make run`.
DIST    := dist/$(BINARY)
IMAGE   := progapandist/hello
# Pristine upstream image, by digest. Deploy layers on this rather than on
# whatever the tag currently points at, so repeated deploys never stack.
BASE    := progapandist/hello@sha256:930112104e1442e2ea8adb6503c11822b2a37e12724227f1cf91cface830525b
PLATFORM:= linux/amd64
SELECTOR:= app.kubernetes.io/name=progapanda-org

.PHONY: run build test clean deploy

run: ## Run the TUI locally
	go run .

test:
	go test ./...

build: test ## Cross-compile for the container (linux/amd64)
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(DIST) .

clean:
	rm -rf dist

# Inject the binary into the Docker-in-Docker sidecar of every web pod and
# rebuild the image tag the daemon serves to visitors. No registry involved:
# the base image is already cached inside each DinD daemon.
# ponytail: emptyDir storage, so this is lost on pod restart. Push the image
# to Docker Hub instead if it needs to survive.
deploy: build
	@pods=$$(kubectl get pods -l $(SELECTOR) -o name) || exit 1; \
	test -n "$$pods" || { echo "no pods match $(SELECTOR) (KUBECONFIG=$$KUBECONFIG)" >&2; exit 1; }; \
	for pod in $$pods; do \
		pod=$${pod#pod/}; \
		echo "==> $$pod"; \
		kubectl cp $(DIST) $$pod:/tmp/$(BINARY) -c dind-daemon || exit 1; \
		kubectl exec $$pod -c dind-daemon -- sh -c '\
			docker pull -q $(BASE) && \
			cd /tmp && printf "FROM $(BASE)\nCOPY $(BINARY) /app/$(BINARY)\n" > Dockerfile.$(BINARY) && \
			chmod +x $(BINARY) && \
			docker build -q -f Dockerfile.$(BINARY) -t $(IMAGE) . ' || exit 1; \
	done; \
	echo "Deployed. New sessions can run ./$(BINARY)"
