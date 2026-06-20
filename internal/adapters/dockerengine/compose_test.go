package dockerengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The project filter selects exactly the project's objects via a structured
// label match — the project name never reaches a shell or command line.
func TestComposeProjectFilter(t *testing.T) {
	f := composeProjectFilter("blog")
	assert.True(t, f.Match("label", composeProjectLabel+"=blog"))
	assert.False(t, f.Match("label", composeProjectLabel+"=other"))
	// A name with shell metacharacters is treated as a literal label value.
	assert.True(t, composeProjectFilter("a;rm -rf /").Match("label", composeProjectLabel+"=a;rm -rf /"))
}
