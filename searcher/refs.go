package searcher

import (
	"github.com/etsy/hound/index"
	"path/filepath"
)

/**
 * Holds a set of IndexRefs that were found in the dbpath at startup,
 * these indexes can be 'claimed' and re-used by newly created Searchers.
 */
type foundRefs struct {
	refs    []*index.IndexRef
	claimed map[*index.IndexRef]bool
}

/**
 * Find an Index ref for the Repo url and rev, returns nil if no such
 * ref exists.
 */
func (r *foundRefs) find(url, rev string) *index.IndexRef {
	for _, ref := range r.refs {
		if ref.Url == url && ref.Rev == rev {
			return ref
		}
	}
	return nil
}

/**
 * Claim a ref for reuse. This ensures they ref will not be garbage
 * collected at the end of startup.
 */
func (r *foundRefs) claim(ref *index.IndexRef) {
	r.claimed[ref] = true
}

/**
 * Delete the directorires associated with all IndexRefs that were
 * found in the dbpath but were not claimed during startup.
 */
func (r *foundRefs) removeUnclaimed() error {
	for _, ref := range r.refs {
		if r.claimed[ref] {
			continue
		}

		if err := ref.Remove(); err != nil {
			return err
		}
	}
	return nil
}

// Read the refs associated with each of the index dirs
// in the given dbpath.
func findExistingRefs(dbpath string) (*foundRefs, error) {
	dirs, err := filepath.Glob(filepath.Join(dbpath, "idx-*"))
	if err != nil {
		return nil, err
	}

	var refs []*index.IndexRef
	for _, dir := range dirs {
		r, _ := index.Read(dir)
		refs = append(refs, r)
	}

	return &foundRefs{
		refs:    refs,
		claimed: map[*index.IndexRef]bool{},
	}, nil
}
