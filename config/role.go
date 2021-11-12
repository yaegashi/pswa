package config

type Role struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}
