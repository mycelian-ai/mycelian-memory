package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestSearch(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.Method != http.MethodPost {
            t.Fatalf("expected POST")
        }
        resp := SearchResponse{Entries: []SearchEntry{{Entry: Entry{ID: "e1"}}}, Count: 1}
        _ = json.NewEncoder(w).Encode(&resp)
    }))
    defer srv.Close()
    c := New(srv.URL)
    res, err := c.Search(context.Background(), SearchRequest{UserID: "user1", MemoryID: "m1", Query: "x"})
    if err != nil || len(res.Entries) != 1 {
        t.Fatalf("Search error: %v", err)
    }
}
