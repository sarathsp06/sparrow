package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

type listRecipesOutput struct {
	Body struct {
		Items []recipes.Recipe `json:"items" doc:"Every shipped recipe."`
	}
}

func registerRecipeRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listRecipes",
		Method:      http.MethodGet,
		Path:        "/v1/recipes",
		Summary:     "List shipped adapter recipes",
		Description: "Lists the adapter recipes built into this Sparrow server. A recipe pre-fills a webhook destination, headers and a subscription transform; clients (the UI, sparrow use) substitute the params and register the webhook. Static and read-only.\n\nPortal tokens: not allowed.",
		Errors:      []int{500},
		Tags:        []string{"Recipes"},
	}, func(ctx context.Context, _ *struct{}) (*listRecipesOutput, error) {
		items, err := recipes.All()
		if err != nil {
			return nil, mapError(ctx, err, "failed to list recipes")
		}
		out := &listRecipesOutput{}
		out.Body.Items = items
		return out, nil
	})
}
