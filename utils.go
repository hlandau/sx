package sx

import "fmt"

// Query first by head yarn.
//
// xs is a list of values. Finds a value of the form (s ...), where s is the
// string given. Returns that value or nil if no such value was found.
//
// For example, suppose xs represents:
//
//   (a ...)
//   (b ...)
//   (c ...)
//
// A call to Q1bhy(xs, "b") would return the list (b ...).
func Q1bhy(xs []interface{}, s string) []interface{} {
  for _, x := range xs {
    if Hhy(x, s) {
      return x.([]interface{})
    }
  }
  return nil
}

// Query first by head yarn tail.
//
// Like Q1bhy, but returns the tail of the list, i.e. (...) rather
// than (b ...).
func Q1bhyt(xs []interface{}, s string) []interface{} {
  v := Q1bhy(xs, s)
  if len(v) == 0 {
    return v
  }
  return v[1:]
}

// Has head yarn?
//
// Returns true iff v is of the form (s ...), where s is the string given.
func Hhy(v interface{}, s string) bool {
  if xs, ok := v.([]interface{}); ok && len(xs) > 0 {
    if ss, ok := xs[0].(string); ok && ss == s {
      return true
    }
  }
  return false
}

// Query first by selector yarn tail.
//
// A selector is an S-expression, for example "a b c".
// Each value in the expression represents a call to Q1bhyt.
//
// For example, given the following:
//
//   (a ...)
//   (b
//     (x ...)
//     (y "foo" "bar")
//     (z ...)
//   )
//   (c ...)
//
// the selector "b y" would return ("foo" "bar").
//
// Returns nil if no match.
func Q1bsyt(xs []interface{}, sel string) []interface{} {
  selvs, err := SX.Parse([]byte(sel))
  if err != nil {
    panic(fmt.Sprintf("bad selector: %v", err))
  }

  cur := xs
  for _, selv := range selvs {
    s, ok := selv.(string)
    if !ok {
      panic(fmt.Sprintf("non-string element in selector: %v", selvs))
    }

    cur = Q1bhyt(cur, s)
    if cur == nil {
      return nil
    }
  }

  return cur
}
