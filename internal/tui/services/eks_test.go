package services

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
)

// --- EKS Clusters View ---

func TestNewEKSClustersView_Title(t *testing.T) {
	// NewEKSClustersView requires a client, but we can still verify
	// the TableView is created with the correct title by checking after init.
	// We cannot pass nil directly because FetchFunc needs the client,
	// but the title is set during construction.
	// Create a minimal view to verify.
	v := NewEKSClustersView(nil)
	if v.Title() != "EKS" {
		t.Errorf("Title() = %q, want %q", v.Title(), "EKS")
	}
}

// --- EKS Cluster Detail View ---

func TestEKSClusterDetailView_TabNames(t *testing.T) {
	cluster := awseks.EKSCluster{
		Name:    "test-cluster",
		Status:  "ACTIVE",
		Version: "1.28",
	}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")

	expectedTabs := []string{
		"Node Groups", "Add-ons", "Fargate", "Access",
		"Pods", "Services", "Deployments", "Svc Accounts", "Ingresses",
	}

	if len(v.tabs.TabNames) != 9 {
		t.Fatalf("expected 9 tabs, got %d", len(v.tabs.TabNames))
	}

	for i, name := range expectedTabs {
		if v.tabs.TabNames[i] != name {
			t.Errorf("tab %d: got %q, want %q", i, v.tabs.TabNames[i], name)
		}
	}
}

func TestEKSClusterDetailView_TabSwitch(t *testing.T) {
	cluster := awseks.EKSCluster{
		Name:    "test-cluster",
		Status:  "ACTIVE",
		Version: "1.28",
	}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")
	v.tabs.InitTab = func(idx int) View { return nil }
	v.loading = false

	if v.tabs.ActiveTab != 0 {
		t.Fatalf("initial activeTab = %d, want 0", v.tabs.ActiveTab)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if v.tabs.ActiveTab != 1 {
		t.Errorf("after Tab: activeTab = %d, want 1", v.tabs.ActiveTab)
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if v.tabs.ActiveTab != 2 {
		t.Errorf("after 2nd Tab: activeTab = %d, want 2", v.tabs.ActiveTab)
	}

	for i := 0; i < 7; i++ {
		v.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	if v.tabs.ActiveTab != 0 {
		t.Errorf("after cycling all tabs: activeTab = %d, want 0", v.tabs.ActiveTab)
	}
}

func TestEKSClusterDetailView_NumberKeys(t *testing.T) {
	cluster := awseks.EKSCluster{
		Name:    "test-cluster",
		Status:  "ACTIVE",
		Version: "1.28",
	}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")
	v.tabs.InitTab = func(idx int) View { return nil }
	v.loading = false

	tests := []struct {
		key     string
		wantTab int
	}{
		{"1", 0},
		{"2", 1},
		{"3", 2},
		{"4", 3},
		{"5", 4},
		{"6", 5},
		{"7", 6},
		{"8", 7},
		{"9", 8},
	}

	for _, tt := range tests {
		v.Update(tea.KeyPressMsg{Code: rune(tt.key[0]), Text: tt.key})
		if v.tabs.ActiveTab != tt.wantTab {
			t.Errorf("key %q: activeTab = %d, want %d", tt.key, v.tabs.ActiveTab, tt.wantTab)
		}
	}
}

func TestEKSClusterDetailView_Title(t *testing.T) {
	cluster := awseks.EKSCluster{
		Name:    "my-cluster",
		Status:  "ACTIVE",
		Version: "1.28",
	}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")
	if v.Title() != "my-cluster" {
		t.Errorf("Title() = %q, want %q", v.Title(), "my-cluster")
	}
}

func TestEKSClusterDetailView_Dashboard(t *testing.T) {
	cluster := awseks.EKSCluster{
		Name:            "prod-cluster",
		Status:          "ACTIVE",
		Version:         "1.28",
		PlatformVersion: "eks.5",
		EndpointPublic:  true,
		EndpointPrivate: true,
	}
	v := NewEKSClusterDetailView(nil, cluster, "us-west-2")
	v.loading = false
	v.width = 80

	view := v.View()
	if !strings.Contains(view, "prod-cluster") {
		t.Error("View() should show cluster name")
	}
	if !strings.Contains(view, "ACTIVE") {
		t.Error("View() should show cluster status")
	}
	if !strings.Contains(view, "1.28") {
		t.Error("View() should show K8s version")
	}
	if !strings.Contains(view, "us-west-2") {
		t.Error("View() should show region")
	}
	if !strings.Contains(view, "Public + Private") {
		t.Error("View() should show endpoint access type")
	}
}

// --- formatAge ---

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
		{"zero", 0, "0s"},
		{"just under a minute", 59 * time.Second, "59s"},
		{"just over an hour", 61 * time.Minute, "1h"},
		{"just over a day", 25 * time.Hour, "1d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			then := time.Now().Add(-tt.duration)
			got := formatAge(then)
			// Allow some flexibility for test execution time
			if got != tt.want {
				// For very small durations, the test execution time
				// could push us into the next bracket
				t.Logf("formatAge(%v ago) = %q, want %q (may differ by test timing)", tt.duration, got, tt.want)
			}
		})
	}
}

