package app

import (
	tea "charm.land/bubbletea/v2"
	"tasnim.dev/aws-tui/internal/plugin"
)

// Router manages a stack of views and implements plugin.Router.
type Router struct {
	stack         []plugin.View
	registry      *plugin.Registry
	toastFn       func(plugin.ToastLevel, string)
	width, height int
}

// NewRouter creates a Router with the given root view on the stack.
func NewRouter(root plugin.View) *Router {
	return &Router{
		stack: []plugin.View{root},
	}
}

// SetRegistry sets the plugin registry used by Navigate and NavigateDetail.
func (r *Router) SetRegistry(reg *plugin.Registry) {
	r.registry = reg
}

// SetToastFn sets the function called by Toast.
func (r *Router) SetToastFn(fn func(plugin.ToastLevel, string)) {
	r.toastFn = fn
}

// SetSize stores the current terminal dimensions so new views receive them.
func (r *Router) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// Push adds a view to the top of the stack and forwards the current window size.
func (r *Router) Push(v plugin.View) {
	r.stack = append(r.stack, v)
	if r.width > 0 && r.height > 0 {
		v.Update(tea.WindowSizeMsg{Width: r.width, Height: r.height})
	}
}

// Pop removes the top view. It is a no-op when at the root.
func (r *Router) Pop() {
	if len(r.stack) > 1 {
		r.stack = r.stack[:len(r.stack)-1]
	}
}

// Home pops all views except the root.
func (r *Router) Home() {
	r.stack = r.stack[:1]
}

// Current returns the view at the top of the stack.
func (r *Router) Current() plugin.View {
	return r.stack[len(r.stack)-1]
}

// Depth returns the number of views on the stack.
func (r *Router) Depth() int {
	return len(r.stack)
}

// Breadcrumbs returns the titles of all views in the stack, ordered root to top.
func (r *Router) Breadcrumbs() []string {
	titles := make([]string, len(r.stack))
	for i, v := range r.stack {
		titles[i] = v.Title()
	}
	return titles
}

// Navigate goes Home, then pushes the ListView for the given plugin.
// It is a no-op if the plugin is not found.
func (r *Router) Navigate(pluginID string) {
	if r.registry == nil {
		return
	}
	p := r.registry.Get(pluginID)
	if p == nil {
		return
	}
	r.Home()
	r.Push(p.ListView(r))
}

// NavigateDetail goes Home, pushes the ListView, then pushes the DetailView.
// It is a no-op if the plugin is not found.
func (r *Router) NavigateDetail(pluginID, id string) {
	if r.registry == nil {
		return
	}
	p := r.registry.Get(pluginID)
	if p == nil {
		return
	}
	r.Home()
	r.Push(p.ListView(r))
	r.Push(p.DetailView(r, id))
}

// Toast calls the configured toast function. It is a no-op if no function is set.
func (r *Router) Toast(level plugin.ToastLevel, msg string) {
	if r.toastFn != nil {
		r.toastFn(level, msg)
	}
}

// Compile-time check that Router implements plugin.Router.
var _ plugin.Router = (*Router)(nil)
