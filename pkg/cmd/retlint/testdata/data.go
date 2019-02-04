package testdata

import "errors"

type BadError struct{}

func (*BadError) Error() string { return "Bad" }

type GoodPtrError struct{}

func (*GoodPtrError) Error() string { return "Good" }

type GoodValError struct{}

func (GoodValError) Error() string { return "Good" }

var (
	_ error = &GoodPtrError{}
	_ error = GoodValError{}
)

func DirectBad() error {
	return errors.New("nope")
}

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

func DirectTupleBad() (int, error) {
	return 0, errors.New("nope")
}

func DirectTupleBadCaller() error {
	_, err := DirectTupleBad()
	return err
}

func DirectTupleBadChain() (int, error) {
	x, err := DirectTupleBad()
	return x + 1, err
}

func PhiBad() error {
	var ret error
	switch choose() {
	case 0:
		ret = GoodValError{}
	case 1:
		ret = &GoodPtrError{}
	case 2:
		ret = DirectGood()
	case 3:
		ret = DirectBad()
	default:
		panic("fake code")
	}
	return ret
}

func PhiGood() error {
	var ret error
	switch choose() {
	case 0:
		ret = GoodValError{}
	case 1:
		ret = &GoodPtrError{}
	case 2:
		ret = DirectGood()
	default:
		panic("fake code")
	}
	return ret
}

// ShortestWhyPath demonstrates that we choose the shortest "why" path.
func ShortestWhyPath() error {
	switch choose() {
	case 0:
		return &BadError{}
	case 1:
		return errors.New("shouldn't see this")
	default:
		return PhiBad()
	}
}

func choose() int {
	return -1
}
