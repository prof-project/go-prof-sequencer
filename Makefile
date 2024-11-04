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
	docker image build --no-cache --platform="linux/amd64" -f ./Dockerfile ./app -t prof-project/prof-sequencer

docker-build-noauth:
	docker image build --no-cache --platform="linux/amd64" --build-arg BUILD_TAGS=noauth -f ./Dockerfile ./app -t prof-project/prof-sequencer-noauth

docker-run:
	docker run --rm -it --name prof-sequencer-container prof-project/prof-sequencer

start-testserver:
	$(MAKE) -C test/ rebuild
	$(MAKE) -C test/ start-testserver