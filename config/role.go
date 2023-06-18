package config

type Role struct {
	Role    string   `json:"role,omitempty"`
	Members []string `json:"members,omitempty"`
}
