# Task: Добавление Retry Interceptor для транзиентных ошибок

**Date:** 2026-01-25  
**Status:** Completed

## Problem Statement

Нужно добавить Retry Interceptor в HTTP-клиент для автоматического повторения запросов при транзиентных ошибках. Интерцептор должен поддерживать:
1. Ретраи при сетевых ошибках (net.Error, context.DeadlineExceeded)
2. Ретраи при HTTP статус-кодах: 408, 429, 500, 502, 503, 504
3. Два варианта использования: с дефолтными настройками и с кастомными настройками

## Proposed Solution

Создать `RetryInterceptor` с конфигурируемыми параметрами:
- `MaxRetries` (по умолчанию 3)
- `Backoff` (экспоненциальный backoff с дефолтными значениями)
- `RetryOnStatus` (список статус-кодов для ретрая)
- `RetryOnError` (функция для определения сетевых ошибок)
- `RetryMethods` (только идемпотентные методы: GET, HEAD, PUT, DELETE, OPTIONS)

Предоставить две фабричные функции:
1. `DefaultRetryInterceptor()` - с дефолтными настройками
2. `NewRetryInterceptor(opts ...RetryOpt)` - с кастомными настройками

## Detailed Steps

1. [ ] **Анализ существующего кода**
   - Files: `interceptor.go`, `client.go`
   - Changes: Изучить механизм интерцепторов и безопасного клонирования запросов

2. [ ] **Определение типов и конфигурации**
   - Files: `interceptor.go`
   - Changes: Добавить типы `RetryConfig`, `RetryOpt`, функции-опции

3. [ ] **Реализация RetryInterceptor**
   - Files: `interceptor.go`
   - Changes: Реализовать логику ретраев с учетом:
     - Безопасного клонирования запроса (особенно тела)
     - Проверки методов (только идемпотентные)
     - Обработки сетевых ошибок и статус-кодов
     - Экспоненциального backoff с jitter

4. [ ] **Добавление тестов**
   - Files: `interceptor_test.go`
   - Changes: Тесты для всех случаев:
     - Сетевые ошибки (временный отказ)
     - Ретраи по статусам (408, 429, 500, 502, 503, 504)
     - Отсутствие ретраев на не-транзиентные статусы
     - Поведение с неидемпотентными методами (POST, PATCH)
     - Успешная попытка после нескольких фейлов
     - Исчерпание попыток
     - Кастомная конфигурация

5. [ ] **Интеграция и проверка**
   - Files: `client_test.go` (опционально)
   - Changes: Добавить интеграционный тест с использованием клиента
   - Проверить сборку: `go build -v .`
   - Запустить тесты: `go test ./... -v`

6. [ ] **Обновление документации**
   - Files: `specs/spec.md`
   - Changes: Добавить описание RetryInterceptor в раздел Built-in Interceptors

## Testing Strategy

- [ ] Unit tests: Тестирование логики ретраев изолированно
- [ ] Integration tests: Тестирование с httptest.Server
- [ ] Edge cases: Пустые тела запросов, большие тела, контекст с таймаутом

### Тестовые сценарии:
1. **Транзиентные сетевые ошибки**: симулировать временный отказ сети
2. **HTTP статус-коды**: сервер возвращает 500, затем 200
3. **Неидемпотентные методы**: POST не должен ретраиться
4. **Исчерпание попыток**: после 3 неудач возвращается ошибка
5. **Кастомная конфигурация**: проверить переопределение параметров
6. **Экспоненциальный backoff**: проверить задержки между попытками

## Open Questions (Решено)

1. **Заголовок `Retry-After`**: Не реализовано в текущей версии. Оставлено для будущих улучшений.
2. **Jitter в backoff**: Реализовано! Добавлен jitter ±15% для предотвращения thundering herd.
3. **Логирование попыток**: Не реализовано. Может быть добавлено как опциональная функция в будущем.

## Risks and Edge Cases

- **Risk 1**: Неправильное клонирование тела запроса может привести к его потере
  - Mitigation: Использовать `req.Clone()` и проверять возможность клонирования тела
- **Risk 2**: Ретраи неидемпотентных методов могут вызвать дублирование операций
  - Mitigation: Проверять метод перед ретраем
- **Risk 3**: Бесконечные ретраи при постоянных ошибках
  - Mitigation: Ограничить максимальное количество попыток
- **Edge case 1**: Запросы с большими телами могут потреблять много памяти при клонировании
  - Handling: Ограничить размер тела для клонирования или использовать буферизацию

## Rollback Strategy

Удалить добавленный код из:
1. `interceptor.go` - удалить RetryInterceptor и связанные типы
2. `interceptor_test.go` - удалить тесты
3. `specs/spec.md` - вернуть предыдущую версию

## Implementation Details

### Конфигурация по умолчанию:
- MaxRetries: 3
- Backoff: экспоненциальный с jitter (100ms base, 1s max)
- RetryOnStatus: [408, 429, 500, 502, 503, 504]
- RetryMethods: ["GET", "HEAD", "PUT", "DELETE", "OPTIONS"]
- RetryOnError: проверка на net.Error.Temporary() и context.DeadlineExceeded

### Реализованные функции:
```go
// Дефолтный интерцептор
func DefaultRetryInterceptor() Interceptor

// Конфигурируемый интерцептор
func NewRetryInterceptor(opts ...RetryOpt) Interceptor

// Опции конфигурации
func WithMaxRetries(n int) RetryOpt
func WithBackoff(base, max time.Duration) RetryOpt
func WithRetryOnStatus(codes []int) RetryOpt
func WithRetryMethods(methods []string) RetryOpt
func WithRetryOnError(fn func(error) bool) RetryOpt

// Вспомогательные функции
func cloneRequest(req *http.Request) (*http.Request, error)
func calculateBackoff(baseDelay, maxDelay time.Duration, attempt int) time.Duration
```

### Особенности реализации:
1. **Безопасное клонирование запросов**: Используется `req.Clone()` с сохранением тела запроса
2. **Проверка методов**: Только идемпотентные методы ретраятся по умолчанию
3. **Экспоненциальный backoff с jitter**: ±15% случайное отклонение для предотвращения thundering herd
4. **Обработка ошибок**: Раздельная обработка сетевых ошибок и HTTP статус-кодов
5. **Тестирование**: Полный набор тестов для всех основных сценариев