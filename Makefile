INSTALL_PATH := ~/.local/bin

.PHONY: *

build:
	cd src && go build -o ../build/iplan

install: build
	cp ./build/iplan ${INSTALL_PATH}
