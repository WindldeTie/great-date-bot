package main

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"greateDateBot/handler"
	"greateDateBot/handler/repo"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	// Загружаем .env только в development (не в production)
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Warning: .env file not found: %v", err)
		} else {
			log.Println("Loaded .env file for development")
		}
	}

	// Получаем DATABASE_URL из переменных окружения
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Printf("Database URL: %s", maskDatabaseURL(databaseURL))

	// Исправляем URL для pgx (заменяем postgres:// на postgresql://)
	if strings.HasPrefix(databaseURL, "postgres://") {
		databaseURL = strings.Replace(databaseURL, "postgres://", "postgresql://", 1)
	}

	// Создаем пул подключений
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	// Для Railway добавляем SSL
	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		config.ConnConfig.RuntimeParams["sslmode"] = "require"
	}

	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close()

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conn.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Printf("Successfully connected to database")

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
		log.Println("RAILWAY_STATIC_URL not set, using polling mode (development)")
		// Если нет webhook URL, используем polling (для локальной разработки)
		_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
		if err != nil {
			log.Printf("Warning: failed to delete webhook: %v", err)
		}

		botHandler := handler.NewHandler(bot, userRepo)
		botHandler.Start(false)
		return
	}

	log.Printf("Using webhook mode with URL: %s", webhookURL)

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
	log.Println("Bot started with webhook mode")
	for update := range updates {
		botHandler.HandleUpdate(update)
	}
}

// Функция для маскирования пароля в логах
func maskDatabaseURL(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		authParts := strings.Split(parts[0], "://")
		if len(authParts) > 1 && strings.Contains(authParts[1], ":") {
			userPass := strings.Split(authParts[1], ":")
			if len(userPass) > 1 {
				masked := authParts[0] + "://" + userPass[0] + ":****@" + parts[1]
				return masked
			}
		}
	}
	return url
}
