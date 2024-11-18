package testkit

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

type settingsRoot struct {
	Index IndexSetting `json:"index"`
}
type getSettingsResponse map[string]settingsWrapper
type settingsWrapper struct {
	Settings settingsRoot `json:"settings"`
}
