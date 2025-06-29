package main

import (
	"bufio"
	"fmt"
	"io"
	"iplan/stack"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	var in io.Reader
	var tfIn io.Writer
	tfCommand := os.Args[1:]
	if len(tfCommand) > 0 {
		stdin, stdout, err := runTerraform(tfCommand)
		if err != nil {
			panic(err)
		}
		tfIn = stdin
		in = stdout
	} else {
		in = os.Stdin
	}

	app := tview.NewApplication().
		EnableMouse(true)
	root := newTreeNode("Terraform plan")
	tree := tview.NewTreeView().
		SetRoot(root).
		SetTopLevel(1). // hide root node
		SetGraphics(false).
		SetAlign(true)
	tree.SetBackgroundColor(tcell.ColorDefault)

	query, err := readTree(root, in)
	if err != nil {
		panic(err)
	}

	postProcess(root)
	tree.SetCurrentNode(firstSelectableNode(root))
	setupInputCapture(tree)

	if query != "" {
		if tfIn != nil {
			setupInputDialog(app, tree, query, tfIn)
		} else {
			if _, err := io.Copy(os.Stdout, in); err != nil {
				panic(err)
			}
			fmt.Fprintln(os.Stderr, "iplan: Piping only works with `terraform plan | iplan`. For apply or destroy run `iplan terraform apply` or `iplan terraform destroy`.")
			os.Exit(1)
		}
	}

	if err := app.SetRoot(tree, true).Run(); err != nil {
		panic(err)
	}

	if tfIn != nil {
		go func() {
			// pass user input to Terraform
			if _, err := io.Copy(tfIn, os.Stdin); err != nil {
				panic(err)
			}
		}()
	}
	// print further Terraform output
	if _, err := io.Copy(os.Stdout, in); err != nil {
		panic(err)
	}
}

func runTerraform(command []string) (io.Writer, io.Reader, error) {
	cmd := exec.Command(command[0], command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create StdinPipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create StdoutPipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to exec command %s: %w", command, err)
	}
	return stdin, stdout, nil
}

func readTree(root *tview.TreeNode, in io.Reader) (string, error) {
	parentStack := stack.New[*tview.TreeNode]()
	parentStack.Push(root)

	start := false
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		coloredLine := scanner.Text()
		fmt.Fprintln(os.Stdout, coloredLine)
		rawLine := stripansi.Strip(coloredLine)
		if !start {
			if isStart(rawLine) {
				start = true
			} else {
				continue
			}
		}
		if needsInput(rawLine) {
			return rawLine, nil
		}

		node := newTreeNode(ansiColorToTview(coloredLine)).Collapse()
		parent := parentStack.MustPeek()
		parent.AddChild(node)
		node.SetReference(parent)

		if isOpener(rawLine) {
			parentStack.Push(node)
			updateSuffix(node)
		} else if isCloser(rawLine) {
			parentStack.Pop()
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func isStart(line string) bool {
	return strings.HasSuffix(line, "Objects have changed outside of Terraform") ||
		strings.HasPrefix(line, "Terraform detected the following changes") ||
		strings.HasPrefix(line, "Terraform used the selected providers") ||
		strings.HasPrefix(line, "Terraform will perform the following actions")
}

func needsInput(line string) bool {
	return line == "Do you want to perform these actions?" ||
		line == "Do you really want to destroy all resources?"
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

func firstSelectableNode(root *tview.TreeNode) *tview.TreeNode {
	for _, child := range root.GetChildren() {
		if len(child.GetChildren()) > 0 {
			return child
		}
	}
	return nil
}

func setupInputCapture(tree *tview.TreeView) {
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		node := tree.GetCurrentNode()
		if node == nil {
			return event
		}

		switch event.Key() {
		case tcell.KeyRight:
			if node.IsExpanded() {
				tree.Move(1)
			} else {
				node.SetExpanded(true)
				updateSuffix(node)
			}
			return nil
		case tcell.KeyLeft:
			if node.IsExpanded() {
				node.SetExpanded(false)
				updateSuffix(node)
			} else {
				parent, ok := node.GetReference().(*tview.TreeNode)
				if ok && parent != tree.GetRoot() {
					tree.SetCurrentNode(parent)
				}
			}
			return nil
		default:
			return event
		}
	})

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
		updateSuffix(node)
	})
}

func setupInputDialog(app *tview.Application, tree *tview.TreeView, query string, tfin io.Writer) {
	inputNode := newTreeNode(query).
		SetSelectable(true).
		SetSelectedFunc(func() {
			modal := tview.NewModal().
				SetText(query).
				AddButtons(dialogButtons()).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					fmt.Fprintln(tfin, buttonLabel)
					app.Stop()
				})
			modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEsc {
					app.SetRoot(tree, true)
					return nil
				}
				return event
			})
			app.SetRoot(modal, true)
		})
	tree.GetRoot().AddChild(inputNode)
}

func dialogButtons() []string {
	const no = "no"
	const yes = "yes"
	buttons := []string{no, no, no, yes}
	shuffleSlice(buttons[1:])
	return buttons
}

func shuffleSlice[T any](slice []T) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func newTreeNode(text string) *tview.TreeNode {
	return tview.NewTreeNode(text).SetTextStyle(tcell.StyleDefault)
}

func updateSuffix(node *tview.TreeNode) {
	if node.IsExpanded() {
		node.SetText(getExpandedText(node.GetText()))
	} else {
		node.SetText(getCollapsedText(node.GetText()))
	}
}

func getCollapsedText(expandedText string) string {
	if strings.HasSuffix(expandedText, "(") {
		return expandedText + "...)"
	} else if strings.HasSuffix(expandedText, "[") {
		return expandedText + "...]"
	} else if strings.HasSuffix(expandedText, "{") {
		return expandedText + "...}"
	} else {
		return expandedText
	}
}

func getExpandedText(collapsedText string) string {
	collapsedSuffixes := []string{
		"(...)",
		"[...]",
		"{...}",
	}

	for _, collapsedSuffix := range collapsedSuffixes {
		if strings.HasSuffix(collapsedText, collapsedSuffix) {
			cutLen := len(collapsedSuffix) - 1
			return collapsedText[:len(collapsedText)-cutLen]
		}
	}

	return collapsedText
}
