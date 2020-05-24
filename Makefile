build:
	rm -rf dist
	parcel build src/index.html
	GOOS=linux GOARCH=amd64 go build .
	docker build -t progapandist/progapanda-org .

deploy:
	docker push progapandist/progapanda-org
	export KUBECONFIG=/Users/andybarnov/code/kubeconfig
	kubectl apply -f k8s
	kubectl delete pod -l app.kubernetes.io/name=progapanda-org