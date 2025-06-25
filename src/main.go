package main

import (
	"bufio"
	"fmt"
	"io"
	"iplan/stack"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	root := tview.NewTreeNode("Root")
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetGraphics(false).
		SetAlign(true)

	if err := readTree(root, os.Stdin); err != nil {
		panic(err)
	}

	setupInputCapture(tree)

	if err := app.SetRoot(tree, true).Run(); err != nil {
		panic(err)
	}
}

func readTree(root *tview.TreeNode, in io.Reader) error {
	scanner := bufio.NewScanner(in)
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
			return fmt.Errorf("indentation increased by %d", indentationDelta)
		} else if indentationDelta <= 0 {
			parentStack.Drop(-indentationDelta + 1)
		}

		node := tview.NewTreeNode(ansiColorToTview(coloredLine)).Collapse()
		if parent, ok := parentStack.Peek(); ok {
			parent.AddChild(node)
		} else {
			root.AddChild(node)
		}

		parentStack.Push(node)
		lastIndentation = indentation
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
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

func ansiColorToTview(line string) string {
	replacer := strings.NewReplacer(
		"\033[30m", "[black]",
		"\033[31m", "[red]",
		"\033[32m", "[green]",
		"\033[33m", "[yellow]",
		"\033[34m", "[blue]",
		"\033[35m", "[magenta]",
		"\033[36m", "[cyan]",
		"\033[37m", "[white]",
		"\033[90m", "[gray]",
		"\033[91m", "[red]",
		"\033[92m", "[green]",
		"\033[93m", "[yellow]",
		"\033[94m", "[blue]",
		"\033[95m", "[magenta]",
		"\033[96m", "[cyan]",
		"\033[97m", "[white]",
		"\033[1m", "[::b]", // bold
		"\033[3m", "[::i]", // italic
		"\033[4m", "[::u]", // underline
		"\033[0m", "[-:-:-]", // reset all
	)
	return replacer.Replace(line)
}

func setupInputCapture(tree *tview.TreeView) {
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		node := tree.GetCurrentNode()
		if node == nil {
			return event
		}

		switch event.Key() {
		case tcell.KeyRight:
			node.SetExpanded(true)
			return nil
		case tcell.KeyLeft:
			node.SetExpanded(false)
			return nil
		default:
			return event
		}
	})
}
