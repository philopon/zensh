package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Ask(question, retry string, hasDefault bool, check func(string) bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(question)

	for {
		text, err := reader.ReadString('\n')

		ans := strings.TrimSpace(text)

		if err != nil {
			return "", err
		}

		if hasDefault && ans == "" {
			return "", err
		}

		if check(ans) {
			return ans, nil
		}

		fmt.Print(retry)
	}
}
