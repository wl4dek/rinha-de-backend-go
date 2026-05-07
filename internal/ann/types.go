package ann

type Reference struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

type SearchRequest struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
}

type SearchResult struct {
	Label string  `json:"label"`
	Dist  float32 `json:"dist"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}
