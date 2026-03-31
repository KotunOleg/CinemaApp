package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"webapp/internal/database"
)

func loadEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if os.Getenv(strings.TrimSpace(k)) == "" {
			os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
}

func main() {
	loadEnv(".env")
	ctx := context.Background()
	cfg := database.ConfigFromEnv()
	pool, err := database.New(ctx, cfg)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer pool.Close()

	sql, err := os.ReadFile("migrations/001_schema.sql")
	if err != nil {
		log.Fatalf("Read migration: %v", err)
	}

	_, err = pool.Exec(ctx, string(sql))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Println("Migration applied successfully")
}
