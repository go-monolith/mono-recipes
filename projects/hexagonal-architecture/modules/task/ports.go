package task

import (
	"context"

	usercontract "github.com/example/hexagonal-architecture/modules/user/contract"
)

// UserPort is a driven port: what the task domain needs from the user module.
// The task module owns it, it speaks the user contract's DTOs and says nothing
// about NATS. Tests can substitute a fake for it.
type UserPort interface {
	ValidateUser(ctx context.Context, req usercontract.ValidateUserRequest) (usercontract.ValidateUserResponse, error)
}
