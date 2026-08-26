KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

PLATFORM := linux/amd64

.PHONY: build deploy

build:
	rm -rf dist
	docker run --rm \
		-v "$(CURDIR):/app" \
		-v /app/node_modules \
		-w /app \
		node:16-bullseye \
		sh -c 'yarn install --frozen-lockfile && yarn run parcel build src/index.html'
	GOOS=linux GOARCH=amd64 go build .
	docker build --platform $(PLATFORM) -t progapandist/progapanda-org .

# Recreating the pods wipes the image cache in each DinD sidecar, so ./hello2
# has to be redeployed from the hello2 repo afterwards.
deploy: build
	docker push progapandist/progapanda-org
	kubectl apply -f k8s
	kubectl delete pod -l app.kubernetes.io/name=progapanda-org
