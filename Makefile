export PATH := $(shell go env GOPATH)/bin:$(PATH)

all: clean install

fyne:
ifeq (, $(shell which fyne))
	go get -v fyne.io/fyne/cmd/fyne
else
	@echo "fyne installed"
endif

fyne-cross-bin:
ifeq (, $(shell which fyne-cross))
	go get -v github.com/lucor/fyne-cross/cmd/fyne-cross
else
	@echo "fyne-cross installed"
endif

rsrc:
ifeq (, $(shell which rsrc))
	go get -v github.com/akavel/rsrc
else
	@echo "rsrc installed"
endif

clean:
	rm -rf build

build:
	mkdir -p build

linux: build
	mkdir -p build/linux
	go build -ldflags "-s -w" -trimpath -v -o build/linux/tinc-desktop ./cmd/tinc-desktop
	cd build/linux && tar -zcvf ../tinc-desktop-linux64.tar.gz .

windows: build rsrc
	mkdir -p build/windows
	rm -rf fyne-cross
	rsrc -manifest assets/admin.xml -o tinc-desktop.syso
	go build -ldflags "-s -w -H=windowsgui" -trimpath -v -o build/windows/tinc-desktop.exe ./cmd/tinc-desktop
	rm tinc-desktop.syso
	cp -r assets/windows/. build/windows/
	cd build/windows && zip -r ../tinc-desktop-win64.zip .

darwin: build
	mkdir -p build/darwin
	go build -ldflags "-s -w" -trimpath -v -o build/darwin/tinc-desktop ./cmd/tinc-desktop
	cd build/darwin && tar -zcvf ../tinc-desktop-darwin64.tar.gz .


install: linux windows darwin

.PHONY: all