package telegram_bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/store"
)

type Service struct {
	bot *tgbotapi.BotAPI
	db  *store.Store
}

func New(token string, db *store.Store) *Service {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panicf("Telegram bot init error: %v", err)
	}

	return &Service{
		bot: bot,
		db:  db,
	}
}

func (s *Service) Run() error {
	log.Printf("Telegram bot authorized as %s", s.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch text {
		case "/start":
			s.showMainMenu(chatID)
		case "📥 Добавить покупку":
			s.bot.Send(tgbotapi.NewMessage(chatID, "Окей! Сейчас добавим покупку 💰"))
			// тут будет запуск пошагового сценария
		case "📊 Получить отчёт":
			s.bot.Send(tgbotapi.NewMessage(chatID, "Собираю отчёт 📈..."))
			// тут будет логика получения отчёта
		default:
			s.bot.Send(tgbotapi.NewMessage(chatID, "Я тебя не понял 🙈. Нажми одну из кнопок."))
		}
	}

	return nil
}

func (s *Service) showMainMenu(chatID int64) {
	menu := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📥 Добавить покупку"),
			tgbotapi.NewKeyboardButton("📊 Получить отчёт"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Выбери действие:")
	msg.ReplyMarkup = menu

	s.bot.Send(msg)
}
