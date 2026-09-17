package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

type listRecipesOutput struct {
	Body struct {
		Items []recipes.Recipe `json:"items"`
	}
}

func registerRecipeRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listRecipes",
		Method:      http.MethodGet,
		Path:        "/v1/recipes",
		Summary:     "List shipped adapter recipes",
		Description: "Lists recipe templates shipped with this Sparrow server. Recipes pre-fill webhook destinations, headers, and optional subscription transforms.",
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
