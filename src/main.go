package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"plant/process"
	"plant/stack"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	var in io.Reader
	var tfProc *process.Process
	tfCommand := os.Args[1:]
	if len(tfCommand) > 0 {
		proc, err := process.Exec(tfCommand)
		if err != nil {
			panic(err)
		}
		in = proc.Stdout
		tfProc = proc
	} else {
		in = os.Stdin
	}

	inTee := io.TeeReader(in, os.Stdout)
	root := newTreeNode("Terraform plan")
	promptMsg, err := readTree(root, inTee)
	if errors.Is(err, ErrTerraform) {
		if _, err := io.Copy(os.Stdout, in); err != nil && !errors.Is(err, os.ErrClosed) {
			panic(err)
		}
		if tfProc != nil {
			if _, err := io.Copy(os.Stderr, tfProc.Stderr); err != nil && !errors.Is(err, os.ErrClosed) {
				panic(err)
			}
			_ = tfProc.Cmd.Wait()
		}
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}

	tree := newTreeView(root)
	app := newApp(tree)

	promptAnswered := false
	if promptMsg != "" {
		if tfProc != nil {
			setupInputDialog(app, tree, promptMsg, tfProc.Stdin, func() { promptAnswered = true })
		} else {
			if _, err := io.Copy(os.Stdout, in); err != nil {
				panic(err)
			}
			fmt.Fprintln(os.Stderr, "plant: Piping only works with `terraform plan | plant`. For apply or destroy run `plant terraform apply` or `plant terraform destroy`.")
			os.Exit(1)
		}
	}

	if tfProc != nil {
		// using `SetAfterDrawFunc` to ensure `app.Stop` is not called before `app.Run`
		app.SetAfterDrawFunc(func(_ tcell.Screen) {
			app.SetAfterDrawFunc(nil)
			go func() {
				err := tfProc.Cmd.Wait()
				if err != nil || promptMsg != "" {
					app.Stop()
				}
			}()
		})
	}
	if err := app.Run(); err != nil {
		panic(err)
	}

	if tfProc != nil && !promptAnswered {
		// user exited the interactive menu without answering the prompt
		// user wants to exit
		tfProc.Cmd.Process.Signal(os.Interrupt)
	}

	// print further Terraform output
	if _, err := io.Copy(os.Stdout, in); err != nil && !errors.Is(err, os.ErrClosed) {
		panic(err)
	}
	if tfProc != nil {
		if _, err := io.Copy(os.Stderr, tfProc.Stderr); err != nil && !errors.Is(err, os.ErrClosed) {
			panic(err)
		}
	}
}

var ErrTerraform = errors.New("terraform encountered a problem")

func readTree(root *tview.TreeNode, in io.Reader) (string, error) {
	parentStack := stack.New[*tview.TreeNode]()
	parentStack.Push(root)

	start := false
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		coloredLine := scanner.Text()
		rawLine := stripansi.Strip(coloredLine)
		if isProblem(rawLine) {
			return "", ErrTerraform
		}
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

		opener := isOpener(rawLine)
		closer := isCloser(rawLine)
		node.SetSelectable(opener)
		if opener {
			node.SetSelectable(true)
			parentStack.Push(node)
			updateSuffix(node)
		} else if closer {
			parentStack.Pop()
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func isStart(line string) bool {
	return strings.Contains(line, "Objects have changed outside of Terraform") ||
		strings.Contains(line, "Terraform detected the following changes") ||
		strings.Contains(line, "Terraform used the selected providers") ||
		strings.Contains(line, "Terraform will perform the following actions")
}

func isProblem(line string) bool {
	return strings.Contains(line, "Terraform planned the following actions, but then encountered a problem")
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
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		return false
	}
	firstChar := trimmedLine[0]
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

func setupInputDialog(app *tview.Application, tree *tview.TreeView, promptMsg string, tfIn io.Writer, done func()) {
	inputNode := newTreeNode(promptMsg).
		SetSelectable(true).
		SetSelectedFunc(func() {
			modal := tview.NewModal().
				SetText(promptMsg).
				AddButtons(dialogButtons()).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					fmt.Fprintln(tfIn, buttonLabel)
					done()
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

func newTreeView(root *tview.TreeNode) *tview.TreeView {
	tree := tview.NewTreeView().
		SetRoot(root).
		SetTopLevel(1). // hide root node
		SetGraphics(false).
		SetAlign(true)

	tree.SetBackgroundColor(tcell.ColorDefault)
	tree.SetCurrentNode(firstSelectableNode(root))
	setupInputCapture(tree)

	return tree
}

func newApp(tree *tview.TreeView) *tview.Application {
	return tview.NewApplication().
		SetRoot(tree, true).
		EnableMouse(true)
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
