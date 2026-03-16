PACKAGE  := xray-vpn
VERSION  := 0.1.0

DEB_ARCH := amd64
DEB_NAME := $(PACKAGE)_$(VERSION)_$(DEB_ARCH).deb
STAGE    := target/deb/$(PACKAGE)_$(VERSION)_$(DEB_ARCH)

IPK_NAME  := $(PACKAGE)_$(VERSION)-1_all.ipk
IPK_STAGE := target/ipk/$(PACKAGE)

LDFLAGS := -s -w -buildid=v0.0.1

.PHONY: all build deb openwrt openwrt-pkg clean

all: deb

build_mipsle:
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpnd-mipsle ./cmd/xray-vpnd
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpn-mipsle  ./cmd/xray-vpn
	upx --best --lzma target/xray-vpnd-mipsle
	upx --best --lzma target/xray-vpn-mipsle
	@echo "Compressed binaries:"
	@ls -lh target/xray-vpnd-mipsle target/xray-vpn-mipsle


openwrt: build_mipsle
	rm -rf $(IPK_STAGE)
	# data tree
	install -Dm755 openwrt/files/xray-vpn.init       $(IPK_STAGE)/data/etc/init.d/xray-vpn
	install -Dm755 openwrt/files/xray-vpn-download   $(IPK_STAGE)/data/usr/bin/xray-vpn-download
	install -Dm644 /dev/null                          $(IPK_STAGE)/data/etc/xray-vpn/state.json
	# control tree
	install -d $(IPK_STAGE)/control
	printf 'Package: $(PACKAGE)\nVersion: $(VERSION)-1\nDepends: ca-certificates, wget\nSection: net\nArchitecture: all\nMaintainer: Gleb Ivanov <realglebivanov@gmail.com>\nDescription: Xray TUN VPN with automatic route management\n' \
		> $(IPK_STAGE)/control/control
	printf '/etc/xray-vpn/state.json\n' > $(IPK_STAGE)/control/conffiles
	printf '#!/bin/sh\n[ -n "$${IPKG_INSTROOT}" ] || /etc/init.d/xray-vpn enable\n' > $(IPK_STAGE)/control/postinst
	printf '#!/bin/sh\n/etc/init.d/xray-vpn stop 2>/dev/null\n/etc/init.d/xray-vpn disable\n' > $(IPK_STAGE)/control/prerm
	chmod 755 $(IPK_STAGE)/control/postinst $(IPK_STAGE)/control/prerm
	# assemble ipk
	echo "2.0" > $(IPK_STAGE)/debian-binary
	cd $(IPK_STAGE)/data    && tar czf ../data.tar.gz .
	cd $(IPK_STAGE)/control && tar czf ../control.tar.gz .
	cd $(IPK_STAGE) && ar rc ../$(IPK_NAME) debian-binary control.tar.gz data.tar.gz
	@echo "Package built: target/ipk/$(IPK_NAME)"

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpnd ./cmd/xray-vpnd
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpn  ./cmd/xray-vpn

deb: build
	rm -rf $(STAGE)
	install -Dm755 target/xray-vpnd        $(STAGE)/usr/bin/xray-vpnd
	install -Dm755 target/xray-vpn         $(STAGE)/usr/bin/xray-vpn
	install -Dm644 debian/xray-vpn.service         $(STAGE)/lib/systemd/system/xray-vpn.service
	install -Dm644 debian/xray-vpn-refresh.service $(STAGE)/lib/systemd/system/xray-vpn-refresh.service
	install -Dm644 debian/xray-vpn-refresh.timer   $(STAGE)/lib/systemd/system/xray-vpn-refresh.timer
	install -Dm644 /dev/null		 	   $(STAGE)/etc/xray-vpn/state.json
	install -d     						   $(STAGE)/DEBIAN
	install -m755  debian/postinst         $(STAGE)/DEBIAN/postinst
	install -m755  debian/prerm            $(STAGE)/DEBIAN/prerm
	install -m755  debian/postrm           $(STAGE)/DEBIAN/postrm
	cp             debian/control          $(STAGE)/DEBIAN/control
	echo "/etc/xray-vpn/state.json" > $(STAGE)/DEBIAN/conffiles
	dpkg-deb --build --root-owner-group $(STAGE)
	@echo "Package built: target/deb/$(DEB_NAME)"

clean:
	rm -rf target
