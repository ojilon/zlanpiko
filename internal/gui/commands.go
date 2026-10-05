package gui

import (
	"fmt"
	"strings"

	"zlanpiko/internal/app"
	"zlanpiko/internal/maintenance"
	"zlanpiko/internal/timeline"
)

// CommandResultDTO tells the frontend what a command line did: navigate to
// a view, refresh the current one, or show a message/error inline.
type CommandResultDTO struct {
	Kind       string `json:"kind"` // navigate|refresh|message|error
	View       string `json:"view,omitempty"`
	Message    string `json:"message,omitempty"`
	Hint       string `json:"hint,omitempty"`
	WeekOffset *int   `json:"week_offset,omitempty"`
	Unit       string `json:"unit,omitempty"`
}

// guiCommandHelp mirrors the TUI help plus GUI-only verbs.
const guiCommandHelp = `Commands:
  /help /dashboard /units /topics [unit] /tasks [unit]
  /timeline [next|prev|today|YYYY-Www] /calendar /files /analytics /settings
  /verify /export /backup /version /clear
  /topics status <unit> <topic> <unread|pending|read>
  /tasks deadline <unit> <task> <YYYY-MM-DD[ HH:MM]>
  /tasks status <unit> <task> <not_started|in_progress|completed|submitted>`

func cmdNavigate(view string) (CommandResultDTO, error) {
	return CommandResultDTO{Kind: "navigate", View: view}, nil
}

func cmdMessage(msg string) (CommandResultDTO, error) {
	return CommandResultDTO{Kind: "message", Message: msg}, nil
}

func cmdErr(msg, hint string) (CommandResultDTO, error) {
	return CommandResultDTO{Kind: "error", Message: msg, Hint: hint}, nil
}

// RunCommand parses one command-bar line and dispatches to GuiApi methods.
// Navigation verbs match internal/tui dispatch exactly; the topics/tasks
// mutation verbs are GUI extensions documented in /help (docs/cs/08).
func (g *GuiApi) RunCommand(raw string) (CommandResultDTO, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return cmdMessage("")
	}
	name := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	args := fields[1:]
	nav := func(view string) (CommandResultDTO, error) {
		return CommandResultDTO{Kind: "navigate", View: view}, nil
	}
	withUnit := func(view string) (CommandResultDTO, error) {
		r := CommandResultDTO{Kind: "navigate", View: view}
		if len(args) > 0 {
			r.Unit = args[0]
			r.Message = "filtered to " + args[0]
		}
		return r, nil
	}
	switch name {
	case "help":
		return cmdMessage(guiCommandHelp)
	case "dashboard":
		return nav("Dashboard")
	case "units":
		if len(args) > 0 && args[0] == "list" {
			return g.cmdUnitsList()
		}
		return nav("Units")
	case "topics":
		if len(args) >= 4 && args[0] == "status" {
			return g.cmdTopicStatus(args[1], args[2], args[3])
		}
		if len(args) > 0 && (args[0] == "list" || !strings.HasPrefix(args[0], "-")) {
			return withUnit("Topics")
		}
		return nav("Topics")
	case "tasks":
		if len(args) >= 4 && args[0] == "deadline" {
			return g.cmdTaskDeadline(args[1], args[2], strings.Join(args[3:], " "))
		}
		if len(args) >= 4 && args[0] == "status" {
			return g.cmdTaskStatus(args[1], args[2], args[3])
		}
		if len(args) > 0 && args[0] == "deadlines" {
			return g.cmdDeadlines()
		}
		if len(args) > 0 && args[0] == "list" {
			return g.cmdTasksList()
		}
		return withUnit("Tasks")
	case "timeline":
		return g.cmdTimeline(args)
	case "calendar":
		return nav("Calendar")
	case "files":
		return nav("Files")
	case "analytics":
		return nav("Analytics")
	case "config", "settings":
		return nav("Settings")
	case "verify":
		rep, err := maintenance.Verify(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger)
		if err != nil {
			return cmdErr(err.Error(), "")
		}
		return cmdMessage(rep.Summary())
	case "export", "backup":
		r, _ := nav("Settings")
		r.Message = "use the Settings view for export and backup"
		return r, nil
	case "version":
		return cmdMessage(app.Info())
	case "quit", "q", "exit":
		return cmdMessage("close the window with ✕ (data is saved continuously)")
	case "search":
		return cmdErr("/search arrives in a later release", "use the view lists for now")
	default:
		return cmdErr("unknown command "+quoteCmd(raw), suggestCmd(fields[0]))
	}
}

func (g *GuiApi) cmdUnitsList() (CommandResultDTO, error) {
	cards, err := g.GetUnitCards()
	if err != nil {
		return CommandResultDTO{}, err
	}
	var b strings.Builder
	for _, c := range cards {
		fmt.Fprintf(&b, "• %s [%s] %s R%d/P%d/U%d %d active\n",
			c.Name, c.Code, c.CoverageText, c.Read, c.Pending, c.Unread, c.ActiveTasks)
	}
	return CommandResultDTO{Kind: "message", View: "Units", Message: strings.TrimSpace(b.String())}, nil
}

