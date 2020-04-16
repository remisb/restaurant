SHELL := /bin/bash

export PROJECT = restaurant-api

# all: restaurant-api metrics

keys:
	go run ./cmd/restaurant-admin/main.go keygen private.pem

admin:
	go run ./cmd/restaurant-admin/main.go --db-disable-tls=1 useradd admin@example.com gophers

migrate:
	go run ./cmd/restaurant-admin/main.go --db-disable-tls=1 migrate

seed: migrate
	go run ./cmd/restaurant-admin/main.go --db-disable-tls=1 seed


# restaurant-api:
#     docker build \
#         -f dockerfile.restaurant-api \
#         -t gcr.io/$(PROJECT)/restaurant-api-amd64:1.0 \
#         --build-arg PACKAGE_NAME=sales-api \
#         --build-arg PACKAGE_PREFIX=sidecar/ \
#         --build-arg VCS_REF=`git rev-parse HEAD` \
#         --build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` .

metrics:
	docker build \
		-f dockerfile.metrics \
		-t gcr.io/$(PROJECT)/metrics-amd64:1.0 \
		--build-arg PACKAGE_NAME=metrics \
		--build-arg PACKAGE_PREFIX=sidecar/ \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` .


up:
	docker-compose up

down:
	docker-compose down

test:
	go test ./... -count=1

clean:
	docker system prune -f

stop-all:
	docker stop $(docker ps -aq)

remove-all:
	docker rm $(docker ps -aq)

tidy:
	go mod tidy
	go mod vendor

deps-upgrade:
	# go get $(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	go get -u -t -d -v ./...

deps-cleancache:
	go clean -modcache

