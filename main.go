package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

func extractHashes(text string) []string {
	normalized := strings.ReplaceAll(text, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\r", " ")
	normalized = strings.ReplaceAll(normalized, "  ", " ")

	flexRegex := regexp.MustCompile(`(?i)(?:[a-f0-9]{64}|(?:[a-f0-9]{2}\s*){64})`)
	rawMatches := flexRegex.FindAllString(normalized, -1)

	var cleaned []string
	for _, match := range rawMatches {
		clean := strings.ToLower(strings.ReplaceAll(match, " ", ""))
		if len(clean) == 64 {
			cleaned = append(cleaned, clean)
		}
	}
	return cleaned
}

func generateSplunkQuery(indicators []string) string {
	if len(indicators) == 0 {
		return "No indicators found."
	}
	var quoted []string
	for _, ioc := range indicators {
		quoted = append(quoted, fmt.Sprintf("\"%s\"", ioc))
	}
	return fmt.Sprintf("index=* sourcetype=* (%s)", strings.Join(quoted, " OR "))
}

func extractTextWithLedongthuc(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var fullText string
	r.NumPage()
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			log.Printf("Error reading page %d: %v", i, err)
			continue
		}
		fullText += content + "\n"
	}
	return fullText, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file.pdf>")
		return
	}

	filePath := os.Args[1]
	fullText, err := extractTextWithLedongthuc(filePath)
	if err != nil {
		log.Fatalf("Error extracting text from PDF: %v", err)
	}

	fmt.Println("\n===== Raw Extracted Text Preview =====")
	fmt.Println(fullText)

	hashes := extractHashes(fullText)
	query := generateSplunkQuery(hashes)

	fmt.Println("\n===== Generated Splunk Query =====")
	fmt.Println(query)

	fmt.Println("\n===== Extracted Indicators =====")
	for _, h := range hashes {
		fmt.Println("-", h)
	}

	fmt.Println("\nSave this query to a file? (y/n):")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(scanner.Text()) == "y" {
		outputFile := "splunk_query.txt"
		err := os.WriteFile(outputFile, []byte(query), 0644)
		if err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}
		fmt.Println("Query saved to", outputFile)
	}
}
