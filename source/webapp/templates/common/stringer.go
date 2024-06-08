// It's inconceivable to me the language doesn't have this infrastructure honestly.
// It's even more crazy go templ only supports strings, and not fmt.Formatter.
package common

import "fmt"

type StringerFormat1 struct {
    Format string
    Value interface{}
}

func (s *StringerFormat1) String() string {
    return fmt.Sprintf(s.Format, s.Value)
}

type StringerString struct {
    Str string
}

func (s *StringerString) String() string {
    return s.Str
}

type StringerFormatter struct {
    Formatter fmt.Formatter
}

func (s *StringerFormatter) String() string {
    return fmt.Sprintf("%v", s.Formatter)
}
