run-server:
	go run main.go

run-client:
	SERVICE_PORT=50051 go run main.go --client

kube-apply:
	kubectl apply -f ./k8s
