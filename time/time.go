// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Abridged copy of /usr/local/go/src/time/time.go, containing only the
// parts potentially used for communication between Go and C code in TCG.
//
// There are several reasons for this abridgement.
//
// (*) We already have some special handling in place within the gotocjson
//     conversion tool for the time.Time data structure.  That is done to
//     support our gwos/milliseconds package, since the C implementation of
//     that structure is not a direct analogue of the Go implementation.
//     Including the actual time.Time data structure from time.go collides
//     with that special handling.
//
// (*) The official time.go code includes two separate const blocks that both
//     define sets of constants of type Duration.  Our gotocjson conversion
//     tool presently treats those blocks as separate enumerations, and the C
//     compiler objects to the collision of having two separate "enum Duration"
//     declarations.  A future version of the conversion tool might consolidate
//     such declarations, but that has not yet been done.
//
// (*) Our present application code as of this writing only needs a tiny part
//     of the official time.go code with respect to what we address with our
//     gotocjson tool.  So at least for the time being, there is insufficient
//     cause to extend the tool to better address the items noted above.

package time

// A Duration represents the elapsed time between two instants
// as an int64 nanosecond count. The representation limits the
// largest representable duration to approximately 290 years.
//
type Duration int64

