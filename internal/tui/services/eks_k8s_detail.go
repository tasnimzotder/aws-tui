package services

// ---------------------------------------------------------------------------
// Shared messages for K8s detail views
// ---------------------------------------------------------------------------

type k8sPodDetailMsg struct {
	containers []PodContainerDetail
	conditions []PodCondition
	events     []PodEvent
	node       string
	podIP      string
}

type k8sServiceDetailMsg struct {
	endpoints  []ServiceEndpoint
	selector   string
	clusterIP  string
	externalIP string
	svcType    string
}

type k8sDeploymentDetailMsg struct {
	revisions  []DeploymentRevision
	strategy   string
	maxSurge   string
	maxUnavail string
}

type k8sServiceAccountDetailMsg struct {
	annotations map[string]string
	labels      map[string]string
	secrets     []string
	automount   string
}

type k8sDetailErrorMsg struct {
	err error
}
