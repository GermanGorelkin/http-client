# Task: Replace interface{} with any

**Date:** 2026-01-25
**Status:** Completed

## Problem Statement
Использование `interface{}` нужно заменить на `any` по современным best practices.

## Proposed Solution
Найти все `interface{}` в исходниках и тестах, заменить на `any` без изменения поведения.

## Detailed Steps
1. [x] Найти все упоминания `interface{}`.
   - Files: `client.go`, `interceptor.go`, `client_test.go`, `interceptor_test.go`
2. [x] Заменить на `any`.
3. [x] Прогнать тесты.

## Testing Strategy
- [x] Unit tests: `go test ./... -v`

## Open Questions
1. Нет.

## Risks and Edge Cases
- Риск: пропустить `interface{}` в примерах/README.

## Rollback Strategy
Вернуть `any` обратно на `interface{}`.
