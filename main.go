package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"greateDateBot/handler"
	"greateDateBot/handler/repo"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	_ = godotenv.Load()

	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = conn.Ping(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	userRepo := repo.NewRepo(conn)

	// Получаем порт из переменных окружения Railway
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Получаем URL приложения из переменных окружения Railway
	webhookURL := os.Getenv("RAILWAY_STATIC_URL")
	if webhookURL == "" {
		log.Println("RAILWAY_STATIC_URL not set, using polling mode")
		// Если нет webhook URL, используем polling (для локальной разработки)
		botHandler := handler.NewHandler(bot, userRepo)
		botHandler.Start(false)
		return
	}

	// Настраиваем webhook для продакшена
	webhookConfig, err := tgbotapi.NewWebhook(webhookURL + "/webhook")
	if err != nil {
		log.Panic(err)
	}
	_, err = bot.Request(webhookConfig)
	if err != nil {
		log.Printf("Error setting webhook: %v", err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Printf("Error getting webhook info: %v", err)
	}

	log.Printf("Webhook Info: %+v", info)

	// Создаем обработчик webhook
	updates := bot.ListenForWebhook("/webhook")

	// Запускаем HTTP сервер
	go func() {
		log.Printf("Starting server on port %s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("\nПолучен сигнал прерывания. Завершение работы...")
		os.Exit(0)
	}()

	botHandler := handler.NewHandler(bot, userRepo)

	// Обрабатываем обновления через webhook
	for update := range updates {
		botHandler.HandleUpdate(update)
	}
}
