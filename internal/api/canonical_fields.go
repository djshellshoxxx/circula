package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// A2-01: encoding/json accepts case-insensitive field aliases even with
// DisallowUnknownFields. Check the declared wire names before it assigns them.
// Maps and raw JSON keep their documented free-form keys; duplicate keys are
// checked separately for every object, including those free-form values.
func checkCanonicalFields(b []byte, typ reflect.Type) error {
	if typ == nil || bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return nil
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if reflect.PointerTo(typ).Implements(reflect.TypeFor[json.Unmarshaler]()) {
		return nil
	}
	switch typ.Kind() {
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(b, &obj); err != nil {
			return err
		}
		fields := map[string]reflect.Type{}
		seen := map[reflect.Type]bool{}
		var addFields func(reflect.Type)
		addFields = func(t reflect.Type) {
			if seen[t] {
				return
			}
			seen[t] = true
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				name := strings.Split(f.Tag.Get("json"), ",")[0]
				if name == "-" || (!f.Anonymous && !f.IsExported()) {
					continue
				}
				ft := f.Type
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if f.Anonymous && name == "" && ft.Kind() == reflect.Struct {
					addFields(ft)
					continue
				}
				if name == "" {
					name = f.Name
				}
				fields[name] = f.Type
			}
		}
		addFields(typ)
		for k, v := range obj {
			ft, ok := fields[k]
			if !ok {
				return errors.New("unknown field")
			}
			if err := checkCanonicalFields(v, ft); err != nil {
				return err
			}
		}
	case reflect.Map:
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(b, &obj); err != nil {
			return err
		}
		for _, v := range obj {
			if err := checkCanonicalFields(v, typ.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		// []byte uses encoding/json's base64 string representation.
		if typ.Elem().Kind() == reflect.Uint8 {
			return nil
		}
		var list []json.RawMessage
		if err := json.Unmarshal(b, &list); err != nil {
			return err
		}
		for _, v := range list {
			if err := checkCanonicalFields(v, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
