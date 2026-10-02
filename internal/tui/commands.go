package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
	"zlanpiko/internal/tui/screens"
)

const tuiCommandHelp = `Commands:
  /help /dashboard /units /topics [unit] /tasks [unit] /files /settings
  /version /quit
  /timeline /analytics /search /export /backup  (later phases)
Keys work too: 1-6 switch screens, see ? for more.`

// dispatch runs a command-input line: history, screen switching, filters,
// notices and quit. Unknown commands get a closest-match suggestion.
func (m *Model) dispatch(raw string) tea.Cmd {
	line := strings.TrimSpace(raw)
	if line != "" {
		m.history = append(m.history, line)
	}
	if line == "" {
		return nil
	}
	name := line
	args := ""
	if i := strings.IndexAny(line, " \t"); i >= 0 {
		name, args = line[:i], strings.TrimSpace(line[i+1:])
	}
	name = strings.TrimPrefix(name, "/")
	switch name {
	case "help":
		m.help = true
		m.status = ""
		return nil
	case "dashboard":
		m.switchTo(screens.Dashboard)
		return nil
	case "units":
		m.switchTo(screens.Units)
		return nil
	case "topics":
		if args != "" {
			m.shared.TopicsUnit = args
		}
		m.switchTo(screens.Topics)
		return nil
	case "tasks":
		if args != "" {
			m.shared.TasksUnit = args
			m.shared.TasksStatus = ""
		}
		m.switchTo(screens.Tasks)
		return nil
	case "files":
		m.switchTo(screens.Files)
		return nil
	case "settings":
		m.switchTo(screens.Settings)
		return nil
	case "version":
		m.status = app.Info()
		return nil
	case "quit", "q", "exit":
		return tea.Quit
	case "timeline", "analytics":
		m.status = "/" + name + " arrives in Phase 6"
		return nil
	case "search", "export", "backup":
		m.status = "/" + name + " arrives in Phase 8"
		return nil
	default:
		m.status = "unknown command " + quote(line) + suggest(line)
		return nil
	}
}

// suggest finds screen names close to the input, for typo recovery.
func suggest(line string) string {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(line), "/"))
	fields := strings.Fields(clean)
	if len(fields) == 0 {
		return ""
	}
	var hits []string
	for _, id := range screens.Order {
		title := strings.ToLower(screens.Titles[id])
		if strings.Contains(title, fields[0]) || strings.Contains(fields[0], title) ||
			editDistance(title, fields[0]) <= 2 {
			hits = append(hits, "/"+strings.ToLower(screens.Titles[id]))
		}
	}
	if len(hits) == 0 {
		return " (try /help)"
	}
	return " (did you mean " + strings.Join(hits, ", ") + "?)"
}

// editDistance is the Levenshtein distance over runes.
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

func quote(s string) string {
	return "\"" + s + "\""
}
