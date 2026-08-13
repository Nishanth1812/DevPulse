package collector

import (
	"sort"
	"strings"
	"time"

	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/utils"
)

func ParseGoals() (models.GoalsData, error) {
	path := utils.GoalsPath()

	content, err := utils.ReadFile(path)
	if err != nil {
		return models.GoalsData{}, err
	}

	var goals models.GoalsData

	sections := splitSections(content)

	goals.Now = sections["Now"]
	goals.Next = sections["Next"]
	goals.Someday = sections["Someday"]
	goals.Deadlines = parseDeadlines(sections["Deadlines"])

	return goals, nil
}

func splitSections(content string) map[string]string {
	result := map[string]string{
		"Now":       "",
		"Next":      "",
		"Deadlines": "",
		"Someday":   "",
	}

	var current string

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Only real markdown headings (lines starting with #) switch sections;
		// a bare "Now" inside body text must not silently change the section.
		if !strings.HasPrefix(trimmed, "#") {
			if current != "" {
				result[current] += line + "\n"
			}
			continue
		}

		heading := strings.TrimPrefix(trimmed, "### ")
		heading = strings.TrimPrefix(heading, "## ")
		heading = strings.TrimPrefix(heading, "# ")

		switch heading {
		case "Now":
			current = "Now"
		case "Next":
			current = "Next"
		case "Deadlines":
			current = "Deadlines"
		case "Someday":
			current = "Someday"
		default:
			// Unknown heading: treat the heading itself as content of the
			// current section rather than dropping it.
			if current != "" {
				result[current] += line + "\n"
			}
		}
	}

	return result
}

func parseDeadlines(content string) []models.Deadline {
	lines := strings.Split(content, "\n")

	var deadlines []models.Deadline

	for _, line := range lines {
		// Accept both the em-dash separator from the template and a plain
		// hyphen or colon for hand-written files.
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "—", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(line, " - ", 2)
		}
		if len(parts) != 2 {
			parts = strings.SplitN(line, ": ", 2)
		}
		if len(parts) != 2 {
			continue
		}

		dateStr := strings.TrimSpace(parts[0])
		desc := strings.TrimSpace(parts[1])

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		deadlines = append(deadlines, models.Deadline{
			Date:        date,
			Description: desc,
			DaysUntil:   utils.DaysUntil(date),
		})
	}

	sort.Slice(deadlines, func(i, j int) bool {
		return deadlines[i].Date.Before(deadlines[j].Date)
	})

	return deadlines
}
