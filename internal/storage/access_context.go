package storage

import "context"

type environmentAccessKey struct{}

// WithEnvironmentAccess carries authenticated restrictions into catalog SQL,
// so authorization predicates are applied before LIMIT/OFFSET.
func WithEnvironmentAccess(ctx context.Context, ids []string) context.Context {
	return context.WithValue(ctx, environmentAccessKey{}, append([]string(nil), ids...))
}
func environmentAccess(ctx context.Context) []string {
	ids, _ := ctx.Value(environmentAccessKey{}).([]string)
	return ids
}
func appendEnvironmentAccess(ctx context.Context, query, column string, args []any) (string, []any) {
	ids := environmentAccess(ctx)
	if len(ids) == 0 {
		return query, args
	}
	query += " AND " + column + " IN ("
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	return query + ")", args
}
