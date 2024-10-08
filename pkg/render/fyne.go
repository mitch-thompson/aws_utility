package render

import (
	awsinterface "aws_utility/pkg/awsInterface"
	"aws_utility/pkg/logger"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"sort"
	"strings"
	"time"
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

	instructions := widget.NewLabel("Please enter your AWS SSO login portal URL.")

	portalEntry := widget.NewEntry()
	portalEntry.SetPlaceHolder("https://your-domain.awsapps.com/start")

	statusLabel := widget.NewLabel("")

	var accountSelect *widget.Select
	var roleSelect *widget.Select

	loginButton := widget.NewButton("Login", func() {
		portalURL := portalEntry.Text

		if portalURL == "" {
			statusLabel.SetText("Error: Portal URL is required")
			return
		}

		statusLabel.SetText("Initiating authentication...")

		go func() {
			var err error
			r.awsInterface, err = awsinterface.NewAWSInterface(portalURL)
			if err != nil {
				logger.Error("Failed to create AWS interface:", err)
				r.window.Canvas().Refresh(statusLabel)
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			err = r.awsInterface.RegisterClient()
			if err != nil {
				logger.Error("Failed to register client:", err)
				r.window.Canvas().Refresh(statusLabel)
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			authInfo, err := r.awsInterface.StartAuthentication()
			if err != nil {
				logger.Error("Failed to start authentication:", err)
				r.window.Canvas().Refresh(statusLabel)
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			authInstructions := widget.NewLabel(fmt.Sprintf("Please visit this URL to complete authentication:\n%s\n\nAnd enter this code: %s", authInfo.VerificationURIComplete, authInfo.UserCode))
			logger.Info(authInfo.VerificationURIComplete, authInfo.UserCode)

			r.contentContainer.RemoveAll()
			r.contentContainer.Add(authInstructions)
			r.contentContainer.Refresh()

			r.menuContainer.Hide()
			r.contentContainer.Show()

			go func() {
				err := r.awsInterface.PollForToken(authInfo)
				if err != nil {
					logger.Error("Failed to complete authentication:", err)
					r.window.Canvas().Refresh(statusLabel)
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
					return
				}

				accounts, err := r.awsInterface.ListAccounts()
				if err != nil {
					logger.Error("Failed to list accounts:", err)
					r.window.Canvas().Refresh(statusLabel)
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
					return
				}

				accountOptions := make([]string, len(accounts))
				for i, account := range accounts {
					accountOptions[i] = fmt.Sprintf("%s (%s)", account.AccountName, account.AccountID)
				}

				accountSelect.Options = accountOptions
				accountSelect.Refresh()
				accountSelect.Show()

				r.contentContainer.Hide()
				r.menuContainer.Show()
				r.window.Canvas().Refresh(statusLabel)
				statusLabel.SetText("Authentication successful. Please select an account.")
			}()
		}()
	})

	var selectedAccountID string

	accountSelect = widget.NewSelect([]string{}, func(value string) {
		logger.Info("Account selected:", value)

		accountID := strings.Split(value, "(")[1]
		selectedAccountID = strings.TrimSuffix(accountID, ")")

		roles, err := r.awsInterface.ListRoles(selectedAccountID)
		if err != nil {
			logger.Error("Failed to list roles:", err)
			statusLabel.SetText(fmt.Sprintf("Error: Failed to list roles: %v", err))
			return
		}

		roleOptions := make([]string, len(roles))
		for i, role := range roles {
			roleOptions[i] = role.RoleName
		}

		roleSelect.Options = roleOptions
		roleSelect.Refresh()
		roleSelect.Show()
		statusLabel.SetText("Please select a role.")
	})
	accountSelect.Hide()

	roleSelect = widget.NewSelect([]string{}, func(value string) {
		logger.Info("Role selected:", value)
		statusLabel.SetText("Assuming role...")

		err := r.awsInterface.AssumeRole(selectedAccountID, value)
		if err != nil {
			logger.Error("Failed to assume role:", err)
			statusLabel.SetText(fmt.Sprintf("Error: Failed to assume role: %v", err))
			return
		}

		statusLabel.SetText("Role assumed successfully. Loading Lambda functions...")

		lambdaFunctions, err := r.awsInterface.ListLambdaFunctions()
		if err != nil {
			logger.Error("Failed to list Lambda functions:", err)
			statusLabel.SetText(fmt.Sprintf("Error: Failed to list Lambda functions: %v", err))
			return
		}

		r.GenerateLambdaContent(lambdaFunctions)
	})
	roleSelect.Hide()

	menuContent := container.NewVBox(
		instructions,
		portalEntry,
		loginButton,
		accountSelect,
		roleSelect,
		statusLabel,
	)

	r.menuContainer.Add(menuContent)
	r.menuContainer.Show()
	r.contentContainer.Hide()
}

func (r *FyneRenderer) GenerateCalendarView() {
	r.ClearScreen()

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	months := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	monthSelect := widget.NewSelect(months, nil)
	monthSelect.SetSelected(currentMonth.String())

	years := make([]string, 4)
	for i := 0; i < 4; i++ {
		years[i] = fmt.Sprintf("%d", currentYear+i)
	}
	yearSelect := widget.NewSelect(years, nil)
	yearSelect.SetSelected(fmt.Sprintf("%d", currentYear))

	createHeader := func() *fyne.Container {
		daysOfWeek := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		headerRow := container.NewGridWithColumns(7)
		for _, day := range daysOfWeek {
			headerRow.Add(widget.NewLabel(day))
		}
		return headerRow
	}

	updateCalendar := func(year int, month time.Month) {
		firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
		lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

		calendarGrid := container.NewGridWithColumns(7)

		for i := 0; i < int(firstOfMonth.Weekday()); i++ {
			calendarGrid.Add(widget.NewLabel(""))
		}

		for day := 1; day <= lastOfMonth.Day(); day++ {
			dayButton := widget.NewButton(fmt.Sprintf("%d", day), func(d int) func() {
				return func() {
					selectedDate := time.Date(year, month, d, 0, 0, 0, 0, now.Location())
					r.GenerateWeeklyView(selectedDate)
				}
			}(day))
			calendarGrid.Add(dayButton)
		}

		r.contentContainer.RemoveAll()
		r.contentContainer.Add(container.NewVBox(
			container.NewHBox(monthSelect, yearSelect),
			createHeader(),
			calendarGrid,
			widget.NewButton("Back to Menu", func() {
				r.GenerateMenu()
			}),
		))
		r.contentContainer.Refresh()
	}

	monthSelect.OnChanged = func(selected string) {
		month := time.Month(monthSelect.SelectedIndex() + 1)
		year, _ := time.Parse("2006", yearSelect.Selected)
		updateCalendar(year.Year(), month)
	}

	yearSelect.OnChanged = func(selected string) {
		year, _ := time.Parse("2006", selected)
		month := time.Month(monthSelect.SelectedIndex() + 1)
		updateCalendar(year.Year(), month)
	}

	updateCalendar(currentYear, currentMonth)

	r.menuContainer.Hide()
	r.contentContainer.Show()
}

func (r *FyneRenderer) GenerateWeeklyView(selectedDate time.Time) {
	r.ClearScreen()

	startOfWeek := selectedDate.AddDate(0, 0, -int(selectedDate.Weekday()))

	headerRow := container.NewGridWithColumns(8)
	headerRow.Add(widget.NewLabel("Time"))
	for i := 0; i < 7; i++ {
		day := startOfWeek.AddDate(0, 0, i)
		headerRow.Add(widget.NewLabel(day.Format("Mon 1/2")))
	}

	weekGrid := container.NewGridWithColumns(8)

	selectedTimes := make(map[time.Time]bool)

	for hour := 0; hour < 24; hour++ {
		weekGrid.Add(widget.NewLabel(fmt.Sprintf("%02d:00", hour)))

		for day := 0; day < 7; day++ {
			cellDate := startOfWeek.AddDate(0, 0, day)
			cellTime := time.Date(cellDate.Year(), cellDate.Month(), cellDate.Day(), hour, 0, 0, 0, cellDate.Location())

			isAvailable := cellTime.Weekday() >= time.Monday &&
				cellTime.Weekday() <= time.Friday &&
				hour >= 9 && hour < 17

			bgColor := color.NRGBA{R: 255, G: 0, B: 0, A: 100}
			if isAvailable {
				bgColor = color.NRGBA{R: 0, G: 255, B: 0, A: 100}
			}

			rect := canvas.NewRectangle(bgColor)

			highlight := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 0, A: 255})
			highlight.Hide()

			cellButton := widget.NewButton("", func(t time.Time, r *canvas.Rectangle, h *canvas.Rectangle) func() {
				return func() {
					if selectedTimes[t] {
						delete(selectedTimes, t)
						r.FillColor = bgColor
						h.Hide()
					} else {
						selectedTimes[t] = true
						r.FillColor = color.NRGBA{R: 0, B: 255, A: 100}
						h.Show()
					}
					r.Refresh()
					h.Refresh()
				}
			}(cellTime, rect, highlight))

			cell := container.New(layout.NewMaxLayout(), rect, highlight, cellButton)

			weekGrid.Add(cell)
		}
	}

	backButton := widget.NewButton("Back to Calendar", func() {
		r.GenerateCalendarView()
	})

	submitButton := widget.NewButton("Submit Selection", func() {
		r.logSelectedTimes(selectedTimes)
	})

	scrollContainer := container.NewVScroll(weekGrid)
	scrollContainer.SetMinSize(fyne.NewSize(600, 400)) //todo adjust size

	r.contentContainer.Add(container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Week of %s", startOfWeek.Format("January 2, 2006"))),
		headerRow,
		scrollContainer,
		container.NewHBox(backButton, submitButton),
	))
	r.contentContainer.Refresh()
}

func (r *FyneRenderer) logSelectedTimes(selectedTimes map[time.Time]bool) {
	var timeSlice []time.Time
	for t := range selectedTimes {
		timeSlice = append(timeSlice, t)
	}
	sort.Slice(timeSlice, func(i, j int) bool { return timeSlice[i].Before(timeSlice[j]) })

	logger.Info("Selected times:")
	for _, t := range timeSlice {
		logger.Info(t.Format("Mon 1/2 15:04"))
	}
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
