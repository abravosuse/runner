package runner

import (
	"encoding/json"
	"os"
	"path"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type InventoryContent struct {
	Groups []*Group
	Nodes  []*Node // this seems unused
}

type Group struct {
	Name  string
	Nodes []*Node
}

type Node struct {
	Name        string
	AnsibleHost string
	AnsibleUser string
	Variables   map[string]interface{}
}

const (
	inventoryTemplate = `{{- range .Nodes }}
{{ .Name }} ansible_host={{ .AnsibleHost }} ansible_user={{ .AnsibleUser }} {{ range $key, $value := .Variables }}{{ $key }}={{ $value }} {{ end }}
{{- end }}
{{- range .Groups }}
[{{ .Name }}]
{{- range .Nodes }}
{{ .Name }} ansible_host={{ .AnsibleHost }} ansible_user={{ .AnsibleUser }} {{ range $key, $value := .Variables }}{{ $key }}={{ $value }} {{ end }}
{{- end }}
{{- end }}
`
	clusterSelectedChecks string = "cluster_selected_checks"
	provider              string = "provider"
)

func CreateInventory(destination string, content *InventoryContent) error {
	t := template.Must(template.New("").Parse(inventoryTemplate))

	if err := os.MkdirAll(path.Dir(destination), 0755); err != nil {
		return err
	}

	f, err := os.Create(destination)
	if err != nil {
		return err
	}
	if err := t.Execute(f, content); err != nil {
		return nil
	}
	f.Close()

	return nil
}

func NewClusterInventoryContent(e *ExecutionEvent) (*InventoryContent, error) {
	content := &InventoryContent{}

	for _, cluster := range e.Clusters {
		nodes := []*Node{}

		jsonChecks, err := json.Marshal(cluster.Checks)
		if err != nil {
			log.Errorf("error marshalling the cluster %s selected checks: %s", cluster.ID.String(), err)
			continue
		}

		for _, host := range cluster.Hosts {
			node := &Node{
				Name:        host.Name,
				AnsibleHost: host.Address,
				AnsibleUser: host.User,
				Variables:   make(map[string]interface{}),
			}

			node.Variables[clusterSelectedChecks] = string(jsonChecks)
			node.Variables[provider] = cluster.Provider

			nodes = append(nodes, node)
		}
		group := &Group{Name: cluster.ID.String(), Nodes: nodes}

		content.Groups = append(content.Groups, group)
	}

	return content, nil
}
