package plank

import (
	"context"
	"testing"
	"time"
)

func TestEnsureAndStatus(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	st := m.GetStatus(ctx)
	if !st.Online {
		t.Fatalf("expected online, got %#v", st)
	}
	if m.Token() == "" {
		t.Fatal("expected session token")
	}
	if st.AgentName != "P.L.A.N.K." {
		t.Fatalf("agent name: %q", st.AgentName)
	}
	t.Logf("model=%s provider=%s skin=%s", st.Model, st.Provider, st.Skin)
}
