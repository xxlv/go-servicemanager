package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed trayicon.png
var trayIcon embed.FS
var menuItems = make(map[string]*fyne.MenuItem)
var defaultSize = fyne.NewSize(600, 400)

const configFile = "config.json"

type AppState struct {
	config *Config
	app    fyne.App
	menu   *fyne.Menu
	window fyne.Window
}

func main() {
	config, err := LoadConfigOrCreate(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	myApp := app.New()
	win := myApp.NewWindow("Service Manager")
	win.SetFixedSize(true)
	state := &AppState{
		config: config,
		app:    myApp,
		menu:   fyne.NewMenu("Service Manager"),
		window: win,
	}

	if desk, ok := myApp.(desktop.App); ok {
		state.updateMenu()
		desk.SetSystemTrayMenu(state.menu)
		myApp.Run()

	} else {
		log.Fatal("System tray not supported on this platform")
	}
	defer cleanup(config.Services)
}

// This will create tray system menu
func (s *AppState) updateMenu() {

	s.menu.Items = nil
	s.menu.Items = append(s.menu.Items, fyne.NewMenuItem("Add Service", func() {
		_ = showAddServiceDialog(s)
	}))
	s.menu.Items = append(s.menu.Items, fyne.NewMenuItem("Settings", func() {
		_ = showSettingsDialog(s)
	}))

	separator := &fyne.MenuItem{}
	separator.IsSeparator = true
	s.menu.Items = append(s.menu.Items, separator)

	for i := range s.config.Services {
		service := &s.config.Services[i]
		item := createMenuItem(s, service)
		s.menu.Items = append(s.menu.Items, item)
	}
	s.menu.Items = append(s.menu.Items, separator)
	s.menu.Items = append(s.menu.Items, fyne.NewMenuItem("Quit", func() {
		s.app.Quit()
	}))
	iconResource := loadIconFromEmbed()
	// Refresh the system tray menu
	if desk, ok := s.app.(desktop.App); ok {
		desk.SetSystemTrayIcon(iconResource)
		desk.SetSystemTrayMenu(s.menu)
	}
}

func toggleService(state *AppState, service *Service) {
	var err error
	if service.Running {
		err = service.Stop()
	} else {
		err = service.Start()
	}
	if err != nil {
		log.Printf("Error toggling service %s: %v", service.Name, err)
	}
	state.updateMenu()
}

func createMenuItem(state *AppState, service *Service) *fyne.MenuItem {
	item := fyne.NewMenuItem(service.Name, func() {
		toggleService(state, service)
	})
	updateMenuItem(item, service)
	menuItems[service.Name] = item
	return item
}

func getMenuItem(serviceName string) *fyne.MenuItem {
	return menuItems[serviceName]
}

func updateMenuItem(item *fyne.MenuItem, service *Service) {
	log.Default().Println("update item for service " + service.Name)
	if service.Running {
		item.Label = service.Name + " (Running)"
	} else {
		item.Label = service.Name + " (Stopped)"
	}
}

func showAddServiceDialog(state *AppState) error {
	if state == nil || state.window == nil {
		return fmt.Errorf("state or state.window is nil")
	}

	dialogWin := state.app.NewWindow("Add Service")
	dialogWin.Resize(defaultSize)

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Service Name")

	workDirEntry := widget.NewEntry()
	workDirEntry.SetPlaceHolder("Working Directory")

	commandEntry := widget.NewEntry()
	commandEntry.SetPlaceHolder("Command")

	submitBtn := widget.NewButton("Add", func() {
		newService := Service{
			Name:    nameEntry.Text,
			WorkDir: workDirEntry.Text,
			Command: commandEntry.Text,
		}
		state.config.Services = append(state.config.Services, newService)
		err := SaveConfig(configFile, state.config)
		if err != nil {
			log.Printf("Error saving config: %v", err)
			dialog.ShowError(err, dialogWin)
			return
		}
		state.updateMenu()
		dialogWin.Close()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		dialogWin.Close()
	})

	buttons := container.NewHBox(submitBtn, cancelBtn)
	content := container.NewVBox(nameEntry, workDirEntry, commandEntry, buttons)
	dialogWin.SetContent(content)
	dialogWin.Show()
	return nil
}

func showSettingsDialog(state *AppState) error {
	if state == nil || state.window == nil {
		return fmt.Errorf("state or state.window is nil")
	}
	dialogWin := state.app.NewWindow("Service Settings")
	dialogWin.Resize(defaultSize)
	var selectedService *Service
	servicesList := widget.NewList(
		func() int {
			return len(state.config.Services)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			item := container.NewBorder(nil, nil, widget.NewIcon(theme.MoreVerticalIcon()), nil, label)
			return item
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {

			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(state.config.Services[i].Name)
		},
	)
	servicesList.OnSelected = func(id widget.ListItemID) {
		selectedService = &state.config.Services[id]
		showServiceDetails(state, selectedService)
	}
	main := container.NewStack(container.NewStack(servicesList))
	dialogWin.SetContent(main)
	dialogWin.Show()
	return nil
}

func showServiceDetails(state *AppState, service *Service) {
	dialogWin := state.app.NewWindow(fmt.Sprintf("Service Details - %s", service.Name))
	dialogWin.Resize(fyne.NewSize(600, 400))

	nameEntry := widget.NewEntry()
	nameEntry.SetText(service.Name)

	workDirEntry := widget.NewEntry()
	workDirEntry.SetText(service.WorkDir)

	commandEntry := widget.NewEntry()
	commandEntry.SetText(service.Command)

	addBtn := widget.NewButton("Save", func() {
		newService := &Service{
			Name:    nameEntry.Text,
			WorkDir: workDirEntry.Text,
			Command: commandEntry.Text,
		}
		saveService(state, service, newService)
		dialogWin.Close()
	})

	deleteBtn := widget.NewButton("Delete", func() {
		deleteService(state, service)
	})

	buttons := container.NewHBox(addBtn, deleteBtn)
	content := container.NewVBox(nameEntry, workDirEntry, commandEntry, buttons)
	dialogWin.SetContent(content)
	dialogWin.Show()
}
func saveService(state *AppState, oldService *Service, newService *Service) {
	log.Printf("Save service: old:%+v,new:%+v", oldService, newService)
	needUpdate := []Service{}
	for i := range state.config.Services {
		if state.config.Services[i] != *oldService {
			needUpdate = append(needUpdate, state.config.Services[i])
		}
	}
	needUpdate = append(needUpdate, *newService)
	state.config.Services = needUpdate
	_ = SaveConfig(configFile, state.config)
	state.updateMenu()
}

func deleteService(state *AppState, service *Service) {
	log.Printf("Deleting service: %s", service.Name)
	for i, s := range state.config.Services {
		if s.Name == service.Name {
			state.config.Services = append(state.config.Services[:i], state.config.Services[i+1:]...)
			break
		}
	}
	err := SaveConfig(configFile, state.config)
	if err != nil {
		log.Printf("Error saving config: %v", err)
		dialog.ShowError(err, state.window)
		return
	}
	// Refresh the menu for updating UI
	state.updateMenu()
}

func cleanup(services []Service) {
	for _, service := range services {
		if service.Running {
			err := service.Stop()
			if err != nil {
				fmt.Printf("Error stopping service %s: %v\n", service.Name, err)
			}
		}
	}
}

type Service struct {
	Name    string `json:"name"`
	WorkDir string `json:"workDir"`
	Command string `json:"command"`
	Running bool   `json:"-"`
	Pid     int    `json:"-"`
}

func (s *Service) Start() error {
	if s.Running {
		return fmt.Errorf("service %s is already running", s.Name)
	}

	cmd := exec.Command("sh", "-c", s.Command)
	cmd.Dir = s.WorkDir

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %v", s.Name, err)
	}

	s.Pid = cmd.Process.Pid
	s.Running = true
	fmt.Printf("Started service: %s (PID: %d)\n", s.Name, s.Pid)

	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Service %s exited with error: %v\n", s.Name, err)
		}
		s.Running = false
		s.Pid = 0
		// Ensure menu item is updated when the service stops
		updateMenuItem(getMenuItem(s.Name), s)
	}()

	// Update menu item immediately after starting
	updateMenuItem(getMenuItem(s.Name), s)

	return nil
}

func (s *Service) Stop() error {
	if !s.Running || s.Pid == 0 {
		fmt.Printf("Service %s is not running\n", s.Name)
		s.Running = false
		s.Pid = 0
		return nil
	}

	process, err := os.FindProcess(s.Pid)
	if err != nil {
		return fmt.Errorf("failed to find process for service %s: %v", s.Name, err)
	}

	pgid, err := syscall.Getpgid(s.Pid)
	if err != nil {
		return fmt.Errorf("failed to get process group ID for service %s: %v", s.Name, err)
	}

	err = syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %v", s.Name, err)
	}

	_, err = process.Wait()
	if err != nil {
		return fmt.Errorf("error waiting for service %s to stop: %v", s.Name, err)
	}
	s.Running = false
	s.Pid = 0
	fmt.Printf("Stopped service: %s\n", s.Name)
	return nil
}

func loadIconFromEmbed() fyne.Resource {
	iconData, err := trayIcon.ReadFile("trayicon.png")
	if err != nil {
		log.Fatalf("Failed to read embedded icon file: %v", err)
	}
	return fyne.NewStaticResource("trayicon.png", iconData)
}
