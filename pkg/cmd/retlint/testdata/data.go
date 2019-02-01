package testdata

type GoodPtrError struct{}

func (*GoodPtrError) Error() string { return "Good" }

type GoodValError struct{}

func (GoodValError) Error() string { return "Good" }

var (
	_ error = &GoodPtrError{}
	_ error = GoodValError{}
)

func DirectGood() error {
	switch choose() {
	case 0:
		return GoodValError{}
	case 1:
		return &GoodPtrError{}
	default:
		panic("fake code")
	}
}

func PhiGood() error {
	var ret error
	switch choose() {
	case 0:
		ret = GoodValError{}
	case 1:
		ret = &GoodPtrError{}
	default:
		panic("fake code")
	}
	return ret
}

//go:noinline
func choose() int {
	return -1
}
