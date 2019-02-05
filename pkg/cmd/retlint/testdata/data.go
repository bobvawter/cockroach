// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package testdata

import "errors"

type Selfish interface {
	Self() error
}

type BadError struct{}

func (*BadError) Error() string { return "Bad" }
func (e *BadError) Self() error { return e }

type GoodPtrError struct{}

func (*GoodPtrError) Error() string { return "Good" }
func (e *GoodPtrError) Self() error { return e }

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

func EnsureGoodValWithCommaOk(err error) error {
	if tested, ok := err.(GoodValError); ok {
		return tested
	}
	return GoodValError{}
}

func ExplictReturnVarNoOp() (err error) {
	return
}

func ExplicitReturnVarBad() (err error) {
	err = DirectBad()
	return
}

func ExplicitReturnVarGood() (err error) {
	err = DirectGood()
	return
}

func ExplicitReturnVarPhiBad() (err error) {
	switch choose() {
	case 0:
		err = DirectBad()
	case 1:
		err = DirectGood()
	}
	return
}

func ExplicitReturnVarPhiGood() (err error) {
	switch choose() {
	case 0:
		err = DirectGood()
	case 1:
		err = &GoodValError{}
	case 2:
		err = &GoodPtrError{}
	}
	return
}

func EnsureGoodValWithSwitch(err error) error {
	switch t := err.(type) {
	case GoodValError:
		return t
	case *GoodPtrError:
		return t
	default:
		return GoodValError{}
	}
}

func EnsureGoodValWithTest(err error) error {
	if _, ok := err.(GoodValError); ok {
		return err
	}
	return GoodValError{}
}

func MakesIndirectCall(fn func() error) error {
	return fn()
}

func NoopGood() {}

func NoopCallGood() {
	NoopGood()
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

func ReturnNilGood() error {
	return nil
}

func UsesSelfBad() error {
	return (&BadError{}).Self()
}

func UsesSelfGood() error {
	return (&GoodPtrError{}).Self()
}

func choose() int {
	return -1
}
