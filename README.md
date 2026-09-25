# itk-tt

## Вопросы, решенные самостоятельно

- Изменил API, для большего соответствия RESTful-подходу
- Добавил ручки для создания кошелька и получения списка всех кошельков (без пагинации) - они удобны для проверки работы приложения
- Добавил валидацию amount (от 1 до 1 000 000)
- Значения operationType используются в нижнем регистре: deposit и withdraw

## Запуск

Параметризовать config.env (по примеру config.env.example)

### Docker

Запустить приложение:

`docker compose --env-file config.env up --build`

### Разработка

Запустить PostgreSQL и миграции:

`docker compose -f docker-compose.dev.yaml --env-file config.env up`

Запустить приложение локально:

`air`

## Тесты

Реализованы только интеграционные тесты: делаем запрос к url, проверяем полученные данные, сверяем с БД при необходимости.

Запустить тесты:

`go test ./...`

## API

### Создание кошелька

`POST /api/v1/wallets`

### Получение кошелька

`GET /api/v1/wallets/{wallet_uuid}`

### Получение всех кошельков

`GET /api/v1/wallets`

### Изменение баланса

`POST /api/v1/wallets/{wallet_id}/transaction`

Пополнение:

```json
{
  "operationType": "deposit",
  "amount": 1000
}
```

Снятие:

```json
{
  "operationType": "withdraw",
  "amount": 500
}
```
