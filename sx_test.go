package sx_test

import "testing"
import "github.com/hlandau/sx"

type testCase struct {
	In, Out string
}

var cases = []testCase{
	{"()", "()"},
	{"(())", "(())"},
	{"(()())", "(()())"},
	{"((()))", "((()))"},
	{"(((())))", "(((())))"},
	{"0", "0"},
	{"1", "1"},
	{"42", "42"},
	{"123", "123"},
	{"1234", "1234"},
	{"4294967296", "4294967296"},
	{"9999999999", "9999999999"},
	{"-1", "-1"},
	{"-2", "-2"},
	{"-42", "-42"},
	{"-9999999999", "-9999999999"},
	{"(42)", "(42)"},
	{"(42 105)", "(42 105)"},
	{"(42 105 -12)", "(42 105 -12)"},
	{"(-10 92 -108)", "(-10 92 -108)"},
	{"5:hello", "5:hello"},
	{"5:h\x00\xFF\x42o", "5:h\x00\xFF\x42o"},
	{"(1:a2:oh3:abc(4:open5:apple42 91))", "(1:a2:oh3:abc(4:open5:apple42 91))"},
	{"||", "0:"},
	{"|YQ==|", "1:a"},
	{"|YXA=|", "2:ap"},
	{"|YXBw|", "3:app"},
	{"|YXBwbA==|", "4:appl"},
	{"|YXB \r\t\nwbGU=|", "5:apple"},
	{"{MDo=}", "0:"},
	{"{MTph}", "1:a"},
	{"{MjphYg==}", "2:ab"},
	{"{MzphYmM=}", "3:abc"},
	{"{NDphYmNk}", "4:abcd"},
	{"{NTpoZ\n \t\rWxsbyAoNTp0aGVyZSk=}", "5:hello(5:there)"},
	{"the elves", "3:the5:elves"},
  {"-token", "6:-token"},
	{"#00010203af#", "5:\x00\x01\x02\x03\xaf"},
	{"5#00010203af#", "5:\x00\x01\x02\x03\xaf"},
  {`"apple"`, "5:apple"},
  {`"app\tfoo"`, "7:app\tfoo"},
  {`"app\x61ae"`, "6:appaae"},
  {`"app\xeeae"`, "6:app\xeeae"},
  {`"app\377ae"`, "6:app\xffae"},

  // rivest samples
  // canonical: input==output
  {"(11:certificate(6:issuer(4:name(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))13:aid-committee))(7:subject(3:ref(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))3:tom6:mother))(10:not-before19:1997-01-01_09:00:00)(9:not-after19:1998-01-01_09:00:00)(3:tag(5:spend(7:account8:12345678)(1:*7:numeric5:range1:14:1000))))", "(11:certificate(6:issuer(4:name(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))13:aid-committee))(7:subject(3:ref(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))3:tom6:mother))(10:not-before19:1997-01-01_09:00:00)(9:not-after19:1998-01-01_09:00:00)(3:tag(5:spend(7:account8:12345678)(1:*7:numeric5:range1:14:1000))))"},
  // advanced and transport: same output as canonical
  {`(certificate
 (issuer
  (name
   (public-key
    rsa-with-md5
    (e |NFGq/E3wh9f4rJIQVXhS|)
    (n |d738/4ghP9rFZ0gAIYZ5q9y6iskDJwASi5rEQpEQq8ZyMZeIZzIAR2I5iGE=|))
   aid-committee))
 (subject
  (ref
   (public-key
    rsa-with-md5
    (e |NFGq/E3wh9f4rJIQVXhS|)
    (n |d738/4ghP9rFZ0gAIYZ5q9y6iskDJwASi5rEQpEQq8ZyMZeIZzIAR2I5iGE=|))
   tom
   mother))
 (not-before "1997-01-01_09:00:00")
 (not-after "1998-01-01_09:00:00")
 (tag
  (spend (account "12345678") (* numeric range "1" "1000"))))`, "(11:certificate(6:issuer(4:name(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))13:aid-committee))(7:subject(3:ref(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))3:tom6:mother))(10:not-before19:1997-01-01_09:00:00)(9:not-after19:1998-01-01_09:00:00)(3:tag(5:spend(7:account8:12345678)(1:*7:numeric5:range1:14:1000))))",},
  {`{KDExOmNlcnRpZmljYXRlKDY6aXNzdWVyKDQ6bmFtZSgxMDpwdWJsaWMta2V5MTI6cnNhLXdpdGgtbWQ1KDE6ZTE1OjRRqvxN8IfX+KySEFV4UikoMTpuNDQ6d738/4ghP9rFZ0gAIYZ5q9y6iskDJwASi5rEQpEQq8ZyMZeIZzIAR2I5iGEpKTEzOmFpZC1jb21taXR0ZWUpKSg3OnN1YmplY3QoMzpyZWYoMTA6cHVibGljLWtleTEyOnJzYS13aXRoLW1kNSgxOmUxNTo0Uar8TfCH1/iskhBVeFIpKDE6bjQ0One9/P+IIT/axWdIACGGeavcuorJAycAEouaxEKREKvGcjGXiGcyAEdiOYhhKSkzOnRvbTY6bW90aGVyKSkoMTA6bm90LWJlZm9yZTE5OjE5OTctMDEtMDFfMDk6MDA6MDApKDk6bm90LWFmdGVyMTk6MTk5OC0wMS0wMV8wOTowMDowMCkoMzp0YWcoNTpzcGVuZCg3OmFjY291bnQ4OjEyMzQ1Njc4KSgxOio3Om51bWVyaWM1OnJhbmdlMToxNDoxMDAwKSkpKQ==
}`, "(11:certificate(6:issuer(4:name(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))13:aid-committee))(7:subject(3:ref(10:public-key12:rsa-with-md5(1:e15:4Q\xaa\xfcM\xf0\x87\xd7\xf8\xac\x92\x10UxR)(1:n44:w\xbd\xfc\xff\x88!?\xda\xc5gH\x00!\x86y\xabܺ\x8a\xc9\x03'\x00\x12\x8b\x9a\xc4B\x91\x10\xab\xc6r1\x97\x88g2\x00Gb9\x88a))3:tom6:mother))(10:not-before19:1997-01-01_09:00:00)(9:not-after19:1998-01-01_09:00:00)(3:tag(5:spend(7:account8:12345678)(1:*7:numeric5:range1:14:1000))))"},
}

