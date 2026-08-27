KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

HELLO2_DIR ?= ../hello2
VISITOR_IMAGE := progapandist/hello
PLATFORM := linux/amd64

.PHONY: build frontend visitor-image dev deploy

frontend:
	rm -rf dist
	docker build -f Dockerfile.frontend --output type=local,dest=dist .

visitor-image:
	@test -f "$(HELLO2_DIR)/entrypoint.sh" || { \
		echo "hello2 not found at $(HELLO2_DIR); set HELLO2_DIR=/path/to/hello2" >&2; \
		exit 1; \
	}
	docker build -t $(VISITOR_IMAGE) $(HELLO2_DIR)

dev: frontend visitor-image
	go run -mod=mod .

build: frontend
	GOOS=linux GOARCH=amd64 go build -mod=mod .
	docker build --platform $(PLATFORM) -t progapandist/progapanda-org .

# Recreating the pods wipes the image cache in each DinD sidecar, so ./hello2
# has to be redeployed from the hello2 repo afterwards.
deploy: build
	docker push progapandist/progapanda-org
	kubectl apply -f k8s
	kubectl delete pod -l app.kubernetes.io/name=progapanda-org
