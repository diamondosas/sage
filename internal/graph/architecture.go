package graph

import (
	"path/filepath"
	"sort"
	"strings"
)

// ClusterID represents a strongly typed domain cluster identifier.
type ClusterID string

// Canonical Architectural Domain Cluster IDs.
const (
	ClusterCLI     ClusterID = "cli"
	ClusterAI      ClusterID = "ai"
	ClusterGit     ClusterID = "git"
	ClusterGraph   ClusterID = "graph"
	ClusterConfig  ClusterID = "config"
	ClusterTesting ClusterID = "testing"
	ClusterUI      ClusterID = "ui"
	ClusterDocs    ClusterID = "docs"
	ClusterCore    ClusterID = "core"
)

// ArchitectureNode represents an individual file in the architecture visualizer.
type ArchitectureNode struct {
	Path         string   `json:"path"`
	Summary      string   `json:"summary"`
	Role         string   `json:"role"`
	Exports      []string `json:"exports"`
	Dependencies []string `json:"dependencies"`
}

// StoredFileRecord is an alias for ArchitectureNode for database ingestion and extraction.
type StoredFileRecord = ArchitectureNode

// ArchitectureCluster represents a coarse-grained architectural domain.
type ArchitectureCluster struct {
	ID          ClusterID          `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	FileCount   int                `json:"file_count"`
	Nodes       []ArchitectureNode `json:"nodes"`
}

// ArchitectureEdge represents a directed dependency link between two architectural clusters.
type ArchitectureEdge struct {
	Source      ClusterID `json:"source"`
	Target      ClusterID `json:"target"`
	Weight      int       `json:"weight"`
	Description string    `json:"description"`
}

// ArchitectureGraph holds the complete cluster-based system view.
type ArchitectureGraph struct {
	Clusters   []ArchitectureCluster `json:"clusters"`
	Edges      []ArchitectureEdge    `json:"edges"`
	TotalFiles int                   `json:"total_files"`
}

// ClusterMeta defines the display properties for an architectural cluster.
type ClusterMeta struct {
	ID          ClusterID
	Name        string
	Description string
}

var clusterMetadata = map[ClusterID]ClusterMeta{
	ClusterCLI: {
		ID:          ClusterCLI,
		Name:        "Command Line Interface",
		Description: "CLI command handling, argument parsing, and user interactions",
	},
	ClusterAI: {
		ID:          ClusterAI,
		Name:        "AI Synthesis Engine",
		Description: "LLM semantic reasoning, heuristic extraction, and architectural summarization",
	},
	ClusterGit: {
		ID:          ClusterGit,
		Name:        "Git Extractor",
		Description: "Commit history parsing, code ownership attribution, and expertise decay scoring",
	},
	ClusterGraph: {
		ID:          ClusterGraph,
		Name:        "Knowledge Graph Storage",
		Description: "Relational persistence, lineage tracking, and architectural graph querying",
	},
	ClusterConfig: {
		ID:          ClusterConfig,
		Name:        "Configuration",
		Description: "Application settings, configuration flags, and environment parameters",
	},
	ClusterTesting: {
		ID:          ClusterTesting,
		Name:        "Automated Test Suite",
		Description: "Unit tests, integration specifications, and regression verification",
	},
	ClusterUI: {
		ID:          ClusterUI,
		Name:        "Web UI & Visualizer",
		Description: "Embedded HTTP server, static asset delivery, and interactive browser visualizer",
	},
	ClusterDocs: {
		ID:          ClusterDocs,
		Name:        "Documentation & ADRs",
		Description: "Architecture decision records, project specifications, and user guides",
	},
	ClusterCore: {
		ID:          ClusterCore,
		Name:        "Core Utilities",
		Description: "Shared domain abstractions and core helper utilities",
	},
}

// roleContainsToken checks if a role string contains any of the specified tokens as discrete words.
func roleContainsToken(role string, tokens ...string) bool {
	fields := strings.FieldsFunc(role, func(r rune) bool {
		return r == ' ' || r == '/' || r == '-' || r == '_' || r == ',' || r == '.'
	})
	for _, f := range fields {
		for _, tok := range tokens {
			if strings.EqualFold(f, tok) {
				return true
			}
		}
	}
	return false
}

// ClassifyCluster assigns a file path and role to an architectural cluster.
func ClassifyCluster(path, role string) ClusterID {
	cleanPath := filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
	cleanRole := strings.ToLower(strings.TrimSpace(role))

	// 1. Automated Testing (takes precedence on file extension or path)
	if strings.HasSuffix(cleanPath, "_test.go") ||
		strings.Contains(cleanPath, "_test.") ||
		strings.HasPrefix(cleanPath, "test/") ||
		strings.Contains(cleanPath, "/test/") ||
		roleContainsToken(cleanRole, "test", "tests", "testing") {
		return ClusterTesting
	}

	// 2. Command Line Interface
	if strings.HasPrefix(cleanPath, "cmd/") ||
		strings.Contains(cleanPath, "/cmd/") ||
		strings.Contains(cleanPath, "cli/") ||
		cleanPath == "main.go" ||
		strings.HasSuffix(cleanPath, "/main.go") ||
		roleContainsToken(cleanRole, "cli", "command", "commands") {
		return ClusterCLI
	}

	// 3. AI Synthesizer Engine
	if strings.HasPrefix(cleanPath, "internal/ai/") ||
		strings.Contains(cleanPath, "/ai/") ||
		roleContainsToken(cleanRole, "ai", "synthesis", "synthesizer", "mind") {
		return ClusterAI
	}

	// 4. Git Extractor
	if strings.HasPrefix(cleanPath, "internal/git/") ||
		strings.Contains(cleanPath, "/git/") ||
		roleContainsToken(cleanRole, "git", "extractor") {
		return ClusterGit
	}

	// 5. Knowledge Graph Storage
	if strings.HasPrefix(cleanPath, "internal/graph/") ||
		strings.Contains(cleanPath, "/graph/") ||
		strings.Contains(cleanPath, "database/") ||
		roleContainsToken(cleanRole, "graph", "storage", "database") {
		return ClusterGraph
	}

	// 6. Web UI & Visualizer
	if strings.HasPrefix(cleanPath, "internal/ui/") ||
		strings.Contains(cleanPath, "/ui/") ||
		strings.Contains(cleanPath, "assets/") ||
		roleContainsToken(cleanRole, "ui", "visualizer", "web") {
		return ClusterUI
	}

	// 7. Configuration
	if strings.HasPrefix(cleanPath, "internal/config/") ||
		strings.Contains(cleanPath, "/config/") ||
		roleContainsToken(cleanRole, "config", "configuration") {
		return ClusterConfig
	}

	// 8. Documentation
	if strings.HasPrefix(cleanPath, "docs/") ||
		strings.Contains(cleanPath, "/docs/") ||
		strings.HasSuffix(cleanPath, ".md") ||
		roleContainsToken(cleanRole, "doc", "docs", "documentation") {
		return ClusterDocs
	}

	return ClusterCore
}

// ClassifyDependencyCluster maps an imported dependency path to an internal architectural cluster.
// Returns empty string if the dependency is standard library or third-party external.
func ClassifyDependencyCluster(dep string) ClusterID {
	d := strings.ToLower(strings.Trim(dep, `"`))

	// Must be an internal package or relative module import
	isInternal := strings.HasPrefix(d, "sage/") ||
		strings.HasPrefix(d, "./") ||
		strings.HasPrefix(d, "../") ||
		strings.HasPrefix(d, "internal/") ||
		strings.HasPrefix(d, "cmd/")

	if !isInternal {
		return ""
	}

	switch {
	case strings.Contains(d, "_test") || strings.Contains(d, "/test/") || strings.HasSuffix(d, "/test"):
		return ClusterTesting
	case strings.Contains(d, "/cli/") || strings.HasSuffix(d, "/cli") || strings.Contains(d, "cmd/"):
		return ClusterCLI
	case strings.Contains(d, "/ai/") || strings.HasSuffix(d, "/ai"):
		return ClusterAI
	case strings.Contains(d, "/git/") || strings.HasSuffix(d, "/git"):
		return ClusterGit
	case strings.Contains(d, "/graph/") || strings.HasSuffix(d, "/graph"):
		return ClusterGraph
	case strings.Contains(d, "/ui/") || strings.HasSuffix(d, "/ui"):
		return ClusterUI
	case strings.Contains(d, "/config/") || strings.HasSuffix(d, "/config"):
		return ClusterConfig
	case strings.Contains(d, "/docs/") || strings.HasSuffix(d, "/docs"):
		return ClusterDocs
	default:
		return ClusterCore
	}
}

