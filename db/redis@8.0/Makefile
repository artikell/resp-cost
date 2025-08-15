REDIS_VERSION := 8.0
REDIS_SOURCE_URL := https://github.com/redis/redis.git
BUILD_DIR := vendor/redis
DATA_DIR := data/

.PHONY: all clean

all: download build

download:
	@if [ ! -d "$(BUILD_DIR)/.git" ]; then \
		echo "正在克隆 Redis $(REDIS_VERSION)..."; \
		git clone --branch $(REDIS_VERSION) --depth 1 $(REDIS_SOURCE_URL) $(BUILD_DIR); \
	else \
		echo "已存在仓库，更新版本..."; \
		cd $(BUILD_DIR) && pwd && git fetch origin && \
        git checkout -B $(REDIS_VERSION) origin/$(REDIS_VERSION); \
	fi

build: download
	@echo "编译 Redis..."
	@$(MAKE) -C $(BUILD_DIR) --silent

run: build
	@echo "运行 Redis..."
	@if [ ! -d "$(DATA_DIR)" ]; then \
		rm -rf $(DATA_DIR); \
	fi
	@mkdir -p $(DATA_DIR)
	@cp redis.conf $(DATA_DIR)
	@cp $(BUILD_DIR)/src/redis-server $(DATA_DIR)
	@cd $(DATA_DIR) && ./redis-server redis.conf

clean:
	@echo "清理编译文件..."
	@rm -rf $(BUILD_DIR)/*
	@rm -f $(BUILD_DIR)/.built

# 安装到系统目录（可选）
install: build
	@echo "安装 Redis..."
	@$(MAKE) -C $(BUILD_DIR) install