func TestSX(t *testing.T) {
	for _, c := range cases {
		L, err := sx.SX.Parse([]byte(c.In))
		if err != nil {
			t.Logf("test case failed: %s: %v", c.In, err)
			t.Fail()
			continue
		}

		out, err := sx.SX.String(L)
		if err != nil {
			t.Logf("test case output failed: %s: %v", c.Out, err)
			t.Fail()
			continue
		}

		if out != c.Out {
			t.Logf("output does not match: %#v != %#v", out, c.Out)
			t.Fail()
			continue
		}
	}
}

func TestQuery(t *testing.T) {
  if !sx.Hhy([]interface{}{"alpha"}, "alpha") {
    t.Fatalf("...")
  }

  xs, err := sx.SX.Parse([]byte(`
    (alpha)
    (beta
      (x)
      (y qwe)
      (z))
    (gamma)
    (delta)
  `))
  if err != nil {
    t.Fatalf("failed to parse: %v", err)
  }

  if !sx.Hhy(xs[0], "alpha") {
    t.Fatalf("...")
  }

  if sx.Q1bhy(xs, "alpha") == nil {
    t.Fatalf("...")
  }

  if sx.Q1bhyt(xs, "beta") == nil {
    t.Fatalf("...")
  }

  r := sx.Q1bsyt(xs, "beta y")
  if r == nil {
    t.Fatalf("no match")
  }

  out, err := sx.SX.String(r)
  if err != nil {
    t.Fatalf("cannot serialize: %v", err)
  }

  if out != "3:qwe" {
    t.Fatalf("mismatch: %#v", out)
  }
}
