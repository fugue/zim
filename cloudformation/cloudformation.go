package cloudformation

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation"
	"github.com/awslabs/goformation/intrinsics"
	"github.com/LuminalHQ/zim/graph"
)

// DiscoverDependencies returns a Graph representing dependencies between
// Cloudformation stacks
func DiscoverDependencies(templates map[string]string) (*graph.Graph, error) {

	g := graph.NewGraph()

	for name, tmplPath := range templates {
		g.AddNode(name)
		options := &intrinsics.ProcessorOptions{
			IntrinsicHandlerOverrides: map[string]intrinsics.IntrinsicHandler{
				"Fn::ImportValue": importValueHandler(g, name, nil),
			},
		}
		_, err := goformation.OpenWithOptions(tmplPath, options)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse template %s: %v", name, err)
		}
	}

	// nodes := res.graph.Nodes()
	// for {
	// 	if !nodes.Next() {
	// 		break
	// 	}
	// 	node := nodes.Node()
	// 	fmt.Println("NODE", node)
	// }

	return g, nil
}

func importValueToTemplate(name string) string {
	stack := strings.SplitN(name, ":", 2)[0]
	trimmed := strings.TrimPrefix(stack, "fugue-risk-manager-")
	return strings.Replace(trimmed, "-", "_", -1)
}

func importValueHandler(g *graph.Graph, from string, ignore map[string]bool) intrinsics.IntrinsicHandler {
	return func(name string, input, template interface{}) interface{} {
		stack := input.(string)
		to := importValueToTemplate(input.(string))
		if ignore[stack] || ignore[to] {
			return nil
		}
		g.AddDependency(from, to)
		fmt.Println("DEP", from, to)
		return nil
	}
}
