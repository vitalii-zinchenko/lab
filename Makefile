.PHONY: infra-up infra-down

infra-up:
	docker compose -f infra/docker-compose.yml up -d
	@echo ""
	@echo "Prometheus is running!"
	@echo "  UI:  http://localhost:9090"
	@echo "  Targets: http://localhost:9090/targets"
	@echo ""

infra-down:
	docker compose -f infra/docker-compose.yml down
