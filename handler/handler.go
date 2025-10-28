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

type userRepository interface {
	CreateUser(ctx context.Context, username string, id int64) error
	GetUserCount(ctx context.Context, userID int64) (int, error)
	UpdateUserCount(ctx context.Context, userID int64) error
	GetUser(ctx context.Context, userID int64) (*model.User, error)
	UserExists(ctx context.Context, userID int64) bool
	GetAllUsers(ctx context.Context) ([]model.User, error)
	DeleteUser(ctx context.Context, userID int64) error
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
		h.forwardToAdmin(update)
		switch update.Message.Text {
		case "/start":
			h.handleStart(update)
			return
		case "Узнать время 🤫":
			log.Printf("Пользователь: %s с id: %d, решил посмотреть сколько осталось до великой даты\n",
				update.Message.From.UserName, update.Message.From.ID)
			h.handleTime(update)
			return
		default:
			log.Printf("Пользователь: %s с id: %d, решил написать: %s \n",
				update.Message.From.UserName, update.Message.From.ID, update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда")
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

//func (h *Handler) console() {
//	reader := bufio.NewReader(os.Stdin)
//	for {
//		command, _ := reader.ReadString('\n')
//		command = strings.TrimSpace(command)
//		username := strings.Split(command, " ")
//
//		switch username[0] {
//		case "exit":
//			log.Println("Завершение работы бота...")
//			os.Exit(0)
//		case "delete":
//			log.Println("Удаление пользователя...")
//
//			userID, err := strconv.Atoi(username[1])
//			if err != nil {
//				log.Println("error converting userID to int", err)
//			}
//
//			err = h.userRepo.DeleteUser(context.Background(), int64(userID))
//			if err != nil {
//				log.Println("error deleteUser: ", err)
//			}
//			log.Println("Пользователь удален")
//		case "list":
//			users, err := h.userRepo.GetAllUsers(context.Background())
//			if err != nil {
//				log.Println("error getAllUsers: ", err)
//			}
//
//			log.Println("Список пользователей:")
//			for _, user := range users {
//				printUser(user)
//			}
//		case "get":
//			userID, err := strconv.Atoi(username[1])
//			if err != nil {
//				log.Println("error converting userID to int", err)
//			}
//			user, err := h.userRepo.GetUser(context.Background(), int64(userID))
//			if err != nil {
//				log.Println("error getUser: ", err)
//			}
//			printUser(*user)
//		case "exists":
//			userID, err := strconv.Atoi(username[1])
//			if err != nil {
//				log.Println("error converting userID to int", err)
//			}
//			exist := h.userRepo.UserExists(context.Background(), int64(userID))
//			if exist {
//				log.Println("Пользователь существует")
//			} else {
//				log.Println("Пользователь не существует")
//			}
//		default:
//			log.Printf("Неизвестная команда: %s \n", command)
//		}
//	}
//}

func (h *Handler) forwardToAdmin(update tgbotapi.Update) {
	adminChatID := int64(5120614747)
	infoText := formatMessageForAdmin(update.Message)
	msg := tgbotapi.NewMessage(adminChatID, infoText)
	h.bot.Send(msg)
}

func formatMessageForAdmin(message *tgbotapi.Message) string {
	return fmt.Sprintf(
		"📩 Новое сообщение от пользователя:\n"+
			"👤 Username: @%s\n"+
			"👤 Имя: %s %s\n"+
			"📝 Текст: %s\n"+
			"⏰ Время: %s",
		message.From.UserName,
		message.From.FirstName,
		message.From.LastName,
		message.Text,
		message.Time().Format("2006-01-02 15:04:05"),
	)
}

func printUser(user model.User) {
	fmt.Printf("Имя: %s, id: %d, count: %d \n", user.Username, user.ID, user.Count)
}

//func (h *Handler) handlePhoto(message *tgbotapi.Message) {
//	// message.Photo уже является срезом, не нужно разыменовывать *
//	adminChatID := int64(5120614747)
//	if len(message.Photo) == 0 {
//		return
//	}
//
//	for _, photo := range message.Photo {
//		photoConfig := tgbotapi.NewPhoto(adminChatID, tgbotapi.FileID(photo.FileID))
//		caption := fmt.Sprintf(
//			"📸 Фото от: %s %s (@%s)\nID: %d",
//			message.From.FirstName,
//			message.From.LastName,
//			message.From.UserName,
//			message.From.ID,
//		)
//		photoConfig.Caption = caption
//
//		_, err := h.bot.Send(photoConfig)
//		if err != nil {
//			log.Printf("Ошибка отправки фото: %v", err)
//		}
//	}
//}
