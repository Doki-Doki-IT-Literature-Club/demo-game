lint:
	go fmt ./...

run_client:
	go run cmd/client/main.go ${addr}

run_server:
	go run cmd/server/main.go ${port}

run_map_editor:
	go run cmd/mapeditor/main.go
