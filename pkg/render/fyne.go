package render

import (
	awsinterface "aws_utility/pkg/awsInterface"
	"aws_utility/pkg/logger"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type FyneRenderer struct {
	window           fyne.Window
	menuContainer    *fyne.Container
	contentContainer *fyne.Container
	awsInterface     *awsinterface.AWSInterface
}

func NewFyneRenderer(window fyne.Window, menuContainer *fyne.Container, contentContainer *fyne.Container) (*FyneRenderer, error) {
	return &FyneRenderer{
		window:           window,
		menuContainer:    menuContainer,
		contentContainer: contentContainer,
	}, nil
}

func (r *FyneRenderer) GenerateMenu() {
	r.ClearScreen()

	instructions := widget.NewLabel("Please enter your AWS SSO profile name.\n" +
		"This should be configured in your AWS CLI (~/.aws/config).")

	profileEntry := widget.NewEntry()
	profileEntry.SetPlaceHolder("Enter AWS SSO Profile Name")

	statusLabel := widget.NewLabel("")

	fetchLambdasButton := widget.NewButton("Fetch Lambda Functions", func() {
		profileName := profileEntry.Text
		var err error
		r.awsInterface, err = awsinterface.NewAWSInterface(profileName)
		if err != nil {
			logger.Error("Failed to create AWS interface:", err)
			statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		statusLabel.SetText("AWS interface created successfully. Fetching Lambda functions...")

		go func() {
			lambdaFunctions, err := r.awsInterface.ListLambdaFunctions()
			if err != nil {
				logger.Error("Failed to list Lambda functions:", err)
				r.window.Canvas().Refresh(statusLabel)
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}
			r.window.Canvas().Refresh(statusLabel)
			statusLabel.SetText(fmt.Sprintf("Successfully fetched %d Lambda functions", len(lambdaFunctions)))
			r.GenerateLambdaContent(lambdaFunctions)
		}()
	})

	menuContent := container.NewVBox(
		instructions,
		profileEntry,
		fetchLambdasButton,
		statusLabel,
	)

	r.menuContainer.Add(menuContent)
	r.menuContainer.Show()
	r.contentContainer.Hide()
}

func (r *FyneRenderer) GenerateLambdaContent(lambdaFunctions []string) {
	r.ClearScreen()

	functionDropdown := widget.NewSelect(lambdaFunctions, func(value string) {
		logger.Info("Lambda function selected:", value)
	})

	clusterEntry := widget.NewEntry()
	clusterEntry.SetPlaceHolder("Enter Cluster Name")

	serviceEntry := widget.NewEntry()
	serviceEntry.SetPlaceHolder("Enter Service Name")

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("Enter Tag Name")

	resultLabel := widget.NewLabel("")

	invokeButton := widget.NewButton("Invoke Lambda", func() {
		selectedFunction := functionDropdown.Selected
		payload := map[string]string{
			"cluster": clusterEntry.Text,
			"service": serviceEntry.Text,
			"ecr_tag": tagEntry.Text,
		}

		payloadJson, err := json.Marshal(payload)
		if err != nil {
			logger.Error("Failed to marshal payload:", err)
			resultLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		result, err := r.awsInterface.InvokeLambda(selectedFunction, payloadJson)
		if err != nil {
			logger.Error("Failed to invoke Lambda:", err)
			resultLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		logger.Info("Lambda invoked successfully. Result:", string(result))
		resultLabel.SetText(fmt.Sprintf("Result: %s", string(result)))
	})

	menuContent := container.NewVBox(
		widget.NewLabel("Select Lambda Function:"),
		functionDropdown,
		widget.NewLabel("Cluster:"),
		clusterEntry,
		widget.NewLabel("Service:"),
		serviceEntry,
		widget.NewLabel("Tag:"),
		tagEntry,
		invokeButton,
		resultLabel,
	)

	r.contentContainer.Add(menuContent)
	r.menuContainer.Hide()
	r.contentContainer.Show()
}

func (r *FyneRenderer) clearMenu() {
	r.menuContainer.RemoveAll()
	r.menuContainer.Refresh()
}

func (r *FyneRenderer) clearContent() {
	r.contentContainer.RemoveAll()
	r.contentContainer.Refresh()
}

func (r *FyneRenderer) ClearScreen() {
	r.clearMenu()
	r.clearContent()
}
