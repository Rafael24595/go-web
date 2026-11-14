package paths

import (
	"testing"

	"github.com/Rafael24595/go-assert/assert"
)

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   bool
	}{
		{
			path:   "/api",
			prefix: "/api",
			want:   true,
		},
		{
			path:   "/api/items",
			prefix: "/api",
			want:   true,
		},
		{
			path:   "/api/items",
			prefix: "/api/",
			want:   true,
		},
		{
			path:   "/apiary",
			prefix: "/api",
			want:   false,
		},
		{
			path:   "/api",
			prefix: "/api/items",
			want:   false,
		},
		{
			path:   "/anything",
			prefix: "/",
			want:   true,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, HasPrefix(tt.path, tt.prefix))
	}
}

func TestGroups(t *testing.T) {
	groups := []string{"/api/admin", "/apiary", "/api", "/other"}

	assert.DeepEqual(t, []string{"/api", "/api/admin"}, Groups("/api/admin/users", groups))
	assert.DeepEqual(t, []string{"/api"}, Groups("/api/items", groups))
	assert.DeepEqual(t, []string{}, Groups("/nowhere", groups))
}
