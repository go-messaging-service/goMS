package config

type Config struct {
	ServerConfig ServerConfig
	TopicConfig  TopicConfig
}

type TopicConfig struct {
	Topics []string `json:"topics"`
}

type ServerConfig struct {
	TopicLocation string      `json:"topic-config"`
	Connectors    []Connector `json:"connectors"`
	DebugLogging  bool        `json:"debug-logging"`
}

type Connector struct {
	Protocol string
	Ip       string
	Port     int
}
