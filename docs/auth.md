# Аутентификация и проброс сессии

## Модель

Сессионная, не JWT. При логине IAM кладёт пользователя в Redis под ключом `session-uuid`
с TTL и отдаёт этот uuid клиенту. Клиент присылает его в `Authorization: Bearer <session-uuid>`.
Проверка сессии — всегда обращение в IAM (`Whoami`), локальной валидации нет: отозванная
сессия перестаёт работать сразу.

Пароли хешируются bcrypt, cost задаётся конфигом.

## Цепочка проброса

Ключевая сложность: session-uuid должен пройти через три разных транспорта, сохранив
одно и то же имя.

```
HTTP-заголовок Authorization
   └─ order/internal/middleware/auth.go — проверяет сессию в IAM, кладёт
      user-uuid и session-uuid в context (platform/pkg/auth)
         └─ gRPC: order/internal/interceptor/auth.go — SessionForwarder,
            клиентский интерцептор: достаёт session-uuid из context
            и добавляет в исходящую metadata
               └─ inventory/internal/interceptor/auth.go — серверный интерцептор:
                  читает metadata, валидирует через IAM.Whoami, кладёт
                  user-uuid и session-uuid в context
                     └─ Kafka: platform/pkg/middleware/kafka/session.go —
                        заголовок сообщения на стороне продюсера,
                        восстановление context на стороне консьюмера
```

Имя ключа одно и то же на всех участках (`auth.SessionTokenKey`), и это сделано
специально — иначе на каждом стыке пришлось бы помнить про переименование.

## Почему нужно middleware для Kafka

`platform/pkg/middleware/kafka/session.go`.

Kafka-обработчик получает от sarama «голый» context, в котором нет ничего из исходного
HTTP-запроса. `SessionForwarder` берёт session-uuid именно из context — значит, без
восстановления любой защищённый gRPC-вызов из обработчика (например,
`InventoryService.CommitParts` при обработке `assembly.ship-assembled`) вернёт
`codes.Unauthenticated`.

Поэтому:

- продюсер добавляет заголовок через `kafkamw.ProducerSessionHeaders(ctx)`;
- консьюмер оборачивается в `kafkamw.ConsumerSession`, которое кладёт значение обратно в context.

Если заголовка в сообщении нет или он невалиден, middleware **не** модифицирует context
и не отклоняет сообщение. Это осознанно: решать, обязательна ли сессия для конкретного
обработчика, должен сам обработчик, а не транспортное middleware.

## Хранение в context

`platform/pkg/auth/context.go`. User-uuid хранится как `uuid.UUID`, session-uuid — **строкой**:
он проходит чистым passthrough из заголовка в исходящую metadata, парсить его по дороге незачем.

## Что нельзя логировать

Session-uuid — это действующий bearer-токен. В логах он равносилен утечке учётной записи,
поэтому не пишется нигде: ни в IAM при создании сессии, ни при выходе. Сессия
идентифицируется через `user_uuid`, а связать запись с конкретным запросом позволяет `trace_id`.

Пароли и bcrypt-хеши не логируются ни на каком уровне.

Неудачные входы, наоборот, пишутся с `login` — это аудит-след, по которому виден перебор
паролей. При этом наружу всегда отдаётся один и тот же `ErrInvalidCredentials`, независимо
от того, не нашёлся пользователь или не сошёлся хеш: иначе по коду ответа можно перечислять
существующие логины.
