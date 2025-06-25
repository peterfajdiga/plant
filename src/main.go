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
	app := tview.NewApplication().
		EnableMouse(true)
	root := tview.NewTreeNode("Root")
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetGraphics(false).
		SetAlign(true)

	if err := readTree(root, os.Stdin); err != nil {
		panic(err)
	}

	postProcess(root)
	setupInputCapture(tree)

	if err := app.SetRoot(tree, true).Run(); err != nil {
		panic(err)
	}
}

func readTree(root *tview.TreeNode, in io.Reader) error {
	parentStack := stack.New[*tview.TreeNode]()
	parentStack.Push(root)

	start := false
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		coloredLine := scanner.Text()
		rawLine := stripansi.Strip(coloredLine)
		if !start {
			if isStart(rawLine) {
				start = true
			} else {
				fmt.Fprintln(os.Stdout, coloredLine)
				continue
			}
		}

		node := tview.NewTreeNode(ansiColorToTview(coloredLine)).Collapse()
		parentStack.MustPeek().AddChild(node)

		if isOpener(rawLine) {
			parentStack.Push(node)
		} else if isCloser(rawLine) {
			parentStack.Pop()
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func isStart(line string) bool {
	return strings.HasSuffix(line, "Objects have changed outside of Terraform") ||
		strings.HasPrefix(line, "Terraform detected the following changes") ||
		strings.HasPrefix(line, "Terraform used the selected providers") ||
		strings.HasPrefix(line, "Terraform will perform the following actions")
}

func isOpener(line string) bool {
	if line == "" {
		return false
	}
	lastChar := line[len(line)-1]
	return lastChar == '(' || lastChar == '[' || lastChar == '{'
}

func isCloser(line string) bool {
	if line == "" {
		return false
	}
	firstChar := strings.TrimSpace(line)[0]
	return firstChar == ')' || firstChar == ']' || firstChar == '}'
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

func postProcess(node *tview.TreeNode) {
	node.SetSelectable(len(node.GetChildren()) > 0)
	for _, child := range node.GetChildren() {
		postProcess(child)
	}
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
