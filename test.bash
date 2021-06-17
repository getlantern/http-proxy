#! /bin/bash

COMPOSE_FILE=docker-compose-redis.yml

function fail() {
    echo $1
    exit 1
}

function tearDown() {
    echo "Shutting down test cluster"
    docker compose -f $COMPOSE_FILE down
}

function printLogs() {
    if [ "$1" == "true" ]; then
        echo "Test cluster logs:"
        docker compose -f $COMPOSE_FILE logs sentinel
        docker compose -f $COMPOSE_FILE logs redis
    fi
}

trap tearDown EXIT

echo "Starting test cluster via Docker compose."
docker compose -f $COMPOSE_FILE up -d || "Failed to start test cluster"
go test -race ./... || printLogs $1 && exit 1