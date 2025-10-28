package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"greateDateBot/model"
	"log"
	"strconv"
	"strings"
	"time"
)

const adminID = int64(5120614747)

type userRepository interface {
	CreateUser(ctx context.Context, username string, id int64) error
	GetUserCount(ctx context.Context, userID int64) (int, error)
	UpdateUserCount(ctx context.Context, userID int64) error
	GetUser(ctx context.Context, userID int64) (*model.User, error)
	UserExists(ctx context.Context, userID int64) bool
	GetAllUsers(ctx context.Context) ([]model.User, error)
	DeleteUser(ctx context.Context, userID int64) error
	GetUserByName(ctx context.Context, username string) (*model.User, error)
}

type Handler struct {
	bot      *tgbotapi.BotAPI
	userRepo userRepository
}

func NewHandler(bot *tgbotapi.BotAPI, userRepo userRepository) *Handler {
	return &Handler{
		bot:      bot,
		userRepo: userRepo,
	}
}

func (h *Handler) Start(debug bool) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	h.bot.Debug = debug
	updates := h.bot.GetUpdatesChan(u)
	// go h.console()

	for update := range updates {
		h.HandleUpdate(update)
	}
}

// Обработка команд --------------------------------------------------------------------------------------------------

func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		// h.forwardToAdmin(update)
		command := strings.TrimSpace(update.Message.Text)
		msgArr := strings.Split(command, " ")
		switch msgArr[0] {
		case "/start":
			h.handleStart(update)
			return
		case "Узнать":
			log.Printf("Пользователь: %s с id: %d, решил посмотреть сколько осталось до великой даты\n",
				update.Message.From.UserName, update.Message.From.ID)
			h.handleTime(update)
			if update.Message.From.ID == adminID {
				return
			} else {
				//msg := tgbotapi.NewMessage(adminID,
				//	fmt.Sprintf("Пользователь: %s с id: `%d`, решил посмотреть сколько осталось до великой даты\n",
				//		update.Message.From.UserName, update.Message.From.ID))
				//msg.ParseMode = tgbotapi.ModeMarkdownV2
				//h.bot.Send(msg)
				info := tgbotapi.NewMessage(adminID,
					fmt.Sprintf("<a href=\"tg://user?id=%d\">Этот пользователь хотел посмотреть сколько осталось до великой даты</a>",
						update.Message.From.ID))
				info.ParseMode = tgbotapi.ModeHTML
				// [inline mention of a user](tg://user?id=123456789)
				h.bot.Send(info)
				info2 := tgbotapi.NewMessage(adminID,
					fmt.Sprintf("[Этот пользователь хотел посмотреть сколько осталось до великой даты](tg://user?id=%d)",
						update.Message.From.ID))
				info2.ParseMode = tgbotapi.ModeMarkdownV2
				h.bot.Send(info2)
			}
			return
		case "delete":
			log.Println("Удаление пользователя...")

			userID, err := strconv.Atoi(msgArr[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}

			err = h.userRepo.DeleteUser(context.Background(), int64(userID))
			if err != nil {
				log.Println("error deleteUser: ", err)
			}
			log.Println("Пользователь удален")
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь удален")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		case "list":
			users, err := h.userRepo.GetAllUsers(context.Background())
			if err != nil {
				log.Println("error getAllUsers: ", err)
			}

			log.Println("Список пользователей:")
			for _, user := range users {
				h.sendUser(user, update)
			}
			return
		case "get":
			userID, err := strconv.Atoi(msgArr[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}
			user, err := h.userRepo.GetUser(context.Background(), int64(userID))
			if err != nil {
				log.Println("error getUser: ", err)
			}
			h.sendUser(*user, update)
			return
		case "exists":
			userID, err := strconv.Atoi(msgArr[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}
			exist := h.userRepo.UserExists(context.Background(), int64(userID))
			if exist {
				log.Println("Пользователь существует")
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь существует")
				msg.ReplyToMessageID = update.Message.MessageID
				h.bot.Send(msg)
			} else {
				log.Println("Пользователь не существует")
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не существует")
				msg.ReplyToMessageID = update.Message.MessageID
				h.bot.Send(msg)
			}
			return
		default:
			log.Printf("Пользователь: %s с id: `%d`, решил написать: %s\n",
				update.Message.From.UserName, update.Message.From.ID, update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда")
			msg.ParseMode = tgbotapi.ModeMarkdownV2
			h.bot.Send(msg)
			return
		}
	}
}

func (h *Handler) handleTime(update tgbotapi.Update) {
	err := h.userRepo.UpdateUserCount(context.Background(), update.Message.From.ID)
	if err != nil {
		log.Println("error userRepo.UpdateUserCount: ", err)
	}

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Узнать время 🤫"),
		),
	)

	l, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println("error loading timezone", err)
	}
	currentTime := time.Now().In(l)
	greatDate := time.Date(2026, 3, 13, 23, 10, 0, 0, l)
	duration := greatDate.Sub(currentTime).Round(time.Second)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, formatDuration(duration))
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *Handler) handleStart(update tgbotapi.Update) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Узнать время 🤫"),
		),
	)

	log.Printf("Зашел пользователь: %s, id: %d \n", update.Message.From.UserName, update.Message.From.ID)
	err := h.ensureUser(context.Background(), update)
	if err != nil {
		log.Println("error ensureUser: ", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка! Скоро мы исправим!")
		h.bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Здравствуйте, этот бот будет показывать "+
		"время до великой даты отправления в Казань! Просто нажмите на кнопку снизу")
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

// Вспомогательные функции --------------------------------------------------------------------------------------------

func (h *Handler) ensureUser(ctx context.Context, update tgbotapi.Update) error {
	userID := update.Message.From.ID
	username := update.Message.From.UserName

	_, err := h.userRepo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err = h.userRepo.CreateUser(ctx, username, userID); err != nil {
				return fmt.Errorf("error userRepo.CreateUser: %w", err)
			}
			log.Println("Пользователь создан:", userID)
			return nil
		}
		return fmt.Errorf("error userRepo.GetUser: %w", err)
	}

	log.Println("Пользователь уже существует:", userID)

	return nil
}

func formatDuration(d time.Duration) string {
	formatted := d.String()
	daysArr := strings.Split(formatted, "h")
	daysInt, err := strconv.Atoi(daysArr[0])
	if err != nil {
		log.Println("error converting daysInt to days", err)
	}
	days := daysInt / 24
	hours := daysInt % 24
	minutesArr := strings.Split(daysArr[1], "m")
	minutesInt, err := strconv.Atoi(minutesArr[0])
	if err != nil {
		log.Println("error converting minutes to int", err)
	}
	secondsArr := strings.Split(minutesArr[1], "s")
	secondsInt, err := strconv.Atoi(secondsArr[0])
	if err != nil {
		log.Println("error converting seconds to int", err)
	}
	return fmt.Sprintf("%02d дней, %02d часов, %02d минут, %02d секунд", days, hours, minutesInt, secondsInt)
}

func (h *Handler) sendUser(user model.User, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("username: @%s, id: `%d`, count: %d\n",
		user.Username, user.ID, user.Count))
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	h.bot.Send(msg)
}