func (g *GuiApi) cmdTasksList() (CommandResultDTO, error) {
	d, err := g.GetDashboard()
	if err != nil {
		return CommandResultDTO{}, err
	}
	var b strings.Builder
	for _, t := range d.Upcoming {
		fmt.Fprintf(&b, "• %s (%s) due %s · %s\n", t.Title, t.UnitName, t.DueDisplay, t.Distance)
	}
	for _, t := range d.Overdue {
		fmt.Fprintf(&b, "• %s (%s) OVERDUE %s · %s\n", t.Title, t.UnitName, t.DueDisplay, t.Distance)
	}
	if b.Len() == 0 {
		return cmdMessage("no open tasks")
	}
	return CommandResultDTO{Kind: "message", View: "Tasks", Message: strings.TrimSpace(b.String())}, nil
}

func (g *GuiApi) cmdDeadlines() (CommandResultDTO, error) {
	r, err := g.cmdTasksList()
	if err != nil {
		return CommandResultDTO{}, err
	}
	r.View = "Timeline"
	return r, nil
}

func (g *GuiApi) cmdTopicStatus(unitID, topicID, status string) (CommandResultDTO, error) {
	t, err := g.SetTopicStatus(unitID, topicID, status)
	if err != nil {
		if gerr, ok := err.(*GuiError); ok {
			return cmdErr(gerr.Message, gerr.Hint)
		}
		return CommandResultDTO{}, err
	}
	return CommandResultDTO{Kind: "refresh", Message: fmt.Sprintf("%s is now %s", t.Name, t.Status)}, nil
}

func (g *GuiApi) cmdTaskDeadline(unitID, taskID, due string) (CommandResultDTO, error) {
	d, err := g.MoveDeadline(unitID, taskID, due)
	if err != nil {
		if gerr, ok := err.(*GuiError); ok {
			return cmdErr(gerr.Message, gerr.Hint)
		}
		return CommandResultDTO{}, err
	}
	return CommandResultDTO{Kind: "refresh", Message: fmt.Sprintf("%s due %s", d.Task.Title, d.Task.DueDisplay)}, nil
}

func (g *GuiApi) cmdTaskStatus(unitID, taskID, status string) (CommandResultDTO, error) {
	d, err := g.SetTaskStatus(unitID, taskID, status)
	if err != nil {
		if gerr, ok := err.(*GuiError); ok {
			return cmdErr(gerr.Message, gerr.Hint)
		}
		return CommandResultDTO{}, err
	}
	return CommandResultDTO{Kind: "refresh", Message: fmt.Sprintf("%s is now %s", d.Task.Title, d.Task.Status)}, nil
}

// cmdTimeline resolves next|prev|today|YYYY-Www to a week offset from the
// current week so the frontend never computes ISO weeks (docs/cs/08).
func (g *GuiApi) cmdTimeline(args []string) (CommandResultDTO, error) {
	y, w := timeline.WeekOf(g.now())
	off := 0
	if len(args) > 0 {
		switch args[0] {
		case "", "today":
			off = 0
		case "next":
			off = 1
		case "prev", "previous":
			off = -1
		default:
			ty, tw, err := timeline.ParseWeek(args[0])
			if err != nil {
				return cmdErr(err.Error(), "want next|prev|today|YYYY-Www")
			}
			off = weekOffset(y, w, ty, tw)
		}
	}
	r := CommandResultDTO{Kind: "navigate", View: "Timeline", WeekOffset: &off}
	return r, nil
}

// weekOffset counts whole weeks from (y,w) to (ty,tw) via Monday dates.
func weekOffset(y, w, ty, tw int) int {
	a := timeline.MondayOf(y, w)
	b := timeline.MondayOf(ty, tw)
	d := b.Sub(a).Hours() / 24
	off := int(d / 7)
	if d < 0 && int(d)%7 != 0 {
		off--
	}
	return off
}

func quoteCmd(s string) string {
	return "\"" + s + "\""
}

// suggestCmd ports the TUI typo recovery to view names (docs/cs/08).
func suggestCmd(raw string) string {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "/"))
	fields := strings.Fields(clean)
	if len(fields) == 0 {
		return "(try /help)"
	}
	var hits []string
	for _, title := range []string{"dashboard", "units", "topics", "tasks", "timeline", "calendar", "files", "analytics", "settings", "help", "verify", "version"} {
		if strings.Contains(title, fields[0]) || strings.Contains(fields[0], title) ||
			editDistance(title, fields[0]) <= 2 {
			hits = append(hits, "/"+title)
		}
	}
	if len(hits) == 0 {
		return "(try /help)"
	}
	return "(did you mean " + strings.Join(hits, ", ") + "?)"
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range ar {
		cur := make([]int, len(br)+1)
		cur[0] = i + 1
		for j, rb := range br {
			cost := 0
			if ra != rb {
				cost = 1
			}
			del, ins, sub := prev[j+1]+1, cur[j]+1, prev[j]+cost
			cur[j+1] = del
			if ins < cur[j+1] {
				cur[j+1] = ins
			}
			if sub < cur[j+1] {
				cur[j+1] = sub
			}
		}
		prev = cur
	}
	return prev[len(br)]
}
