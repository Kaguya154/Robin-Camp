package boxoffice

// BoxOffice mirrors the upstream Box Office API payload.
type BoxOffice struct {
	Title       string  `json:"title"`
	Distributor string  `json:"distributor"`
	ReleaseDate string  `json:"releaseDate"`
	Budget      *int64  `json:"budget,omitempty"`
	Revenue     Revenue `json:"revenue"`
	MpaRating   string  `json:"mpaRating"`
	Currency    string  `json:"currency"`
	Source      string  `json:"source"`
	LastUpdated string  `json:"lastUpdated"`
}

// Revenue breaks down gross revenue values returned by the upstream API.
type Revenue struct {
	Worldwide         *int64 `json:"worldwide,omitempty"`
	OpeningWeekendUSA *int64 `json:"openingWeekendUSA,omitempty"`
}

type Error struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
