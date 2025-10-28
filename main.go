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
	"strings"
	"time"
)

func main() {
	// Шаг 1: Проверить все переменные окружения
	log.Println("=== Checking environment variables ===")
	checkEnvVariables()

	// Загружаем .env только для локальной разработки
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Note: .env file not found (this is normal on Railway)")
		} else {
			log.Println("Loaded .env file for local development")
		}
	}

	// Шаг 2: Получить DATABASE_URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Println("❌ ERROR: DATABASE_URL environment variable is required")
		log.Println("Please add DATABASE_URL to Railway variables")
		log.Println("Waiting 60 seconds before exit...")
		time.Sleep(60 * time.Second)
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Printf("✅ DATABASE_URL found: %s", maskDatabaseURL(databaseURL))

	// Шаг 3: Подключиться к базе данных
	conn, err := connectToDatabase(databaseURL)
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer conn.Close()

	log.Println("✅ Successfully connected to database")

	// Шаг 4: Создать бота
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panicf("❌ Bot creation failed: %v", err)
	}

	log.Printf("✅ Authorized on account %s", bot.Self.UserName)

	// Шаг 5: Инициализировать базу данных
	userRepo := repo.NewRepo(conn)

	// Шаг 6: Запустить бота
	startBot(bot, userRepo)
}

// Функция проверки переменных окружения
func checkEnvVariables() {
	variables := []string{"BOT_TOKEN", "DATABASE_URL", "RAILWAY_ENVIRONMENT", "PORT"}

	for _, v := range variables {
		value := os.Getenv(v)
		if value == "" {
			log.Printf("❌ %s: NOT SET", v)
		} else {
			if v == "BOT_TOKEN" || v == "DATABASE_URL" {
				log.Printf("✅ %s: SET (value hidden for security)", v)
			} else {
				log.Printf("✅ %s: %s", v, value)
			}
		}
	}
}

// Функция подключения к базе данных
func connectToDatabase(databaseURL string) (*pgxpool.Pool, error) {
	// Исправить URL если нужно
	if strings.HasPrefix(databaseURL, "postgres://") {
		databaseURL = strings.Replace(databaseURL, "postgres://", "postgresql://", 1)
	}

	// Создать конфиг
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Для Railway добавить SSL
	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		config.ConnConfig.RuntimeParams["sslmode"] = "require"
	}

	// Подключиться
	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("create connection: %w", err)
	}

	// Проверить подключение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conn.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return conn, nil
}

// Функция запуска бота
func startBot(bot *tgbotapi.BotAPI, userRepo *repo.Repo) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	webhookURL := os.Getenv("RAILWAY_STATIC_URL")
	if webhookURL == "" {
		// Режим polling для разработки
		log.Println("🚀 Starting bot in POLLING mode (development)")
		_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

		botHandler := handler.NewHandler(bot, userRepo)
		botHandler.Start(false)
		return
	}

	// Режим webhook для продакшена
	log.Printf("🚀 Starting bot in WEBHOOK mode: %s", webhookURL)

	webhookConfig, err := tgbotapi.NewWebhook(webhookURL + "/webhook")
	if err != nil {
		log.Println(fmt.Sprintf("❌ Error creating webhook: %v", err))
	}
	_, err = bot.Request(webhookConfig)
	if err != nil {
		log.Printf("⚠️ Error setting webhook: %v", err)
	}

	updates := bot.ListenForWebhook("/webhook")

	go func() {
		log.Printf("🌐 Starting HTTP server on port %s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// Обработка сигналов завершения
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("🛑 Received interrupt signal. Shutting down...")
		os.Exit(0)
	}()

	// Обработка обновлений
	botHandler := handler.NewHandler(bot, userRepo)
	log.Println("✅ Bot is running and ready to receive messages")

	for update := range updates {
		botHandler.HandleUpdate(update)
	}
}

// Функция для скрытия пароля в логах
func maskDatabaseURL(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		authParts := strings.Split(parts[0], "://")
		if len(authParts) > 1 && strings.Contains(authParts[1], ":") {
			userPass := strings.Split(authParts[1], ":")
			if len(userPass) > 1 {
				return authParts[0] + "://" + userPass[0] + ":****@" + parts[1]
			}
		}
	}
	return url
}