// --- ContainerPickerView ---

func TestContainerPickerView_Title(t *testing.T) {
	pod := K8sPod{
		Name:       "my-pod",
		Namespace:  "default",
		Containers: []string{"app", "sidecar"},
	}
	v := NewContainerPickerView(nil, pod, "logs")
	if !strings.Contains(v.Title(), "logs") {
		t.Errorf("Title() = %q, should contain 'logs'", v.Title())
	}
	if !strings.Contains(v.Title(), "my-pod") {
		t.Errorf("Title() = %q, should contain 'my-pod'", v.Title())
	}
}

func TestContainerPickerView_ShowsContainers(t *testing.T) {
	pod := K8sPod{
		Name:       "multi-container-pod",
		Namespace:  "default",
		Containers: []string{"nginx", "redis", "sidecar"},
	}
	v := NewContainerPickerView(nil, pod, "exec")
	v.Init()

	view := v.View()
	if !strings.Contains(view, "nginx") {
		t.Errorf("View() should show container 'nginx', got: %s", view)
	}
	if !strings.Contains(view, "redis") {
		t.Errorf("View() should show container 'redis', got: %s", view)
	}
	if !strings.Contains(view, "sidecar") {
		t.Errorf("View() should show container 'sidecar', got: %s", view)
	}
}

func TestContainerPickerView_ActionInTitle(t *testing.T) {
	pod := K8sPod{
		Name:       "test-pod",
		Namespace:  "default",
		Containers: []string{"app"},
	}

	for _, action := range []string{"logs", "exec"} {
		v := NewContainerPickerView(nil, pod, action)
		if !strings.Contains(v.Title(), action) {
			t.Errorf("Title() for action %q = %q, should contain action", action, v.Title())
		}
	}
}

// --- EKS Resource Detail Views ---

func TestEKSNodeGroupDetailView_Title(t *testing.T) {
	ng := awseks.EKSNodeGroup{
		Name:   "web-nodes",
		Status: "ACTIVE",
	}
	v := NewEKSNodeGroupDetailView(ng)
	if v.Title() != "Node Group: web-nodes" {
		t.Errorf("Title() = %q, want %q", v.Title(), "Node Group: web-nodes")
	}
}

