package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

// SetETag sets a strong entity tag derived from a resource version.
func SetETag(w http.ResponseWriter, version int64) {
	w.Header().Set("ETag", etag(version))
}

// CheckIfMatch enforces the If-Match precondition against the current version
// of a resource. A request without the header is unconditional.
func CheckIfMatch(r *http.Request, version int64) error {
	header := r.Header.Get("If-Match")
	if header == "" || strings.TrimSpace(header) == "*" {
		return nil
	}

	current := etag(version)
	for _, candidate := range strings.Split(header, ",") {
		if strings.TrimSpace(candidate) == current {
			return nil
		}
	}
	return errs.New(errs.CodePreconditionFailed, "the resource has changed since it was last read")
}

func etag(version int64) string {
	return `"` + strconv.FormatInt(version, 10) + `"`
}
