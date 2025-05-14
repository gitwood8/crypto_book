wood_post — Dev README

Telegram-бот на Go + PostgreSQL для отслеживания криптовалютных транзакций и управления портфелями.

📁 Структура проекта

.
├── .air.toml                    # Конфигурация hot-reload
├── .env                         # Переменные окружения
├── .gitignore                   # Git-исключения
├── bin/                         # Бинарники и тулзы, собранные после запуска docker-compose
│   ├── air
│   └── wood_post
├── cmd/
│   └── service/
│       └── service.go           # Точка входа в приложение
├── config/
│   └── config.go                # Загрузка конфигурации
├── docker-compose.override.yml # Dev-переопределения
├── docker-compose.yml          # Compose-описание сервисов
├── Dockerfile                  # Многоступенчатая сборка (dev/prod)
├── go.mod / go.sum             # Модули проекта
├── IMPORTANT_NOTES             # Важные заметки
├── internal/
│   ├── service.go              # Инициализация сервисов
│   └── telegram_bot/
│       ├── service.go          # Основная логика Telegram-бота
│       └── sessions.go         # Управление сессиями пользователей
├── Makefile                    # Сборка, миграции, dev-команды
├── migrations/
│   └── 20250422150545_init.sql # Goose-миграции
├── pkg/
│   └── log/
│       └── log.go              # Кастомный логгер с уровнем и caller
├── README.md
├── store/
│   ├── schema.dbml             # DBML-схема для dbdiagram.io
│   └── service.go              # SQL-логика и доступ к базе
└── volume/                     # Docker volume для базы

🛠 Используемые технологии

Golang 1.23

PostgreSQL (в контейнере)

Air (горячая перезагрузка)

Goose (миграции)

Squirrel (SQL query builder)

go-telegram-bot-api (v5)

⚙️ Основной функционал

Telegram Bot:

Команда /start создаёт пользователя, если его ещё нет.

Показываются кнопки, пользователь создаёт "портфель".

Поддержка состояний пользователя через UserSession.

Состояние сбрасывается по таймеру (30 мин неактивности).

Отправка временных сообщений, удаляемых через sendTemporaryMessage().

Store:

CreateUserIfNotExists, UserExists и другие CRUD-запросы.

Все запросы построены через squirrel.

Миграции выполняются вручную (через make db-migrate).

Sessions:

Хранятся в SessionManager внутри Telegram-сервиса.

Поля: State, TempData, LastUpdated.

Фоновая горутина очищает сессии каждые 10 минут.

🔁 Горячая перезагрузка

Используется Air + .air.toml + make watch. Команда:

make watch

🧪 Команды Makefile

make build-service    # Собрать бинарник wood_post
make watch            # Запуск с Air (dev)
make db-migrate       # Применить миграции через Goose
make install-tools    # Установка air, goose и прочих тулов

📋 Заметки

Миграции не запускаются автоматически — только руками.

Планируется добавить поддержку webhook.

Vault для хранения секретов подключим позже.

📌 Пример БД в DBML

Находится в store/schema.dbml. Используется на https://dbdiagram.io.

📓 Логгирование

Используется собственный log из pkg/log/log.go с поддержкой:

Уровней Info, Warn, Error, Debug

Подключения runtime.Caller для вывода точного места вызова

Пример:

log.Infof("user %d created portfolio %q", userID, name)

✅ To Do

Поддержка загрузки CSV

Графики и аналитика по активам

Установка default-портфеля

Интеграция с Vault

CI/CD + автоматические миграции