package config

type ConfigDataObjectEmbed struct {
	// ConfigDataObject `json:",inline" gorm:"config_data_object;type:jsonb;not null"`
	ConfigDataObject ConfigDataObject `json:",inline" gorm:"embedded"`
}
