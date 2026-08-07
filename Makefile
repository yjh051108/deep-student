# DeepStudent (Go) Makefile

.PHONY: help tidy build dev test lint run migrate clean installer nsis nsis-check

GO          ?= go
APP         ?= deepstudent
BINDIR      ?= build/bin
MAKENSIS    ?= makensis
VERSION     ?= 1.0.0

help:
	@echo "Targets:"
	@echo "  tidy        - go mod tidy"
	@echo "  dev         - wails dev (frontend hot reload)"
	@echo "  build       - wails build (current platform)"
	@echo "  test        - go test ./..."
	@echo "  lint        - golangci-lint run"
	@echo "  run         - go run ./cmd/$(APP)"
	@echo "  migrate     - go run ./cmd/migrate --from <old> --to <new>"
	@echo "  installer   - build Windows installer (DeepStudent-Setup-$(VERSION).exe)"
	@echo "  nsis-check  - syntax-check installer.nsi without producing output"
	@echo "  clean       - remove build artifacts"

tidy:
	$(GO) mod tidy

dev:
	wails dev

build:
	wails build -clean -o $(BINDIR)/$(APP).exe

test:
	$(GO) test ./... -count=1

lint:
	golangci-lint run

run:
	$(GO) run ./cmd/$(APP)

migrate:
	$(GO) run ./cmd/migrate --from $(FROM) --to $(TO)

# 仅做 NSIS 语法检查（不产出安装包，不要求 build/bin 存在）
nsis-check:
	@cmd /c "if exist $(MAKENSIS) ( $(MAKENSIS) /NOUNLOAD /DVERSION=$(VERSION) cmd/installer/installer.nsi ) else ( echo [nsis] makensis not found, skipping dry-run )"

# 完整产出安装包：先 wails 构建，再用 NSIS 包装
installer: build
	@cmd /c "if exist $(MAKENSIS) ( $(MAKENSIS) /DVERSION=$(VERSION) /V2 cmd/installer/installer.nsi ) else ( echo [nsis] makensis not found, please install NSIS and re-run )"
	@echo "Installer -> build\installer\DeepStudent-Setup-$(VERSION).exe"

clean:
	rm -rf $(BINDIR) build/installer
