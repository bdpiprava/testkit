package search

// Index represents the index on elasticsearch
type Index struct {
	Name         string `json:"index"`
	Health       string `json:"health"`
	Status       string `json:"status"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

// Indices represents the list of index
type Indices []Index

// QueryResponse represents the response from elasticsearch or opensearch
type QueryResponse struct {
	Hits Hits `json:"hits"`
}

// Hits represents the hits from the response
type Hits struct {
	Hits []Hit `json:"hits"`
}

// Hit represents the hit from the response
type Hit struct {
	Index  string         `json:"_index"`
	ID     string         `json:"_id"`
	Score  float64        `json:"_score"`
	Source map[string]any `json:"_source"`
}
