# Test Fakes

Fakes são implementações simplificadas de portas (interfaces) usadas para injetar comportamento controlado em testes.

Segundo `testing.md` (seção 5), fakes:
- Implementam interfaces de portas (ports)
- Mantêm estado para verificação de comportamento
- Permittem simular erros e comportamentos específicos
- São reutilizáveis entre suite rápida e suite de integração

## Fakes Disponíveis

### FakeClock

Implementa `Clock` port com tempo controlável.

**Uso:**
```go
clock := fakes.NewFakeClock()
clock.Advance(1 * time.Hour)
clock.SetTime(now)
t := clock.Now()
```

**Características:**
- Thread-safe com `sync.RWMutex`
- Suporta `Advance()` para mover tempo para frente
- Suporta `SetTime()` para definir tempo específico
- Inicializa em `time.Now()` por padrão

### FakeIDGenerator

Implementa `IDGenerator` port com UUIDs determinísticos.

**Uso:**
```go
gen := fakes.NewFakeIDGenerator()
id1 := gen.Generate()
id2 := gen.Generate()

// Ou com UUIDs predefinidos
uuids := []uuid.UUID{...}
gen := fakes.NewFakeIDGeneratorWithUUIDs(uuids)
```

**Características:**
- Thread-safe com `sync.Mutex`
- Gera UUIDs determinísticos baseados em contador
- Suporta UUIDs predefinidos (cicla através deles)
- Permite acesso ao contador interno para teste

### FakeKafkaPublisher

Implementa `KafkaPublisher` port com rastreamento de eventos publicados.

**Uso:**
```go
pub := fakes.NewFakeKafkaPublisher()
pub.PublishAsync(ctx, event, callback)

events := pub.GetPublishedEvents()
count := pub.GetPublishedEventCount()
```

**Características:**
- Thread-safe com `sync.RWMutex`
- Registra todos os eventos publicados
- Suporta simulação de erros com `SetFailNextPublish()`
- Suporta simulação contínua de erros com `SetSimulateError()`
- Suporta delay de publicação com `SetPublishDelay()`
- Método `WaitForEvent()` para esperar por eventos em testes assincronos

### FakeRateLimiter

Implementa `RateLimiter` port com limites configuráveis.

**Uso:**
```go
limiter := fakes.NewFakeRateLimiter()
limiter.SetLimit(12)
limiter.AllowVehicle("VH-001")
limiter.BlockVehicle("VH-BAD")
limiter.CheckLimit(ctx, "VH-001")
```

**Características:**
- Thread-safe com `sync.RWMutex`
- Suporta whitelist e blacklist de veículos
- Suporta `AlwaysAllow` e `AlwaysDeny` para comportamentos extremos
- Rastreia chamadas e por veículo
- Suporta `Reset()` para limpar estado

## Padrões de Uso em Testes

### Suite Rápida

Os fakes são injetados nos casos de uso:

```go
// Arrange
clock := fakes.NewFakeClock()
idGen := fakes.NewFakeIDGenerator()
kafkaPub := fakes.NewFakeKafkaPublisher()
rateLim := fakes.NewFakeRateLimiter()

useCase := &application.ReceivePositionUseCase{
    Clock:           clock,
    IDGenerator:     idGen,
    KafkaPublisher:  kafkaPub,
    RateLimiter:     rateLim,
}

// Act
entity, err := useCase.Execute(ctx, cmd)

// Assert
assert.NoError(t, err)
assert.Equal(t, 1, kafkaPub.GetPublishedEventCount())
```

### Object Mother

Os Object Mothers usam os mesmos fakes para construir dados de teste:

```go
func PositionRequestMother() *PositionRequestBuilder {
    // Usa FakeClock.Now() para timestamp determinístico
    clock := fakes.NewFakeClock()
    return &PositionRequestBuilder{
        capturedAt: clock.Now(),
    }
}
```

## Princípios de Implementação

1. **Sem Dependências Externas:** Fakes usam apenas stdlib
2. **Thread-Safe:** Todas as operações de estado com locks apropriados
3. **Determinísticos:** Mesmas entradas = mesmas saídas em testes
4. **Rastreáveis:** Estado interno acessível para verificação
5. **Flexíveis:** Múltiplos modos de operação (allow/deny, fail/succeed)

## Não Confundir com Mocks

- **Fakes:** Implementações simplificadas, com estado rastreável (usado neste projeto)
- **Mocks:** Verificam chamadas e interações (evitados conforme `testing.md`)

Exemplos de erro:
```go
// ❌ ERRADO: Mock que verifica chamadas
mock.ExpectCall("Publish").Times(1)

// ✅ CORRETO: Fake que rastreia estado
pub.GetPublishedEventCount() == 1
```

## Ciclo de Vida em Testes

Fakes são criados fresh em cada teste:

```go
func TestPositionValid(t *testing.T) {
    fakes := createFakes() // Novo para cada teste
    defer fakes.Cleanup()  // Limpar se necessário
    
    // Test...
}

func createFakes() *TestFakes {
    return &TestFakes{
        Clock:           fakes.NewFakeClock(),
        IDGenerator:     fakes.NewFakeIDGenerator(),
        KafkaPublisher:  fakes.NewFakeKafkaPublisher(),
        RateLimiter:     fakes.NewFakeRateLimiter(),
    }
}
```

## Garantias de Testes

Usar fakes garante:

1. **Isolamento:** Testes não dependem de Kafka real
2. **Determinismo:** Sem raceconditions ou timing issues
3. **Velocidade:** Operações em memória, <1s total
4. **Observabilidade:** Estado disponível para asserções
5. **Reprodutibilidade:** Mesmo seed = mesmos resultados
