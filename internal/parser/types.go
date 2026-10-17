package parser

// ShapeType represents the supported geometric visual shapes in SAGE.
type ShapeType string

const (
	// Default and supported shape types.
	// You can add or rename shape types here to support different diagram elements.
	ShapeRect     ShapeType = "rect"
	ShapeCircle   ShapeType = "circle"
	ShapeDiamond  ShapeType = "diamond"
	ShapeTriangle ShapeType = "triangle"
	ShapeCylinder ShapeType = "cylinder"
)

// Node represents an architectural component in the diagram.
type Node struct {
	ID       string    `json:"id"`
	Shape    ShapeType `json:"shape"`
	Label    string    `json:"label"`
	Color    string    `json:"color,omitempty"`
	Group    string    `json:"group,omitempty"`
	FilePath string    `json:"filePath,omitempty"`
}

// Edge represents a directed or bidirectional dependency connector between components.
type Edge struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Label         string `json:"label,omitempty"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
}

// Graph contains all extracted nodes, edges, non-fatal syntax warnings, and optional Mermaid syntax.
type Graph struct {
	Nodes    []Node   `json:"nodes"`
	Edges    []Edge   `json:"edges"`
	Warnings []string `json:"warnings,omitempty"`
	Mermaid  string   `json:"mermaid,omitempty"`
}
