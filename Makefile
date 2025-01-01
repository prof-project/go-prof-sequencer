all:
	@echo "This is a dummy to prevent running make without explicit target!"

init:
	git submodule update --init --recursive

clean:
	$(MAKE) -C app/ clean
	$(MAKE) -C test/ clean

build:
	$(MAKE) -C app/ build

rebuild: clean init
	$(MAKE) -C app/ rebuild

run:
	$(MAKE) -C app/ run

docker-build:
	docker image build --platform="linux/amd64" -f ./Dockerfile ./app -t prof-project/prof-sequencer

docker-build-noauth:
	docker image build --platform="linux/amd64" --build-arg BUILD_TAGS=noauth -f ./Dockerfile ./app -t prof-project/prof-sequencer-noauth

docker-run:
	docker run --rm -it -p 8084:80 --name prof-sequencer-container prof-project/prof-sequencer

docker-run-noauth:
	docker run --rm -it -p 8084:80 --name prof-sequencer-container prof-project/prof-sequencer-noauth

docker-build-run-test: docker-stop
	docker network create prof-network || true
	docker image build --platform="linux/amd64" -f ./Dockerfile ./app -t prof-project/prof-sequencer
	docker image build --platform="linux/amd64" -f ./test/bundlemerger/Dockerfile ./test/bundlemerger/ -t prof-project/prof-testserver
	docker run --rm -d --network prof-network --name prof-testserver-container prof-project/prof-testserver
	docker run --rm -d --network prof-network -p 8084:80 --name prof-sequencer-container prof-project/prof-sequencer --grpc-url=prof-testserver-container:50051

docker-build-run-test-noauth: docker-stop
	docker network create prof-network || true
	docker image build --platform="linux/amd64" --build-arg BUILD_TAGS=noauth -f ./Dockerfile ./app -t prof-project/prof-sequencer-noauth
	docker image build --platform="linux/amd64" -f ./test/bundlemerger/Dockerfile ./test/bundlemerger/ -t prof-project/prof-testserver
	docker run --rm -d --network prof-network --name prof-testserver-container prof-project/prof-testserver
	docker run --rm -d --network prof-network -p 8084:80 --name prof-sequencer-container prof-project/prof-sequencer-noauth --grpc-url=prof-testserver-container:50051

docker-stop:
	docker stop prof-sequencer-container || true
	docker stop prof-testserver-container || true
	docker network rm prof-network || true

start-testserver:
	$(MAKE) -C test/ rebuild
	$(MAKE) -C test/ start-testserver