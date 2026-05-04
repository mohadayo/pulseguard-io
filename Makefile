.PHONY: up down build test test-python test-go test-ts lint lint-python lint-go lint-ts clean

up:
	docker compose up --build -d

down:
	docker compose down

build:
	docker compose build

test: test-python test-go test-ts

test-python:
	cd services/api-gateway && pip install -r requirements.txt -q && pytest -v

test-go:
	cd services/health-checker && go test -v ./...

test-ts:
	cd services/analytics-dashboard && npm install --silent && npm test

lint: lint-python lint-go lint-ts

lint-python:
	cd services/api-gateway && flake8 app.py --max-line-length=100

lint-go:
	cd services/health-checker && go vet ./...

lint-ts:
	cd services/analytics-dashboard && npx eslint src/

clean:
	docker compose down --rmi local --volumes --remove-orphans
	rm -rf services/analytics-dashboard/node_modules services/analytics-dashboard/dist
	rm -rf services/api-gateway/__pycache__ services/api-gateway/.pytest_cache

logs:
	docker compose logs -f

health:
	@echo "API Gateway:"; curl -s http://localhost:8001/health | python3 -m json.tool 2>/dev/null || echo "  not running"
	@echo "Health Checker:"; curl -s http://localhost:8002/health | python3 -m json.tool 2>/dev/null || echo "  not running"
	@echo "Analytics Dashboard:"; curl -s http://localhost:8003/health | python3 -m json.tool 2>/dev/null || echo "  not running"
