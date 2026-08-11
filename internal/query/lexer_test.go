package query

import "testing"

func TestReadString(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple string",
			query:    `"GET"`,
			expected: "GET",
		},
		{
			name:     "path string",
			query:    `"/users"`,
			expected: "/users",
		},
		{
			name:     "string with spaces",
			query:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "empty string",
			query:    `""`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := Lexer{
				Query: tt.query,
			}

			lexer.readString()

			if len(lexer.Tokens) != 1 {
				t.Fatalf(
					"expected 1 token, got %d",
					len(lexer.Tokens),
				)
			}

			got := lexer.Tokens[0]

			if got.Type != TokenString {
				t.Errorf(
					"expected token type %v, got %v",
					TokenString,
					got.Type,
				)
			}

			if got.Literal != tt.expected {
				t.Errorf(
					"expected literal %q, got %q",
					tt.expected,
					got.Literal,
				)
			}
		})
	}
}

func TestReadNumber(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		startPosition    int
		expectedLiteral  string
		expectedPosition int
		wantErr          bool
	}{
		{
			name:             "integer",
			query:            "400",
			startPosition:    0,
			expectedLiteral:  "400",
			expectedPosition: 3,
		},
		{
			name:             "integer followed by space",
			query:            "400 host",
			startPosition:    0,
			expectedLiteral:  "400",
			expectedPosition: 3,
		},
		{
			name:             "integer followed by identifier",
			query:            "400host",
			startPosition:    0,
			expectedLiteral:  "400",
			expectedPosition: 3,
		},
		{
			name:             "integer inside query",
			query:            `status=400host="test-service"`,
			startPosition:    7,
			expectedLiteral:  "400",
			expectedPosition: 10,
		},
		{
			name:          "decimal is not allowed",
			query:         "400.5",
			startPosition: 0,
			wantErr:       true,
		},
		{
			name:          "invalid character",
			query:         "400@",
			startPosition: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := Lexer{
				Query:    tt.query,
				Position: tt.startPosition,
			}

			err := lexer.readNumber()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(lexer.Tokens) != 1 {
				t.Fatalf(
					"expected 1 token, got %d",
					len(lexer.Tokens),
				)
			}

			token := lexer.Tokens[0]

			if token.Type != TokenNumber {
				t.Errorf(
					"expected TokenNumber, got %v",
					token.Type,
				)
			}

			if token.Literal != tt.expectedLiteral {
				t.Errorf(
					"expected literal %q, got %q",
					tt.expectedLiteral,
					token.Literal,
				)
			}

			if token.Position != tt.startPosition {
				t.Errorf(
					"expected token position %d, got %d",
					tt.startPosition,
					token.Position,
				)
			}

			if lexer.Position != tt.expectedPosition {
				t.Errorf(
					"expected lexer position %d, got %d",
					tt.expectedPosition,
					lexer.Position,
				)
			}
		})
	}
}

func TestReadOperation(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		startPosition    int
		expectedType     TokenType
		expectedLiteral  string
		expectedPosition int
		wantErr          bool
	}{
		{
			name:             "equal",
			query:            "=",
			startPosition:    0,
			expectedType:     TokenEqual,
			expectedLiteral:  "=",
			expectedPosition: 1,
		},
		{
			name:             "not equal",
			query:            "!=",
			startPosition:    0,
			expectedType:     TokenNotEqual,
			expectedLiteral:  "!=",
			expectedPosition: 2,
		},
		{
			name:             "less than",
			query:            "<",
			startPosition:    0,
			expectedType:     TokenLess,
			expectedLiteral:  "<",
			expectedPosition: 1,
		},
		{
			name:             "less than or equal",
			query:            "<=",
			startPosition:    0,
			expectedType:     TokenLessEqual,
			expectedLiteral:  "<=",
			expectedPosition: 2,
		},
		{
			name:             "greater than",
			query:            ">",
			startPosition:    0,
			expectedType:     TokenGreater,
			expectedLiteral:  ">",
			expectedPosition: 1,
		},
		{
			name:             "greater than or equal",
			query:            ">=",
			startPosition:    0,
			expectedType:     TokenGreaterEqual,
			expectedLiteral:  ">=",
			expectedPosition: 2,
		},
		{
			name:          "standalone not is invalid",
			query:         "!",
			startPosition: 0,
			wantErr:       true,
		},
		{
			name:             "operator inside query",
			query:            `status>=400`,
			startPosition:    6,
			expectedType:     TokenGreaterEqual,
			expectedLiteral:  ">=",
			expectedPosition: 8,
		},
		{
			name:             "equal inside query",
			query:            `host="test-service"`,
			startPosition:    4,
			expectedType:     TokenEqual,
			expectedLiteral:  "=",
			expectedPosition: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := Lexer{
				Query:    tt.query,
				Position: tt.startPosition,
			}

			err := lexer.readOperator()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(lexer.Tokens) != 1 {
				t.Fatalf(
					"expected 1 token, got %d",
					len(lexer.Tokens),
				)
			}

			token := lexer.Tokens[0]

			if token.Type != tt.expectedType {
				t.Errorf(
					"expected token type %v, got %v",
					tt.expectedType,
					token.Type,
				)
			}

			if token.Literal != tt.expectedLiteral {
				t.Errorf(
					"expected literal %q, got %q",
					tt.expectedLiteral,
					token.Literal,
				)
			}

			if token.Position != tt.startPosition {
				t.Errorf(
					"expected token position %d, got %d",
					tt.startPosition,
					token.Position,
				)
			}

			if lexer.Position != tt.expectedPosition {
				t.Errorf(
					"expected lexer position %d, got %d",
					tt.expectedPosition,
					lexer.Position,
				)
			}
		})
	}
}

