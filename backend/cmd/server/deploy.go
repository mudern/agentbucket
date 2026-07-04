package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (app *App) createDeployment(req DeployRequest) (Deployment, error) {
	state := app.store.snapshot()
	repos := app.scanRepositories(state.Repositories)
	repo, commit, agent, err := findDeploymentTarget(repos, req)
	if err != nil {
		return Deployment{}, err
	}
	if req.Runtime == "" {
		req.Runtime = agent.Runtime
	}
	if req.RuntimeVersion == "" {
		req.RuntimeVersion = agent.RuntimeVersion
	}
	if req.Model == "" {
		req.Model = agent.Model
	}
	if len(req.Skills) == 0 {
		req.Skills = agent.Skills
	}
	if len(req.MCPs) == 0 {
		req.MCPs = agent.MCPs
	}
	if len(req.ExtraInstall) == 0 {
		req.ExtraInstall = agent.ExtraInstall
	}
	if !isSupportedRuntime(req.Runtime) {
		return Deployment{}, fmt.Errorf("unsupported runtime %q", req.Runtime)
	}

	id := fmt.Sprintf("dep-%s-%d", slug(agent.ID), time.Now().Unix())
	contextDir := filepath.Join(app.dataDir, "deployments", id, "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return Deployment{}, err
	}
	if err := app.writeBuildContext(contextDir, repo, commit, agent, req); err != nil {
		return Deployment{}, err
	}

	deployment := Deployment{
		ID:             id,
		RepositoryID:   repo.ID,
		CommitHash:     commit.Hash,
		AgentID:        agent.ID,
		APITokenID:     req.APITokenID,
		Model:          req.Model,
		Runtime:        req.Runtime,
		RuntimeVersion: req.RuntimeVersion,
		Skills:         req.Skills,
		MCPs:           req.MCPs,
		AuthTokens:     req.AuthTokens,
		ImageTag:       "agentbucket/" + slug(agent.ID) + ":" + commit.Hash,
		ContainerName:  "agentbucket-" + slug(agent.ID),
		Status:         "packaged",
		Message:        "Docker build context generated",
		BuildContext:   contextDir,
		HostPort:       hostPortFor(agent.ID),
		CreatedAt:      time.Now(),
	}
	deployment.SidecarURL = fmt.Sprintf("http://%s:%d", sidecarHost(), deployment.HostPort)

	if _, err := exec.LookPath("docker"); err != nil {
		deployment.Message = "Docker CLI not found; generated build context only"
		return deployment, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout())
	defer cancel()
	build := exec.CommandContext(ctx, "docker", "build", "-t", deployment.ImageTag, contextDir)
	buildOut, err := build.CombinedOutput()
	buildLog := string(buildOut)
	if err != nil {
		deployment.Status = "build_failed"
		if ctx.Err() == context.DeadlineExceeded {
			deployment.Message = "docker build timed out: " + buildLog
		} else {
			deployment.Message = buildLog
		}
		return deployment, nil
	}
	_ = exec.Command("docker", "rm", "-f", deployment.ContainerName).Run()
	run := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", deployment.ContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:8088", deployment.HostPort),
		"--add-host", "host.docker.internal:host-gateway",
		"-e", fmt.Sprintf("AGENTBUCKET_URL=http://host.docker.internal:%d", mustPort()),
		deployment.ImageTag,
	)
	runOut, err := run.CombinedOutput()
	if err != nil {
		deployment.Status = "run_failed"
		deployment.Message = buildLog + "\n---\n" + string(runOut)
		return deployment, nil
	}
	deployment.Status = "running"
	// Combine build log with container ID for complete traceability
	deployment.Message = buildLog + "\n---\ncontainer: " + strings.TrimSpace(string(runOut))
	return deployment, nil
}

