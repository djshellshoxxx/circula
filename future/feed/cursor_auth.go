package feed

import (
 "crypto/hmac"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "errors"
 "strings"
 "time"
)

type cursorPayload struct {
 Version int `json:"v"`
 Network string `json:"n"`
 Kind string `json:"k"`
 ID string `json:"i"`
 Created int64 `json:"t"`
}

func validCursor(key []byte, network string, c Cursor) bool {
 return len(key)>=32 && network!="" && len(network)<=200 && c.ID!="" && len(c.ID)<=256 && (c.Kind=="listing"||c.Kind=="announcement") && !c.Created.IsZero() && c.Created.UnixNano()>0
}

// SealCursor signs a network-bound, versioned pagination position. The signing
// key must be stable across restarts and supplied by a secret store.
func SealCursor(key []byte, network string, c Cursor)(string,error){
 if !validCursor(key,network,c){return "",errors.New("invalid cursor inputs")}
 p,_:=json.Marshal(cursorPayload{1,network,c.Kind,c.ID,c.Created.UnixNano()})
 h:=hmac.New(sha256.New,key);h.Write(p)
 return base64.RawURLEncoding.EncodeToString(p)+"."+base64.RawURLEncoding.EncodeToString(h.Sum(nil)),nil
}

// OpenCursor verifies integrity and tenant binding before exposing a cursor.
func OpenCursor(key []byte, network, token string)(Cursor,error){
 invalid:=errors.New("invalid cursor")
 if len(key)<32 || network=="" || len(token)>2048 || strings.Count(token,".")!=1{return Cursor{},invalid}
 a,b,ok:=strings.Cut(token,".");if !ok{return Cursor{},invalid}
 p,err:=base64.RawURLEncoding.DecodeString(a);if err!=nil{return Cursor{},invalid}
 sig,err:=base64.RawURLEncoding.DecodeString(b);if err!=nil||len(sig)!=sha256.Size{return Cursor{},invalid}
 h:=hmac.New(sha256.New,key);h.Write(p)
 if !hmac.Equal(sig,h.Sum(nil)){return Cursor{},invalid}
 var v cursorPayload
 if err=json.Unmarshal(p,&v);err!=nil||v.Version!=1||v.Network!=network{return Cursor{},invalid}
 c:=Cursor{Created:time.Unix(0,v.Created).UTC(),Kind:v.Kind,ID:v.ID}
 if !validCursor(key,network,c) || c.Created.UnixNano()!=v.Created{return Cursor{},invalid}
 return c,nil
}
