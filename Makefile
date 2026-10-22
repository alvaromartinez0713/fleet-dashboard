.PHONY: help build up down logs status clean

help:
	@echo "Fleet Platform - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Build all Docker images (2-3 minutes)"
	@echo "  make up       - Start all services"
	@echo "  make down     - Stop all services"
	@echo "  make logs     - Follow logs from all services"
	@echo "  make status   - Check service health"
	@echo "  make clean    - Remove containers and volumes"
	@echo ""
	@echo "Quick Start:"
	@echo "  make build && make up"
	@echo ""
	@echo "Or use: ./deploy.sh"

build:
	@cd infrastructure/docker && docker-compose build --no-cache

up:
	@cd infrastructure/docker && docker-compose up -d && \
	  echo "Waiting 10 seconds for services to start..." && \
	  sleep 10 && \
	  docker-compose ps

down:
	@cd infrastructure/docker && docker-compose down

logs:
	@cd infrastructure/docker && docker-compose logs -f

status:
	@cd infrastructure/docker && docker-compose ps

clean:
	@cd infrastructure/docker && docker-compose down -v && docker system prune -f

