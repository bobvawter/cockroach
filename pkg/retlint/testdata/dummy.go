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

// This file contains test declarations which will be parsed by
// the test suite.

import "math/rand"

type Target interface {
	isTarget()
}

type Concrete struct{}

func (Concrete) isTarget() {}

type Other struct{}

func (*Other) isTarget() {}

func returnsConcrete() Concrete {
	return Concrete{}
}

func returnsConcretePtr() *Concrete {
	return &Concrete{}
}

func returnsConcreteAsTarget() Target {
	return Concrete{}
}

func returnsConcretePtrAsTarget() Target {
	return &Concrete{}
}

func phiSimple() Target {
	switch rand.Int() {
	case 0:
		return returnsConcrete()
	case 1:
		return returnsConcretePtr()
	case 2:
		return returnsConcreteAsTarget()
	case 3:
		return returnsConcretePtrAsTarget()
	case 4:
		return &Other{}
	case 5:
		return phiSimple()
	default:
		panic("unimplemented")
	}
}

var indirectInvocation func() Target

func callsIndirect() Target {
	return indirectInvocation()
}

func returnsMultiple() (Target, Target) {
	return &Concrete{}, &Other{}
}

func usesMultiple() Target {
	a, _ := returnsMultiple()
	return a
}
