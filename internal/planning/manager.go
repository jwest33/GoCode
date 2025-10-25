package planning

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jake/gocode/internal/memory"
	"github.com/jake/gocode/internal/tools"
)

// PlanManager coordinates hierarchical planning across sessions
// It manages: Goals → Milestones → Tasks
// And syncs with TODO.md for immediate tasks
type PlanManager struct {
	ltm      *memory.LongTermMemory
	todoTool *tools.TodoWriteTool
}

// Plan represents a hierarchical plan structure
type Plan struct {
	Goal       string      `json:"goal"`        // High-level goal
	Milestones []Milestone `json:"milestones"`  // Major checkpoints
	Context    string      `json:"context"`     // Additional context
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Milestone represents a major checkpoint toward the goal
type Milestone struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`      // pending, in_progress, completed
	Tasks       []string `json:"tasks"`       // Task IDs or descriptions
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

const (
	PlanMemoryTag  = "plan"
	GoalMemoryType = memory.TypeDecision
)

func NewPlanManager(ltm *memory.LongTermMemory, todoTool *tools.TodoWriteTool) *PlanManager {
	return &PlanManager{
		ltm:      ltm,
		todoTool: todoTool,
	}
}

// StorePlan stores a plan in long-term memory
func (pm *PlanManager) StorePlan(plan *Plan) error {
	now := time.Now()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now

	// Serialize plan
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	// Store as a decision memory with high importance
	mem := &memory.Memory{
		Type:       GoalMemoryType,
		Summary:    fmt.Sprintf("Project Goal: %s", plan.Goal),
		Content:    string(planJSON),
		Tags:       []string{PlanMemoryTag, "goal", "active"},
		Importance: 0.9, // High importance for plans
	}

	return pm.ltm.Store(mem)
}

// GetActivePlan retrieves the most recent active plan
func (pm *PlanManager) GetActivePlan() (*Plan, error) {
	// Search for active plans
	memories, err := pm.ltm.GetByTags([]string{PlanMemoryTag, "active"}, 1)
	if err != nil {
		return nil, err
	}

	if len(memories) == 0 {
		return nil, fmt.Errorf("no active plan found")
	}

	// Deserialize plan
	var plan Plan
	if err := json.Unmarshal([]byte(memories[0].Content), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// UpdateMilestone updates a milestone's status
func (pm *PlanManager) UpdateMilestone(milestoneID, status string) error {
	plan, err := pm.GetActivePlan()
	if err != nil {
		return err
	}

	// Find and update milestone
	found := false
	for i := range plan.Milestones {
		if plan.Milestones[i].ID == milestoneID {
			plan.Milestones[i].Status = status
			if status == "completed" {
				now := time.Now()
				plan.Milestones[i].CompletedAt = &now
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("milestone %s not found", milestoneID)
	}

	// Store updated plan
	return pm.StorePlan(plan)
}

// GetCurrentContext generates context string about current plan status
// This is injected into the agent's context when relevant
func (pm *PlanManager) GetCurrentContext() string {
	plan, err := pm.GetActivePlan()
	if err != nil {
		return ""
	}

	var ctx strings.Builder
	ctx.WriteString("# Active Project Plan\n\n")
	ctx.WriteString(fmt.Sprintf("**Goal:** %s\n\n", plan.Goal))

	if plan.Context != "" {
		ctx.WriteString(fmt.Sprintf("**Context:** %s\n\n", plan.Context))
	}

	ctx.WriteString("## Milestones\n\n")
	for i, milestone := range plan.Milestones {
		statusIcon := ""
		switch milestone.Status {
		case "completed":
			statusIcon = "✓"
		case "in_progress":
			statusIcon = "→"
		default:
			statusIcon = "○"
		}

		ctx.WriteString(fmt.Sprintf("%d. [%s] **%s**\n", i+1, statusIcon, milestone.Title))
		if milestone.Description != "" {
			ctx.WriteString(fmt.Sprintf("   %s\n", milestone.Description))
		}

		if len(milestone.Tasks) > 0 {
			ctx.WriteString(fmt.Sprintf("   Tasks: %d total\n", len(milestone.Tasks)))
		}

		if milestone.CompletedAt != nil {
			ctx.WriteString(fmt.Sprintf("   Completed: %s\n", milestone.CompletedAt.Format("2006-01-02")))
		}
		ctx.WriteString("\n")
	}

	// Include current TODO status
	todos := pm.todoTool.GetTodos()
	if len(todos) > 0 {
		ctx.WriteString("## Current Tasks (TODO)\n\n")
		pending := 0
		inProgress := 0
		completed := 0

		for _, todo := range todos {
			switch todo.Status {
			case "pending":
				pending++
			case "in_progress":
				inProgress++
			case "completed":
				completed++
			}
		}

		ctx.WriteString(fmt.Sprintf("- %d pending | %d in progress | %d completed\n\n",
			pending, inProgress, completed))
	}

	return ctx.String()
}

// ShouldInjectPlanContext determines if plan context should be injected
// based on the user's query
func (pm *PlanManager) ShouldInjectPlanContext(userQuery string) bool {
	// Keywords that suggest the user wants plan context
	keywords := []string{
		"plan", "goal", "progress", "milestone", "status",
		"what's next", "what next", "continue", "resume",
		"where are we", "what have we done", "what's left",
	}

	lowerQuery := strings.ToLower(userQuery)
	for _, keyword := range keywords {
		if strings.Contains(lowerQuery, keyword) {
			return true
		}
	}

	return false
}

// GenerateMilestoneID generates a unique milestone ID
func GenerateMilestoneID(title string) string {
	// Simple ID generation - can be enhanced
	clean := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	return fmt.Sprintf("milestone-%s", clean)
}
