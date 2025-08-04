# Wallet Service

Сервис для управления кошельками с поддержкой операций пополнения и снятия средств.

## Описание

Сервис предоставляет REST API для:
- Пополнения кошелька (DEPOSIT)
- Снятия средств с кошелька (WITHDRAW) 
- Получения баланса кошелька
- Проверки состояния сервиса (health check)

## Команды

### Запуск проекта
```bash
# Собрать и запустить сервис
make build
make up

# Или одной командой
make build && make up
```

### Остановка
```bash
make down
```

### Перезапуск
```bash
make restart
```

### Просмотр логов
```bash
make logs
```

### Очистка (удаление контейнеров и данных)
```bash
make clean
```

### Тестирование
```bash
# Запуск всех тестов
make test

# Или напрямую
go test ./...
```

### Нагрузочное тестирование
```bash

# Запустить нагрузочный тест (5 секунд)
make load-test

# Короткий тест (1 секунда)
make load-test-short

# Все тесты нагрузки
make load-test-all
```

### Проверка состояния
```bash
make health-check
```

## API Endpoints

- `GET /health` - проверка состояния сервиса
- `POST /api/v1/wallet` - операции с кошельком
- `GET /api/v1/wallets/:id` - получение баланса

## Примеры запросов

```bash
# Пополнение кошелька
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"DEPOSIT","amount":100}'

# Снятие средств
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"WITHDRAW","amount":50}'

# Получение баланса
curl http://localhost:8080/api/v1/wallets/123e4567-e89b-12d3-a456-426614174000
```
```
