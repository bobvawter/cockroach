// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in licenses/BSD-vitess.txt.

// Portions of this file are additionally subject to the following
// license and copyright.
//
// Copyright 2015 The Cockroach Authors.
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

// This code was derived from https://github.com/youtube/vitess.

package parser

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/sql/coltypes"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase/intsize"
	"github.com/pkg/errors"
)

// Parser wraps a scanner, parser and other utilities present in the parser
// package.
type Parser struct {
	scanner    Scanner
	parserImpl sqlParserImpl
}

// Parse parses the sql and returns a list of statements.
func (p *Parser) Parse(ctx Context, sql string) (stmts tree.StatementList, err error) {
	return parseWithDepth(ctx, 1, sql)
}

var defaultContext = &simpleContext{defaultIntSize: INT8}

func Default() Context {
	return defaultContext
}

type Context interface {
	getDefaultIntSize() intsize.IntSize
}

func IntSize(size intsize.IntSize) Context {
	return &simpleContext{defaultIntSize: size}
}

type simpleContext struct {
	defaultIntSize intsize.IntSize
}

func (c simpleContext) getDefaultIntSize() intsize.IntSize {
	return c.defaultIntSize
}

func (p *Parser) parseWithDepth(
	ctx Context, depth int, sql string,
) (stmts tree.StatementList, err error) {
	p.scanner.init(sql)
	switch ctx.getDefaultIntSize() {
	case intsize.INT4:
		p.scanner.defaultIntType = coltypes.Int4
	case intsize.Unknown, intsize.INT8:
		p.scanner.defaultIntType = coltypes.Int8
	default:
		return nil, errors.Errorf("unsupported default int size: %v", ctx.getDefaultIntSize())
	}

	if p.parserImpl.Parse(&p.scanner) != 0 {
		var err *pgerror.Error
		if feat := p.scanner.lastError.unimplementedFeature; feat != "" {
			// UnimplementedWithDepth populates the generic hint. However
			// in some cases we have a more specific hint. This is overridden
			// below.
			err = pgerror.UnimplementedWithDepth(depth+1, "syntax."+feat, p.scanner.lastError.msg)
		} else {
			err = pgerror.NewErrorWithDepth(depth+1, pgerror.CodeSyntaxError, p.scanner.lastError.msg)
		}
		if p.scanner.lastError.hint != "" {
			// If lastError.hint is not set, e.g. from (*Scanner).Unimplemented(),
			// we're OK with the default hint. Otherwise, override it.
			err.Hint = p.scanner.lastError.hint
		}
		err.Detail = p.scanner.lastError.detail
		return nil, err
	}
	return p.scanner.stmts, nil
}

// unaryNegation constructs an AST node for a negation. This attempts
// to preserve constant NumVals and embed the negative sign inside
// them instead of wrapping in an UnaryExpr. This in turn ensures
// that negative numbers get considered as a single constant
// for the purpose of formatting and scrubbing.
func unaryNegation(e tree.Expr) tree.Expr {
	if cst, ok := e.(*tree.NumVal); ok {
		cst.Negative = !cst.Negative
		return cst
	}

	// Common case.
	return &tree.UnaryExpr{Operator: tree.UnaryMinus, Expr: e}
}

// Parse parses a sql statement string and returns a list of Statements.
func Parse(ctx Context, sql string) (tree.StatementList, error) {
	return parseWithDepth(ctx, 1, sql)
}

func parseWithDepth(ctx Context, depth int, sql string) (tree.StatementList, error) {
	var p Parser
	return p.parseWithDepth(ctx, depth+1, sql)
}

// ParseOne parses a sql statement string, ensuring that it contains only a
// single statement, and returns that Statement.
func ParseOne(ctx Context, sql string) (tree.Statement, error) {
	stmts, err := parseWithDepth(ctx, 1, sql)
	if err != nil {
		return nil, err
	}
	if len(stmts) != 1 {
		return nil, pgerror.NewAssertionErrorf("expected 1 statement, but found %d", len(stmts))
	}
	return stmts[0], nil
}

// ParseTableNameWithIndex parses a table name with index.
func ParseTableNameWithIndex(sql string) (tree.TableNameWithIndex, error) {
	// We wrap the name we want to parse into a dummy statement since our parser
	// can only parse full statements.
	stmt, err := ParseOne(Default(), fmt.Sprintf("ALTER INDEX %s RENAME TO x", sql))
	if err != nil {
		return tree.TableNameWithIndex{}, err
	}
	rename, ok := stmt.(*tree.RenameIndex)
	if !ok {
		return tree.TableNameWithIndex{}, pgerror.NewAssertionErrorf("expected an ALTER INDEX statement, but found %T", stmt)
	}
	return *rename.Index, nil
}

// ParseTableName parses a table name.
func ParseTableName(sql string) (*tree.TableName, error) {
	// We wrap the name we want to parse into a dummy statement since our parser
	// can only parse full statements.
	stmt, err := ParseOne(Default(), fmt.Sprintf("ALTER TABLE %s RENAME TO x", sql))
	if err != nil {
		return nil, err
	}
	rename, ok := stmt.(*tree.RenameTable)
	if !ok {
		return nil, pgerror.NewAssertionErrorf("expected an ALTER TABLE statement, but found %T", stmt)
	}
	return &rename.Name, nil
}

// parseExprs parses one or more sql expressions.
func parseExprs(ctx Context, exprs []string) (tree.Exprs, error) {
	stmt, err := ParseOne(ctx, fmt.Sprintf("SET ROW (%s)", strings.Join(exprs, ",")))
	if err != nil {
		return nil, err
	}
	set, ok := stmt.(*tree.SetVar)
	if !ok {
		return nil, pgerror.NewAssertionErrorf("expected a SET statement, but found %T", stmt)
	}
	return set.Values, nil
}

// ParseExprs is a short-hand for parseExprs(sql)
func ParseExprs(ctx Context, sql []string) (tree.Exprs, error) {
	if len(sql) == 0 {
		return tree.Exprs{}, nil
	}
	return parseExprs(ctx, sql)
}

// ParseExpr is a short-hand for parseExprs([]string{sql})
func ParseExpr(ctx Context, sql string) (tree.Expr, error) {
	exprs, err := parseExprs(ctx, []string{sql})
	if err != nil {
		return nil, err
	}
	if len(exprs) != 1 {
		return nil, pgerror.NewAssertionErrorf("expected 1 expression, found %d", len(exprs))
	}
	return exprs[0], nil
}

// ParseType parses a column type.
func ParseType(sql string) (coltypes.CastTargetType, error) {
	expr, err := ParseExpr(Default(), fmt.Sprintf("1::%s", sql))
	if err != nil {
		return nil, err
	}

	cast, ok := expr.(*tree.CastExpr)
	if !ok {
		return nil, pgerror.NewAssertionErrorf("expected a tree.CastExpr, but found %T", expr)
	}

	return cast.Type, nil
}
