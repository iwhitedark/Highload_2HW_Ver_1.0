# Разработка и интеграция микросервиса на Go
Домашнее задание представляет собой комплексный проект, направленный на закрепление практических навыков разработки высоконагруженных микросервисов на языке Go с интеграцией отечественных облачных платформ. Задание имитирует реальный производственный сценарий, где микросервисы обрабатывают запросы под высокой нагрузкой.

Целью данного домашнего задания является разработка полнофункционального микросервиса на языке Go, способного:
•	Обрабатывать CRUD-операции для пользовательских данных.
•	Использовать механизмы конкурентного программирования.
•	Выдерживать высокую нагрузку.
•	Быть готовым к развертыванию в контейнерной среде.

## Техническое задание
Требования к функциональности
1. HTTP API для управления пользователями
Разработайте RESTful API с операциями:
•	GET /api/users – получение списка всех пользователей.
•	GET /api/users/{id} – получение конкретного пользователя.
•	POST /api/users – создание нового пользователя.
•	PUT /api/users/{id} – обновление данных пользователя.
•	DELETE /api/users/{id} – удаление пользователя.
3. Конкурентная обработка
Используйте goroutines для асинхронной обработки:
•	Логирование операций (audit log).
•	Отправка уведомлений.
•	Обработка ошибок.
4. Мониторинг и метрики
•	Реализуйте endpoint для Prometheus метрик.
•	Добавьте сбор основных метрик (RPS, latency, ошибки).
5. Ограничение скорости (Rate Limiting)
•	Реализуйте механизм ограничения количества запросов.
•	Настройте rate limiter на 1000 запросов в секунду.
6. Контейнеризация
•	Создайте Dockerfile для сборки и запуска сервиса.
•	Обеспечьте возможность запуска через docker-compose.
Технические требования:
Язык программирования
•	Go версии 1.22+.
Используемые библиотеки
•	gorilla/mux – HTTP-роутинг.
•	minio-go – клиент для S3-совместимого хранилища.
•	golang.org/x/time/rate – rate limiting.
•	prometheus/client_golang – сбор метрик.
Инструменты для тестирования
•	wrk – нагрузочное тестирование.
•	Docker – контейнеризация.
•	MinIO – локальное S3-совместимое хранилище.

Подробное описание реализации:
## Структура проекта

```text
go-microservice/
├── main.go
├── handlers/
│   ├── user_handler.go
│   └── integration_handler.go
├── services/
│   ├── user_service.go
│   └── integration_service.go
├── models/
│   └── user.go
├── utils/
│   ├── logger.go
│   └── rate_limiter.go
├── metrics/
│   └── prometheus.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```
2. Реализация CRUD-операций
```
Модель пользователя:
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```
Обработчик запросов (пример для Create; аналогично для остальных. Импорты: encoding/json, net/http, go-microservice/services, go-microservice/utils):

```
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Сохранение пользователя (присваивается ID)
    savedUser := userService.Create(user)


    // Асинхронное логирование (после присвоения ID, чтобы избежать ID=0)
    go logUserAction("CREATE", savedUser.ID)


    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(savedUser)
}
```
3. Rate Limiting

```
var limiter = rate.NewLimiter(rate.Limit(1000), 5000)  // 1000 req/s + burst 5000 для стабильности

func rateLimitMiddleware(next http.Handler) http.Handler {  // Для Gorilla Mux (http.Handler)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServerHTTP(w, r)
    })
}
```
4. Метрики Prometheus

```
var (
    TotalRequests = prometheus.NewCounterVec(  // Consistent имя
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint"},
    )

    RequestDuration = prometheus.NewHistogramVec(  // Добавлен для latency
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Request duration",
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(TotalRequests)
    prometheus.MustRegister(RequestDuration)
}

func metricsMiddleware(next http.Handler) http.Handler {  // Для Gorilla Mux (http.Handler)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()  // Таймер для latency
        TotalRequests.WithLabelValues(r.Method, r.URL.Path).Inc()
        next.ServeHTTP(w, r)
        // Фиксируем latency
        RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
    })
}
```
# Нагрузочное тестирование
Команда wrk:

wrk -t12 -c500 -d60s http://localhost:8080/api/users

Ожидаемые результаты:

RPS: > 1000 запросов в секунду.
Latency: < 10 мс.
Ошибок: 0.
Мониторинг метрик:

curl http://localhost:8080/metrics

## Формат сдачи работы
Содержимое архива:

Исходный код (все .go файлы)
- go.mod и go.sum
- Dockerfile и docker-compose.yml
- Отчет в формате .docx, включающий:
- Скриншоты результатов нагрузочного тестирования (wrk)
- Описание реализации интеграции
- Сравнительный анализ Go vs Java в high-load сценариях (минимум 500 слов)
- Анализ альтернативных решений (Quarkus, Rust) с примерами

Критерии оценивания (максимум 16 баллов)

Критерий 1. Реализация функциональности микросервиса (CRUD + конкурентность) - Реализованы все 5 CRUD-операций (GET /users, GET /users/{id}, POST, PUT, DELETE) без ошибок. Использованы goroutines (или аналог на другом языке программрования) для асинхронного логирования или уведомлений. Код компилируется, не содержит паник при корректных/некорректных запросах.

Критерий 2. Инфраструктурные компоненты (Rate Limiting, Metrics, Docker) - Реализован rate limiter (1000 req/s), работает корректно (не блокирует тест при wrk -c500). Настроен endpoint /metrics для Prometheus с метриками RPS и latency. Собран Docker-образ, запуск через docker-compose up успешен.

Критерий 3. Нагрузочное тестирование и результаты - Приложены скриншоты wrk с результатами: RPS > 1000, avg latency < 10 мс, 0% errors. Тест выполнен на работающем сервисе (запущенном через Docker). Все параметры ( -t12 -c500 -d60s) соблюдены.

Критерий 4. Отчет и аналитическая часть - Отчет содержит сравнительный анализ Go vs Java (минимум 500 слов), включает аргументированный анализ альтернатив (Quarkus, Rust) с примерами. Описаны изменения, внесенные в исходный код (например, настройка burst, валидация email, обработка ошибок).


Важное замечание:
Данное задание разработано как комплексный проект, где предоставленные примеры кода служат ориентиром, а не готовым решением «под ключ». Мы ожидаем, что вы самостоятельно доработаете и адаптируете код под ваш проект: исправите потенциальные ошибки, добавите недостающие элементы (например, валидацию email в модели User или обработку ошибок в middleware), а также протестируете и настроите параметры (например, burst в rate limiter для вашего окружения). Простое копирование и вставка примеров может привести к неожиданным проблемам: компиляционным ошибкам (дубликаты функций), провалу нагрузочного теста (из-за слишком строгого лимитера/ограничителя) или некорректному поведению (ID=0 в логах).
Это часть обучения — экспериментируйте, отлаживайте с помощью go build -v и curl, и документируйте свои изменения в отчёте (например, «Я увеличил burst до 5000, чтобы тест прошёл с 0 ошибками»). Только так вы закрепите навыки и получите полные баллы (16/16). Если застряли — используйте документацию Go (pkg.go.dev) или форумы, но не готовые репозитории! Это поможет вам подготовиться к реальным проектам, где код всегда требует тюнинга под нагрузку и окружение.
