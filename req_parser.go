package apitool

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func ParserDataRequest(req *http.Request, data interface{}) error {
	kindOfJ := reflect.ValueOf(data).Kind()
	if kindOfJ != reflect.Ptr {
		return errors.New("data is not pointer")
	}
	switch req.Header.Get("Content-Type") {
	case "application/json":
		err := json.NewDecoder(req.Body).Decode(data)
		if err != nil {
			return err
		}
	case "application/x-www-form-urlencoded":
		rt := reflect.TypeOf(data).Elem()
		keys := []string{}
		for i := 0; i < rt.NumField(); i++ {
			if key, ok := rt.Field(i).Tag.Lookup("json"); ok {
				keys = append(keys, key)
			} else {
				keys = append(keys, strings.ToLower(rt.Field(i).Name))
			}
		}
		vars, err := GetPostValue(req, false, keys)
		if err != nil {
			return err
		}
		decoderConf := &mapstructure.DecoderConfig{
			TagName:  "json",
			Metadata: nil,
			Result:   data}
		decoder, err := mapstructure.NewDecoder(decoderConf)
		if err != nil {
			return err
		}
		err = decoder.Decode(vars)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetPostValue(req *http.Request, defaultEmpty bool, keys []string) (map[string]interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	for _, key := range keys {
		if vs := req.PostForm[key]; len(vs) > 0 {
			result[key] = vs[0]
		} else if defaultEmpty {
			result[key] = ""
		}
	}
	return result, nil
}

func GetHost(req *http.Request) string {
	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}
	return host
}
