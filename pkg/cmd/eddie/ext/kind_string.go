// Code generated by "stringer -type Kind -trimprefix Kind"; DO NOT EDIT.

package ext

import "strconv"

const _Kind_name = "UnknownMethodFunctionInterfaceInterfaceMethodType"

var _Kind_index = [...]uint8{0, 7, 13, 21, 30, 45, 49}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}