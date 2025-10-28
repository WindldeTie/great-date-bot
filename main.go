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
	// Загрузка .env только для локальной разработки
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		godotenv.Load()
	}

	// Инициализация базы данных
	db := initDatabase()
	defer db.Close()

	// Инициализация бота
	bot := initBot()

	// Создание репозитория и запуск бота
	userRepo := repo.NewRepo(db)
	startBot(bot, userRepo)
}

func initDatabase() *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Исправляем URL для pgx
	dbURL = strings.Replace(dbURL, "postgres://", "postgresql://", 1)

	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = conn.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	log.Println("✅ Database connected successfully")
	return conn
}

func initBot() *tgbotapi.BotAPI {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panicf("Bot creation failed: %v", err)
	}

	log.Printf("✅ Authorized as @%s", bot.Self.UserName)
	return bot
}

func startBot(bot *tgbotapi.BotAPI, userRepo *repo.Repo) {
	port := getPort()
	webhookURL := os.Getenv("RAILWAY_STATIC_URL")

	if webhookURL == "" {
		startPolling(bot, userRepo)
	} else {
		startWebhook(bot, userRepo, webhookURL, port)
	}
}

func startPolling(bot *tgbotapi.BotAPI, userRepo *repo.Repo) {
	log.Println("🚀 Starting in POLLING mode (development)")
	bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

	handler.NewHandler(bot, userRepo).Start(false)
}

func startWebhook(bot *tgbotapi.BotAPI, userRepo *repo.Repo, webhookURL, port string) {
	log.Printf("🚀 Starting in WEBHOOK mode: %s", webhookURL)

	// Настройка webhook
	webhookConfig, err := tgbotapi.NewWebhook(webhookURL + "/webhook")
	if err != nil {
		log.Panicf("Webhook creation failed: %v", err)
	}
	bot.Request(webhookConfig)

	// Запуск HTTP сервера
	updates := bot.ListenForWebhook("/webhook")
	go func() {
		log.Printf("🌐 HTTP server listening on port %s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// Обработка сигналов завершения
	setupGracefulShutdown()

	// Обработка обновлений
	botHandler := handler.NewHandler(bot, userRepo)
	log.Println("✅ Bot is running and ready")

	for update := range updates {
		botHandler.HandleUpdate(update)
	}
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("🛑 Shutting down...")
		os.Exit(0)
	}()
}
