.PHONY: build run run-local run-server docker-build docker-run clean test test-ruby

build:
	go build -o bin/ical_merger ./cmd

run: build
	./bin/ical_merger

run-local: build
	./bin/ical_merger -local -calendar-dir=./calendars

run-server: build
	./bin/ical_merger -serve

docker-build:
	docker build -t ical_merger .

docker-run:
	docker-compose up -d

docker-logs:
	docker-compose logs -f

test: build
	go test ./...

test-ruby: build
	ruby tests/validate_ical_with_ruby.rb

test-all: test test-ruby

clean:
	rm -rf bin/ output/
	docker-compose down