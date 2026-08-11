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
