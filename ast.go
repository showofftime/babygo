package main


// --- ast ---
var astCon string = "Con"
var astTyp string = "Typ"
var astVar string = "Var"
var astFun string = "Fun"

type signature struct {
	params  *astFieldList
	results *astFieldList
}

type ObjDecl struct {
	dtype     string
	valueSpec *astValueSpec
	funcDecl  *astFuncDecl
	typeSpec  *astTypeSpec
	field     *astField
}

type astObject struct {
	Kind     string
	Name     string
	Decl     *ObjDecl
	Variable *Variable
}

type astExpr struct {
	dtype        string
	ident        *astIdent
	arrayType    *astArrayType
	basicLit     *astBasicLit
	callExpr     *astCallExpr
	binaryExpr   *astBinaryExpr
	unaryExpr    *astUnaryExpr
	selectorExpr *astSelectorExpr
	indexExpr    *astIndexExpr
	sliceExpr    *astSliceExpr
	starExpr     *astStarExpr
	parenExpr    *astParenExpr
	structType   *astStructType
	compositeLit *astCompositeLit
	ellipsis     *astEllipsis
}

type astField struct {
	Name   *astIdent
	Type   *astExpr
	Offset int
}

type astFieldList struct {
	List []*astField
}

type astIdent struct {
	Name string
	Obj  *astObject
}

type astEllipsis struct {
	Elt *astExpr
}

type astBasicLit struct {
	Kind  string // token.INT, token.CHAR, or token.STRING
	Value string
}

type astCompositeLit struct {
	Type *astExpr
	Elts []*astExpr
}

type astParenExpr struct {
	X *astExpr
}

type astSelectorExpr struct {
	X   *astExpr
	Sel *astIdent
}

type astIndexExpr struct {
	X     *astExpr
	Index *astExpr
}

type astSliceExpr struct {
	X      *astExpr
	Low    *astExpr
	High   *astExpr
	Max    *astExpr
	Slice3 bool
}

type astCallExpr struct {
	Fun  *astExpr   // function expression
	Args []*astExpr // function arguments; or nil
}

type astStarExpr struct {
	X *astExpr
}

type astUnaryExpr struct {
	X  *astExpr
	Op string
}

type astBinaryExpr struct {
	X  *astExpr
	Y  *astExpr
	Op string
}

// Type nodes
type astArrayType struct {
	Len *astExpr
	Elt *astExpr
}

type astStructType struct {
	Fields *astFieldList
}

type astFuncType struct {
	params  *astFieldList
	results *astFieldList
}

type astStmt struct {
	dtype      string
	DeclStmt   *astDeclStmt
	exprStmt   *astExprStmt
	blockStmt  *astBlockStmt
	assignStmt *astAssignStmt
	returnStmt *astReturnStmt
	ifStmt     *astIfStmt
	forStmt    *astForStmt
	incDecStmt *astIncDecStmt
	isRange    bool
	rangeStmt  *astRangeStmt
	branchStmt *astBranchStmt
	switchStmt *astSwitchStmt
	caseClause *astCaseClause
}

type astDeclStmt struct {
	Decl *astDecl
}

type astExprStmt struct {
	X *astExpr
}

type astIncDecStmt struct {
	X   *astExpr
	Tok string
}

type astAssignStmt struct {
	Lhs []*astExpr
	Tok string
	Rhs []*astExpr
}

type astReturnStmt struct {
	Results []*astExpr
}

type astBranchStmt struct {
	Tok        string
	Label      string
	currentFor *astStmt
}

type astBlockStmt struct {
	List []*astStmt
}

type astIfStmt struct {
	Init *astStmt
	Cond *astExpr
	Body *astBlockStmt
	Else *astStmt
}

type astCaseClause struct {
	List []*astExpr
	Body []*astStmt
}

type astSwitchStmt struct {
	Tag  *astExpr
	Body *astBlockStmt
	// lableExit string
}

type astForStmt struct {
	Init      *astStmt
	Cond      *astExpr
	Post      *astStmt
	Body      *astBlockStmt
	Outer     *astStmt // outer loop
	labelPost string
	labelExit string
}

type astRangeStmt struct {
	Key       *astExpr
	Value     *astExpr
	X         *astExpr
	Body      *astBlockStmt
	Outer     *astStmt // outer loop
	labelPost string
	labelExit string
	lenvar    *Variable
	indexvar  *Variable
}

// Declarations
type astSpec struct {
	dtype     string
	valueSpec *astValueSpec
	typeSpec  *astTypeSpec
}

type astImportSpec struct {
	Path string
}

type astValueSpec struct {
	Name  *astIdent
	Type  *astExpr
	Value *astExpr
}

type astTypeSpec struct {
	Name *astIdent
	Type *astExpr
}

// Pseudo interface for *ast.Decl
type astDecl struct {
	dtype    string
	genDecl  *astGenDecl
	funcDecl *astFuncDecl
}

type astGenDecl struct {
	Spec *astSpec
}

type astFuncDecl struct {
	Name *astIdent
	Type *astFuncType
	Body *astBlockStmt
}

type astFile struct {
	Name       string
	Decls      []*astDecl
	Unresolved []*astIdent
}

type astScope struct {
	Outer   *astScope
	Objects []*objectEntry
}

func astNewScope(outer *astScope) *astScope {
	var r = new(astScope)
	r.Outer = outer
	return r
}

func scopeInsert(s *astScope, obj *astObject) {
	if s == nil {
		panic2(__func__, "s sholud not be nil\n")
	}
	var oe = new(objectEntry)
	oe.name = obj.Name
	oe.obj = obj
	s.Objects = append(s.Objects, oe)
}

func scopeLookup(s *astScope, name string) *astObject {
	var oe *objectEntry
	for _, oe = range s.Objects {
		if oe.name == name {
			return oe.obj
		}
	}
	var r *astObject
	return r
}
