package services

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type regionChangeMsg struct{ region string }

type regionItem struct {
	code string
	name string
}

func (r regionItem) Title() string       { return r.code }
func (r regionItem) Description() string { return r.name }
func (r regionItem) FilterValue() string { return r.code + " " + r.name }

type regionPickerView struct {
	list list.Model
}

func newRegionPickerView() *regionPickerView {
	regions := []list.Item{
		regionItem{code: "us-east-1", name: "US East (N. Virginia)"},
		regionItem{code: "us-east-2", name: "US East (Ohio)"},
		regionItem{code: "us-west-1", name: "US West (N. California)"},
		regionItem{code: "us-west-2", name: "US West (Oregon)"},
		regionItem{code: "eu-west-1", name: "Europe (Ireland)"},
		regionItem{code: "eu-west-2", name: "Europe (London)"},
		regionItem{code: "eu-central-1", name: "Europe (Frankfurt)"},
		regionItem{code: "ap-southeast-1", name: "Asia Pacific (Singapore)"},
		regionItem{code: "ap-southeast-2", name: "Asia Pacific (Sydney)"},
		regionItem{code: "ap-northeast-1", name: "Asia Pacific (Tokyo)"},
		regionItem{code: "ap-northeast-2", name: "Asia Pacific (Seoul)"},
		regionItem{code: "ap-south-1", name: "Asia Pacific (Mumbai)"},
		regionItem{code: "ca-central-1", name: "Canada (Central)"},
		regionItem{code: "sa-east-1", name: "South America (São Paulo)"},
	}

	l := list.New(regions, list.NewDefaultDelegate(), 60, 16)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	return &regionPickerView{list: l}
}

func (v *regionPickerView) Title() string { return "Region" }
func (v *regionPickerView) Init() tea.Cmd { return nil }

func (v *regionPickerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Only handle Enter when not filtering
		if msg.String() == "enter" && !v.list.SettingFilter() {
			selected, ok := v.list.SelectedItem().(regionItem)
			if !ok {
				return v, nil
			}
			return v, func() tea.Msg {
				return regionChangeMsg{region: selected.code}
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *regionPickerView) View() string { return v.list.View() }

func (v *regionPickerView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// CapturesInput returns true when the list is filtering, so keystrokes
// go to the filter input rather than being intercepted by the model.
func (v *regionPickerView) CapturesInput() bool {
	return v.list.SettingFilter()
}
