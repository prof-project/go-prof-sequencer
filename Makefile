all:
	@echo "This is a dummy to prevent running make without explicit target!"

init:
	git submodule update --init --recursive

clean:
	$(MAKE) -C app/ clean
	$(MAKE) -C test/ clean

build:
	$(MAKE) -C app/ build
	$(MAKE) -C test/ build

rebuild: clean init
	$(MAKE) -C app/ rebuild
	$(MAKE) -C test/ rebuild

run:
	$(MAKE) -C app/ run

docker-build:
	docker image build --platform="linux/arm64" -f ./Dockerfile ./app -t prof-project/prof-sequencer

docker-run:
	docker run --rm -it --name prof-sequencer-container prof-project/prof-sequencer