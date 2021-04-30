dev:
	docker build . -t infrahq/infra
	kubectl apply -f deploy/docker-desktop.yaml 
	kubectl delete pods --all --namespace infra
