controller-run:
	go run ./cmd/controller/main.go -config ./deploy/controller.yaml

build:
	docker build -t zhuwt/controller .


alertengine-run:
	go run ./cmd/alertengine/main.go -config ./deploy/alertengine.yaml