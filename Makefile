KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

IMAGE := progapandist/progapanda-org
VISITOR_IMAGE := progapandist/hello
# Pristine upstream visitor image, by digest. deploy-tui layers on this rather
# than on whatever the tag currently points at, so repeated deploys never stack.
VISITOR_BASE := progapandist/hello@sha256:930112104e1442e2ea8adb6503c11822b2a37e12724227f1cf91cface830525b
PLATFORM := linux/amd64
DEPLOYMENT := deployment/progapanda-org
SELECTOR := app.kubernetes.io/name=progapanda-org

.PHONY: test frontend visitor-image dev run-tui build deploy deploy-tui clean

test:
	go test ./...

frontend:
	rm -rf dist
	docker build -f Dockerfile.frontend --output type=local,dest=dist .

visitor-image:
	docker build -f Dockerfile.visitor -t $(VISITOR_IMAGE) .

dev: frontend visitor-image ## Serve the whole site on :4567
	go run ./cmd/webterm

run-tui: ## Run the TUI on its own, in this terminal
	go run ./cmd/hello2

build: test frontend
	GOOS=linux GOARCH=amd64 go build -o webterm ./cmd/webterm
	docker build --platform $(PLATFORM) -t $(IMAGE) .

# Rolling the pods recreates the Docker-in-Docker sidecars, whose image cache is
# an emptyDir, so the visitor image is wiped every time. deploy-tui puts it back.
deploy: build
	docker push $(IMAGE)
	kubectl apply -f k8s
	kubectl rollout restart $(DEPLOYMENT)
	kubectl rollout status $(DEPLOYMENT) --timeout=300s
	$(MAKE) deploy-tui

# Inject the visitor payload into the Docker-in-Docker sidecar of every ready
# web pod, then rebuild the image tag served to visitors. No registry involved:
# the base image is already cached inside each DinD daemon.
#
# Pods still terminating from a rollout are skipped — they stay listed for a
# while after the rollout reports done, and copying into one fails. Ready alone
# is not enough, since a terminating pod stays Ready for a bit; the absent
# deletionTimestamp (a two-field line) is what marks a pod as staying.
#
# ponytail: emptyDir storage, so this is lost on pod restart. Push the image to
# Docker Hub instead if it needs to survive.
deploy-tui: payload
	@pods=$$(kubectl get pods -l $(SELECTOR) \
		-o jsonpath='{range .items[*]}{.metadata.name} {.status.conditions[?(@.type=="Ready")].status} {.metadata.deletionTimestamp}{"\n"}{end}' \
		| awk 'NF == 2 && $$2 == "True" { print $$1 }') || exit 1; \
	test -n "$$pods" || { echo "no ready pods match $(SELECTOR) (KUBECONFIG=$$KUBECONFIG)" >&2; exit 1; }; \
	for pod in $$pods; do \
		echo "==> $$pod"; \
		kubectl exec $$pod -c dind-daemon -- rm -rf /tmp/payload || exit 1; \
		kubectl cp build/payload $$pod:/tmp/payload -c dind-daemon || exit 1; \
		kubectl exec $$pod -c dind-daemon -- sh -c '\
			docker pull -q $(VISITOR_BASE) && \
			printf "FROM $(VISITOR_BASE)\nCOPY . /app/\n" > /tmp/Dockerfile.payload && \
			docker build -q -f /tmp/Dockerfile.payload -t $(VISITOR_IMAGE) /tmp/payload ' || exit 1; \
	done; \
	echo "Deployed. New sessions start with ./hello2 pre-filled."

# Everything that goes into the visitor container, staged in one directory so
# the deploy is a single copy per pod. Built to build/ and not dist/: dist/ is
# the frontend bundle, and these are linux/amd64 and will not run on a Mac.
payload: test
	@rm -rf build/payload && mkdir -p build/payload
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/payload/hello2 ./cmd/hello2
	docker build --platform $(PLATFORM) -f Dockerfile.visitor --target stripeek-export --output type=local,dest=build/payload .
	cp visitor/entrypoint.sh visitor/canihackit.hack visitor/stripeek visitor/stripeek-history.json build/payload/
	chmod +x build/payload/hello2 build/payload/stripeek build/payload/stripeek.bin build/payload/entrypoint.sh
	@# The cluster is amd64. An arm64 binary here does not fail loudly: exec
	@# returns ENOEXEC and the shell tries to read the ELF as a script.
	@for b in hello2 stripeek.bin; do \
		file build/payload/$$b | grep -q x86-64 || { \
			echo "build/payload/$$b is not x86-64:" >&2; \
			file build/payload/$$b >&2; exit 1; }; \
	done
	@echo "payload staged, both binaries x86-64"

clean:
	rm -rf build webterm
