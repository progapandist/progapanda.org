KUBECONFIG ?= $(HOME)/kubeconfig
export KUBECONFIG

build:
	rm -rf dist
	parcel build src/index.html
	GOOS=linux GOARCH=amd64 go build .
	docker build -t progapandist/progapanda-org .

# Recreating the pods wipes the image cache in each DinD sidecar, so ./hello2
# has to be redeployed from the hello2 repo afterwards.
deploy:
	docker push progapandist/progapanda-org
	kubectl apply -f k8s
	kubectl delete pod -l app.kubernetes.io/name=progapanda-org