func findDeploymentTarget(repos []Repository, req DeployRequest) (Repository, Commit, Agent, error) {
	for _, repo := range repos {
		if repo.ID != req.RepositoryID {
			continue
		}
		for _, commit := range repo.Commits {
			if req.CommitHash != "" && commit.Hash != req.CommitHash {
				continue
			}
			for _, agent := range commit.Agents {
				if agent.ID == req.AgentID {
					return repo, commit, agent, nil
				}
			}
		}
	}
	return Repository{}, Commit{}, Agent{}, fmt.Errorf("deployment target not found")
}

func (app *App) writeBuildContext(contextDir string, repo Repository, commit Commit, agent Agent, req DeployRequest) error {
	repoRoot := repoPath(repo)
	agentSource := filepath.Join(repoRoot, filepath.FromSlash(agent.Path))
	agentDir := filepath.Dir(agentSource)
	if err := copyDir(agentDir, filepath.Join(contextDir, "agent")); err != nil {
		return err
	}
	if err := copySelectedSkills(filepath.Join(repoRoot, "skills"), filepath.Join(contextDir, "skills"), req.Skills); err != nil {
		return err
	}
	if err := copyOptionalDir(filepath.Join(repoRoot, "mcp"), filepath.Join(contextDir, "mcp")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(contextDir, "sidecar"), 0o755); err != nil {
		return err
	}
	config := map[string]any{
		"repositoryId":   repo.ID,
		"commitHash":     commit.Hash,
		"agentId":        agent.ID,
		"model":          req.Model,
		"runtime":        req.Runtime,
		"runtimeVersion": req.RuntimeVersion,
		"apiTokenId":     req.APITokenID,
		"skills":         req.Skills,
		"mcps":           req.MCPs,
		"authTokens":     req.AuthTokens,
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "agentbucket.config.json"), raw, 0o644); err != nil {
		return err
	}
	sidecarSource, err := os.ReadFile(filepath.Join(app.rootDir, "cmd", "sidecar", "main.go"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "sidecar", "main.go"), sidecarSource, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "Dockerfile"), []byte(dockerfileFor(req.Runtime, req.RuntimeVersion, req.ExtraInstall)), 0o644); err != nil {
		return err
	}
	return writeTarball(contextDir)
}

func dockerfileFor(runtime string, version string, extraInstall []string) string {
	if version == "" {
		version = "latest"
	}
	runtimeLine := fmt.Sprintf("ENV AGENTBUCKET_RUNTIME=%s\nENV AGENTBUCKET_RUNTIME_VERSION=%s\n", runtime, version)
	installLine := runtimeInstallLine(runtime, version)
	extraLine := ""
	for _, cmd := range extraInstall {
		extraLine += "RUN " + cmd + "\n"
	}
	return `FROM golang:1.22-alpine AS sidecar-build
WORKDIR /src
COPY sidecar/main.go .
RUN go build -o /out/agentbucket-sidecar main.go

FROM node:20-alpine
RUN apk add --no-cache ca-certificates bash curl git
` + extraLine + installLine + `
WORKDIR /app
` + runtimeLine + `COPY --from=sidecar-build /out/agentbucket-sidecar /usr/local/bin/agentbucket-sidecar
COPY agentbucket.config.json /app/agentbucket.config.json
COPY agent /app/agent
COPY skills /app/skills
COPY mcp /app/mcp
EXPOSE 8088
ENTRYPOINT ["/usr/local/bin/agentbucket-sidecar"]
`
}

func sidecarHost() string {
	if h := os.Getenv("AGENTBUCKET_SIDECAR_HOST"); h != "" {
		return h
	}
	return "127.0.0.1"
}

func runtimeInstallLine(runtime string, version string) string {
	switch runtime {
	case "codex":
		return fmt.Sprintf("RUN npm install -g @openai/codex@%s", version)
	case "claudecode":
		return fmt.Sprintf("RUN npm install -g @anthropic-ai/claude-code@%s", version)
	case "opencode":
		return fmt.Sprintf("RUN npm install -g opencode-ai@%s", version)
	default:
		return "RUN true"
	}
}
