package generator

import "github.com/lucasjones/reggen"

func Reggen(pattern string, len int) (string, error) {
	return reggen.Generate(pattern, len)
}
