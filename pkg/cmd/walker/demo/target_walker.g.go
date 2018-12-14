//+build !walkerAnalysis

package demo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type TargetW interface {
	TargetKind() TargetKind
}

type TargetKind int

const (
	_ TargetKind = iota
	Target_ByRefType
	Target_ByValType
	Target_ContainerType
)
