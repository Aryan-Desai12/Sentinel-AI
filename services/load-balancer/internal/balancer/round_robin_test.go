package balancer

import "testing"

func TestRoundRobin(t *testing.T) {
	r, err := NewRoundRobin([]string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
		"http://localhost:8081",
		"http://localhost:8082",
	}

	for i, expectedURL := range expected {
		backend := r.Next()

		if backend == nil {
			t.Fatalf("request %d returned nil backend", i)
		}

		if backend.URL.String() != expectedURL {
			t.Fatalf(
				"request %d: expected %s, got %s",
				i,
				expectedURL,
				backend.URL.String(),
			)
		}
	}
}
