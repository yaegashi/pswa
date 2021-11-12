package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Routes []*Route `json:"routes"`
	Roles  []*Role  `json:"roles"`
}

func (c *Config) MemberRoles(members []string) []string {
	memberMap := map[string]struct{}{}
	for _, m := range members {
		memberMap[m] = struct{}{}
	}
	roles := []string{"authenticated"}
	for _, r := range c.Roles {
		for _, m := range r.Members {
			if _, ok := memberMap[m]; ok {
				roles = append(roles, r.Role)
				break
			}
		}
	}
	return roles
}

func New(configPath string) (*Config, error) {
	c := &Config{}
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	for _, r := range c.Routes {
		err = r.Compile()
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}
