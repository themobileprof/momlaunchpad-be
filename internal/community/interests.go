package community

// InterestGroup is a category grouping for onboarding and filters.
type InterestGroup struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Items []Interest `json:"items"`
}

// Interest is a selectable community topic.
type Interest struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

const MaxUserInterests = 5

// AllInterestGroups returns the full interest catalog for onboarding.
func AllInterestGroups() []InterestGroup {
	return []InterestGroup{
		{
			Key: "pregnancy", Label: "Pregnancy",
			Items: []Interest{
				{Key: "first_trimester", Label: "First Trimester"},
				{Key: "second_trimester", Label: "Second Trimester"},
				{Key: "third_trimester", Label: "Third Trimester"},
			},
		},
		{
			Key: "health", Label: "Health",
			Items: []Interest{
				{Key: "pregnancy_health", Label: "Pregnancy Health"},
				{Key: "mental_health", Label: "Mental Health"},
				{Key: "nutrition", Label: "Nutrition"},
				{Key: "fitness", Label: "Fitness"},
			},
		},
		{
			Key: "baby", Label: "Baby",
			Items: []Interest{
				{Key: "newborn_care", Label: "Newborn Care"},
				{Key: "breastfeeding", Label: "Breastfeeding"},
				{Key: "baby_sleep", Label: "Baby Sleep"},
				{Key: "baby_health", Label: "Baby Health"},
			},
		},
		{
			Key: "parenthood", Label: "Parenthood",
			Items: []Interest{
				{Key: "first_time_moms", Label: "First-Time Moms"},
				{Key: "experienced_moms", Label: "Experienced Moms"},
				{Key: "dads_partners", Label: "Dads & Partners"},
				{Key: "single_parents", Label: "Single Parents"},
			},
		},
		{
			Key: "support", Label: "Support",
			Items: []Interest{
				{Key: "ask_midwife", Label: "Ask a Midwife"},
				{Key: "ask_doctor", Label: "Ask a Doctor"},
				{Key: "emotional_support", Label: "Emotional Support"},
			},
		},
		{
			Key: "local", Label: "Local",
			Items: []Interest{
				{Key: "local_recommendations", Label: "Local Recommendations"},
				{Key: "local_services", Label: "Local Services"},
				{Key: "events_meetups", Label: "Events & Meetups"},
			},
		},
		{
			Key: "community", Label: "Community",
			Items: []Interest{
				{Key: "introductions", Label: "Introductions"},
				{Key: "success_stories", Label: "Success Stories"},
			},
		},
	}
}

// ValidInterestKeys returns all allowed interest keys.
func ValidInterestKeys() map[string]bool {
	keys := make(map[string]bool)
	for _, group := range AllInterestGroups() {
		for _, item := range group.Items {
			keys[item.Key] = true
		}
	}
	return keys
}

// IsValidInterest returns true when key is in the catalog.
func IsValidInterest(key string) bool {
	return ValidInterestKeys()[key]
}

// BadgeLabels maps badge types to display names.
var BadgeLabels = map[string]string{
	"midwife":               "Midwife",
	"doctor":                "Doctor",
	"pediatrician":          "Pediatrician",
	"lactation_consultant":  "Lactation Consultant",
	"community_moderator":   "Community Moderator",
}

// ValidBadgeTypes returns allowed badge type keys.
func ValidBadgeTypes() map[string]bool {
	keys := make(map[string]bool)
	for k := range BadgeLabels {
		keys[k] = true
	}
	return keys
}
