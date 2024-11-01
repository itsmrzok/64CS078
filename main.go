package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/aandrew-me/tgpt/v2/utils"
	"github.com/atotto/clipboard"
	Prompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
)

const localVersion = "2.8.2"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var blue = color.New(color.FgBlue)
var boldViolet = color.New(color.Bold, color.FgMagenta)
var codeText = color.New(color.BgBlack, color.FgGreen, color.Bold)
var stopSpin = false
var programLoop = true
var userInput = ""
var lastResponse = ""
var executablePath = ""
var provider *string
var apiModel *string
var apiKey *string
var temperature *string
var top_p *string
var max_length *string
var preprompt *string
var url *string
var logFile *string
var shouldExecuteCommand *bool
var disableInputLimit *bool

func main() {
	execPath, err := os.Executable()
	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	args := os.Args

	apiModel = flag.String("model", "", "Choose which model to use")
	provider = flag.String("provider", os.Getenv("AI_PROVIDER"), "Choose which provider to use")
	apiKey = flag.String("key", "", "Use personal API Key")
	temperature = flag.String("temperature", "", "Set temperature")
	top_p = flag.String("top_p", "", "Set top_p")
	max_length = flag.String("max_length", "", "Set max length of response")
	preprompt = flag.String("preprompt", "", "Set preprompt")
	url = flag.String("url", "https://api.openai.com/v1/chat/completions", "url for openai providers")
	logFile = flag.String("log", "", "Filepath to log conversation to.")
	shouldExecuteCommand = flag.Bool(("y"), false, "Instantly execute the shell command")

	isMultiline := flag.Bool("m", false, "Start multi-line interactive mode")
	flag.BoolVar(isMultiline, "multiline", false, "Start multi-line interactive mode")

	isVersion := flag.Bool("v", false, "Gives response back as a whole text")
	flag.BoolVar(isVersion, "version", false, "Gives response back as a whole text")


	disableInputLimit := flag.Bool("disable-input-limit", false, "Disables the checking of 4000 character input limit")

	flag.Parse()

	prompt := flag.Arg(0)

	pipedInput := ""
	cleanPipedInput := ""

	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "accessing standard input:", err)
		os.Exit(1)
	}

	// Checking for piped text
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			pipedInput += scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			os.Exit(1)
		}
	}

	if len(pipedInput) > 0 {
		cleanPipedInputByte, err := json.Marshal(pipedInput)
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		cleanPipedInput = string(cleanPipedInputByte)
		cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

		safePipedBytes, err := json.Marshal(pipedInput + "\n")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		pipedInput = string(safePipedBytes)
		pipedInput = pipedInput[1 : len(pipedInput)-1]
	}

	contextTextByte, _ := json.Marshal("\n\nHere is text for the context:\n")
	contextText := string(contextTextByte)

	if len(args) > 1 {
		switch {

		case *isVersion:
			fmt.Println("tgpt", localVersion)
		
		
		case *isMultiline:
			/////////////////////
			// Multiline interactive
			/////////////////////

			fmt.Print("\nPress Ctrl + D to submit, Ctrl + C to exit, Esc to unfocus, i to focus. When unfocused, press p to paste, c to copy response, b to copy last code block in response\n")

			previousMessages := ""
			threadID := utils.RandomString(36)

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(initialModel())
				_, err := p.Run()

				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if len(userInput) > 0 {
					if len(*logFile) > 0 {
						utils.LogToFile(userInput, "USER_QUERY", *logFile)
					}

					responseJson, responseTxt := getData(userInput, structs.Params{
						PrevMessages: previousMessages,
						Provider:     *provider,
						ThreadID:     threadID,
					}, structs.ExtraOptions{IsInteractive: true, DisableInputLimit: *disableInputLimit, IsNormal: true})
					previousMessages += responseJson
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}

			}

		

			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -img "cat"`)
					os.Exit(1)
				}
				generateImageBlackbox(trimmedPrompt)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				generateImageBlackbox(*preprompt + formattedInput)
			}
		default:
			go loading(&stopSpin)
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) < 1 {
				fmt.Fprintln(os.Stderr, "You need to write something")
				os.Exit(1)
			}

			getData(*preprompt+formattedInput+contextText+pipedInput, structs.Params{}, structs.ExtraOptions{IsNormal: true, IsInteractive: false, DisableInputLimit: *disableInputLimit})
		}

	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		go loading(&stopSpin)
		formattedInput := strings.TrimSpace(input)
		getData(*preprompt+formattedInput+pipedInput, structs.Params{}, structs.ExtraOptions{IsInteractive: false, DisableInputLimit: *disableInputLimit})
	}
}

// Multiline input
type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
	ti.SetValue(*preprompt)
	*preprompt = ""
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyCtrlD:
			userInput = m.textarea.Value()

			if len(userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}
		case tea.KeyTab:
			if m.textarea.Focused() {
				m.textarea.InsertString("\t")
			}
		default:
			if m.textarea.Focused() {
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

		// Command mode
		if !m.textarea.Focused() {
			switch msg.String() {
			case "i":
				m.textarea.Focus()
			case "c":
				if len(lastResponse) == 0 {
					break
				}
				err := clipboard.WriteAll(lastResponse)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "b":
				if len(lastResponse) == 0 {
					break
				}
				lastCodeBlock := getLastCodeBlock(lastResponse)
				err := clipboard.WriteAll(lastCodeBlock)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "p":
				m.textarea.Focus()
				clip, err := clipboard.ReadAll()
				msg.Runes = []rune(clip)
				if err != nil {
					fmt.Println("Could not read from clipboard")
				}
				userInput = clip
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.textarea.View()
}

func getFormattedInputStdin() (formattedInput string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	return strings.TrimSpace(input)
}

func showHelpMessage() {
	boldBlue.Println(`Usage: tgpt [Flags] [Prompt]`)

	boldBlue.Println("\nFlags:")
	fmt.Printf("%-50v Generate and Execute shell commands. (Experimental) \n", "-s, --shell")
	fmt.Printf("%-50v Generate Code. (Experimental)\n", "-c, --code")
	fmt.Printf("%-50v Gives response back without loading animation\n", "-q, --quiet")
	fmt.Printf("%-50v Gives response back as a whole text\n", "-w, --whole")
	fmt.Printf("%-50v Generate images from text\n", "-img, --image")
	fmt.Printf("%-50v Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)\n", "--provider")

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key\n", "--key")
	fmt.Printf("%-50v Set OpenAI API endpoint url\n", "--url")
	fmt.Printf("%-50v Set temperature\n", "--temperature")
	fmt.Printf("%-50v Set top_p\n", "--top_p")
	fmt.Printf("%-50v Set max response length\n", "--max_length")
	fmt.Printf("%-50v Set filepath to log conversation to (For interactive modes)\n", "--log")
	fmt.Printf("%-50v Set preprompt\n", "--preprompt")
	fmt.Printf("%-50v Execute shell command without confirmation\n", "-y")

	boldBlue.Println("\nOptions:")
	fmt.Printf("%-50v Print version \n", "-v, --version")
	fmt.Printf("%-50v Print help message \n", "-h, --help")
	fmt.Printf("%-50v Start normal interactive mode \n", "-i, --interactive")
	fmt.Printf("%-50v Start multi-line interactive mode \n", "-m, --multiline")
	fmt.Printf("%-50v See changelog of versions \n", "-cl, --changelog")

	if runtime.GOOS != "windows" {
		fmt.Printf("%-50v Update program \n", "-u, --update")
	}

	boldBlue.Println("\nProviders:")
	fmt.Println("The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.")
	fmt.Println("Available providers to use: openai ")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url")

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
}

func historyCompleter(d Prompt.Document) []Prompt.Suggest {
	s := []Prompt.Suggest{}
	return Prompt.FilterHasPrefix(s, d.GetWordAfterCursor(), true)
}

func exit(_ *Prompt.Buffer) {
	bold.Println("Exiting...")

	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
		rawModeOff.Wait()
	}

	os.Exit(0)
}
