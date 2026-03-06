# GOLLM CLI - Детали реализации

## Содержание

1. [Обзор проекта](#обзор-проекта)
2. [Архитектура](#архитектура)
3. [Основные компоненты](#основные-компоненты)
4. [Команды CLI](#команды-cli)
5. [Провайдеры LLM](#провайдеры-llm)
6. [Пользовательские интерфейсы](#пользовательские-интерфейсы)
7. [Конфигурация](#конфигурация)
8. [Безопасность](#безопасность)
9. [Производительность](#производительность)
10. [Тестирование](#тестирование)

## Обзор проекта

**GOLLM** - это высокопроизводительный CLI инструмент для работы с большими языковыми моделями (LLM), написанный на Go. Проект предоставляет единый интерфейс для взаимодействия с различными провайдерами LLM.

### Ключевые особенности:
- ⚡ **Высокая производительность**: время запуска < 100мс, использование памяти < 10МБ
- 🔌 **Мультипровайдерность**: поддержка OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter, Ollama
- 🎨 **Богатые UI возможности**: текстовый режим, интерактивный режим, TUI с различными темами
- 🔐 **Безопасность**: валидация входных данных, аудит логирование, безопасное управление учетными данными
- 📦 **Модульная архитектура**: чистая архитектура с четким разделением ответственности

## Архитектура

### Структура проекта

```
gollm-cli/
├── cmd/gollm/              # Точка входа приложения
│   └── main.go            # Главный файл с обработкой сигналов
├── internal/              # Приватный код приложения
│   ├── cli/              # Слой CLI интерфейса
│   │   ├── commands/     # Реализация команд
│   │   ├── root.go       # Корневая команда и глобальные флаги
│   │   └── logo.go       # ASCII арт и брендинг
│   ├── config/           # Управление конфигурацией
│   ├── core/             # Основная бизнес-логика
│   ├── providers/        # Реализации провайдеров LLM
│   ├── display/          # Форматирование вывода
│   ├── security/         # Функции безопасности
│   ├── transport/        # HTTP транспорт с circuit breaker
│   ├── tui/              # Terminal UI компоненты
│   └── version/          # Информация о версии
└── tests/                # Тестовые наборы
```

### Основные паттерны проектирования

1. **Provider Registry Pattern**: Динамическая регистрация провайдеров
2. **Command Pattern**: Изоляция команд в отдельных файлах
3. **Strategy Pattern**: Различные стратегии отображения (text, json, yaml)
4. **Observer Pattern**: Streaming ответов с каналами
5. **Circuit Breaker**: Защита от сбоев внешних сервисов

## Основные компоненты

### 1. Точка входа (cmd/gollm/main.go)

```go
// Обработка паники
defer func() {
    if r := recover(); r != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %v\n", r)
        os.Exit(ExitCodeGeneralError)
    }
}()

// Graceful shutdown
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
```

Особенности:
- Восстановление после паники с трассировкой стека
- Graceful shutdown с 5-секундным таймаутом
- Контекстное управление для отмены операций
- Специальные коды выхода для разных типов ошибок

### 2. CLI Framework (internal/cli/root.go)

Корневая команда управляет:
- Глобальными флагами (provider, model, temperature и т.д.)
- Инициализацией конфигурации
- Регистрацией подкоманд
- Контекстом выполнения с таймаутами

```go
type GlobalFlags struct {
    ConfigFile   string
    LogLevel     string
    OutputFormat string
    Provider     string
    Model        string
    Temperature  float64
    MaxTokens    int
    // ...
}
```

### 3. Provider Interface (internal/core/provider.go)

```go
type Provider interface {
    Name() string
    CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
    ListModels(ctx context.Context) ([]Model, error)
    GetModel(ctx context.Context, modelID string) (*Model, error)
    ValidateCredentials(ctx context.Context) error
}
```

## Команды CLI

### 1. chat - Отправка сообщений

```bash
gollm chat "Что такое Go?" --provider openai --model gpt-4
```

Возможности:
- Поддержка streaming и non-streaming режимов
- Системные сообщения для контекста
- Сохранение ответов в файл
- Чтение из stdin
- Настройка параметров генерации (temperature, max-tokens и т.д.)

### 2. interactive - Интерактивный режим

```bash
gollm interactive --provider anthropic --model claude-3-sonnet
```

Функции:
- Непрерывная беседа с сохранением контекста
- История сообщений
- Команды сессии (/help, /clear, /stats, /system)
- Многострочный ввод
- Отслеживание использования токенов

### 3. interactive-enhanced - Улучшенный интерактивный режим

Дополнительные возможности:
- Анимированные индикаторы загрузки
- Цветовое кодирование сообщений
- Расширенная статистика сессии
- Сохранение истории чата

### 4. tui - Terminal UI режимы

```bash
gollm tui --theme cyberpunk
```

Доступные темы:
- **simple**: Минималистичный интерфейс для отладки
- **cyberpunk**: Неоновый стиль с эффектом Matrix rain
- **minimal**: Чистый интерфейс без излишеств
- **professional**: Бизнес-ориентированный стиль
- **advanced**: Расширенный интерфейс с панелями
- **ultimate**: Максимально функциональный интерфейс

### 5. complete - Завершение кода

```bash
gollm complete "def fibonacci(n):" --provider openai
```

### 6. models - Управление моделями

```bash
gollm models list --provider gemini
gollm models info gpt-4 --provider openai
```

### 7. config - Управление конфигурацией

```bash
gollm config init
gollm config set default-provider openai
gollm config get
```

### 8. profile - Предустановленные профили

```bash
gollm profile list
gollm profile use creative
```

### 9. benchmark - Тестирование производительности

```bash
gollm benchmark --provider openai --iterations 10
```

### 10. version - Информация о версии

```bash
gollm version --detailed
```

## Провайдеры LLM

### Реализованные провайдеры:

#### 1. OpenAI (internal/providers/openai/)
- Модели: GPT-3.5, GPT-4, GPT-4-Vision
- Особенности:
  - Function calling
  - Vision capabilities
  - Streaming с Server-Sent Events
  - Автоматические повторы с экспоненциальной задержкой

#### 2. Anthropic (internal/providers/anthropic/)
- Модели: Claude 3 (Opus, Sonnet, Haiku)
- Особенности:
  - Системные сообщения
  - Streaming ответы
  - Специфичный формат API с версионированием

#### 3. Google Gemini (internal/providers/gemini/)
- Модели: Gemini Pro, Gemini Pro Vision
- Особенности:
  - Мультимодальность (текст + изображения)
  - Safety settings
  - Генеративная конфигурация

#### 4. DeepSeek (internal/providers/deepseek/)
- Модели: DeepSeek Chat, DeepSeek Coder
- Особенности:
  - Оптимизация для кода
  - Совместимость с OpenAI API

#### 5. OpenRouter (internal/providers/openrouter/)
- Агрегатор множества моделей
- Единая точка доступа к различным провайдерам

#### 6. Ollama (internal/providers/ollama/)
- Локальные модели
- Поддержка различных open-source моделей

#### 7. Mock (internal/providers/mock/)
- Для тестирования
- Конфигурируемые ответы и задержки

### Общие возможности провайдеров:

```go
// Метрики производительности
type ProviderMetrics struct {
    TotalRequests   int64
    TotalErrors     int64
    TotalTokens     int64
    TotalCost       float64
    AverageLatency  time.Duration
}

// Circuit Breaker для устойчивости
type CircuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    halfOpenTimeout time.Duration
}
```

## Пользовательские интерфейсы

### 1. Simple TUI
```go
type SimpleChat struct {
    config    *config.Config
    aiService *AIService
    messages  []string
    colors    struct {
        primary   *color.Color
        user      *color.Color
        ai        *color.Color
        error     *color.Color
    }
}
```

Особенности:
- Базовый чат интерфейс
- Цветовое выделение
- Fallback режим при ошибках AI сервиса

### 2. Cyberpunk TUI
```go
type CyberpunkTheme struct {
    Primary    lipgloss.Color // Matrix green
    Secondary  lipgloss.Color // Electric blue
    Accent     lipgloss.Color // Neon pink
    Glow       lipgloss.Color // Glow effect
}
```

Эффекты:
- Matrix rain анимация
- Glitch эффекты
- Неоновое свечение
- Анимированные переходы

### 3. Professional TUI
- Минималистичный дизайн
- Фокус на продуктивности
- Панели со статистикой
- Горячие клавиши

## Конфигурация

### Иерархия конфигурации:
1. Флаги командной строки (высший приоритет)
2. Переменные окружения (GOLLM_*)
3. Конфигурационный файл (~/.gollm/config.yaml)
4. Значения по умолчанию (низший приоритет)

### Структура конфигурации:
```yaml
default_provider: openai
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    models:
      default: gpt-4
      available:
        - gpt-4
        - gpt-3.5-turbo
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    base_url: https://api.anthropic.com
settings:
  temperature: 0.7
  max_tokens: 2048
  timeout: 30s
  output_format: text
logging:
  level: info
  file: ~/.gollm/logs/gollm.log
```

## Безопасность

### 1. Валидация входных данных (internal/security/validators.go)
```go
func ValidatePrompt(prompt string) error {
    // Проверка длины
    if len(prompt) > MaxPromptLength {
        return ErrPromptTooLong
    }
    // Проверка на вредоносные паттерны
    if containsMaliciousPatterns(prompt) {
        return ErrMaliciousContent
    }
    return nil
}
```

### 2. Управление учетными данными
- Автоматическая очистка из памяти
- Загрузка из переменных окружения
- Безопасное хранение в конфигурации

### 3. Аудит логирование
- Все запросы к API логируются
- Sensitive данные маскируются
- Ротация логов

## Производительность

### Целевые показатели:
- **Время запуска**: < 100мс
- **Использование памяти**: < 10МБ на операцию
- **Конкурентность**: 1000+ одновременных запросов
- **Размер бинарника**: ~15МБ (оптимизированный)

### Оптимизации:
1. **HTTP клиент с пулом соединений**
```go
Transport: &http.Transport{
    MaxIdleConns:       100,
    IdleConnTimeout:    90 * time.Second,
    DisableCompression: false,
}
```

2. **Кеширование моделей**
```go
type Provider struct {
    modelsCache     []Model
    modelsCacheTime time.Time
    modelsCacheTTL  time.Duration
}
```

3. **Streaming с буферизацией**
```go
chunks := make(chan StreamChunk, 10) // Буферизированный канал
```

## Тестирование

### Уровни тестирования:

1. **Unit тесты** (покрытие > 75%)
```bash
make test
```

2. **Интеграционные тесты**
```bash
make test-integration
```

3. **E2E тесты**
```bash
make test-e2e
```

4. **Benchmark тесты**
```bash
make benchmark
```

5. **Security тесты**
```bash
make security
```

### Примеры тестов:

```go
// Тест провайдера
func TestProvider_CreateCompletion(t *testing.T) {
    provider := NewMockProvider()
    resp, err := provider.CreateCompletion(ctx, &CompletionRequest{
        Model:    "gpt-3.5-turbo",
        Messages: []Message{{Role: "user", Content: "test"}},
    })
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

## Сборка и развертывание

### Makefile команды:

```bash
# Сборка
make build              # Текущая платформа
make build-all          # Все платформы
make build-debug        # С отладочной информацией
make build-race         # С детектором гонок

# Качество кода
make fmt                # Форматирование
make lint               # Линтер
make vet                # go vet
make security           # Проверка безопасности

# Установка
make install            # В GOPATH/bin
make install-local      # В /usr/local/bin
```

### CI/CD Pipeline (GitHub Actions):
- **CI workflow**: тесты, линтинг, проверка безопасности
- **Release workflow**: мультиплатформенная сборка, Docker образы

## Расширяемость

### Добавление нового провайдера:

1. Создать пакет в `internal/providers/yourprovider/`
2. Реализовать интерфейс `Provider`
3. Зарегистрировать в `init()` функции:
```go
func init() {
    core.RegisterProvider("yourprovider", NewFromConfig)
}
```
4. Добавить конфигурацию
5. Написать тесты

### Добавление новой команды:

1. Создать файл в `internal/cli/commands/`
2. Реализовать команду с Cobra
3. Добавить в `addSubcommands()` в root.go
4. Написать тесты и документацию

## Заключение

GOLLM CLI представляет собой мощный и гибкий инструмент для работы с LLM, построенный с учетом лучших практик Go и современных паттернов проектирования. Проект активно развивается, добавляются новые провайдеры и функциональность.
