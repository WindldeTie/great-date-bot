package main

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"greateDateBot/handler"
	"greateDateBot/handler/repo"
	"log"
	"os"
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
	bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

	handler.NewHandler(bot, userRepo).Start(false)
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
