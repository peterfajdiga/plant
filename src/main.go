package main

import (
	"bufio"
	"fmt"
	"iplan/stack"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	root := tview.NewTreeNode("Root")
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	scanner := bufio.NewScanner(os.Stdin)
	lastIndentation := 0
	parentStack := stack.New[*tview.TreeNode]()
	for scanner.Scan() {
		coloredLine := scanner.Text()
		var indentation int
		if coloredLine == "" {
			indentation = lastIndentation
		} else {
			rawLine := stripansi.Strip(coloredLine)
			indentation = getIndentation(rawLine)
		}

		indentationDelta := indentation - lastIndentation
		if indentationDelta > 1 {
			panic("indentation increased by " + fmt.Sprint(indentationDelta))
		} else if indentationDelta <= 0 {
			parentStack.Drop(-indentationDelta + 1)
		}

		node := tview.NewTreeNode(coloredLine)
		if parent, ok := parentStack.Peek(); ok {
			parent.AddChild(node)
		} else {
			root.AddChild(node)
		}

		parentStack.Push(node)
		lastIndentation = indentation
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
		return
	}

	if err := app.SetRoot(tree, true).Run(); err != nil {
		panic(err)
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
