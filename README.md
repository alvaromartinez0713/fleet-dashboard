# Fleet Dashboard

## Start

```bash
make build && make up
```

Visit: http://localhost:3000

## Commands

```bash
make build    # Build Docker images (2-3 min)
make up       # Start all services
make down     # Stop services
make logs     # Watch logs
make ps       # Show status
```

## Services

- Frontend: http://localhost:3000
- API: http://localhost:8080
- DB: localhost:5433

## Config

Edit: `infrastructure/docker/.env`

## Verify

Open http://localhost:3000 and check browser console (F12).