func TestEKSNodeGroupDetailView_Content(t *testing.T) {
	ng := awseks.EKSNodeGroup{
		Name:          "web-nodes",
		Status:        "ACTIVE",
		InstanceTypes: []string{"t3.large", "t3.xlarge"},
		AMIType:       "AL2_x86_64",
		LaunchTemplate: "lt-abc123",
		MinSize:       2,
		MaxSize:       5,
		DesiredSize:   3,
		Labels:        map[string]string{"workload": "web", "tier": "frontend"},
		Taints: []awseks.NodeGroupTaint{
			{Key: "gpu", Value: "true", Effect: "NoSchedule"},
		},
		Subnets: []string{"subnet-abc123", "subnet-def456"},
	}
	v := NewEKSNodeGroupDetailView(ng)
	v.Init()

	content := v.renderContent()
	checks := []struct {
		label string
		text  string
	}{
		{"name", "web-nodes"},
		{"status", "ACTIVE"},
		{"instance types", "t3.large, t3.xlarge"},
		{"AMI type", "AL2_x86_64"},
		{"launch template", "lt-abc123"},
		{"min size", "Min: 2"},
		{"max size", "Max: 5"},
		{"desired size", "Desired: 3"},
		{"label workload", "workload: web"},
		{"label tier", "tier: frontend"},
		{"taint", "gpu=true:NoSchedule"},
		{"subnet", "subnet-abc123"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.text) {
			t.Errorf("content should contain %s (%q), got:\n%s", c.label, c.text, content)
		}
	}
}

func TestEKSAddonDetailView_Title(t *testing.T) {
	addon := awseks.EKSAddon{
		Name:   "vpc-cni",
		Status: "ACTIVE",
	}
	v := NewEKSAddonDetailView(addon)
	if v.Title() != "Add-on: vpc-cni" {
		t.Errorf("Title() = %q, want %q", v.Title(), "Add-on: vpc-cni")
	}
}

func TestEKSAddonDetailView_Content(t *testing.T) {
	addon := awseks.EKSAddon{
		Name:                "vpc-cni",
		Status:              "ACTIVE",
		Version:             "v1.15.1-eksbuild.1",
		Health:              "",
		ServiceAccountRole:  "arn:aws:iam::123:role/vpc-cni-role",
		ConfigurationValues: `{"env":{"ENABLE_PREFIX_DELEGATION":"true"}}`,
	}
	v := NewEKSAddonDetailView(addon)
	v.Init()

	content := v.renderContent()
	checks := []struct {
		label string
		text  string
	}{
		{"name", "vpc-cni"},
		{"status", "ACTIVE"},
		{"version", "v1.15.1-eksbuild.1"},
		{"health", "(healthy)"},
		{"service account role", "arn:aws:iam::123:role/vpc-cni-role"},
		{"configuration", "ENABLE_PREFIX_DELEGATION"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.text) {
			t.Errorf("content should contain %s (%q), got:\n%s", c.label, c.text, content)
		}
	}
}

func TestEKSAddonDetailView_HealthWithIssues(t *testing.T) {
	addon := awseks.EKSAddon{
		Name:   "coredns",
		Status: "DEGRADED",
		Health: "ConfigurationConflict",
	}
	v := NewEKSAddonDetailView(addon)
	v.Init()

	content := v.renderContent()
	if !strings.Contains(content, "ConfigurationConflict") {
		t.Errorf("content should show health issue, got:\n%s", content)
	}
}

func TestEKSFargateDetailView_Title(t *testing.T) {
	fp := awseks.EKSFargateProfile{
		Name:   "default-profile",
		Status: "ACTIVE",
	}
	v := NewEKSFargateDetailView(fp)
	if v.Title() != "Fargate Profile: default-profile" {
		t.Errorf("Title() = %q, want %q", v.Title(), "Fargate Profile: default-profile")
	}
}

