package config

import (
	"encoding/json"
	"io/ioutil"
	"sort"
)

type Config struct {
	TestRoot           bool                `json:"testRoot"`
	Routes             []*Route            `json:"routes"`
	Roles              []*Role             `json:"roles"`
	NavigationFallback *NavigationFallback `json:"navigationFallback"`
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
	sort.Strings(roles)
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
	if c.NavigationFallback != nil {
		err = c.NavigationFallback.Compile()
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

var Unconfigured = &Config{TestRoot: true}
