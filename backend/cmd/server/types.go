package main

import (
	"database/sql"
	"sync"
	"time"
)

type App struct {
	rootDir string
	dataDir string
	store   *Store
	bus     *AgentBus
}

type AgentBus struct {
	mu       sync.RWMutex
	agents   map[string]BusAgent
	messages []BusMessage
}

type BusAgent struct {
	AgentID  string    `json:"agentId"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Endpoint string    `json:"endpoint"`
	LastSeen time.Time `json:"lastSeen"`
}

type BusMessage struct {
	ID        string    `json:"id"`
	FromAgent string    `json:"fromAgent"`
	ToAgent   string    `json:"toAgent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct {
	mu    sync.RWMutex
	db    *sql.DB
	state State
}

type State struct {
	CurrentUser  CurrentUser              `json:"currentUser"`
	Repositories []Repository             `json:"repositories"`
	AITokens     []AIToken                `json:"aiTokens"`
	AuthTokens   []AuthToken              `json:"authTokens"`
	Users        []User                   `json:"users"`
	Approvals    []Approval               `json:"approvals"`
	Deployments  []Deployment             `json:"deployments"`
	ChatSessions map[string][]ChatSession `json:"chatSessions"`
	ChatMessages map[string][]ChatMessage `json:"chatMessages"`
}

type CurrentUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Repository struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	URL        string   `json:"url"`
	Branch     string   `json:"branch"`
	AgentsPath string   `json:"agentsPath"`
	LocalPath  string   `json:"localPath"`
	Status     string   `json:"status"`
	LastSync   string   `json:"lastSync,omitempty"`
	Commits    []Commit `json:"commits"`
}

type Commit struct {
	Hash        string  `json:"hash"`
	Message     string  `json:"message"`
	CommittedAt string  `json:"committedAt"`
	Agents      []Agent `json:"agents"`
}

type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Description    string   `json:"description"`
	Model          string   `json:"model"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtimeVersion"`
	APIToken       string   `json:"apiToken"`
	Skills         []string `json:"skills"`
	MCPs           []string `json:"mcps"`
	ExtraInstall   []string `json:"extraInstall,omitempty"`
	Status         string   `json:"status,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	UpdatedAt      string   `json:"updatedAt,omitempty"`
}

type AIToken struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Scope    string `json:"scope"`
	Usage    string `json:"usage"`
	Status   string `json:"status"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Model    string `json:"model,omitempty"`
	Secret   string `json:"-"`
}

type AuthToken struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	AccessTarget string `json:"accessTarget"`
	Script       string `json:"script"`
	FunctionName string `json:"functionName"`
	Argument     string `json:"argument"`
	Status       string `json:"status"`
	UpdatedAt    string `json:"updatedAt"`
}

type User struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Active       bool   `json:"active"`
	PasswordHash string `json:"-"` // never exposed via API
}

type Approval struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Applicant string `json:"applicant"`
	Summary   string `json:"summary"`
	Priority  string `json:"priority"`
	Status    string `json:"status"`
	Reviewer  string `json:"reviewer,omitempty"`
}

type MCPServer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type DeployOptions struct {
	Repositories []Repository `json:"repositories"`
	Models       []string     `json:"models"`
	Runtimes     []string     `json:"runtimes"`
	RuntimeTags  []string     `json:"runtimeTags"`
	MCPServers   []MCPServer  `json:"mcpServers"`
	AITokens     []AIToken    `json:"aiTokens"`
	AuthTokens   []AuthToken  `json:"authTokens"`
}

type Deployment struct {
	ID             string    `json:"id"`
	RepositoryID   string    `json:"repositoryId"`
	CommitHash     string    `json:"commitHash"`
	AgentID        string    `json:"agentId"`
	APITokenID     int       `json:"apiTokenId"`
	Model          string    `json:"model"`
	Runtime        string    `json:"runtime"`
	RuntimeVersion string    `json:"runtimeVersion"`
	Skills         []string  `json:"skills"`
	MCPs           []string  `json:"mcps"`
	AuthTokens     []int     `json:"authTokens"`
	ImageTag       string    `json:"imageTag"`
	ContainerName  string    `json:"containerName"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	BuildContext   string    `json:"buildContext"`
	SidecarURL     string    `json:"sidecarUrl"`
	HostPort       int       `json:"hostPort"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DeployRequest struct {
	RepositoryID   string   `json:"repositoryId"`
	CommitHash     string   `json:"commitHash"`
	AgentID        string   `json:"agentId"`
	APITokenID     int      `json:"apiTokenId"`
	Model          string   `json:"model"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtimeVersion"`
	AuthTokens     []int    `json:"authTokens"`
	Skills         []string `json:"skills"`
	MCPs           []string `json:"mcps"`
	ExtraInstall   []string `json:"extraInstall"`
}

type TokenResolveRequest struct {
	TokenID int    `json:"tokenId"`
	Param   string `json:"param"`
}

type ChatSession struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agentId"`
	Title     string    `json:"title"`
	Preview   string    `json:"preview"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	AgentID   string    `json:"agentId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}
