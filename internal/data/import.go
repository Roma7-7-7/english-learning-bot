package data

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type (
	Line struct {
		Word        string
		Translation string
		Description string
	}

	ParsingError struct {
		InvalidLines []int
	}
)

func (e *ParsingError) Error() string {
	return fmt.Sprintf("parsing error: invalidLines=%v", e.InvalidLines)
}

func Parse(ctx context.Context, in io.ReadCloser, out chan<- Line) error {
	defer close(out)
	defer in.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(in)
	invalidLines := make([]int, 0, 10) //nolint:mnd // 10 is the expected capacity
	linNum := 0
	for scanner.Scan() {
		linNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(strings.ToLower(line), ":")
		if len(parts) < 2 || len(parts) > 3 {
			invalidLines = append(invalidLines, linNum)
			continue
		}

		word := strings.TrimSpace(parts[0])
		translation := strings.TrimSpace(parts[1])
		description := ""
		if len(parts) == 3 { //nolint:mnd // 3 is the expected length
			description = strings.TrimSpace(parts[2])
		}

		select {
		case <-ctx.Done():
			return nil
		case out <- Line{
			Word:        word,
			Translation: translation,
			Description: description,
		}: // continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file: %w", err)
	}
	if len(invalidLines) > 0 {
		return &ParsingError{InvalidLines: invalidLines}
	}

	return nil
}
