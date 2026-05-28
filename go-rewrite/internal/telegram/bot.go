package telegram

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"meridian-go-rewrite/internal/config"
)

var (
	bot             *tgbotapi.BotAPI
	chatID          int64
	messageHandlers []func(*tgbotapi.Message)
)

func StartPolling(cfg *config.Config, handler func(*tgbotapi.Message)) error {
	if cfg.Telegram.BotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	var err error
	bot, err = tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return err
	}

	if cfg.Telegram.ChatID != "" {
		chatID = parseInt64(cfg.Telegram.ChatID)
	}

	messageHandlers = append(messageHandlers, handler)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message != nil {
				for _, h := range messageHandlers {
					h(update.Message)
				}
			}
			if update.CallbackQuery != nil {
				handleCallback(update.CallbackQuery)
			}
		}
	}()

	log.Printf("[telegram] Bot polling started")
	return nil
}

func SendMessage(text string) error {
	if bot == nil || chatID == 0 {
		return fmt.Errorf("telegram not configured")
	}
	text = truncate(text, 4096)
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	return err
}

func SendHTML(html string) error {
	if bot == nil || chatID == 0 {
		return fmt.Errorf("telegram not configured")
	}
	html = truncate(html, 4096)
	msg := tgbotapi.NewMessage(chatID, html)
	msg.ParseMode = "HTML"
	_, err := bot.Send(msg)
	return err
}

func EditMessage(text string, messageID int) error {
	if bot == nil || chatID == 0 {
		return nil
	}
	text = truncate(text, 4096)
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	_, err := bot.Send(msg)
	return err
}

func SendWithButtons(text string, keyboard [][]tgbotapi.InlineKeyboardButton) error {
	if bot == nil || chatID == 0 {
		return fmt.Errorf("telegram not configured")
	}
	text = truncate(text, 4096)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err := bot.Send(msg)
	return err
}

func EditWithButtons(text string, messageID int, keyboard [][]tgbotapi.InlineKeyboardButton) error {
	if bot == nil || chatID == 0 {
		return nil
	}
	text = truncate(text, 4096)
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
	_, err := bot.Send(msg)
	return err
}

func handleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	if strings.HasPrefix(data, "cfg:") {
		answerCallback(callback.ID, "Config updated")
		EditMessage("Settings updated.", callback.Message.MessageID)
		return
	}
	answerCallback(callback.ID, "")
}

func answerCallback(id, text string) {
	if bot == nil {
		return
	}
	callback := tgbotapi.NewCallback(id, text)
	bot.Request(callback)
}

type LiveMessage struct {
	bot       *tgbotapi.BotAPI
	chatID    int64
	msgID     int
	title     string
	toolLines []string
	footer    string
}

func CreateLiveMessage(title, intro string) (*LiveMessage, error) {
	if bot == nil || chatID == 0 {
		return nil, nil
	}
	text := title + "\n\n" + intro
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := bot.Send(msg)
	if err != nil {
		return nil, err
	}
	return &LiveMessage{
		bot:    bot,
		chatID: chatID,
		msgID:  sent.MessageID,
		title:  title,
	}, nil
}

func (lm *LiveMessage) ToolStart(name string) {
	label := name
	switch name {
	case "deploy_position":
		label = "deploy position"
	case "close_position":
		label = "close position"
	case "claim_fees":
		label = "claim fees"
	case "get_top_candidates":
		label = "get top candidates"
	case "get_my_positions":
		label = "get positions"
	case "get_wallet_balance":
		label = "get wallet balance"
	case "get_active_bin":
		label = "get active bin"
	case "get_token_info":
		label = "get token info"
	}
	lm.toolLines = append(lm.toolLines, "ℹ️ "+label+" ...")
	lm.flush()
}

func (lm *LiveMessage) ToolFinish(name string, result any, success bool) {
	label := name
	switch name {
	case "deploy_position":
		label = "deploy position"
	case "close_position":
		label = "close position"
	case "claim_fees":
		label = "claim fees"
	default:
		label = name
	}
	icon := "✅"
	if !success {
		icon = "❌"
	}
	lm.toolLines = append(lm.toolLines, icon+" "+label)
	lm.flush()
}

func (lm *LiveMessage) Finalize(text string) error {
	lm.footer = text
	return lm.flush()
}

func (lm *LiveMessage) Fail(text string) error {
	lm.footer = "❌ " + text
	return lm.flush()
}

func (lm *LiveMessage) flush() error {
	if lm.bot == nil || lm.msgID == 0 {
		return nil
	}
	parts := []string{lm.title}
	if len(lm.toolLines) > 0 {
		parts = append(parts, strings.Join(lm.toolLines, "\n"))
	}
	if lm.footer != "" {
		parts = append(parts, lm.footer)
	}
	text := strings.Join(parts, "\n\n")
	text = truncate(text, 4096)
	msg := tgbotapi.NewEditMessageText(lm.chatID, lm.msgID, text)
	_, err := lm.bot.Send(msg)
	return err
}

func (lm *LiveMessage) Note(text string) {
	lm.toolLines = append(lm.toolLines, text)
	lm.flush()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

func NotifyDeploy(pair string, amountSol float64, position, tx string) {
	if bot == nil || chatID == 0 {
		return
	}
	text := fmt.Sprintf("✅ <b>Deployed</b> %s\nAmount: %.2f SOL\nPosition: <code>%s</code>\nTx: <code>%s</code>",
		pair, amountSol, truncateMID(position, 8), truncateMID(tx, 16))
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func NotifyClose(pair string, pnlUSD, pnlPct float64) {
	if bot == nil || chatID == 0 {
		return
	}
	sign := ""
	if pnlUSD >= 0 {
		sign = "+"
	}
	text := fmt.Sprintf("🔒 <b>Closed</b> %s\nPnL: %s$%.2f (%s%.2f%%)",
		pair, sign, pnlUSD, sign, pnlPct)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func NotifyOOR(pair string, minutesOOR int) {
	if bot == nil || chatID == 0 {
		return
	}
	text := fmt.Sprintf("⚠️ <b>Out of Range</b> %s\n%d minutes OOR", pair, minutesOOR)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func truncateMID(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func IsEnabled() bool {
	return bot != nil && chatID != 0
}
