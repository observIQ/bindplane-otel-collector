# Regex Matcher

The `matcher` package encapsulates the problem of matching a string to one of many possible regexes. 
For example, when processing logs of many formats, it can be used to efficiently determine which of several
expected formats the log matches.

## Usage

```go
// Instantate a `Matcher` by providing `[]NamedRegex`. The order of these values is respected
// so that the name of the first regex to match will be returned.
matcher := matcher.New(
	[]matcher.NamedRegex{
    // use an rfc timestamp format
		{Name: "five_digits", Regex: regexp.MustCompile(`^<(\d{5})>`)},
		{Name: "json", Regex: regexp.MustCompile(`^\{.*\}$`)},
		{Name: "logfmt", Regex: regexp.MustCompile(`^.*=.*$`)},
	},
)

// Match will return the name of the first regex that matches the input string.
// If no regex matches, it will return "".
regexName := matcher.Match("some log line")
if regexName == "" {
	// no regex matched
}
```

## Performance

The primary purpose of this package is to improve upon the performance we might expect from a naive
approach of evaluating each regex independently against the input string. Therefore, a benchmark is
included which compares the performance of the `Matcher` against this naive approach, given the same
ordered set of regexes.