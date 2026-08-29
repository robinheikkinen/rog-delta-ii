.PHONY: build test run install deb clean

VERSION ?= 0.1.0
ARCH ?= $(shell dpkg --print-architecture 2>/dev/null || echo amd64)

build:
	go build -buildvcs=false -buildmode=pie -ldflags "-s -w -X main.version=$(VERSION)" -o bin/rogdelta ./cmd/rogdelta
test:
	go test ./...
run:
	go run ./cmd/rogdelta serve
install: build
	install -Dm755 bin/rogdelta $(HOME)/.local/bin/rogdelta
	install -Dm644 packaging/rogdelta.service $(HOME)/.config/systemd/user/rogdelta.service
	sed -i 's|@BINDIR@|$(HOME)/.local/bin|' $(HOME)/.config/systemd/user/rogdelta.service
	install -Dm644 packaging/rogdelta.desktop $(HOME)/.local/share/applications/rogdelta.desktop
	sed -i 's|@BINDIR@|$(HOME)/.local/bin|' $(HOME)/.local/share/applications/rogdelta.desktop
	install -Dm644 packaging/rogdelta.svg $(HOME)/.local/share/icons/hicolor/scalable/apps/rogdelta.svg
	systemctl --user daemon-reload
	-update-desktop-database $(HOME)/.local/share/applications
	@echo "ROG Delta II is now available in the application menu"

deb: test build
	rm -rf build/deb
	install -d -m755 build/deb/DEBIAN build/deb/usr/bin build/deb/usr/share/applications
	install -d -m755 build/deb/usr/share/icons/hicolor/scalable/apps build/deb/usr/lib/systemd/user
	install -d -m755 build/deb/usr/lib/udev/rules.d build/deb/usr/share/metainfo
	install -d -m755 build/deb/usr/share/doc/rogdelta build/deb/usr/share/man/man1
	install -m755 bin/rogdelta build/deb/usr/bin/rogdelta
	install -m644 packaging/rogdelta.desktop build/deb/usr/share/applications/rogdelta.desktop
	sed -i 's|@BINDIR@|/usr/bin|' build/deb/usr/share/applications/rogdelta.desktop
	install -m644 packaging/rogdelta.svg build/deb/usr/share/icons/hicolor/scalable/apps/rogdelta.svg
	install -m644 packaging/rogdelta.metainfo.xml build/deb/usr/share/metainfo/io.github.morten.rogdelta.metainfo.xml
	install -m644 packaging/rogdelta.service build/deb/usr/lib/systemd/user/rogdelta.service
	sed -i 's|@BINDIR@|/usr/bin|' build/deb/usr/lib/systemd/user/rogdelta.service
	install -m644 packaging/99-rog-delta-ii.rules build/deb/usr/lib/udev/rules.d/99-rog-delta-ii.rules
	install -m644 packaging/debian/control build/deb/DEBIAN/control
	sed -i 's|@VERSION@|$(VERSION)|; s|@ARCH@|$(ARCH)|' build/deb/DEBIAN/control
	install -m755 packaging/debian/postinst packaging/debian/postrm build/deb/DEBIAN/
	install -m644 packaging/debian/copyright build/deb/usr/share/doc/rogdelta/copyright
	gzip -9n -c packaging/debian/changelog > build/deb/usr/share/doc/rogdelta/changelog.gz
	gzip -9n -c packaging/debian/rogdelta.1 > build/deb/usr/share/man/man1/rogdelta.1.gz
	chmod -R u=rwX,go=rX build/deb
	install -d -m755 dist
	dpkg-deb --root-owner-group --build build/deb dist/rogdelta_$(VERSION)_$(ARCH).deb

clean:
	rm -rf bin build dist
