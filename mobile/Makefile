.PHONY: help install start android emulator test test-watch clean

help:
	@echo "Fleet Dashboard Mobile - Makefile Commands"
	@echo "==========================================="
	@echo "make install       - Instalar dependencias"
	@echo "make android       - Iniciar emulador Android + app"
	@echo "make emulator      - Solo iniciar emulador Android"
	@echo "make start         - Iniciar app (requiere emulador ejecutándose)"
	@echo "make test          - Ejecutar tests una sola vez"
	@echo "make test-watch    - Ejecutar tests en modo watch"
	@echo "make test-coverage - Ejecutar tests con cobertura"
	@echo "make clean         - Limpiar node_modules y cache"
	@echo "make type-check    - Verificar tipos TypeScript"
	@echo "make all           - install + emulator + start + test"

install:
	@echo "Instalando dependencias..."
	npm install
	@echo "✓ Dependencias instaladas"

emulator:
	@echo "Iniciando emulador Android..."
	@echo "Asegúrate de tener Android Studio instalado"
	@which emulator > /dev/null || (echo "ERROR: emulator no encontrado. Instala Android Studio." && exit 1)
	emulator -avd Pixel_4_API_30 &
	@echo "✓ Emulador iniciando en background..."
	@echo "Esperando 30 segundos para que arranque..."
	sleep 30

start:
	@echo "Iniciando app Expo..."
	npx expo start --android

android: emulator start

test:
	@echo "Ejecutando tests..."
	node ./node_modules/.bin/jest --passWithNoTests

test-watch:
	@echo "Ejecutando tests en modo watch..."
	node ./node_modules/.bin/jest --watch --passWithNoTests

test-coverage:
	@echo "Ejecutando tests con cobertura..."
	node ./node_modules/.bin/jest --coverage --passWithNoTests

type-check:
	@echo "Verificando tipos TypeScript..."
	npm run type-check

clean:
	@echo "Limpiando cache y dependencias..."
	rm -rf node_modules
	rm -rf .expo
	rm -rf coverage
	npm cache clean --force
	@echo "✓ Cache limpiado"

all: install emulator start test

.DEFAULT_GOAL := help
