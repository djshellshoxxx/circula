package feed

import("testing";"time")
func TestCursorSealVerifyAndTenantBinding(t *testing.T){
 key:=[]byte("01234567890123456789012345678901")
 c:=Cursor{Created:time.Unix(1234,34).UTC(),Kind:"listing",ID:"item-1"}
 token,err:=SealCursor(key,"network-A",c);if err!=nil {t.Fatal(err)}
 got,err:=OpenCursor(key,"network-A",token);if err!=nil||!got.Created.Equal(c.Created)||got.Kind!=c.Kind||got.ID!=c.ID {t.Fatalf("round trip: %+v %v",got,err)}
 if _,err=OpenCursor(key,"network-B",token);err==nil {t.Fatal("cross tenant cursor accepted")}
 if _,err=OpenCursor([]byte("different-key-0123456789012345678901"),"network-A",token);err==nil {t.Fatal("wrong key accepted")}
 if _,err=OpenCursor(key,"network-A",token+"A");err==nil {t.Fatal("tampered cursor accepted")}
 if _,err=SealCursor(key,"network-A",Cursor{Created:c.Created,Kind:"transaction",ID:"x"});err==nil {t.Fatal("private type cursor accepted")}
 if _,err=SealCursor([]byte("short"),"network-A",c);err==nil {t.Fatal("short signing key accepted")}
}
