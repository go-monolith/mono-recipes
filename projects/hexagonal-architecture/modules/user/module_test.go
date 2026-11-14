package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/hexagonal-architecture/modules/user"
	"github.com/example/hexagonal-architecture/modules/user/contract"
)

// TestUserModuleConformance checks the user module against the Ports and
// Adapters rules and calls every contract it provides.
func TestUserModuleConformance(t *testing.T) {
	monotest.CheckModule(t, user.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			got, err := contract.GetUser.Call(ctx, h.Container(), contract.GetUserRequest{UserID: "user-1"})
			if err != nil {
				return err
			}
			if !got.Found || got.User == nil || got.User.Name != "Alice Johnson" {
				return fmt.Errorf("get-user user-1 = %+v", got)
			}

			valid, err := contract.ValidateUser.Call(ctx, h.Container(), contract.ValidateUserRequest{UserID: "user-2"})
			if err != nil {
				return err
			}
			if !valid.Valid {
				return fmt.Errorf("validate-user user-2 = %+v, want valid", valid)
			}
			return nil
		}))
}

func TestUnknownUser(t *testing.T) {
	ctx := context.Background()
	h := monotest.New(t, user.NewModule())
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	got, err := contract.GetUser.Call(ctx, h.Container(), contract.GetUserRequest{UserID: "nobody"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Found || got.User != nil {
		t.Errorf("get-user nobody = %+v, want not found", got)
	}

	valid, err := contract.ValidateUser.Call(ctx, h.Container(), contract.ValidateUserRequest{UserID: "nobody"})
	if err != nil {
		t.Fatal(err)
	}
	if valid.Valid {
		t.Error("validate-user nobody = valid, want invalid")
	}
}
