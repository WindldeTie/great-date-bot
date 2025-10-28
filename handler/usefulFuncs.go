package handler

import (
	"bufio"
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"greateDateBot/model"
	"log"
	"os"
	"strconv"
	"strings"
)

func (h *Handler) forwardToAdmin(update tgbotapi.Update) {
	adminChatID := int64(5120614747)
	infoText := formatMessageForAdmin(update.Message)
	if update.Message.From.ID == adminChatID {
		return
	} else {
		msg := tgbotapi.NewMessage(adminChatID, infoText)
		h.bot.Send(msg)
	}
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

func (h *Handler) handlePhoto(message *tgbotapi.Message) {
	// message.Photo уже является срезом, не нужно разыменовывать *
	adminChatID := int64(5120614747)
	if len(message.Photo) == 0 {
		return
	}

	for _, photo := range message.Photo {
		photoConfig := tgbotapi.NewPhoto(adminChatID, tgbotapi.FileID(photo.FileID))
		caption := fmt.Sprintf(
			"📸 Фото от: %s %s (@%s)\nID: %d",
			message.From.FirstName,
			message.From.LastName,
			message.From.UserName,
			message.From.ID,
		)
		photoConfig.Caption = caption

		_, err := h.bot.Send(photoConfig)
		if err != nil {
			log.Printf("Ошибка отправки фото: %v", err)
		}
	}
}

func (h *Handler) console() {
	reader := bufio.NewReader(os.Stdin)
	for {
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)
		username := strings.Split(command, " ")

		switch username[0] {
		case "exit":
			log.Println("Завершение работы бота...")
			os.Exit(0)
		case "delete":
			log.Println("Удаление пользователя...")

			userID, err := strconv.Atoi(username[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}

			err = h.userRepo.DeleteUser(context.Background(), int64(userID))
			if err != nil {
				log.Println("error deleteUser: ", err)
			}
			log.Println("Пользователь удален")
		case "list":
			users, err := h.userRepo.GetAllUsers(context.Background())
			if err != nil {
				log.Println("error getAllUsers: ", err)
			}

			log.Println("Список пользователей:")
			for _, user := range users {
				printUser(user)
			}
		case "get":
			userID, err := strconv.Atoi(username[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}
			user, err := h.userRepo.GetUser(context.Background(), int64(userID))
			if err != nil {
				log.Println("error getUser: ", err)
			}
			printUser(*user)
		case "exists":
			userID, err := strconv.Atoi(username[1])
			if err != nil {
				log.Println("error converting userID to int", err)
			}
			exist := h.userRepo.UserExists(context.Background(), int64(userID))
			if exist {
				log.Println("Пользователь существует")
			} else {
				log.Println("Пользователь не существует")
			}
		default:
			log.Printf("Неизвестная команда: %s \n", command)
		}
	}
}
