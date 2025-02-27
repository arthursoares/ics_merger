.PHONY: build run docker-build docker-run clean

build:
	go build -o bin/ical_merger ./cmd

run: build
	./bin/ical_merger

docker-build:
	docker build -t ical_merger .

docker-run:
	docker-compose up -d

docker-logs:
	docker-compose logs -f

clean:
	rm -rf bin/
	docker-compose down