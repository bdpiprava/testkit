package search

// IndexSetting represents the index settings
type IndexSetting struct {
	CreationDate     string  `json:"creation_date"`
	NumberOfShards   string  `json:"number_of_shards"`
	NumberOfReplicas string  `json:"number_of_replicas"`
	UUID             string  `json:"uuid"`
	Blocks           *Blocks `json:"blocks"`
	ProvidedName     string  `json:"provided_name"`
}

// Blocks represents the blocks on index
type Blocks struct {
	Write    string `json:"write"`
	Read     string `json:"read"`
	Metadata string `json:"metadata"`
	ReadOnly string `json:"read_only"`
}

// SettingsRoot is wrapper for one index settings
type SettingsRoot struct {
	Index IndexSetting `json:"index"`
}

// GetSettingsResponse is the response for get settings
type GetSettingsResponse map[string]SettingsWrapper

// SettingsWrapper is the wrapper for settings
type SettingsWrapper struct {
	Settings SettingsRoot `json:"settings"`
}
