// Example: Module Boundaries (Ports and Adapters), mono v0.1.0+
//
// Layout (Go module path "example.com/shop"):
//
//	main.go
//	inventory/contract/contract.go   published port: DTOs + typed service contract
//	inventory/module.go              domain logic + RegisterServices via contract
//	order/contract/contract.go       published port: service contract + emitted event
//	order/ports.go                   driven ports the order domain needs
//	order/adapter.go                 adapters: ports implemented over provider contracts
//	order/module.go                  domain logic, Dependencies(), ConsumeServices()
//	order/module_test.go             monotest.CheckModule conformance test
//
// What the framework checks:
//   - compile time: both sides share one contract value
//   - startup: ConsumeServices() is validated against the dependencies
//   - runtime: app.ModuleBoundaries() reports use and violations;
//     mono.WithStrictModuleBoundaries() rejects violations
//   - tests: monotest.CheckModule
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-monolith/mono"

	"example.com/shop/inventory"
	"example.com/shop/order"
	ordercontract "example.com/shop/order/contract"
)

func main() {
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithShutdownTimeout(10*time.Second),
		// Recommended path: run without this first, read ModuleBoundaries()
		// and the Stop log, fix what they report, then turn it on.
		mono.WithStrictModuleBoundaries(),
	)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}
	for _, m := range []mono.Module{inventory.NewModule(), order.NewModule()} {
		if err := app.Register(m); err != nil {
			log.Fatalf("register %s: %v", m.Name(), err)
		}
	}

	ctx := context.Background()
	// A consumed contract that no dependency provides fails here, not later.
	if err := app.Start(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	// The host program calls a module through its contract too. app.Services is
	// a host-level escape hatch for main.go, HTTP handlers and tests; module
	// code must use the container from SetDependencyServiceContainer instead.
	resp, err := ordercontract.PlaceOrder.Call(ctx, app.Services("order"),
		ordercontract.PlaceOrderRequest{ProductID: "laptop", Quantity: 1})
	if err != nil {
		log.Fatalf("place order: %v", err)
	}
	log.Printf("order %s: %s", resp.OrderID, resp.Status)

	// How each module actually used its boundaries during this run.
	for _, r := range app.ModuleBoundaries() {
		log.Printf("module=%s declared=%v observed=%+v unused=%v violations=%+v",
			r.Module, r.Declared, r.Observed, r.UnusedDependencies, r.Violations)
	}

	if err := app.Stop(ctx); err != nil {
		log.Fatalf("stop: %v", err)
	}
}
