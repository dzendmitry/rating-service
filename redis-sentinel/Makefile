VERSION ?= 0.0.1
SERVICE ?= redis-sentinel

docker:
	docker build --no-cache=true -t dzendmitry/$(SERVICE):$(VERSION) -f ./Dockerfile .

push: docker
	docker push dzendmitry/$(SERVICE):$(VERSION)