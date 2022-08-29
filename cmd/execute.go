package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/phuangpheth/feedback/database"
	"github.com/phuangpheth/feedback/feedback"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func failOnError(err error, mgs string) {
	if err != nil {
		log.Printf("%s: %s", mgs, err)
		os.Exit(1)
	}
}

func Execute() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5455")
	timeZone := getEnv("TZ", "Asia/Vientiane")

	dbUser := os.Getenv("DB_USER")
	dbPasswd := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dbConnection := fmt.Sprintf(`
    host=%s 
    port=%s 
    user=%s 
    password=%s 
    dbname=%s 
    sslmode=disable 
    TimeZone=%s
  `,
		dbHost,
		dbPort,
		dbUser,
		dbPasswd,
		dbName,
		timeZone,
	)
	db, err := database.Open("postgres", dbConnection)
	failOnError(err, "failed to open database")
	defer func() {
		err := db.Close()
		failOnError(err, "failed to close database")
	}()

	feedbackSvc := feedback.NewService(db)

	e := echo.New()
	e.Use(middleware.Logger())

	NewHandler(e, feedbackSvc)

	go func() {
		err := e.Start(fmt.Sprintf(":%s", getEnv("PORT", "3001")))
		e.Logger.Fatal(err)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = e.Shutdown(ctx)
	failOnError(err, "failed to shutdown server")
}
