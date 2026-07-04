package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	log.Printf("[DEPLOY] TARGET: repo=%s commit=%s agent=%s agent.model=%q agent.runtime=%q agent.apiToken=%q",
		repo.ID, commit.Hash[:8], agent.ID, agent.Model, agent.Runtime, agent.APIToken)
	if req.Runtime == "" {
		req.Runtime = agent.Runtime
	}
	if req.RuntimeVersion == "" {
		req.RuntimeVersion = agent.RuntimeVersion
	}
	if req.Model == "" {
		req.Model = agent.Model
	}
	// Always include the built-in agentbucket-api skill
	hasAPI := false
	for _, s := range req.Skills {
		if s == "agentbucket-api" {
			hasAPI = true
			break
		}
	}
	if !hasAPI {
		req.Skills = append([]string{"agentbucket-api"}, req.Skills...)
	}
	if len(req.ExtraInstall) == 0 {
		req.ExtraInstall = agent.ExtraInstall
	}
	if !isSupportedRuntime(req.Runtime) {
		return Deployment{}, fmt.Errorf("unsupported runtime %q", req.Runtime)
	}

	log.Printf("[DEPLOY] FINAL: model=%q runtime=%q runtimeVersion=%q skills=%v mcps=%v apiTokenId=%d",
		req.Model, req.Runtime, req.RuntimeVersion, req.Skills, req.MCPs, req.APITokenID)

	id := fmt.Sprintf("dep-%s-%d", slug(agent.ID), time.Now().Unix())
	contextDir := filepath.Join(app.dataDir, "deployments", id, "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return Deployment{}, err
	}

	// Build context is fast — do it synchronously so errors surface immediately
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
		Status:         "building_context",
		Message:        "构建上下文已生成，等待 Docker 构建...",
		BuildContext:   contextDir,
		HostPort:       hostPortFor(agent.ID),
		CreatedAt:      time.Now(),
	}
	deployment.SidecarURL = fmt.Sprintf("http://%s:%d", sidecarHost(), deployment.HostPort)

	if _, err := exec.LookPath("docker"); err != nil {
		deployment.Status = "build_failed"
		deployment.Message = "Docker CLI 未找到，仅生成了构建上下文"
		return deployment, nil
	}

	// Launch async build + run — returns immediately with "building_context" status
	go app.runDeployment(deployment)
	return deployment, nil
}

// runDeployment performs the slow Docker build + run in background.
// It updates the deployment status in the store at each step.
func (app *App) runDeployment(d Deployment) {
	// Step 2: build image
	_ = app.store.update(func(s *State) error {
		for i := range s.Deployments {
			if s.Deployments[i].ID == d.ID {
				s.Deployments[i].Status = "building_image"
				s.Deployments[i].Message = "正在构建 Docker 镜像..."
			}
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout())
	defer cancel()
	build := exec.CommandContext(ctx, "docker", "build", "-t", d.ImageTag, d.BuildContext)
	buildOut, err := build.CombinedOutput()
	buildLog := string(buildOut)
	if err != nil {
		msg := buildLog
		if ctx.Err() == context.DeadlineExceeded {
			msg = "Docker 构建超时: " + buildLog
		}
		_ = app.store.update(func(s *State) error {
			for i := range s.Deployments {
				if s.Deployments[i].ID == d.ID {
					s.Deployments[i].Status = "build_failed"
					s.Deployments[i].Message = msg
				}
			}
			return nil
		})
		log.Printf("[DEPLOY] BUILD FAILED: %s — %s", d.ID, msg[:min(200, len(msg))])
		return
	}

	// Step 3: start container
	_ = app.store.update(func(s *State) error {
		for i := range s.Deployments {
			if s.Deployments[i].ID == d.ID {
				s.Deployments[i].Status = "starting_container"
				s.Deployments[i].Message = "镜像构建完成，正在启动容器..."
			}
		}
		return nil
	})

	_ = exec.Command("docker", "rm", "-f", d.ContainerName).Run()
	run := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", d.ContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:8088", d.HostPort),
		"--add-host", "host.docker.internal:host-gateway",
		"-e", fmt.Sprintf("AGENTBUCKET_URL=http://host.docker.internal:%d", mustPort()),
		d.ImageTag,
	)
	runOut, err := run.CombinedOutput()
	if err != nil {
		msg := buildLog + "\n---\n容器启动失败: " + string(runOut)
		_ = app.store.update(func(s *State) error {
			for i := range s.Deployments {
				if s.Deployments[i].ID == d.ID {
					s.Deployments[i].Status = "run_failed"
					s.Deployments[i].Message = msg
				}
			}
			return nil
		})
		log.Printf("[DEPLOY] RUN FAILED: %s — %s", d.ID, msg[:min(200, len(msg))])
		return
	}

	// Step 4: running
	containerID := strings.TrimSpace(string(runOut))
	_ = app.store.update(func(s *State) error {
		for i := range s.Deployments {
			if s.Deployments[i].ID == d.ID {
				s.Deployments[i].Status = "running"
				s.Deployments[i].Message = buildLog + "\n---\ncontainer: " + containerID
			}
		}
		return nil
	})
	log.Printf("[DEPLOY] RUNNING: %s — %s", d.ID, containerID)
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
	// Copy skills from the repo
	repoSkills := make([]string, 0, len(req.Skills))
	builtinSkills := make([]string, 0)
	for _, s := range req.Skills {
		if s == "agentbucket-api" {
			builtinSkills = append(builtinSkills, s)
		} else {
			repoSkills = append(repoSkills, s)
		}
	}
	if err := copySelectedSkills(filepath.Join(repoRoot, "skills"), filepath.Join(contextDir, "skills"), repoSkills); err != nil {
		return err
	}
	// Copy built-in agentbucket-api skill from AgentBucket's own source
	for _, s := range builtinSkills {
		src := filepath.Join(app.rootDir, "examples", "agent-repo", "skills", s)
		if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err == nil {
			if err := copyDir(src, filepath.Join(contextDir, "skills", s)); err != nil {
				return fmt.Errorf("builtin skill %s: %w", s, err)
			}
		}
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
	log.Printf("[DEPLOY] CONFIG written: %s", string(raw))
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
