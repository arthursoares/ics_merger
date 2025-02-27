.PHONY: build run run-local run-server docker-build docker-run clean

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

clean:
	rm -rf bin/ output/
	docker-compose down