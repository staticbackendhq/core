package backend_test

import (
	"fmt"

	"github.com/staticbackendhq/core/backend"
)

func ExampleBuildQueryFilters() {
	filters, err := backend.BuildQueryFilters(
		"done", "=", true,
		"effort", ">=", 15,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(filters)

	// Output: [[done = true] [effort >= 15]]
}