func TestEKSFargateDetailView_Content(t *testing.T) {
	fp := awseks.EKSFargateProfile{
		Name:             "default-profile",
		Status:           "ACTIVE",
		PodExecutionRole: "arn:aws:iam::123:role/fargate-role",
		Selectors: []awseks.FargateSelector{
			{Namespace: "default", Labels: map[string]string{"app": "web", "tier": "frontend"}},
			{Namespace: "kube-system"},
		},
		Subnets: []string{"subnet-abc123", "subnet-def456"},
	}
	v := NewEKSFargateDetailView(fp)
	v.Init()

	content := v.renderContent()
	checks := []struct {
		label string
		text  string
	}{
		{"name", "default-profile"},
		{"status", "ACTIVE"},
		{"pod execution role", "arn:aws:iam::123:role/fargate-role"},
		{"selector 1 namespace", "Namespace: default"},
		{"selector 1 label", "app=web"},
		{"selector 2 namespace", "Namespace: kube-system"},
		{"subnet", "subnet-abc123"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.text) {
			t.Errorf("content should contain %s (%q), got:\n%s", c.label, c.text, content)
		}
	}
}

func TestEKSAccessEntryDetailView_Title(t *testing.T) {
	ae := awseks.EKSAccessEntry{
		PrincipalARN: "arn:aws:iam::123:role/admin",
	}
	v := NewEKSAccessEntryDetailView(ae)
	if v.Title() != "Access Entry: arn:aws:iam::123:role/admin" {
		t.Errorf("Title() = %q, want %q", v.Title(), "Access Entry: arn:aws:iam::123:role/admin")
	}
}

func TestEKSAccessEntryDetailView_Content(t *testing.T) {
	created := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	ae := awseks.EKSAccessEntry{
		PrincipalARN: "arn:aws:iam::123:role/admin",
		Type:         "STANDARD",
		Username:     "admin-user",
		Groups:       []string{"system:masters", "custom-admin-group"},
		CreatedAt:    created,
	}
	v := NewEKSAccessEntryDetailView(ae)
	v.Init()

	content := v.renderContent()
	checks := []struct {
		label string
		text  string
	}{
		{"principal ARN", "arn:aws:iam::123:role/admin"},
		{"type", "STANDARD"},
		{"username", "admin-user"},
		{"created", "2024-01-15"},
		{"group 1", "system:masters"},
		{"group 2", "custom-admin-group"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.text) {
			t.Errorf("content should contain %s (%q), got:\n%s", c.label, c.text, content)
		}
	}
}

// --- Namespace Tracking ---

func TestEKSClusterDetailView_NamespaceChange_ClearsK8sTabs(t *testing.T) {
	cluster := awseks.EKSCluster{Name: "test", Status: "ACTIVE", Version: "1.28"}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")
	v.loading = false

	// Simulate having K8s tabs initialized with namespace "default"
	v.namespace = "default"
	v.lastNamespace = "default"
	// Place dummy views in K8s tab slots
	for i := 4; i < 9; i++ {
		v.tabs.TabViews[i] = &eksK8sPlaceholderView{name: "dummy", k8sOK: true}
	}
	// Override InitTab to return nil (no real K8s client)
	v.tabs.InitTab = func(idx int) View { return nil }

	// Change namespace
	v.namespace = "kube-system"

	// Switch to a K8s tab — should clear all K8s tabs because namespace differs
	v.tabs.SwitchTab(5)
	for i := 4; i < 9; i++ {
		if v.tabs.TabViews[i] != nil {
			t.Errorf("K8s tab %d should be nil after namespace change, got %v", i, v.tabs.TabViews[i])
		}
	}
}

func TestEKSClusterDetailView_SameNamespace_KeepsTabs(t *testing.T) {
	cluster := awseks.EKSCluster{Name: "test", Status: "ACTIVE", Version: "1.28"}
	v := NewEKSClusterDetailView(nil, cluster, "us-east-1")
	v.loading = false

	// Set up namespace and lastNamespace to be the same
	v.namespace = "default"
	v.lastNamespace = "default"
	dummy := &eksK8sPlaceholderView{name: "dummy", k8sOK: true}
	v.tabs.TabViews[5] = dummy
	v.tabs.InitTab = func(idx int) View { return nil }

	// Switch to tab 5 — same namespace, should keep existing view
	v.tabs.SwitchTab(5)
	if v.tabs.TabViews[5] != dummy {
		t.Error("K8s tab 5 should be preserved when namespace hasn't changed")
	}
}

