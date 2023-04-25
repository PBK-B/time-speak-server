package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.30

import (
	"context"
	"fmt"
	"time_speak_server/graph/generated"
)

// UpdateHashTag is the resolver for the updateHashTag field.
func (r *mutationResolver) UpdateHashTag(ctx context.Context, input generated.HashTagInput) (bool, error) {
	panic(fmt.Errorf("not implemented: UpdateHashTag - updateHashTag"))
}

// DeleteHashTag is the resolver for the deleteHashTag field.
func (r *mutationResolver) DeleteHashTag(ctx context.Context, input string) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteHashTag - deleteHashTag"))
}

// AllHashTags is the resolver for the allHashTags field.
func (r *queryResolver) AllHashTags(ctx context.Context, page int, size int, desc bool) ([]*generated.HashTag, error) {
	panic(fmt.Errorf("not implemented: AllHashTags - allHashTags"))
}

// HashTags is the resolver for the hashTags field.
func (r *queryResolver) HashTags(ctx context.Context, input string) ([]*generated.Memory, error) {
	panic(fmt.Errorf("not implemented: HashTags - hashTags"))
}