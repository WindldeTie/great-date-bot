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

	// Delete webhook if it exists (to allow getUpdates)
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		log.Printf("Warning: failed to delete webhook: %v", err)
	}

	userRepo := repo.NewRepo(conn)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("\nПолучен сигнал прерывания. Завершение работы...")
		os.Exit(0)
	}()

	botHandler := handler.NewHandler(bot, userRepo)
	botHandler.Start(false)
}
