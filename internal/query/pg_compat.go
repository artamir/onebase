package query

// isComparisonContext returns true when the token at position i (a string literal)
// appears in a comparison context: ident = 'literal', ident != 'literal',
// or ident IN ('literal', ...). This is used to add ::text on PostgreSQL to
// avoid "operator does not exist: uuid = text" errors.
func isComparisonContext(tokens []tok, i int) bool {
	// Look backwards for = or != operator
	for j := i - 1; j >= 0; j-- {
		switch tokens[j].kind {
		case tOp:
			return tokens[j].val == "=" || tokens[j].val == "!=" ||
				tokens[j].val == "<>" || tokens[j].val == "IN" ||
				tokens[j].val == "in"
		case tIdent:
			// Could be "IN" keyword
			if tokens[j].val == "IN" || tokens[j].val == "in" {
				return true
			}
			continue // skip identifiers
		case tComma, tLParen:
			// Inside IN (...), 'literal', ... — check if we're in an IN context
			for k := j - 1; k >= 0; k-- {
				if tokens[k].kind == tIdent && (tokens[k].val == "IN" || tokens[k].val == "in") {
					return true
				}
				if tokens[k].kind == tOp {
					break
				}
			}
			return false
		case tStr, tNum, tParam, tDot:
			continue
		default:
			return false
		}
	}
	return false
}
