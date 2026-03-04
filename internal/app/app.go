package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	internalaws "tasnim.dev/aws-tui/internal/aws"
	"tasnim.dev/aws-tui/internal/cache"
	"tasnim.dev/aws-tui/internal/config"
	"tasnim.dev/aws-tui/internal/log"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// tickMsg is sent periodically to drive toast expiry and other timers.
type tickMsg time.Time

// paletteNavigateMsg is sent when a palette entry navigates to a service.
type paletteNavigateMsg struct {
	pluginID string
}

// AppConfig holds the dependencies needed to create the App.
type AppConfig struct {
	Registry *plugin.Registry
	Cache    *cache.DB
	Logger   *log.Logger
	Config   *config.Config
	Session  *internalaws.Session
	Region   string
	Profile  string
}

// refreshMsg is sent when the auto-refresh timer fires.
type refreshMsg struct{}

// App is the root Bubble Tea model that composes all UI components.
type App struct {
	router        *Router
	palette       CommandPalette
	toasts        *ui.ToastStack
	statusBar     StatusBar
	breadcrumb    Breadcrumb
	helpOverlay   *ui.HelpOverlay
	registry      *plugin.Registry
	cache         *cache.DB
	logger        *log.Logger
	config        *config.Config
	width, height int
	quitFirst     bool
	quitTime      time.Time
	regionPicker  *ui.Picker
	profilePicker *ui.Picker
	autoRefresh   bool
	refreshCountdown int // seconds until next refresh
	refreshInterval  int // seconds between refreshes
}

// New creates an App with all sub-components wired together.
func New(cfg AppConfig) *App {
	dashboard := NewDashboard(cfg.Registry, nil, cfg.Session, cfg.Cache, cfg.Region, cfg.Profile) // router set below
	router := NewRouter(dashboard)
	router.SetRegistry(cfg.Registry)

	// Wire the dashboard's router reference now that it exists.
	dashboard.router = router

	toasts := ui.NewToastStack()
	router.SetToastFn(func(level plugin.ToastLevel, msg string) {
		toasts.Push(level, msg)
	})

	// Build palette entries from registered plugins.
	plugins := cfg.Registry.All()
	entries := make([]PaletteEntry, 0, len(plugins))
	for _, p := range plugins {
		p := p
		entries = append(entries, PaletteEntry{
			Title:    p.Name(),
			Keywords: []string{p.ID()},
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return paletteNavigateMsg{pluginID: p.ID()}
				}
			},
		})
	}

	interval := cfg.Config.AutoRefreshInterval
	if interval <= 0 {
		interval = 15
	}

	a := &App{
		router:           router,
		palette:          NewCommandPalette(entries),
		toasts:           toasts,
		statusBar:        NewStatusBar(cfg.Region, cfg.Profile),
		breadcrumb:       NewBreadcrumb(),
		helpOverlay:      ui.NewHelpOverlay(nil),
		registry:         cfg.Registry,
		cache:            cfg.Cache,
		logger:           cfg.Logger,
		config:           cfg.Config,
		autoRefresh:      true,
		refreshInterval:  interval,
		refreshCountdown: interval,
	}
	a.statusBar.SetAutoRefresh(true)
	a.statusBar.SetNextRefresh(time.Duration(interval) * time.Second)
	return a
}

// Init runs the dashboard's Init and starts the tick timer.
func (a *App) Init() tea.Cmd {
	dashCmd := a.router.Current().Init()
	tickCmd := tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
	return tea.Batch(dashCmd, tickCmd)
}

