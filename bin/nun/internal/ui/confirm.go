package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Confirm(question string) (bool, error) {
	fmt.Printf("%s [y/N] ", question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return false, err
	}
	return strings.ToLower(strings.TrimSpace(line)) == "y", nil
}

func Choice(question string, allowed ...string) (string, error) {
	allowedSet := map[string]bool{}
	for _, value := range allowed {
		allowedSet[value] = true
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(question + " ")
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return "", err
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		if allowedSet[answer] {
			return answer, nil
		}
		fmt.Printf("Enter one of: %s\n", strings.Join(allowed, ", "))
	}
}

func Prompt(question, fallback string) (string, error) {
	if fallback == "" {
		fmt.Printf("%s: ", question)
	} else {
		fmt.Printf("%s [%s]: ", question, fallback)
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return fallback, nil
	}
	return answer, nil
}
