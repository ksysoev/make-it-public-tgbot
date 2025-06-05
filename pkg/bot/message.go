package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

// newMessage constructs a Telegram message configuration with optional inline keyboard buttons based on given responses.
func newMessage(chatID int64, r *core.Response) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, r.Message)

	if len(r.Answers) > 0 {
		keyboard := make([][]tgbotapi.KeyboardButton, len(r.Answers))
		for i, answer := range r.Answers {
			keyboard[i] = []tgbotapi.KeyboardButton{
				{Text: answer},
			}
		}
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard:        keyboard,
			OneTimeKeyboard: true,
			ResizeKeyboard:  true,
		}
	} else {
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
			RemoveKeyboard: true,
			Selective:      false,
		}
	}

	return msg
}

// newTextMessage constructs a Telegram message configuration with text and removes the keyboard from the chat.
func newTextMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)

	msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
		Selective:      false,
	}

	return msg
}
