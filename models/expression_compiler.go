package models

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/almighty/almighty-core/criteria"
)

const (
	jsonAnnotation = "JSON"
)

// Compile takes an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and a slice of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, err []error) {
	criteria.IteratePostOrder(where, bubbleUpJSONContext)

	compiler := newExpressionCompiler()
	compiled := where.Accept(&compiler)

	return compiled.(string), compiler.parameters, compiler.err
}

// mark expression tree nodes that reference json fields
func bubbleUpJSONContext(exp criteria.Expression) bool {
	switch t := exp.(type) {
	case *criteria.FieldExpression:
		if isJSONField(t.FieldName) {
			t.SetAnnotation(jsonAnnotation, true)
		}
	case *criteria.EqualsExpression:
		if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
			t.SetAnnotation(jsonAnnotation, true)
		}
	}
	return true
}

// does the field name reference a json field or a column?
func isJSONField(fieldName string) bool {
	switch fieldName {
	case "ID", "Type", "Version":
		return false
	}
	return true
}

func newExpressionCompiler() expressionCompiler {
	return expressionCompiler{parameters: []interface{}{}}
}

// expressionCompiler takes an expression and compiles it to a where clause for our gorm models
// implements criteria.ExpressionVisitor
type expressionCompiler struct {
	parameters []interface{} // records the number of parameter expressions encountered
	err        []error       // record any errors found in the expression
}

// visitor implementation
// the convention is to return nil when the expression cannot be compiled and to append an error to the err field

func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	if !isJSONField(f.FieldName) {
		return f.FieldName
	}
	if strings.Contains(f.FieldName, "'") {
		// beware of injection, it's a reasonable restriction for field names, make sure it's not allowed when creating wi types
		c.err = append(c.err, fmt.Errorf("single quote not allowed in field name"))
		return nil
	}
	return "Fields->'" + f.FieldName + "'"
}

func (c *expressionCompiler) And(a *criteria.AndExpression) interface{} {
	return c.binary(a, "and")
}

func (c *expressionCompiler) binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left().Accept(c)
	right := a.Right().Accept(c)
	if left != nil && right != nil {
		return "(" + left.(string) + " " + op + " " + right.(string) + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return nil
}

func (c *expressionCompiler) Or(a *criteria.OrExpression) interface{} {
	return c.binary(a, "or")
}

func (c *expressionCompiler) Equals(e *criteria.EqualsExpression) interface{} {
	return c.binary(e, "=")
}

func (c *expressionCompiler) Matches(e *criteria.MatchesExpression) interface{} {
	left := e.Left().Accept(c)
	right := e.Right().Accept(c)
	if left != nil && right != nil {
		return "(to_tsvector('english', " + left.(string) + ") @@ to_tsquery('english', " + right.(string) + "))"
	}
	// something went wrong in either compilation, errors have been accumulated
	return nil
}

func (c *expressionCompiler) Parameter(v *criteria.ParameterExpression) interface{} {
	c.err = append(c.err, fmt.Errorf("Parameter expression not supported"))
	return nil
}

// iterate the parent chain to see if this expression references json fields
func isInJSONContext(exp criteria.Expression) bool {
	result := false
	criteria.IterateParents(exp, func(exp criteria.Expression) bool {
		if exp.Annotation(jsonAnnotation) == true {
			result = true
			return false
		}
		return true
	})
	return result
}

// literal values need to be converted differently depending on whether they are used in a JSON context or a regular SQL expression.
// JSON values are always strings (delimited with "'"), but operators can be used depending on the dynamic type. For example,
// you can write "a->'foo' < '5'" and it will return true for the json object { "a": 40 }.
func (c *expressionCompiler) Literal(v *criteria.LiteralExpression) interface{} {
	json := isInJSONContext(v)
	if json {
		stringVal, err := c.convertToString(v.Value)
		if err == nil {
			c.parameters = append(c.parameters, stringVal)
			return "?::jsonb"
		}
		c.err = append(c.err, err)
		return nil
	}
	c.parameters = append(c.parameters, v.Value)
	return "?"
}

func (c *expressionCompiler) convertToString(value interface{}) (string, error) {
	var result string
	switch t := value.(type) {
	case float64:
		result = strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		result = strconv.FormatInt(int64(t), 10)
	case int64:
		result = strconv.FormatInt(t, 10)
	case uint:
		result = strconv.FormatUint(uint64(t), 10)
	case uint64:
		result = strconv.FormatUint(t, 10)
	case string:
		result = "\"" + t + "\""
	case bool:
		result = strconv.FormatBool(t)
	default:
		return "", fmt.Errorf("unknown value type of %v: %T", value, value)
	}
	return result, nil
}
