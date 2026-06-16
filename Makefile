BINARY := wtt
MODULE := wtt
GO := go
INSTALL_DIR ?= $(HOME)/.local/bin
INSTALL_BIN := $(INSTALL_DIR)/$(BINARY)
HOME_DIR ?= $(HOME)
USER_SHELL ?= $(shell printf '%s' "$$SHELL")

ifneq ($(strip $(SUDO_USER)$(SUDO_UID)),)
$(error Do not run make with sudo. Run 'make install' as your normal user; it installs to $(HOME)/.local/bin)
endif

.DEFAULT_GOAL := all
.PHONY: all build dev build-release test test-cover vet fmt lint clean install uninstall

all: build

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd

dev:
	$(GO) build -o $(BINARY) ./cmd

build-release: build

test:
	$(GO) test ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -d .; exit 1; }

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out

install: build
	install -d "$(INSTALL_DIR)"
	install -m 0755 "$(BINARY)" "$(INSTALL_BIN)"
	install -m 0644 .wtt "$(HOME_DIR)/.wtt"
	@rc=""; \
	home_dir="$(HOME_DIR)"; \
	user_shell="$(USER_SHELL)"; \
	shell_name=""; \
	if [ -n "$$user_shell" ]; then \
		shell_name="$$(basename "$$user_shell")"; \
	fi; \
	case "$$shell_name" in \
		zsh) rc="$$home_dir/.zshrc" ;; \
		bash) rc="$$home_dir/.bashrc" ;; \
		*) rc="" ;; \
	esac; \
	if [ -n "$$rc" ]; then \
		touch "$$rc"; \
		if ! grep -Fqx "source ~/.wtt" "$$rc"; then \
			printf "\nsource ~/.wtt\n" >> "$$rc"; \
		fi; \
		printf "Installed %s\nInstalled %s\nConfigured %s\n" "$(INSTALL_BIN)" "$$home_dir/.wtt" "$$rc"; \
	else \
		printf "Installed %s\nInstalled %s\nSkipped shell rc configuration for unsupported shell: %s\n" "$(INSTALL_BIN)" "$$home_dir/.wtt" "$${shell_name:-unknown}"; \
	fi

uninstall:
	rm -f "$(INSTALL_BIN)" "$(HOME_DIR)/.wtt"
	@rc=""; \
	home_dir="$(HOME_DIR)"; \
	user_shell="$(USER_SHELL)"; \
	shell_name=""; \
	if [ -n "$$user_shell" ]; then \
		shell_name="$$(basename "$$user_shell")"; \
	fi; \
	case "$$shell_name" in \
		zsh) rc="$$home_dir/.zshrc" ;; \
		bash) rc="$$home_dir/.bashrc" ;; \
		*) rc="" ;; \
	esac; \
	if [ -n "$$rc" ] && [ -f "$$rc" ]; then \
		tmp="$${rc}.tmp.$$$$"; \
		grep -Fvx "source ~/.wtt" "$$rc" > "$$tmp" || true; \
		mv "$$tmp" "$$rc"; \
		printf "Removed %s\nRemoved %s\nUpdated %s\n" "$(INSTALL_BIN)" "$$home_dir/.wtt" "$$rc"; \
	elif [ -n "$$rc" ]; then \
		printf "Removed %s\nRemoved %s\nNo shell rc file found at %s\n" "$(INSTALL_BIN)" "$$home_dir/.wtt" "$$rc"; \
	else \
		printf "Removed %s\nRemoved %s\nSkipped shell rc cleanup for unsupported shell: %s\n" "$(INSTALL_BIN)" "$$home_dir/.wtt" "$${shell_name:-unknown}"; \
	fi
