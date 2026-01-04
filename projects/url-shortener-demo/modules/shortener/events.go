package shortener

import "github.com/go-monolith/mono/pkg/helper"

// Event definitions for the shortener module.
var (
	// URLCreatedV1 is published when a new short URL is created.
	URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](
		"shortener",
		"URLCreated",
		"v1",
	)

	// URLAccessedV1 is published when a short URL is accessed for redirect.
	URLAccessedV1 = helper.EventDefinition[URLAccessedEvent](
		"shortener",
		"URLAccessed",
		"v1",
	)
)