// Update handles all messages for the application.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.router.SetSize(msg.Width, msg.Height)
		// Forward to current view.
		_, cmd := a.router.Current().Update(msg)
		return a, cmd

	case tickMsg:
		a.toasts.Tick()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

		if a.autoRefresh {
			a.refreshCountdown--
			if a.refreshCountdown <= 0 {
				a.refreshCountdown = a.refreshInterval
				cmds = append(cmds, func() tea.Msg { return refreshMsg{} })
			}
			a.statusBar.SetNextRefresh(time.Duration(a.refreshCountdown) * time.Second)
		}
		return a, tea.Batch(cmds...)

	case refreshMsg:
		// Re-init the current view to trigger a data refresh.
		return a, a.router.Current().Init()

	case paletteNavigateMsg:
		a.router.Navigate(msg.pluginID)
		return a, a.router.Current().Init()

	case PaletteSelectMsg:
		if msg.Entry.Action != nil {
			return a, msg.Entry.Action()
		}
		return a, nil

	case ui.PickerResult:
		if msg.Canceled {
			a.regionPicker = nil
			a.profilePicker = nil
			return a, nil
		}
		if a.regionPicker != nil {
			a.statusBar.SetRegion(msg.Selected)
			a.config.LastRegion = msg.Selected
			a.config.Save()
			a.regionPicker = nil
		}
		if a.profilePicker != nil {
			a.statusBar.SetProfile(msg.Selected)
			a.config.LastProfile = msg.Selected
			a.config.Save()
			a.profilePicker = nil
		}
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	// Forward unknown messages to current view.
	_, cmd := a.router.Current().Update(msg)
	return a, cmd
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// If help overlay is visible, only handle ? and esc.
	if a.helpOverlay.Visible() {
		switch msg.String() {
		case "?", "esc", "backspace":
			a.helpOverlay.Toggle()
		}
		return a, nil
	}

	// If palette is active, forward to palette.
	if a.palette.Active() {
		var cmd tea.Cmd
		a.palette, cmd = a.palette.Update(msg)
		return a, cmd
	}

	// If a picker is active, forward to picker.
	if a.regionPicker != nil {
		p, cmd := a.regionPicker.Update(msg)
		a.regionPicker = &p
		return a, cmd
	}
	if a.profilePicker != nil {
		p, cmd := a.profilePicker.Update(msg)
		a.profilePicker = &p
		return a, cmd
	}

	switch msg.String() {
	case "q":
		if a.quitFirst && time.Since(a.quitTime) < 2*time.Second {
			return a, tea.Quit
		}
		a.quitFirst = true
		a.quitTime = time.Now()
		a.toasts.Push(plugin.ToastInfo, "Press q again to quit")
		return a, nil

	case "?":
		hints := a.router.Current().KeyHints()
		a.helpOverlay.SetHints(hints)
		a.helpOverlay.Toggle()
		return a, nil

	case "esc", "backspace":
		// Forward to current view first — it may handle esc internally
		// (e.g., S3 preview close, folder navigation up).
		_, cmd := a.router.Current().Update(msg)
		return a, cmd

	case "ctrl+k":
		a.palette.Open()
		return a, nil

	case "R":
		p := ui.NewPicker("Select Region", internalaws.ListRegions())
		a.regionPicker = &p
		return a, nil

	case "P":
		profiles := internalaws.ListProfiles()
		if len(profiles) == 0 {
			profiles = []string{"default"}
		}
		p := ui.NewPicker("Select Profile", profiles)
		a.profilePicker = &p
		return a, nil

	case "a":
		a.autoRefresh = !a.autoRefresh
		a.statusBar.SetAutoRefresh(a.autoRefresh)
		if a.autoRefresh {
			a.refreshCountdown = a.refreshInterval
			a.statusBar.SetNextRefresh(time.Duration(a.refreshInterval) * time.Second)
			a.toasts.Push(plugin.ToastInfo, "Auto-refresh enabled")
		} else {
			a.toasts.Push(plugin.ToastInfo, "Auto-refresh disabled")
		}
		return a, nil
	}

	// Forward to current view.
	_, cmd := a.router.Current().Update(msg)
	return a, cmd
}

// View renders the full application UI.
func (a *App) View() tea.View {
	var b strings.Builder

	// Line 1: breadcrumb.
	crumbs := a.router.Breadcrumbs()
	b.WriteString(a.breadcrumb.View(crumbs, time.Time{}, a.width))
	b.WriteByte('\n')
	b.WriteByte('\n') // margin below breadcrumb

	// Determine main content: overlay takes precedence over the view.
	hasOverlay := a.palette.Active() || a.regionPicker != nil || a.profilePicker != nil || a.helpOverlay.Visible()
	if hasOverlay {
		if a.palette.Active() {
			b.WriteString(a.palette.View())
		} else if a.regionPicker != nil {
			b.WriteString(a.regionPicker.View())
		} else if a.profilePicker != nil {
			b.WriteString(a.profilePicker.View())
		} else if a.helpOverlay.Visible() {
			b.WriteString(a.helpOverlay.View())
		}
	} else {
		currentView := a.router.Current().View()
		b.WriteString(currentView.Content)
	}

	// Ensure we fill remaining height for the status bar at the bottom.
	contentLines := strings.Count(b.String(), "\n")
	remaining := a.height - contentLines - 2 // 1 for status bar, 1 for safety
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	// Toasts (rendered just above status bar).
	visible := a.toasts.Visible()
	for _, t := range visible {
		b.WriteString(t.Message)
		b.WriteByte('\n')
	}

	// Last line: status bar.
	b.WriteString(a.statusBar.View(a.width))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
