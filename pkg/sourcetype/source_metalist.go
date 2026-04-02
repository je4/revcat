package sourcetype

import "encoding/json"

type Metalist map[string]string

func (ml *Metalist) UnmarshalJSON(b []byte) error {
	type kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var arr []kv

	m := Metalist{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, val := range arr {
		m[val.Key] = val.Value
	}
	*ml = m
	return nil
}

func (ml Metalist) MarshalJSON() ([]byte, error) {
	type kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var arr []kv
	for key, val := range ml {
		arr = append(arr, kv{Key: key, Value: val})
	}
	return json.Marshal(arr)
}
