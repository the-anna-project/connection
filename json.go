package connection

import (
	"encoding/json"
)

func (c *connection) MarshalJSON() ([]byte, error) {
	type ConnectionClone connection

	b, err := json.Marshal(&struct {
		*ConnectionClone
	}{
		ConnectionClone: (*ConnectionClone)(c),
	})
	if err != nil {
		return nil, maskAny(err)
	}

	return b, nil
}

func (c *connection) UnmarshalJSON(b []byte) error {
	type ConnectionClone connection

	aux := &struct {
		*ConnectionClone
	}{
		ConnectionClone: (*ConnectionClone)(c),
	}
	err := json.Unmarshal(b, &aux)
	if err != nil {
		return maskAny(err)
	}

	return nil
}
