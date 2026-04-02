package sourcetype

import "encoding/json"

type Varlist map[string][]string

func (vl *Varlist) UnmarshalJSON(b []byte) error {
	type kv struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	}
	var arr []kv

	m := Varlist{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, val := range arr {
		m[val.Key] = val.Value
	}
	*vl = m
	return nil
}

func (vl Varlist) MarshalJSON() ([]byte, error) {
	type kv struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	}
	var arr []kv
	for key, val := range vl {
		arr = append(arr, kv{Key: key, Value: val})
	}
	return json.Marshal(arr)
}

func (vl Varlist) Append(key string, values []string) {
	if _, ok := vl[key]; !ok {
		vl[key] = []string{}
	}
	vl[key] = append(vl[key], values...)
}

func (vl Varlist) AppendMap(mv map[string][]string) {
	for key, values := range mv {
		vl.Append(key, values)
	}
}

func (vl Varlist) Unique() *Varlist {
	// todo: optimize it
	unique := func(arr []string) []string {
		occured := map[string]bool{}
		result := []string{}
		for e := range arr {
			// check if already the mapped
			// variable is set to true or not
			if occured[arr[e]] != true {
				occured[arr[e]] = true
				// Append to result slice.
				result = append(result, arr[e])
			}
		}

		return result
	}
	result := Varlist{}
	for key, values := range vl {
		result.Append(key, unique(values))
	}
	return &result
}
