# Task: Improve README

**Date:** 2026-01-25
**Status:** Completed

## Problem Statement
README слишком краткий и не отражает best practices использования библиотеки.

## Proposed Solution
Расширить README: описание, установка, быстрый старт, опции, перехватчики, обработка ошибок, тестирование, вклад.

## Detailed Steps
1. [x] Извлечь актуальное API из кода и спецификации.
   - Files: `client.go`, `interceptor.go`, `specs/spec.md`
2. [x] Обновить README согласно best practices.
   - Files: `README.md`
   - Changes: секции Installation, Quick Start, Options, Interceptors, Errors, Testing, Contributing
3. [x] Добавить рабочие примеры.
   - Files: `README.md`

## Testing Strategy
- [x] Manual check: примеры согласованы с API
- [x] All tests pass: `go test ./... -v`

## Open Questions
1. Нужны ли дополнительные разделы (например, Roadmap/FAQ)?
   - Ответ: Добавлены все основные разделы. Roadmap можно добавить позже при необходимости.

## Risks and Edge Cases
- Риск: расхождение README и API в будущем.

## Rollback Strategy
Вернуть `README.md` к предыдущей версии.