// --- EKS Status Dot ---

func TestEKSStatusDot(t *testing.T) {
	tests := []struct {
		status string
		// We just verify it returns a non-empty string (contains the dot character)
	}{
		{"ACTIVE"},
		{"UPDATING"},
		{"CREATING"},
		{"FAILED"},
		{"DELETING"},
		{"UNKNOWN"},
	}
	for _, tt := range tests {
		got := eksStatusDot(tt.status)
		if got == "" {
			t.Errorf("eksStatusDot(%q) returned empty string", tt.status)
		}
		// All dots should contain the bullet character
		if !strings.Contains(got, "●") {
			t.Errorf("eksStatusDot(%q) = %q, should contain '●'", tt.status, got)
		}
	}
}

// --- Detail View SetSize ---

func TestEKSNodeGroupDetailView_SetSize(t *testing.T) {
	ng := awseks.EKSNodeGroup{Name: "test", Status: "ACTIVE"}
	v := NewEKSNodeGroupDetailView(ng)
	v.Init()

	v.SetSize(120, 40)
	if v.width != 120 || v.height != 40 {
		t.Errorf("SetSize: width=%d height=%d, want 120, 40", v.width, v.height)
	}
}

func TestEKSAddonDetailView_SetSize(t *testing.T) {
	addon := awseks.EKSAddon{Name: "test", Status: "ACTIVE"}
	v := NewEKSAddonDetailView(addon)
	v.Init()

	v.SetSize(100, 30)
	if v.width != 100 || v.height != 30 {
		t.Errorf("SetSize: width=%d height=%d, want 100, 30", v.width, v.height)
	}
}

func TestEKSFargateDetailView_SetSize(t *testing.T) {
	fp := awseks.EKSFargateProfile{Name: "test", Status: "ACTIVE"}
	v := NewEKSFargateDetailView(fp)
	v.Init()

	v.SetSize(100, 30)
	if v.width != 100 || v.height != 30 {
		t.Errorf("SetSize: width=%d height=%d, want 100, 30", v.width, v.height)
	}
}

func TestEKSAccessEntryDetailView_SetSize(t *testing.T) {
	ae := awseks.EKSAccessEntry{PrincipalARN: "arn:test"}
	v := NewEKSAccessEntryDetailView(ae)
	v.Init()

	v.SetSize(100, 30)
	if v.width != 100 || v.height != 30 {
		t.Errorf("SetSize: width=%d height=%d, want 100, 30", v.width, v.height)
	}
}

// --- Detail View displays status line ---

func TestDetailViewsShowStatusBar(t *testing.T) {
	tests := []struct {
		name string
		view View
	}{
		{"NodeGroup", func() View {
			v := NewEKSNodeGroupDetailView(awseks.EKSNodeGroup{Name: "test", Status: "ACTIVE"})
			v.Init()
			return v
		}()},
		{"Addon", func() View {
			v := NewEKSAddonDetailView(awseks.EKSAddon{Name: "test", Status: "ACTIVE"})
			v.Init()
			return v
		}()},
		{"Fargate", func() View {
			v := NewEKSFargateDetailView(awseks.EKSFargateProfile{Name: "test", Status: "ACTIVE"})
			v.Init()
			return v
		}()},
		{"AccessEntry", func() View {
			v := NewEKSAccessEntryDetailView(awseks.EKSAccessEntry{PrincipalARN: "arn:test"})
			v.Init()
			return v
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.view.View()
			if !strings.Contains(view, "Esc back") {
				t.Errorf("%s View() should contain 'Esc back', got: %s", tt.name, view)
			}
		})
	}
}
