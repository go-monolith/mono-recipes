package task

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"

	usercontract "github.com/example/hexagonal-architecture/modules/user/contract"
)

// userAdapter is the driven adapter behind UserPort: it calls the user
// module's contract through the container received in
// SetDependencyServiceContainer. It is the only place in the task module that
// knows the call crosses a module boundary.
type userAdapter struct {
	container mono.ServiceContainer
}

var _ UserPort = userAdapter{}

func (a userAdapter) ValidateUser(ctx context.Context, req usercontract.ValidateUserRequest) (usercontract.ValidateUserResponse, error) {
	resp, err := usercontract.ValidateUser.Call(ctx, a.container, req)
	if err != nil {
		return usercontract.ValidateUserResponse{}, fmt.Errorf("validate user failed: %w", err)
	}
	return resp, nil
}
