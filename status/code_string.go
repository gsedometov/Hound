// Code generated by "stringer -type=Code"; DO NOT EDIT.

package status

import "fmt"

const _Code_name = "StartingIndexingReadyStoppingErrorUndefined"

var _Code_index = [...]uint8{0, 8, 16, 21, 29, 34, 43}

func (i Code) String() string {
	if i < 0 || i >= Code(len(_Code_index)-1) {
		return fmt.Sprintf("Code(%d)", i)
	}
	return _Code_name[_Code_index[i]:_Code_index[i+1]]
}
