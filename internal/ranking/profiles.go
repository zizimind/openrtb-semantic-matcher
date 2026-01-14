package ranking

import "sync"

// UserProfileStore handles retrieval of user profiles
type UserProfileStore struct {
	profiles map[string]*UserProfile
	mu       sync.RWMutex
}

// NewUserProfileStore creates a new in-memory store with mock data
func NewUserProfileStore() *UserProfileStore {
	store := &UserProfileStore{
		profiles: make(map[string]*UserProfile),
	}
	store.seedMockData()
	return store
}

// GetProfile returns a user profile by ID
func (s *UserProfileStore) GetProfile(userID string) *UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profiles[userID]
}

// seedMockData populates the store with test users
func (s *UserProfileStore) seedMockData() {
	// User 1: Loves Travel
	s.profiles["user_travel"] = &UserProfile{
		UserID:              "user_travel",
		PreferredCategories: []string{"travel", "hotel", "flight"},
		MinBidPrice:         0.0,
		MaxBidPrice:         100.0,
	}

	// User 2: Loves Tech
	s.profiles["user_tech"] = &UserProfile{
		UserID:              "user_tech",
		PreferredCategories: []string{"technology", "software", "computer"},
		MinBidPrice:         0.0,
		MaxBidPrice:         100.0,
	}

	// User 3: High Roller (Prefers expensive items)
	s.profiles["user_luxury"] = &UserProfile{
		UserID:              "user_luxury",
		PreferredCategories: []string{"fashion", "travel"},
		MinBidPrice:         5.0, // Only interested in premium/high-bid ads
		MaxBidPrice:         100.0,
	}
}
