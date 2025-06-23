package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	lastIndentation := 0
	for scanner.Scan() {
		coloredLine := scanner.Text()
		var indentation int
		if coloredLine == "" {
			indentation = lastIndentation
		} else {
			rawLine := stripansi.Strip(coloredLine)
			indentation = getIndentation(rawLine)
			lastIndentation = indentation
		}
		fmt.Println(indentation, coloredLine)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
	}
}

func getIndentation(line string) int {
	spaces := countPrefixedSpaces(line)
	spaces += matchedPrefixLen(line[spaces:])
	return spaces / 4
}

func countPrefixedSpaces(str string) int {
	count := 0
	for _, char := range str {
		if char == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

func matchedPrefixLen(str string) int {
	prefixes := []string{"- ", "+ ", "~ ", "<= "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return len(prefix)
		}
	}
	return 0
}
