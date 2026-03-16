PACKAGE  := xray-vpn
VERSION  := 0.1.0
ARCH     := amd64
DEB_NAME := $(PACKAGE)_$(VERSION)_$(ARCH).deb
STAGE    := target/deb/$(PACKAGE)_$(VERSION)_$(ARCH)

LDFLAGS := -s -w -buildid=

.PHONY: all build deb clean

all: deb

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpnd ./cmd/xray-vpnd
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpn  ./cmd/xray-vpn

deb: build
	rm -rf $(STAGE)
	install -Dm755 target/xray-vpnd        $(STAGE)/usr/bin/xray-vpnd
	install -Dm755 target/xray-vpn         $(STAGE)/usr/bin/xray-vpn
	install -Dm644 debian/xray-vpn.service         $(STAGE)/lib/systemd/system/xray-vpn.service
	install -Dm644 debian/xray-vpn-refresh.service $(STAGE)/lib/systemd/system/xray-vpn-refresh.service
	install -Dm644 debian/xray-vpn-refresh.timer   $(STAGE)/lib/systemd/system/xray-vpn-refresh.timer
	install -Dm644 debian/config.json 	   $(STAGE)/etc/xray-vpn/config.json
	install -d     						   $(STAGE)/DEBIAN
	install -m755  debian/postinst         $(STAGE)/DEBIAN/postinst
	install -m755  debian/prerm            $(STAGE)/DEBIAN/prerm
	install -m755  debian/postrm           $(STAGE)/DEBIAN/postrm
	cp             debian/control          $(STAGE)/DEBIAN/control
	echo "/etc/xray-vpn/config.json" > $(STAGE)/DEBIAN/conffiles
	dpkg-deb --build --root-owner-group $(STAGE)
	@echo "Package built: target/deb/$(DEB_NAME)"

clean:
	rm -rf target
