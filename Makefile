# Daily driver commands. Run `make help` for a summary.

.PHONY: help up down logs test import play edit dev-server psql art

help: ## Show available targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

up: ## Build and start the backend stack (postgres, valkey, api, game-server)
	docker compose up -d --build

down: ## Stop the stack (postgres data volume survives)
	docker compose down

logs: ## Follow api and game-server logs
	docker compose logs -f api game-server

test: ## Run server tests
	cd server && go test ./...

import: ## Refresh Godot's asset cache (auto-runs before play; needed after changing assets outside the editor)
	godot --headless --path client --import

play: import ## Launch the game client (backend must be up)
	godot --path client

edit: ## Open the client in the Godot editor
	godot -e --path client

dev-server: ## Run game-server natively for fast iteration (stop the container first: docker compose stop game-server)
	cd server && go run ./cmd/game-server

psql: ## Open a psql shell against the dev database
	docker compose exec postgres psql -U corsair corsair

art: ## Regenerate all generated art and audio
	godot --headless --path client -s res://tools/gen_title_art.gd
	godot --headless --path client -s res://tools/gen_world_art.gd
	python3 client/tools/gen_waves.py
