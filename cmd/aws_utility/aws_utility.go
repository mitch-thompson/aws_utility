package main

import (
	"aws_utility/pkg/render"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"log"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("AWS Utility")

	menuContainer := container.NewVBox()
	contentContainer := container.NewVBox()

	mainContainer := container.NewVBox(menuContainer, contentContainer)

	renderer, err := render.NewFyneRenderer(myWindow, menuContainer, contentContainer)
	if err != nil {
		log.Fatalf("Failed to create renderer: %v", err)
	}

	renderer.GenerateMenu()

	myWindow.SetContent(mainContainer)
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.ShowAndRun()
}
