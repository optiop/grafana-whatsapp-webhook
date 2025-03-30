

dev:
	docker compose down
	docker compose up --build

run_grafanaexample:
	docker compose -f ./example/docker-compose.yaml down
	docker compose -f ./example/docker-compose.yaml up --build
