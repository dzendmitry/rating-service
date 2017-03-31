VERSION ?= 0.0.1
SERVICE ?= kinopoisk-service

build:
	env GOOS=linux GOARCH=amd64 go build .

docker: build
	docker build --no-cache=true -t dzendmitry/$(SERVICE):$(VERSION) -f ./Dockerfile .

push: docker
	docker push dzendmitry/$(SERVICE):$(VERSION)

clean:
	rm $(SERVICE)