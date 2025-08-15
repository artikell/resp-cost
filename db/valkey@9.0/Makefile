SERVER_VERSION := 9.0
SERVER_SOURCE_URL := https://github.com/valkey-io/valkey.git
BUILD_DIR := vendor/valkey
DATA_DIR := data/

.PHONY: all clean

all: download build

download:
	@if [ ! -d "$(BUILD_DIR)/.git" ]; then \
		echo "正在克隆 $(SERVER_VERSION)..."; \
		git clone --branch $(SERVER_VERSION) --depth 1 $(SERVER_SOURCE_URL) $(BUILD_DIR); \
	else \
		echo "已存在仓库，更新版本..."; \
		cd $(BUILD_DIR) && pwd && git fetch origin && \
        git checkout -B $(SERVER_VERSION) origin/$(SERVER_VERSION); \
	fi

build: download
	@echo "编译..."
	@$(MAKE) -C $(BUILD_DIR) --silent

run: build
	@echo "运行..."
	@if [ ! -d "$(DATA_DIR)" ]; then \
		rm -rf $(DATA_DIR); \
	fi
	@mkdir -p $(DATA_DIR)
	@cp valkey.conf $(DATA_DIR)
	@cp $(BUILD_DIR)/src/valkey-server $(DATA_DIR)
	@cd $(DATA_DIR) && ./valkey-server valkey.conf

clean:
	@echo "清理编译文件..."
	@rm -rf $(BUILD_DIR)/*
	@rm -f $(BUILD_DIR)/.built

# 安装到系统目录（可选）
install: build
	@echo "安装..."
	@$(MAKE) -C $(BUILD_DIR) install