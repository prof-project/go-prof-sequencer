all:
	@echo "This is a dummy to prevent running make without explicit target!"

init:
	$(MAKE) -C api/ build

clean:
	$(MAKE) -C api/ clean
	$(MAKE) -C app/ clean

build:
	$(MAKE) -C app/ build

rebuild: clean init
	$(MAKE) -C app/ rebuild
