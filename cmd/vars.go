package main

import (
	"fmt"
	"strings"
)

// applyVars substitutes {{key}} placeholders in sql for each "key=value" entry in vars.
func applyVars(sql string, vars []string) (string, error) {
	for _, v := range vars {
		k, val, ok := strings.Cut(v, "=")
		if !ok {
			return "", fmt.Errorf("invalid --var %q: expected key=value", v)
		}
		sql = strings.ReplaceAll(sql, "{{"+k+"}}", val)
	}
	return sql, nil
}
