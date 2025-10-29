# Subscriptions Service

REST API для управления онлайн-подписками пользователей.

Особенности

- CRUD операции над подписками

- Подсчет суммарной стоимости подписок за период с фильтрацией по user_id и service_name

- PostgreSQL с миграциями

- Swagger документация

- Логи через zap

- Конфигурация через .env или .yaml

- Поддержка retry/backoff для операций с БД

## Тесты
### Unit
```bash
go test ./...
```

### Integration
```bash
go test ./... -tags=integration
```

## Запуск через Docker Compose
```bash
docker-compose up --build
```


Сервис будет доступен по адресу: http://localhost:8080.


## Swagger

Документация доступна по адресу:

http://localhost:8080/swagger/index.html

## Примеры запросов
### Создание подписки
```http
POST /subscriptions/
Content-Type: application/json

{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025"
}
```

### Получение списка подписок
```http
GET /subscriptions/
```

### Получение подписки по ID
```http
GET /subscriptions/{id}
```

### Обновление подписки
```http
PUT /subscriptions/{id}
Content-Type: application/json

{
  "service_name": "Yandex Plus",
  "price": 450,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025"
}
```

### Удаление подписки
```http
DELETE /subscriptions/{id}
```

### Подсчет суммы подписок
```http
POST /subscriptions/summary
Content-Type: application/json

{
  "from": "07-2025",
  "to": "10-2025",
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba"
}
```