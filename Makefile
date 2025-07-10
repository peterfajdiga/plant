INSTALL_PATH := ~/.local/bin

.PHONY: *

build:
	cd src && go build -o ../build/plant

install: build
	cp ./build/plant ${INSTALL_PATH}
