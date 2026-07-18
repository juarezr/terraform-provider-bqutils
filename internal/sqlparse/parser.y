%{
package sqlparse

// Parser grammar for BigQuery CREATE routine/view headers.
// The production rules document the supported surface; the runtime entrypoint
// is Parse() in parse.go which uses the same lexical conventions.
//
//go:generate goyacc -o y.go -p yy parser.y

%}

%union {
	str string
	num int
}

%token <str> IDENT STRING NUMBER
%token CREATE OR REPLACE TEMPORARY TEMP FUNCTION TABLE PROCEDURE AGGREGATE
%token VIEW MATERIALIZED IF NOT EXISTS RETURNS LANGUAGE OPTIONS AS
%token REMOTE WITH CONNECTION PARTITION BY CLUSTER ANY TYPE
%token IN OUT INOUT TRUE FALSE BEGIN END
%token LPAREN RPAREN LBRACKET RBRACKET LANGLE RANGLE COMMA DOT EQ SEMI STAR

%start statement

%%

statement:
	create_header
	;

create_header:
	CREATE opt_or_replace opt_temp object_kind opt_if_not_exists
	;

opt_or_replace:
	| OR REPLACE
	;

opt_temp:
	| TEMPORARY
	| TEMP
	;

object_kind:
	FUNCTION
	| TABLE FUNCTION
	| AGGREGATE FUNCTION
	| PROCEDURE
	| VIEW
	| MATERIALIZED VIEW
	;

opt_if_not_exists:
	| IF NOT EXISTS
	;

%%

// yySymType and token constants are generated into y.go.
// Runtime parsing is implemented in parse.go for full OPTIONS/type/body handling.
func init() {
	_ = yyToknames
}
