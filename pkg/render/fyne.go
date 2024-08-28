package render

import (
	awsinterface "aws_utility/pkg/awsInterface"
	"aws_utility/pkg/logger"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	MARGINE_SIZE = float32(5)
	BLOCK_SIZE   = float32(35)
)

type FyneRenderer struct {
	window           fyne.Window
	menuContainer    *fyne.Container
	contentContainer *fyne.Container
	awsInterface     *awsinterface.AWSInterface
}

func NewFyneRenderer(window fyne.Window, menuContainer *fyne.Container, contentContainer *fyne.Container) (*FyneRenderer, error) {
	awsInterface, err := awsinterface.NewAWSInterface()
	if err != nil {
		return nil, err
	}

	return &FyneRenderer{
		window:           window,
		menuContainer:    menuContainer,
		contentContainer: contentContainer,
		awsInterface:     awsInterface,
	}, nil
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

func (r *FyneRenderer) GenerateMenu() {
	r.ClearScreen()

	options := []string{"Invoke Lambda", "Schedule EventBridge"}
	dropdown := widget.NewSelect(options, func(value string) {
		logger.Info("Menu selected:", value)
	})

	goButton := widget.NewButton("Go", func() {
		logger.Info("Menu button pressed!")
		logger.Info("Menu dropdown selection:", dropdown.Selected)
		switch dropdown.Selected {
		case "Invoke Lambda":
			r.GenerateLambdaContent()
		case "Schedule EventBridge":
			r.GenerateEventBridgeContent()
		}
	})
	goButton.Importance = widget.HighImportance

	menuContent := container.NewVBox(
		dropdown,
		goButton,
	)

	r.menuContainer.Add(menuContent)
	r.menuContainer.Show()
	r.contentContainer.Hide()
}

func (r *FyneRenderer) GenerateLambdaContent() {
	r.clearContent()

	functionNameEntry := widget.NewEntry()
	functionNameEntry.SetPlaceHolder("Lambda Function Name")

	payloadEntry := widget.NewMultiLineEntry()
	payloadEntry.SetPlaceHolder("JSON Payload")

	invokeButton := widget.NewButton("Invoke Lambda", func() {
		functionName := functionNameEntry.Text
		payload := payloadEntry.Text

		// TODO: Parse payload as JSON

		result, err := r.awsInterface.InvokeLambda(functionName, payload)
		if err != nil {
			logger.Error("Failed to invoke Lambda:", err)
			// TODO: Show error in UI
		} else {
			logger.Info("Lambda invoked successfully. Result:", string(result))
			// TODO: Show result in UI
		}
	})
	invokeButton.Importance = widget.HighImportance

	backButton := widget.NewButton("Back to Menu", func() {
		r.GenerateMenu()
	})

	content := container.NewVBox(
		functionNameEntry,
		payloadEntry,
		invokeButton,
		backButton,
	)

	r.contentContainer.Add(content)
	r.menuContainer.Hide()
	r.contentContainer.Show()
}

func (r *FyneRenderer) GenerateEventBridgeContent() {
	r.clearContent()

	ruleNameEntry := widget.NewEntry()
	ruleNameEntry.SetPlaceHolder("Rule Name")

	eventBusNameEntry := widget.NewEntry()
	eventBusNameEntry.SetPlaceHolder("Event Bus Name")

	scheduleExpressionEntry := widget.NewEntry()
	scheduleExpressionEntry.SetPlaceHolder("Schedule Expression (e.g., rate(1 hour))")

	payloadEntry := widget.NewMultiLineEntry()
	payloadEntry.SetPlaceHolder("JSON Payload")

	scheduleButton := widget.NewButton("Schedule Event", func() {}) //, func() {
	//	ruleName := ruleNameEntry.Text
	//	eventBusName := eventBusNameEntry.Text
	//	scheduleExpression := scheduleExpressionEntry.Text
	//	payload := payloadEntry.Text
	//
	//	// TODO: Parse payload as JSON
	//
	//	err := r.awsInterface.ScheduleEventBridge(ruleName, eventBusName, scheduleExpression, payload)
	//	if err != nil {
	//		logger.Error("Failed to schedule EventBridge event:", err)
	//		// TODO: Show error in UI
	//	} else {
	//		logger.Info("EventBridge event scheduled successfully")
	//		// TODO: Show success message in UI
	//	}
	//})
	scheduleButton.Importance = widget.HighImportance

	backButton := widget.NewButton("Back to Menu", func() {
		r.GenerateMenu()
	})

	content := container.NewVBox(
		ruleNameEntry,
		eventBusNameEntry,
		scheduleExpressionEntry,
		payloadEntry,
		scheduleButton,
		backButton,
	)

	r.contentContainer.Add(content)
	r.menuContainer.Hide()
	r.contentContainer.Show()
}
