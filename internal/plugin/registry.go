package plugin

type Registry struct {
	plugins map[string]ServicePlugin
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]ServicePlugin)}
}

func (r *Registry) Add(p ServicePlugin) {
	r.plugins[p.ID()] = p
	r.order = append(r.order, p.ID())
}

func (r *Registry) Get(id string) ServicePlugin {
	return r.plugins[id]
}

func (r *Registry) All() []ServicePlugin {
	var result []ServicePlugin
	for _, id := range r.order {
		result = append(result, r.plugins[id])
	}
	return result
}