// BuildArchitectureGraph groups file records into clusters and calculates inter-cluster dependency edges.
func BuildArchitectureGraph(files []StoredFileRecord) *ArchitectureGraph {
	clustersMap := make(map[ClusterID]*ArchitectureCluster)
	fileToCluster := make(map[string]ClusterID)

	// Group files into clusters
	for _, f := range files {
		clusterID := ClassifyCluster(f.Path, f.Role)
		fileToCluster[f.Path] = clusterID

		cluster, exists := clustersMap[clusterID]
		if !exists {
			meta, ok := clusterMetadata[clusterID]
			if !ok {
				meta = ClusterMeta{
					ID:          clusterID,
					Name:        strings.Title(string(clusterID)),
					Description: "Architectural component group",
				}
			}
			cluster = &ArchitectureCluster{
				ID:          meta.ID,
				Name:        meta.Name,
				Description: meta.Description,
				Nodes:       []ArchitectureNode{},
			}
			clustersMap[clusterID] = cluster
		}

		cluster.Nodes = append(cluster.Nodes, ArchitectureNode{
			Path:         f.Path,
			Summary:      f.Summary,
			Role:         f.Role,
			Exports:      f.Exports,
			Dependencies: f.Dependencies,
		})
		cluster.FileCount = len(cluster.Nodes)
	}

	// Calculate inter-cluster dependency edges
	type edgeKey struct {
		source ClusterID
		target ClusterID
	}
	edgeWeights := make(map[edgeKey]int)

	for _, f := range files {
		sourceCluster := fileToCluster[f.Path]
		if sourceCluster == "" {
			continue
		}

		for _, dep := range f.Dependencies {
			var targetCluster ClusterID

			// 1. Check if dep directly refers to another stored file
			if targetCls, ok := fileToCluster[dep]; ok {
				targetCluster = targetCls
			} else {
				// 2. Classify package import path
				targetCluster = ClassifyDependencyCluster(dep)
			}

			// Inter-cluster links only: omit self-loops (source == target) and unknown external dependencies
			if targetCluster != "" && targetCluster != sourceCluster {
				edgeWeights[edgeKey{source: sourceCluster, target: targetCluster}]++
			}
		}
	}

	// Prepare ordered clusters
	clusterOrder := []ClusterID{
		ClusterCLI,
		ClusterAI,
		ClusterGit,
		ClusterGraph,
		ClusterUI,
		ClusterConfig,
		ClusterTesting,
		ClusterDocs,
		ClusterCore,
	}

	var clusters []ArchitectureCluster
	for _, id := range clusterOrder {
		if c, ok := clustersMap[id]; ok {
			// Sort nodes within cluster by path
			sort.Slice(c.Nodes, func(i, j int) bool {
				return c.Nodes[i].Path < c.Nodes[j].Path
			})
			clusters = append(clusters, *c)
			delete(clustersMap, id)
		}
	}

	// Any remaining clusters
	for _, c := range clustersMap {
		sort.Slice(c.Nodes, func(i, j int) bool {
			return c.Nodes[i].Path < c.Nodes[j].Path
		})
		clusters = append(clusters, *c)
	}

	// Prepare sorted edges
	var edges []ArchitectureEdge
	for key, weight := range edgeWeights {
		sourceName := string(key.source)
		targetName := string(key.target)
		if sourceMeta, ok := clusterMetadata[key.source]; ok {
			sourceName = sourceMeta.Name
		}
		if targetMeta, ok := clusterMetadata[key.target]; ok {
			targetName = targetMeta.Name
		}

		edges = append(edges, ArchitectureEdge{
			Source:      key.source,
			Target:      key.target,
			Weight:      weight,
			Description: sourceName + " → " + targetName,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})

	return &ArchitectureGraph{
		Clusters:   clusters,
		Edges:      edges,
		TotalFiles: len(files),
	}
}
