package clicommands

import (
	awsinterface "aws_utility/pkg/awsInterface"
	"aws_utility/pkg/logger"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	list            list.Model
	selectedLambda  string
	awsProfile      string
	clusterInput    textinput.Model
	serviceInput    textinput.Model
	tagInput        textinput.Model
	state           string
	err             error
	awsInterface    *awsinterface.AWSInterface
	lambdaFunctions []string
}

func InitialModel() model {
	return model{
		list:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		state:        "profile_input",
		clusterInput: textinput.New(),
		serviceInput: textinput.New(),
		tagInput:     textinput.New(),
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "profile_input":
			switch msg.String() {
			case "enter":
				m.awsProfile = m.clusterInput.Value()
				return m, m.fetchLambdaFunctions
			}
		case "lambda_selection":
			switch msg.String() {
			case "enter":
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.selectedLambda = i.title
					m.state = "cluster_input"
					m.clusterInput.Focus()
					return m, textinput.Blink
				}
			}
		case "cluster_input":
			switch msg.String() {
			case "enter":
				m.state = "service_input"
				m.serviceInput.Focus()
				return m, textinput.Blink
			}
		case "service_input":
			switch msg.String() {
			case "enter":
				m.state = "tag_input"
				m.tagInput.Focus()
				return m, textinput.Blink
			}
		case "tag_input":
			switch msg.String() {
			case "enter":
				return m, m.invokeLambda
			}
		}
	case fetchLambdaFunctionsMsg:
		m.lambdaFunctions = msg
		items := make([]list.Item, len(m.lambdaFunctions))
		for i, fn := range m.lambdaFunctions {
			items[i] = item{title: fn, desc: ""}
		}
		m.list.SetItems(items)
		m.state = "lambda_selection"
		return m, nil
	case lambdaInvokeResultMsg:
		m.state = "result"
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	switch m.state {
	case "profile_input":
		return fmt.Sprintf(
			"Enter AWS SSO profile name:\n\n%s\n\n%s",
			m.clusterInput.View(),
			"(press enter to confirm)",
		)
	case "lambda_selection":
		return fmt.Sprintf(
			"Select a Lambda function:\n\n%s\n\n%s",
			m.list.View(),
			"(press enter to select)",
		)
	case "cluster_input":
		return fmt.Sprintf(
			"Enter cluster name:\n\n%s\n\n%s",
			m.clusterInput.View(),
			"(press enter to confirm)",
		)
	case "service_input":
		return fmt.Sprintf(
			"Enter service name:\n\n%s\n\n%s",
			m.serviceInput.View(),
			"(press enter to confirm)",
		)
	case "tag_input":
		return fmt.Sprintf(
			"Enter tag name:\n\n%s\n\n%s",
			m.tagInput.View(),
			"(press enter to confirm)",
		)
	case "result":
		if m.err != nil {
			return fmt.Sprintf("Error: %v", m.err)
		}
		return fmt.Sprintf("Lambda function '%s' invoked successfully", m.selectedLambda)
	default:
		return "Loading..."
	}
}

func (m *model) fetchLambdaFunctions() tea.Msg {
	awsInterface, err := awsinterface.NewAWSInterface(m.awsProfile)
	if err != nil {
		logger.Error("Failed to create AWS interface:", err)
		return fetchLambdaFunctionsMsg{}
	}
	m.awsInterface = awsInterface

	lambdaFunctions, err := m.awsInterface.ListLambdaFunctions()
	if err != nil {
		logger.Error("Failed to list Lambda functions:", err)
		return fetchLambdaFunctionsMsg{}
	}
	return fetchLambdaFunctionsMsg(lambdaFunctions)
}

func (m *model) invokeLambda() tea.Msg {
	payload := map[string]string{
		"cluster": m.clusterInput.Value(),
		"service": m.serviceInput.Value(),
		"ecr_tag": m.tagInput.Value(),
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed to marshal payload:", err)
		return nil
	}

	result, err := m.awsInterface.InvokeLambda(m.selectedLambda, payloadJson)
	if err != nil {
		logger.Error("Failed to invoke Lambda:", err)
		return lambdaInvokeResultMsg{err: err}
	}

	logger.Info("Lambda invoked successfully. Result:", string(result))
	return lambdaInvokeResultMsg{result: result}
}

type fetchLambdaFunctionsMsg []string
type lambdaInvokeResultMsg struct {
	result []byte
	err    error
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func ListLambdas(profile string) error {
	awsInterface, err := awsinterface.NewAWSInterface(profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS interface: %v", err)
	}

	lambdaFunctions, err := awsInterface.ListLambdaFunctions()
	if err != nil {
		return fmt.Errorf("failed to list Lambda functions: %v", err)
	}

	fmt.Println("Available Lambda functions:")
	for _, fn := range lambdaFunctions {
		fmt.Println("-", fn)
	}

	return nil
}

func ExecuteLambda(profile, lambdaName, cluster, service, tag string) error {
	awsInterface, err := awsinterface.NewAWSInterface(profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS interface: %v", err)
	}

	payload := map[string]string{
		"cluster": cluster,
		"service": service,
		"ecr_tag": tag,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed to marshal payload:", err)
		return nil
	}

	result, err := awsInterface.InvokeLambda(lambdaName, payloadJson)
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %v", err)
	}

	fmt.Printf("Lambda function '%s' invoked successfully. Result: %s\n", lambdaName, string(result))
	return nil
}
