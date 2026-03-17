PACKAGE  := xray-vpn
VERSION  := 0.1.0

DEB_ARCH := amd64
DEB_NAME := $(PACKAGE)_$(VERSION)_$(DEB_ARCH).deb
STAGE    := target/deb/$(PACKAGE)_$(VERSION)_$(DEB_ARCH)

APK_NAME  := $(PACKAGE)-$(VERSION)-r1.apk
APK_STAGE := target/apk/$(PACKAGE)

LDFLAGS := -s -w -buildid=v0.0.1

.PHONY: all build deb openwrt clean

all: deb

build_mipsle:
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpnd-mipsle ./cmd/xray-vpnd
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -trimpath -ldflags="$(LDFLAGS)" -o target/xray-vpn-mipsle  ./cmd/xray-vpn
	upx --best --lzma target/xray-vpnd-mipsle
	upx --best --lzma target/xray-vpn-mipsle
	@echo "Compressed binaries:"
	@ls -lh target/xray-vpnd-mipsle target/xray-vpn-mipsle


openwrt: build_mipsle
	rm -rf $(APK_STAGE)
	# data tree
	install -Dm755 openwrt/files/xray-vpn.init       $(APK_STAGE)/root/etc/init.d/xray-vpn
	install -Dm755 openwrt/files/xray-vpn-download   $(APK_STAGE)/root/usr/bin/xray-vpn-download
	install -Dm644 /dev/null                          $(APK_STAGE)/root/etc/xray-vpn/state.json
	# control metadata
	printf 'pkgname = $(PACKAGE)\npkgver = $(VERSION)-r1\npkgdesc = Xray TUN VPN with automatic route management\narch = all\nmaintainer = Gleb Ivanov <realglebivanov@gmail.com>\ndepend = ca-certificates\ndepend = wget\n' \
		> $(APK_STAGE)/.PKGINFO
	# lifecycle scripts
	printf '#!/bin/sh\n/etc/init.d/xray-vpn enable\n' > $(APK_STAGE)/.post-install
	printf '#!/bin/sh\n/etc/init.d/xray-vpn stop 2>/dev/null\n/etc/init.d/xray-vpn disable\n' > $(APK_STAGE)/.pre-deinstall
	chmod 755 $(APK_STAGE)/.post-install $(APK_STAGE)/.pre-deinstall
	# assemble apk
	cd $(APK_STAGE) && tar czf ../$(APK_NAME) .PKGINFO .post-install .pre-deinstall -C root .
	@echo "Package built: target/apk/$(APK_NAME)"

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
