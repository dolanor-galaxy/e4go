package keys

import (
	"encoding/json"
	"fmt"
)

type keyType int

const (
	symKeyMaterialType keyType = iota
	pubKeyMaterialType
)

type jsonKey struct {
	KeyType keyType     `json:"keyType"`
	KeyData interface{} `json:"keyData"`
}

// FromRawJSON allows to unmarshal a json encoded client key from a json RawMessage
func FromRawJSON(raw json.RawMessage) (KeyMaterial, error) {
	m := make(map[string]json.RawMessage)
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return nil, err
	}

	if _, ok := m["keyType"]; !ok {
		return nil, fmt.Errorf("invalid json raw message, expected \"keyType\"")
	}
	if _, ok := m["keyData"]; !ok {
		return nil, fmt.Errorf("invalid json raw message, expected \"keyData\"")
	}

	var t keyType
	if err := json.Unmarshal(m["keyType"], &t); err != nil {
		return nil, err
	}

	var clientKey KeyMaterial
	switch t {
	case symKeyMaterialType:
		clientKey = &symKeyMaterial{}
	case pubKeyMaterialType:
		clientKey = &pubKeyMaterial{}
	default:
		return nil, fmt.Errorf("unsupported json key type: %v", t)
	}

	if err := json.Unmarshal(m["keyData"], clientKey); err != nil {
		return nil, err
	}

	return clientKey, nil

}
