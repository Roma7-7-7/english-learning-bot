package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	source string
	dbURL  string
	chatID int
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Printf("failed to connect to database: %v\n", err)
		os.Exit(2)
	}
	defer conn.Close(ctx)

	lines, err := parseLines(source)
	if err != nil {
		fmt.Printf("failed to parse lines: %v\n", err)
		os.Exit(3)
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(strings.ToLower(line), ":")
		if len(parts) < 2 || len(parts) > 3 {
			fmt.Printf("invalid line: %s\n", line)
			continue
		}

		word := strings.TrimSpace(parts[0])
		translation := strings.TrimSpace(parts[1])
		description := ""
		if len(parts) == 3 {
			description = strings.TrimSpace(parts[2])
		}

		_, err = conn.Exec(
			ctx,
			`INSERT INTO word_translations (chat_id, word, translation, description) VALUES ($1, $2, $3, $4) 
				   ON CONFLICT (chat_id, word) DO UPDATE SET translation = $3, description = $4`,
			chatID, word, translation, description,
		)
		if err != nil {
			fmt.Printf("failed to insert word translation: %v\n", err)
			os.Exit(4)
		}
	}

	fmt.Println("done")
}

func parseLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return lines, nil
}

func validate() error {
	if source == "" {
		return errors.New("source file is required")
	}

	if dbURL == "" {
		return errors.New("database URL is required")
	}

	if chatID == 0 {
		return errors.New("chat ID is required")
	}

	return nil
}

func init() {
	flag.StringVar(&source, "source", "", "source file")
	flag.StringVar(&dbURL, "db-url", "", "database URL")
	flag.IntVar(&chatID, "chat-id", 0, "chat ID")
	flag.Parse()
}
