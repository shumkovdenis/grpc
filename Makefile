run-server:
	go run server/main.go

run-client:
	SERVICE_PORT=50051 go run client/main.go
