STOW_DIR := $(CURDIR)
STOW_TARGET := $(HOME)
STOW_CMD := stow -d $(STOW_DIR) -t $(STOW_TARGET)
PKGS := claudecode opencode

YELLOW := \033[33m
GREEN := \033[32m
WHITE := \033[37m
CYAN := \033[36m
CLR := \033[0m

.PHONY: default
default: help

.PHONY: help
help: ## Show this help message (default)
	@awk 'BEGIN {FS = ":.*?## "}; \
		/^[^\t][a-zA-Z0-9_-]+:.*?##/ \
		{ printf "\033[36m%-24s$(CLR) %s\n", $$1, $$2 } \
		/^##/ { printf "$(YELLOW)%s$(CLR)\n", substr($$0, 4) }' $(MAKEFILE_LIST)

.PHONY: up update
up: update
update: ## Restow all packages (use after adding/removing files)
	@for pkg in $(PKGS); do $(STOW_CMD) --restow $$pkg; done
	@echo "$(GREEN)All packages restowed$(CLR)"

.PHONY: link
link: ## Create symlinks (first time setup)
	@for pkg in $(PKGS); do $(STOW_CMD) $$pkg; done
	@echo "$(GREEN)Packages linked: $(PKGS)$(CLR)"

.PHONY: unlink
unlink: ## Remove all symlinks
	@for pkg in $(PKGS); do $(STOW_CMD) --delete $$pkg; done
	@echo "$(WHITE)All package symlinks removed$(CLR)"

.PHONY: status
status: ## Show what stow would do (dry run)
	@for pkg in $(PKGS); do \
		echo "$(CYAN)Package: $$pkg$(CLR)"; \
		$(STOW_CMD) --restow -n -v $$pkg 2>&1 || true; \
		echo ""; \
	done

.PHONY: push
push: ## Commit and push config changes
	@git add -A
	@git diff --cached --quiet && echo "$(YELLOW)Nothing to commit$(CLR)" || \
		(git commit -m "Update config" && git push origin master && echo "$(GREEN)Pushed to origin$(CLR)")
