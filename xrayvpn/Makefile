PACKAGE  := xrayvpn
VERSION  := 0.1.0

DEB_ARCH := amd64
DEB_NAME := $(PACKAGE)_$(VERSION)_$(DEB_ARCH).deb
STAGE    := target/deb/$(PACKAGE)_$(VERSION)_$(DEB_ARCH)

LDFLAGS := -s -w -buildid=

.PHONY: xrayconnectord build deb clean

GO_BUILD := CGO_ENABLED=0 GOOS=linux GOARCH=$(DEB_ARCH) go -C

xrayconnectord:
	$(GO_BUILD) xrayconnectord build -trimpath -ldflags="$(LDFLAGS)" -o $(CURDIR)/target/clientrotate  ./cmd/clientrotate
	$(GO_BUILD) xrayconnectord build -trimpath -ldflags="$(LDFLAGS)" -o $(CURDIR)/target/subsrv  ./cmd/subsrv

build:
	$(GO_BUILD) xrayvpnd  build -trimpath -ldflags="$(LDFLAGS)" -o $(CURDIR)/target/xrayvpnd  ./cmd/xrayvpnd
	$(GO_BUILD) xrayvpnd  build -trimpath -ldflags="$(LDFLAGS)" -o $(CURDIR)/target/xrayvpn   ./cmd/xrayvpn
	$(GO_BUILD) tun2socksd build -trimpath -ldflags="$(LDFLAGS)" -o $(CURDIR)/target/tun2socksd  .

deb: build
	rm -rf $(STAGE)
	install -Dm755 target/xrayvpnd        $(STAGE)/usr/bin/xrayvpnd
	install -Dm755 target/tun2socksd      $(STAGE)/usr/bin/tun2socksd
	install -Dm755 target/xrayvpn         $(STAGE)/usr/bin/xrayvpn
	install -Dm644 debian/xrayvpnd.service         $(STAGE)/lib/systemd/system/xrayvpnd.service
	install -Dm644 debian/tun2socksd.service       $(STAGE)/lib/systemd/system/tun2socksd.service
	install -Dm644 debian/xrayvpnd-refresh.service $(STAGE)/lib/systemd/system/xrayvpnd-refresh.service
	install -Dm644 debian/xrayvpnd-refresh.timer   $(STAGE)/lib/systemd/system/xrayvpnd-refresh.timer
	install -d     						   $(STAGE)/etc/xrayvpn
	install -d     						   $(STAGE)/DEBIAN
	install -m755  debian/postinst         $(STAGE)/DEBIAN/postinst
	install -m755  debian/prerm            $(STAGE)/DEBIAN/prerm
	install -m755  debian/postrm           $(STAGE)/DEBIAN/postrm
	cp             debian/control          $(STAGE)/DEBIAN/control
	SOURCE_DATE_EPOCH=0 dpkg-deb --build --root-owner-group $(STAGE)
	@echo "Package built: target/deb/$(DEB_NAME)"

clean:
	rm -rf target
