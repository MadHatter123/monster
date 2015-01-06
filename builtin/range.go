//  Copyright (c) 2013 Couchbase, Inc.

package builtin

import "fmt"
import "math/rand"

import "github.com/prataprc/monster/common"

func Range(scope common.Scope, args ...interface{}) interface{} {
	var min, max int64
	var err error

	rnd := scope["_random"].(*rand.Rand)
	if len(args) == 2 {
		min, max = args[0].(int64), args[1].(int64)
		if err != nil {
			panic(fmt.Errorf("parsing argument %v\n", args[1]))
		}

	} else if len(args) == 1 {
		max = args[0].(int64)

	} else {
		panic(fmt.Errorf("Atleast one argument expected for range-form\n"))
	}
	return rnd.Int63n(max-min) + min
}
