STOW_DIR := $(HOME)/Developer
STOW_PKG := atlas-config
STOW_TARGET := $(HOME)
STOW_CMD := stow -d $(STOW_DIR) -t $(STOW_TARGET)

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
update: ## Restow PAI config (use after adding/removing files)
	@$(STOW_CMD) --restow $(STOW_PKG)
	@echo "$(GREEN)PAI config restowed$(CLR)"

.PHONY: link
link: ## Create symlinks (first time setup)
	@$(STOW_CMD) $(STOW_PKG)
	@echo "$(GREEN)PAI config linked to ~/.claude$(CLR)"

.PHONY: unlink
unlink: ## Remove all symlinks from ~/.claude
	@$(STOW_CMD) --delete $(STOW_PKG)
	@echo "$(WHITE)PAI symlinks removed$(CLR)"

.PHONY: status
status: ## Show what stow would do (dry run)
	@$(STOW_CMD) --restow -n -v $(STOW_PKG) 2>&1 || true
	@echo ""
	@echo "$(CYAN)Symlinks in ~/.claude:$(CLR)"
	@ls -la $(STOW_TARGET)/.claude/ | grep "^l" | awk '{print "  " $$NF " -> " $$(NF-2)}'

.PHONY: push
push: ## Commit and push PAI config changes
	@git add -A
	@git diff --cached --quiet && echo "$(YELLOW)Nothing to commit$(CLR)" || \
		(git commit -m "Update PAI config" && git push origin master && echo "$(GREEN)Pushed to origin$(CLR)")
