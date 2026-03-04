package app

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tasnim.dev/aws-tui/internal/plugin"
)

// fakeView implements plugin.View for testing.
type fakeView struct {
	title string
}

func (f *fakeView) Init() tea.Cmd                    { return nil }
func (f *fakeView) Update(tea.Msg) (tea.Model, tea.Cmd) { return f, nil }
func (f *fakeView) View() tea.View                   { return tea.NewView("") }
func (f *fakeView) Title() string                    { return f.title }
func (f *fakeView) KeyHints() []plugin.KeyHint       { return nil }

func newFakeView(title string) *fakeView {
	return &fakeView{title: title}
}

func TestRouter(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "initial view is root with depth 1",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				assert.Equal(t, 1, r.Depth())
				assert.Equal(t, root, r.Current())
			},
		},
		{
			name: "push adds view and increases depth",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				detail := newFakeView("detail")
				r.Push(detail)

				assert.Equal(t, 2, r.Depth())
				assert.Equal(t, detail, r.Current())
			},
		},
		{
			name: "pop removes top and falls back to previous",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				detail := newFakeView("detail")
				r.Push(detail)
				r.Pop()

				assert.Equal(t, 1, r.Depth())
				assert.Equal(t, root, r.Current())
			},
		},
		{
			name: "pop at root is no-op",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				r.Pop()

				assert.Equal(t, 1, r.Depth())
				assert.Equal(t, root, r.Current())
			},
		},
		{
			name: "home returns to root regardless of depth",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				r.Push(newFakeView("a"))
				r.Push(newFakeView("b"))
				r.Push(newFakeView("c"))

				require.Equal(t, 4, r.Depth())

				r.Home()

				assert.Equal(t, 1, r.Depth())
				assert.Equal(t, root, r.Current())
			},
		},
		{
			name: "breadcrumbs returns ordered titles",
			fn: func(t *testing.T) {
				root := newFakeView("Home")
				r := NewRouter(root)

				r.Push(newFakeView("Pods"))
				r.Push(newFakeView("pod-abc"))

				assert.Equal(t, []string{"Home", "Pods", "pod-abc"}, r.Breadcrumbs())
			},
		},
		{
			name: "navigate goes home then pushes list view",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)
				r.Push(newFakeView("stale"))

				reg := plugin.NewRegistry()
				fp := &fakePlugin{id: "pods", listTitle: "Pods List"}
				reg.Add(fp)
				r.SetRegistry(reg)

				r.Navigate("pods")

				assert.Equal(t, 2, r.Depth())
				assert.Equal(t, "Pods List", r.Current().Title())
			},
		},
		{
			name: "navigate with unknown plugin is no-op",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)
				r.SetRegistry(plugin.NewRegistry())

				r.Navigate("nonexistent")

				assert.Equal(t, 1, r.Depth())
				assert.Equal(t, root, r.Current())
			},
		},
		{
			name: "navigate detail pushes list then detail",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				reg := plugin.NewRegistry()
				fp := &fakePlugin{id: "pods", listTitle: "Pods List", detailTitle: "pod-xyz"}
				reg.Add(fp)
				r.SetRegistry(reg)

				r.NavigateDetail("pods", "xyz")

				assert.Equal(t, 3, r.Depth())
				assert.Equal(t, "pod-xyz", r.Current().Title())
				assert.Equal(t, []string{"root", "Pods List", "pod-xyz"}, r.Breadcrumbs())
			},
		},
		{
			name: "toast calls toastFn",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				var gotLevel plugin.ToastLevel
				var gotMsg string
				r.SetToastFn(func(level plugin.ToastLevel, msg string) {
					gotLevel = level
					gotMsg = msg
				})

				r.Toast(plugin.ToastError, "something broke")

				assert.Equal(t, plugin.ToastError, gotLevel)
				assert.Equal(t, "something broke", gotMsg)
			},
		},
		{
			name: "toast without fn set is no-op",
			fn: func(t *testing.T) {
				root := newFakeView("root")
				r := NewRouter(root)

				// Should not panic
				r.Toast(plugin.ToastInfo, "hello")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.fn)
	}
}

// fakePlugin implements plugin.ServicePlugin for Navigate/NavigateDetail tests.
type fakePlugin struct {
	id          string
	listTitle   string
	detailTitle string
}

func (f *fakePlugin) ID() string   { return f.id }
func (f *fakePlugin) Name() string { return f.id }
func (f *fakePlugin) Icon() string { return "" }
func (f *fakePlugin) Summary(_ context.Context) (plugin.ServiceSummary, error) {
	return plugin.ServiceSummary{}, nil
}
func (f *fakePlugin) ListView(router plugin.Router) plugin.View {
	return newFakeView(f.listTitle)
}
func (f *fakePlugin) DetailView(router plugin.Router, id string) plugin.View {
	return newFakeView(f.detailTitle)
}
func (f *fakePlugin) Commands() []plugin.Command   { return nil }
func (f *fakePlugin) PollConfig() plugin.PollConfig { return plugin.PollConfig{} }
