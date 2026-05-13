package domain

import (
	"strings"
	"time"
)

type ClientMetadata struct {
	ClientID   string                 `json:"client_id"`
	OS         string                 `json:"os"`
	Labels     map[string]string      `json:"labels"`
	InnerState map[string]interface{} `json:"inner_state"`
	Active     bool                   `json:"active"`
	LastSeenAt time.Time              `json:"last_seen_at"`
}

func (c *ClientMetadata) GetField(path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, ErrUnknownField
	}
	switch parts[0] {
	case "client_id", "id":
		return c.ClientID, nil
	case "os":
		return c.OS, nil
	case "active":
		return c.Active, nil
	case "labels":
		if len(parts) < 2 {
			return c.Labels, nil
		}
		v, ok := c.Labels[parts[1]]
		if !ok {
			return nil, ErrUnknownField
		}
		return v, nil
	case "state", "inner_state":
		if len(parts) < 2 {
			return c.InnerState, nil
		}
		return navigateMap(c.InnerState, parts[1:])
	}
	return nil, ErrUnknownField
}

func navigateMap(m map[string]interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return m, nil
	}
	v, ok := m[parts[0]]
	if !ok {
		return nil, ErrUnknownField
	}
	if len(parts) == 1 {
		return v, nil
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return nil, ErrUnknownField
	}
	return navigateMap(sub, parts[1:])
}
