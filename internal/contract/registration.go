package contract

import "encoding/json"

type Registration struct {
	Service     string          `json:"-" yaml:"service"`
	Environment string          `json:"environment" yaml:"environment"`
	Instance    InstancePatch   `json:"instance" yaml:"instance"`
	Endpoints   []EndpointPatch `json:"endpoints" yaml:"endpoints"`
	Mode        string          `json:"mode,omitempty" yaml:"mode,omitempty"`
	Replace     bool            `json:"replaceEndpoints,omitempty" yaml:"replaceEndpoints,omitempty"`
}
type InstancePatch struct {
	Name        string            `json:"name" yaml:"name"`
	Address     *string           `json:"address,omitempty" yaml:"address,omitempty"`
	Description *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
type EndpointPatch struct {
	Name     string            `json:"name" yaml:"name"`
	Kind     *string           `json:"kind,omitempty" yaml:"kind,omitempty"`
	Port     *int32            `json:"port,omitempty" yaml:"port,omitempty"`
	Path     *string           `json:"path,omitempty" yaml:"path,omitempty"`
	Primary  *bool             `json:"primary,omitempty" yaml:"primary,omitempty"`
	Enabled  *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Tags     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func Pointer[T any](value T) *T { return &value }

func (p InstancePatch) MarshalJSON() ([]byte, error) {
	type plain InstancePatch
	return marshalMaps(plain(p), p.Tags, p.Metadata)
}

func (p EndpointPatch) MarshalJSON() ([]byte, error) {
	type plain EndpointPatch
	return marshalMaps(plain(p), p.Tags, p.Metadata)
}

// An explicit empty map clears values; an omitted map preserves them.
func marshalMaps(value any, tags, metadata map[string]string) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if tags != nil {
		fields["tags"], err = json.Marshal(tags)
		if err != nil {
			return nil, err
		}
	}
	if metadata != nil {
		fields["metadata"], err = json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(fields)
}
