KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

HELLO2_DIR ?= ../hello2
IMAGE := progapandist/progapanda-org
VISITOR_IMAGE := progapandist/hello
PLATFORM := linux/amd64
DEPLOYMENT := deployment/progapanda-org

.PHONY: build frontend check-hello2 visitor-image dev deploy

frontend:
	rm -rf dist
	docker build -f Dockerfile.frontend --output type=local,dest=dist .

check-hello2:
	@test -f "$(HELLO2_DIR)/entrypoint.sh" || { \
		echo "hello2 not found at $(HELLO2_DIR); set HELLO2_DIR=/path/to/hello2" >&2; \
		exit 1; \
	}

visitor-image: check-hello2
	docker build -t $(VISITOR_IMAGE) $(HELLO2_DIR)

dev: frontend visitor-image
	go run .

build: frontend
	GOOS=linux GOARCH=amd64 go build .
	docker build --platform $(PLATFORM) -t $(IMAGE) .

# check-hello2 runs first, and deliberately: new pods come up with empty DinD
# image caches, so a deploy that cannot reach the hello2 repo would leave
# visitors on the stale upstream image. Better to fail before rolling than
# after.
deploy: check-hello2 build
	docker push $(IMAGE)
	kubectl apply -f k8s
	kubectl rollout restart $(DEPLOYMENT)
	kubectl rollout status $(DEPLOYMENT) --timeout=300s
	$(MAKE) -C $(HELLO2_DIR) deploy