func TestReadDelimiter(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		startPosition    int
		expectedType     TokenType
		expectedLiteral  string
		expectedPosition int
		wantErr          bool
	}{
		{
			name:             "left parenthesis",
			query:            "(",
			startPosition:    0,
			expectedType:     TokenLeftParen,
			expectedLiteral:  "(",
			expectedPosition: 1,
		},
		{
			name:             "right parenthesis",
			query:            ")",
			startPosition:    0,
			expectedType:     TokenRightParen,
			expectedLiteral:  ")",
			expectedPosition: 1,
		},
		{
			name:             "left parenthesis inside query",
			query:            `status=400 AND (host="test-service")`,
			startPosition:    15,
			expectedType:     TokenLeftParen,
			expectedLiteral:  "(",
			expectedPosition: 16,
		},
		{
			name:             "right parenthesis inside query",
			query:            `(status=400)`,
			startPosition:    11,
			expectedType:     TokenRightParen,
			expectedLiteral:  ")",
			expectedPosition: 12,
		},
		{
			name:          "invalid delimiter",
			query:         "[",
			startPosition: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := Lexer{
				Query:    tt.query,
				Position: tt.startPosition,
			}

			err := lexer.readDelimiter()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(lexer.Tokens) != 1 {
				t.Fatalf(
					"expected 1 token, got %d",
					len(lexer.Tokens),
				)
			}

			token := lexer.Tokens[0]

			if token.Type != tt.expectedType {
				t.Errorf(
					"expected token type %v, got %v",
					tt.expectedType,
					token.Type,
				)
			}

			if token.Literal != tt.expectedLiteral {
				t.Errorf(
					"expected literal %q, got %q",
					tt.expectedLiteral,
					token.Literal,
				)
			}

			if token.Position != tt.startPosition {
				t.Errorf(
					"expected token position %d, got %d",
					tt.startPosition,
					token.Position,
				)
			}

			if lexer.Position != tt.expectedPosition {
				t.Errorf(
					"expected lexer position %d, got %d",
					tt.expectedPosition,
					lexer.Position,
				)
			}
		})
	}
}

func TestReadIdentifier(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		startPosition    int
		expectedType     TokenType
		expectedLiteral  string
		expectedPosition int
	}{
		{
			name:             "host identifier",
			query:            "host",
			startPosition:    0,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "host",
			expectedPosition: 4,
		},
		{
			name:             "arbitrary identifier",
			query:            "banana",
			startPosition:    0,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "banana",
			expectedPosition: 6,
		},
		{
			name:             "identifier with underscore",
			query:            "request_id",
			startPosition:    0,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "request_id",
			expectedPosition: 10,
		},
		{
			name:             "identifier with number",
			query:            "field123",
			startPosition:    0,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "field123",
			expectedPosition: 8,
		},
		{
			name:             "and keyword",
			query:            "AND",
			startPosition:    0,
			expectedType:     TokenAnd,
			expectedLiteral:  "AND",
			expectedPosition: 3,
		},
		{
			name:             "or keyword",
			query:            "OR",
			startPosition:    0,
			expectedType:     TokenOr,
			expectedLiteral:  "OR",
			expectedPosition: 2,
		},
		{
			name:             "not keyword",
			query:            "NOT",
			startPosition:    0,
			expectedType:     TokenNot,
			expectedLiteral:  "NOT",
			expectedPosition: 3,
		},
		{
			name:             "lowercase keyword",
			query:            "and",
			startPosition:    0,
			expectedType:     TokenAnd,
			expectedLiteral:  "and",
			expectedPosition: 3,
		},
		{
			name:             "identifier before operator",
			query:            "status>=400",
			startPosition:    0,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "status",
			expectedPosition: 6,
		},
		{
			name:             "identifier after number",
			query:            `400host="test-service"`,
			startPosition:    3,
			expectedType:     TokenIdentifier,
			expectedLiteral:  "host",
			expectedPosition: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := Lexer{
				Query:    tt.query,
				Position: tt.startPosition,
			}

			err := lexer.readIdentifier()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(lexer.Tokens) != 1 {
				t.Fatalf(
					"expected 1 token, got %d",
					len(lexer.Tokens),
				)
			}

			token := lexer.Tokens[0]

			if token.Type != tt.expectedType {
				t.Errorf(
					"expected token type %v, got %v",
					tt.expectedType,
					token.Type,
				)
			}

			if token.Literal != tt.expectedLiteral {
				t.Errorf(
					"expected literal %q, got %q",
					tt.expectedLiteral,
					token.Literal,
				)
			}

			if token.Position != tt.startPosition {
				t.Errorf(
					"expected token position %d, got %d",
					tt.startPosition,
					token.Position,
				)
			}

			if lexer.Position != tt.expectedPosition {
				t.Errorf(
					"expected lexer position %d, got %d",
					tt.expectedPosition,
					lexer.Position,
				)
			}
		})
	}
}
