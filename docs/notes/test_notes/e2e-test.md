```go
package e2e

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestUserFlow_E2E(t *testing.T) {
    // Setup real app (no mocks)
    router := setupRouter()

    server := httptest.NewServer(router)
    defer server.Close()

    // Step 1: Create user
    payload := map[string]string{
        "name": "John",
    }

    body, _ := json.Marshal(payload)

    resp, err := http.Post(server.URL+"/users", "application/json", bytes.NewBuffer(body))
    if err != nil {
        t.Fatal(err)
    }

    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("expected 201 got %d", resp.StatusCode)
    }

    // Step 2: Fetch user
    resp, err = http.Get(server.URL + "/users/1")
    if err != nil {
        t.Fatal(err)
    }

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200 got %d", resp.StatusCode)
    }
}
```
