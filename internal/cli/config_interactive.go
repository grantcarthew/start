package cli

import "fmt"

// allConfigCategories is the ordered set of interactive config categories.
var allConfigCategories = []string{"agents", "roles", "contexts", "tasks"}

// loadNamesForCategory loads the ordered list of names for a config category.
func loadNamesForCategory(category string, local bool) ([]string, error) {
	switch category {
	case "agents":
		_, order, err := loadAgentsForScope(local)
		return order, err
	case "roles":
		_, order, err := loadRolesForScope(local)
		return order, err
	case "contexts":
		_, order, err := loadContextsForScope(local)
		return order, err
	case "tasks":
		_, order, err := loadTasksForScope(local)
		return order, err
	default:
		return nil, fmt.Errorf("unknown category %q", category)
	}
}
