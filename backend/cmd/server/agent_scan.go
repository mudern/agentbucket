package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (app *App) scanRepositories(repos []Repository) []Repository {
	scanned := make([]Repository, 0, len(repos))
	for _, repo := range repos {
		next := repo
		// Auto-generate localPath for Remote repos that don't have one yet
		if next.Provider != "Local" && next.LocalPath == "" && next.URL != "" {
			next.LocalPath = filepath.Join(app.dataDir, "repos", slug(next.URL))
		}
		// Auto-clone Remote repos that haven't been cloned yet
		if next.Provider != "Local" && next.URL != "" {
			gitDir := filepath.Join(next.LocalPath, ".git")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				go func() {
					os.MkdirAll(filepath.Dir(next.LocalPath), 0o755)
					cmd := exec.Command("git", "clone", "-b", next.Branch, next.URL, next.LocalPath)
					out, err := cmd.CombinedOutput()
					if err != nil {
						log.Printf("git clone failed for %s: %v - %s", next.ID, err, string(out))
					} else {
						log.Printf("git clone succeeded for %s", next.ID)
					}
				}()
			}
		}
		next.Commits = scanCommits(repo)
		if next.Commits == nil {
			next.Commits = []Commit{}
		}
		if len(next.Commits) == 0 {
			// No git history — create a synthetic commit from filesystem scan
			agents := scanAgents(repo)
			if len(agents) > 0 {
				next.Commits = []Commit{{
					Hash:        shortHash(repo.URL + repo.LocalPath + repo.AgentsPath),
					Message:     "scanned local agent manifests",
					CommittedAt: "刚刚",
					Agents:      agents,
				}}
			}
		}
		scanned = append(scanned, next)
	}
	return scanned
}

func scanCommits(repo Repository) []Commit {
	root := repoPath(repo)
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("git", "-C", root, "log",
		"--format=%H||%s||%aI", "-20")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	agents := scanAgents(repo)
	var commits []Commit
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "||", 3)
		if len(parts) < 2 {
			continue
		}
		hash := parts[0]
		message := ""
		committedAt := ""
		if len(parts) >= 2 {
			message = parts[1]
		}
		if len(parts) >= 3 {
			committedAt = parts[2]
		}
		// All scanned commits share the same agent definitions from HEAD
		commits = append(commits, Commit{
			Hash:        hash,
			Message:     message,
			CommittedAt: committedAt,
			Agents:      agents,
		})
	}
	return commits
}

func scanAgents(repo Repository) []Agent {
	root := repoPath(repo)
	gitDir := filepath.Join(root, ".git")
	hasGit := true
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		hasGit = false
	}

	agentsDirRel := filepath.FromSlash(repo.AgentsPath)
	var entries []string
	if hasGit {
		// Read agent directories from git tree at HEAD — only committed files count
		cmd := exec.Command("git", "-C", root, "ls-tree", "--name-only", "HEAD:"+agentsDirRel)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return []Agent{}
		}
		for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				entries = append(entries, name)
			}
		}
	} else {
		// Fallback: read from filesystem for non-git local dirs
		agentsDir := filepath.Join(root, agentsDirRel)
		dirEntries, err := os.ReadDir(agentsDir)
		if err != nil {
			return []Agent{}
		}
		for _, de := range dirEntries {
			if de.IsDir() {
				entries = append(entries, de.Name())
			}
		}
	}

	agents := make([]Agent, 0)
	for _, name := range entries {
		var raw []byte
		var manifestPath string
		if hasGit {
			manifestPath = filepath.Join(agentsDirRel, name, "agent.toml")
			cmd := exec.Command("git", "-C", root, "show", "HEAD:"+manifestPath)
			out, err := cmd.CombinedOutput()
			if err != nil {
				continue
			}
			raw = out
		} else {
			manifestPath = filepath.Join(root, agentsDirRel, name, "agent.toml")
			var err error
			raw, err = os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
		}
		agent, err := parseAgentManifestFrom(raw)
		if err != nil {
			continue
		}
		agent.Path = filepath.ToSlash(filepath.Join(repo.AgentsPath, name, "agent.toml"))
		if agent.ID == "" {
			agent.ID = name
		}
		if agent.Name == "" {
			agent.Name = agent.ID
		}
		if agent.Runtime == "" {
			agent.Runtime = "codex"
		}
		if agent.RuntimeVersion == "" {
			agent.RuntimeVersion = "latest"
		}
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	return agents
}

func scanMCPServers(repos []Repository) []MCPServer {
	seen := map[string]MCPServer{}
	for _, repo := range repos {
		mcpDir := filepath.Join(repoPath(repo), "mcp")
		entries, err := os.ReadDir(mcpDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var server MCPServer
			raw, err := os.ReadFile(filepath.Join(mcpDir, entry.Name()))
			if err != nil || json.Unmarshal(raw, &server) != nil || server.ID == "" {
				continue
			}
			seen[server.ID] = server
		}
	}
	for _, fallback := range []MCPServer{
		{ID: "github-mcp", Name: "GitHub MCP", Scope: "代码仓库"},
		{ID: "notion-mcp", Name: "Notion MCP", Scope: "知识库"},
		{ID: "filesystem-mcp", Name: "Filesystem MCP", Scope: "文档读取"},
		{ID: "jira-mcp", Name: "Jira MCP", Scope: "项目管理"},
		{ID: "grafana-mcp", Name: "Grafana MCP", Scope: "监控查询"},
	} {
		if _, ok := seen[fallback.ID]; !ok {
			seen[fallback.ID] = fallback
		}
	}
	var servers []MCPServer
	for _, server := range seen {
		servers = append(servers, server)
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].ID < servers[j].ID })
	return servers
}

func parseAgentManifest(path string) (Agent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Agent{}, err
	}
	return parseAgentManifestFrom(raw)
}

func parseAgentManifestFrom(raw []byte) (Agent, error) {
	values := parseSimpleTOML(string(raw))
	return Agent{
		ID:             values["id"].scalar,
		Name:           values["name"].scalar,
		Description:    values["description"].scalar,
		Model:          values["model"].scalar,
		Runtime:        values["runtime"].scalar,
		RuntimeVersion: values["runtime_version"].scalar,
		APIToken:       values["api_token"].scalar,
		Skills:         values["skills"].list,
		MCPs:           values["mcps"].list,
		ExtraInstall:   values["extra_install"].list,
	}, nil
}

type tomlValue struct {
	scalar string
	list   []string
}

func parseSimpleTOML(raw string) map[string]tomlValue {
	result := map[string]tomlValue{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			result[key] = tomlValue{list: parseTOMLStringList(value)}
			continue
		}
		result[key] = tomlValue{scalar: strings.Trim(value, `"'`)}
	}
	return result
}

func parseTOMLStringList(value string) []string {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if value == "" {
		return []string{}
	}
	var items []string
	for _, item := range strings.Split(value, ",") {
		item = strings.Trim(strings.TrimSpace(item), `"'`)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
