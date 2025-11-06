BUILD_DIR=./build
SRC=./cmd
BIN=$(BUILD_DIR)/discover
DISCOVERY_DESKTOP_FILE=$(HOME)/.local/share/applications/discover.desktop

.PHONY: clean void cluster build_discovery install_discovery register_discovery

clean:
	rm -rf $(BUILD_DIR)

void:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/void $(SRC)/void

cluster:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/cluster $(SRC)/cluster

build_discovery:
	@echo "🔧 Building discover..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BIN) $(SRC)/discover

install_discovery: build_discovery
	@echo "📦 Installing binary to /usr/bin/discover (requires sudo)"
	sudo cp $(BIN)/discover /usr/bin/discover
	sudo chmod 755 /usr/bin/discover

register_discovery:
	@echo "🗂️  Installing desktop entry..."
	mkdir -p $(shell dirname $(DISCOVERY_DESKTOP_FILE))
	cp $(SRC)/discover/discover.desktop $(DISCOVERY_DESKTOP_FILE)
	@echo "🖇️  Registering discover:// protocol handler..."
	update-desktop-database $(HOME)/.local/share/applications > /dev/null 2>&1 || true
	xdg-mime default discover.desktop x-scheme-handler/discover
	@echo "✅ discover:// protocol registered for current user."
