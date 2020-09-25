package main

// --- parser ---
const O_READONLY int = 0
const FILE_SIZE int = 2000000

func readFile(filename string) []uint8 {
	var fd int
	// @TODO check error
	fd, _ = syscall.Open(filename, O_READONLY, 0)
	var buf = make([]uint8, FILE_SIZE, FILE_SIZE)
	var n int
	// @TODO check error
	n, _ = syscall.Read(fd, buf)
	var readbytes = buf[0:n]
	return readbytes
}

func readSource(filename string) []uint8 {
	return readFile(filename)
}

func parserInit(src []uint8) {
	scannerInit(src)
	parserNext()
}

type objectEntry struct {
	name string
	obj  *astObject
}

var ptok *TokenContainer

var parserUnresolved []*astIdent

var parserTopScope *astScope
var parserPkgScope *astScope

func openScope() {
	parserTopScope = astNewScope(parserTopScope)
}

func closeScope() {
	parserTopScope = parserTopScope.Outer
}

func parserConsumeComment() {
	parserNext0()
}

func parserNext0() {
	ptok = scannerScan()
}

func parserNext() {
	parserNext0()
	if ptok.tok == ";" {
		logf(" [parser] pointing at : \"%s\" newline (%s)\n", ptok.tok, Itoa(scannerOffset))
	} else if ptok.tok == "IDENT" {
		logf(" [parser] pointing at: IDENT \"%s\" (%s)\n", ptok.lit, Itoa(scannerOffset))
	} else {
		logf(" [parser] pointing at: \"%s\" %s (%s)\n", ptok.tok, ptok.lit, Itoa(scannerOffset))
	}

	if ptok.tok == "COMMENT" {
		for ptok.tok == "COMMENT" {
			parserConsumeComment()
		}
	}
}

func parserExpect(tok string, who string) {
	if ptok.tok != tok {
		var s = fmtSprintf("%s expected, but got %s", []string{tok, ptok.tok})
		panic2(who, s)
	}
	logf(" [%s] consumed \"%s\"\n", who, ptok.tok)
	parserNext()
}

func parserExpectSemi(caller string) {
	if ptok.tok != ")" && ptok.tok != "}" {
		switch ptok.tok {
		case ";":
			logf(" [%s] consumed semicolon %s\n", caller, ptok.tok)
			parserNext()
		default:
			panic2(caller, "semicolon expected, but got token "+ptok.tok)
		}
	}
}

func parseIdent() *astIdent {
	var name string
	if ptok.tok == "IDENT" {
		name = ptok.lit
		parserNext()
	} else {
		panic2(__func__, "IDENT expected, but got "+ptok.tok)
	}
	logf(" [%s] ident name = %s\n", __func__, name)
	var r = new(astIdent)
	r.Name = name
	return r
}

func parserParseImportDecl() *astImportSpec {
	parserExpect("import", __func__)
	var path = ptok.lit
	parserNext()
	parserExpectSemi(__func__)
	var spec = new(astImportSpec)
	spec.Path = path
	return spec
}

func tryVarType(ellipsisOK bool) *astExpr {
	if ellipsisOK && ptok.tok == "..." {
		parserNext() // consume "..."
		var typ = tryIdentOrType()
		if typ != nil {
			parserResolve(typ)
		} else {
			panic2(__func__, "Syntax error")
		}
		var elps = new(astEllipsis)
		elps.Elt = typ
		var r = new(astExpr)
		r.dtype = "*astEllipsis"
		r.ellipsis = elps
		return r
	}
	return tryIdentOrType()
}

func parseVarType(ellipsisOK bool) *astExpr {
	logf(" [%s] begin\n", __func__)
	var typ = tryVarType(ellipsisOK)
	if typ == nil {
		panic2(__func__, "nil is not expected")
	}
	logf(" [%s] end\n", __func__)
	return typ
}

func tryType() *astExpr {
	logf(" [%s] begin\n", __func__)
	var typ = tryIdentOrType()
	if typ != nil {
		parserResolve(typ)
	}
	logf(" [%s] end\n", __func__)
	return typ
}

func parseType() *astExpr {
	var typ = tryType()
	return typ
}

func parsePointerType() *astExpr {
	parserExpect("*", __func__)
	var base = parseType()
	var starExpr = new(astStarExpr)
	starExpr.X = base
	var r = new(astExpr)
	r.dtype = "*astStarExpr"
	r.starExpr = starExpr
	return r
}

func parseArrayType() *astExpr {
	parserExpect("[", __func__)
	var ln *astExpr
	if ptok.tok != "]" {
		ln = parseRhs()
	}
	parserExpect("]", __func__)
	var elt = parseType()
	var arrayType = new(astArrayType)
	arrayType.Elt = elt
	arrayType.Len = ln
	var r = new(astExpr)
	r.dtype = "*astArrayType"
	r.arrayType = arrayType
	return r
}

func parseFieldDecl(scope *astScope) *astField {

	var varType = parseVarType(false)
	var typ = tryVarType(false)

	parserExpectSemi(__func__)

	var field = new(astField)
	field.Type = typ
	field.Name = varType.ident
	declareField(field, scope, astVar, varType.ident)
	parserResolve(typ)
	return field
}

func parseStructType() *astExpr {
	parserExpect("struct", __func__)
	parserExpect("{", __func__)

	var _nil *astScope
	var scope = astNewScope(_nil)

	var structType = new(astStructType)
	var list []*astField
	for ptok.tok == "IDENT" || ptok.tok == "*" {
		var field *astField = parseFieldDecl(scope)
		list = append(list, field)
	}
	parserExpect("}", __func__)

	var fields = new(astFieldList)
	fields.List = list
	structType.Fields = fields
	var r = new(astExpr)
	r.dtype = "*astStructType"
	r.structType = structType
	return r
}

func parseTypeName() *astExpr {
	logf(" [%s] begin\n", __func__)
	var ident = parseIdent()
	var typ = new(astExpr)
	typ.ident = ident
	typ.dtype = "*astIdent"
	logf(" [%s] end\n", __func__)
	return typ
}

func tryIdentOrType() *astExpr {
	logf(" [%s] begin\n", __func__)
	switch ptok.tok {
	case "IDENT":
		return parseTypeName()
	case "[":
		return parseArrayType()
	case "struct":
		return parseStructType()
	case "*":
		return parsePointerType()
	case "(":
		parserNext()
		var _typ = parseType()
		parserExpect(")", __func__)
		var parenExpr = new(astParenExpr)
		parenExpr.X = _typ
		var typ = new(astExpr)
		typ.dtype = "*astParenExpr"
		typ.parenExpr = parenExpr
		return typ
	}
	var _nil *astExpr
	return _nil
}

func parseParameterList(scope *astScope, ellipsisOK bool) []*astField {
	logf(" [%s] begin\n", __func__)
	var list []*astExpr
	for {
		var varType = parseVarType(ellipsisOK)
		list = append(list, varType)
		if ptok.tok != "," {
			break
		}
		parserNext()
		if ptok.tok == ")" {
			break
		}
	}
	logf(" [%s] collected list n=%s\n", __func__, Itoa(len(list)))

	var params []*astField

	var typ = tryVarType(ellipsisOK)
	if typ != nil {
		if len(list) > 1 {
			panic2(__func__, "Ident list is not supported")
		}
		var eIdent = list[0]
		if eIdent.dtype != "*astIdent" {
			panic2(__func__, "Unexpected dtype")
		}
		var ident = eIdent.ident
		var field = new(astField)
		if ident == nil {
			panic2(__func__, "Ident should not be nil")
		}
		logf(" [%s] ident.Name=%s\n", __func__, ident.Name)
		logf(" [%s] typ=%s\n", __func__, typ.dtype)
		field.Name = ident
		field.Type = typ
		logf(" [%s]: Field %s %s\n", __func__, field.Name.Name, field.Type.dtype)
		params = append(params, field)
		declareField(field, scope, astVar, ident)
		parserResolve(typ)
		if ptok.tok != "," {
			logf("  end %s\n", __func__)
			return params
		}
		parserNext()
		for ptok.tok != ")" && ptok.tok != "EOF" {
			ident = parseIdent()
			typ = parseVarType(ellipsisOK)
			field = new(astField)
			field.Name = ident
			field.Type = typ
			params = append(params, field)
			declareField(field, scope, astVar, ident)
			parserResolve(typ)
			if ptok.tok != "," {
				break
			}
			parserNext()
		}
		logf("  end %s\n", __func__)
		return params
	}

	// Type { "," Type } (anonymous parameters)
	params = make([]*astField, len(list), len(list))
	var i int
	for i, typ = range list {
		parserResolve(typ)
		var field = new(astField)
		field.Type = typ
		params[i] = field
		logf(" [DEBUG] range i = %s\n", Itoa(i))
	}
	logf("  end %s\n", __func__)
	return params
}

func parseParameters(scope *astScope, ellipsisOk bool) *astFieldList {
	logf(" [%s] begin\n", __func__)
	var params []*astField
	parserExpect("(", __func__)
	if ptok.tok != ")" {
		params = parseParameterList(scope, ellipsisOk)
	}
	parserExpect(")", __func__)
	var afl = new(astFieldList)
	afl.List = params
	logf(" [%s] end\n", __func__)
	return afl
}

func parserResult(scope *astScope) *astFieldList {
	logf(" [%s] begin\n", __func__)

	if ptok.tok == "(" {
		var r = parseParameters(scope, false)
		logf(" [%s] end\n", __func__)
		return r
	}

	var r = new(astFieldList)
	if ptok.tok == "{" {
		r = nil
		logf(" [%s] end\n", __func__)
		return r
	}
	var typ = tryType()
	var field = new(astField)
	field.Type = typ
	r.List = append(r.List, field)
	logf(" [%s] end\n", __func__)
	return r
}

func parseSignature(scope *astScope) *signature {
	logf(" [%s] begin\n", __func__)
	var params *astFieldList
	var results *astFieldList
	params = parseParameters(scope, true)
	results = parserResult(scope)
	var sig = new(signature)
	sig.params = params
	sig.results = results
	return sig
}

func declareField(decl *astField, scope *astScope, kind string, ident *astIdent) {
	// delcare
	var obj = new(astObject)
	var objDecl = new(ObjDecl)
	objDecl.dtype = "*astField"
	objDecl.field = decl
	obj.Decl = objDecl
	obj.Name = ident.Name
	obj.Kind = kind
	ident.Obj = obj

	// scope insert
	if ident.Name != "_" {
		scopeInsert(scope, obj)
	}
}

func declare(objDecl *ObjDecl, scope *astScope, kind string, ident *astIdent) {
	logf(" [declare] ident %s\n", ident.Name)

	var obj = new(astObject) //valSpec.Name.Obj
	obj.Decl = objDecl
	obj.Name = ident.Name
	obj.Kind = kind
	ident.Obj = obj

	// scope insert
	if ident.Name != "_" {
		scopeInsert(scope, obj)
	}
	logf(" [declare] end\n")

}

func parserResolve(x *astExpr) {
	tryResolve(x, true)
}
func tryResolve(x *astExpr, collectUnresolved bool) {
	if x.dtype != "*astIdent" {
		return
	}
	var ident = x.ident
	if ident.Name == "_" {
		return
	}

	var s *astScope
	for s = parserTopScope; s != nil; s = s.Outer {
		var obj = scopeLookup(s, ident.Name)
		if obj != nil {
			ident.Obj = obj
			return
		}
	}

	if collectUnresolved {
		parserUnresolved = append(parserUnresolved, ident)
		logf(" appended unresolved ident %s\n", ident.Name)
	}
}

func parseOperand() *astExpr {
	logf("   begin %s\n", __func__)
	switch ptok.tok {
	case "IDENT":
		var eIdent = new(astExpr)
		eIdent.dtype = "*astIdent"
		var ident = parseIdent()
		eIdent.ident = ident
		tryResolve(eIdent, true)
		logf("   end %s\n", __func__)
		return eIdent
	case "INT", "STRING", "CHAR":
		var basicLit = new(astBasicLit)
		basicLit.Kind = ptok.tok
		basicLit.Value = ptok.lit
		var r = new(astExpr)
		r.dtype = "*astBasicLit"
		r.basicLit = basicLit
		parserNext()
		logf("   end %s\n", __func__)
		return r
	case "(":
		parserNext() // consume "("
		parserExprLev++
		var x = parserRhsOrType()
		parserExprLev--
		parserExpect(")", __func__)
		var p = new(astParenExpr)
		p.X = x
		var r = new(astExpr)
		r.dtype = "*astParenExpr"
		r.parenExpr = p
		return r
	}

	var typ = tryIdentOrType()
	if typ == nil {
		panic2(__func__, "# typ should not be nil\n")
	}
	logf("   end %s\n", __func__)

	return typ
}

func parserRhsOrType() *astExpr {
	var x = parseExpr()
	return x
}

func parseCallExpr(fn *astExpr) *astExpr {
	parserExpect("(", __func__)
	var callExpr = new(astCallExpr)
	callExpr.Fun = fn
	logf(" [parsePrimaryExpr] ptok.tok=%s\n", ptok.tok)
	var list []*astExpr
	for ptok.tok != ")" {
		var arg = parseExpr()
		list = append(list, arg)
		if ptok.tok == "," {
			parserNext()
		} else if ptok.tok == ")" {
			break
		}
	}
	parserExpect(")", __func__)
	callExpr.Args = list
	var r = new(astExpr)
	r.dtype = "*astCallExpr"
	r.callExpr = callExpr
	return r
}

var parserExprLev int // < 0: in control clause, >= 0: in expression

func parsePrimaryExpr() *astExpr {
	logf("   begin %s\n", __func__)
	var x = parseOperand()

	var cnt int

	for {
		cnt++
		logf("    [%s] tok=%s\n", __func__, ptok.tok)
		if cnt > 100 {
			panic2(__func__, "too many iteration")
		}

		switch ptok.tok {
		case ".":
			parserNext() // consume "."
			if ptok.tok != "IDENT" {
				panic2(__func__, "tok should be IDENT")
			}
			// Assume CallExpr
			var secondIdent = parseIdent()
			var sel = new(astSelectorExpr)
			sel.X = x
			sel.Sel = secondIdent
			if ptok.tok == "(" {
				var fn = new(astExpr)
				fn.dtype = "*astSelectorExpr"
				fn.selectorExpr = sel
				// string = x.ident.Name + "." + secondIdent
				x = parseCallExpr(fn)
				logf(" [parsePrimaryExpr] 741 ptok.tok=%s\n", ptok.tok)
			} else {
				logf("   end parsePrimaryExpr()\n")
				x = new(astExpr)
				x.dtype = "*astSelectorExpr"
				x.selectorExpr = sel
			}
		case "(":
			x = parseCallExpr(x)
		case "[":
			parserResolve(x)
			x = parseIndexOrSlice(x)
		case "{":
			if isLiteralType(x) && parserExprLev >= 0 {
				x = parseLiteralValue(x)
			} else {
				return x
			}
		default:
			logf("   end %s\n", __func__)
			return x
		}
	}

	logf("   end %s\n", __func__)
	return x
}

func parseLiteralValue(x *astExpr) *astExpr {
	logf("   start %s\n", __func__)
	parserExpect("{", __func__)
	var elts []*astExpr
	var e *astExpr
	for ptok.tok != "}" {
		e = parseExpr()
		elts = append(elts, e)
		if ptok.tok == "}" {
			break
		} else {
			parserExpect(",", __func__)
		}
	}
	parserExpect("}", __func__)

	var compositeLit = new(astCompositeLit)
	compositeLit.Type = x
	compositeLit.Elts = elts
	var r = new(astExpr)
	r.dtype = "*astCompositeLit"
	r.compositeLit = compositeLit
	logf("   end %s\n", __func__)
	return r
}

func isLiteralType(x *astExpr) bool {
	switch x.dtype {
	case "*astIdent":
	case "*astSelectorExpr":
		return x.selectorExpr.X.dtype == "*astIdent"
	case "*astArrayType":
	case "*astStructType":
	case "*astMapType":
	default:
		return false
	}

	return true
}

func parseIndexOrSlice(x *astExpr) *astExpr {
	parserExpect("[", __func__)
	var index = make([]*astExpr, 3, 3)
	if ptok.tok != ":" {
		index[0] = parseRhs()
	}
	var ncolons int
	for ptok.tok == ":" && ncolons < 2 {
		ncolons++
		parserNext() // consume ":"
		if ptok.tok != ":" && ptok.tok != "]" {
			index[ncolons] = parseRhs()
		}
	}
	parserExpect("]", __func__)

	if ncolons > 0 {
		// slice expression
		if ncolons == 2 {
			panic2(__func__, "TBI: ncolons=2")
		}
		var sliceExpr = new(astSliceExpr)
		sliceExpr.Slice3 = false
		sliceExpr.X = x
		sliceExpr.Low = index[0]
		sliceExpr.High = index[1]
		var r = new(astExpr)
		r.dtype = "*astSliceExpr"
		r.sliceExpr = sliceExpr
		return r
	}

	var indexExpr = new(astIndexExpr)
	indexExpr.X = x
	indexExpr.Index = index[0]
	var r = new(astExpr)
	r.dtype = "*astIndexExpr"
	r.indexExpr = indexExpr
	return r
}

func parseUnaryExpr() *astExpr {
	var r *astExpr
	logf("   begin parseUnaryExpr()\n")
	switch ptok.tok {
	case "+", "-", "!", "&":
		var tok = ptok.tok
		parserNext()
		var x = parseUnaryExpr()
		r = new(astExpr)
		r.dtype = "*astUnaryExpr"
		r.unaryExpr = new(astUnaryExpr)
		logf(" [DEBUG] unary op = %s\n", tok)
		r.unaryExpr.Op = tok
		r.unaryExpr.X = x
		return r
	case "*":
		parserNext() // consume "*"
		var x = parseUnaryExpr()
		r = new(astExpr)
		r.dtype = "*astStarExpr"
		r.starExpr = new(astStarExpr)
		r.starExpr.X = x
		return r
	}
	r = parsePrimaryExpr()
	logf("   end parseUnaryExpr()\n")
	return r
}

const LowestPrec int = 0

func precedence(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case "==", "!=", "<", "<=", ">", ">=":
		return 3
	case "+", "-":
		return 4
	case "*", "/", "%":
		return 5
	default:
		return 0
	}
	return 0
}

func parseBinaryExpr(prec1 int) *astExpr {
	logf("   begin parseBinaryExpr() prec1=%s\n", Itoa(prec1))
	var x = parseUnaryExpr()
	var oprec int
	for {
		var op = ptok.tok
		oprec = precedence(op)
		logf(" oprec %s\n", Itoa(oprec))
		logf(" precedence \"%s\" %s < %s\n", op, Itoa(oprec), Itoa(prec1))
		if oprec < prec1 {
			logf("   end parseBinaryExpr() (NonBinary)\n")
			return x
		}
		parserExpect(op, __func__)
		var y = parseBinaryExpr(oprec + 1)
		var binaryExpr = new(astBinaryExpr)
		binaryExpr.X = x
		binaryExpr.Y = y
		binaryExpr.Op = op
		var r = new(astExpr)
		r.dtype = "*astBinaryExpr"
		r.binaryExpr = binaryExpr
		x = r
	}
	logf("   end parseBinaryExpr()\n")
	return x
}

func parseExpr() *astExpr {
	logf("   begin parseExpr()\n")
	var e = parseBinaryExpr(1)
	logf("   end parseExpr()\n")
	return e
}

func parseRhs() *astExpr {
	var x = parseExpr()
	return x
}

// Extract Expr from ExprStmt. Returns nil if input is nil
func makeExpr(s *astStmt) *astExpr {
	logf(" begin %s\n", __func__)
	if s == nil {
		var r *astExpr
		return r
	}
	if s.dtype != "*astExprStmt" {
		panic2(__func__, "unexpected dtype="+s.dtype)
	}
	if s.exprStmt == nil {
		panic2(__func__, "exprStmt is nil")
	}
	return s.exprStmt.X
}

func parseForStmt() *astStmt {
	logf(" begin %s\n", __func__)
	parserExpect("for", __func__)
	openScope()

	var s1 *astStmt
	var s2 *astStmt
	var s3 *astStmt
	var isRange bool
	parserExprLev = -1
	if ptok.tok != "{" {
		if ptok.tok != ";" {
			s2 = parseSimpleStmt(true)
			isRange = s2.isRange
			logf(" [%s] isRange=true\n", __func__)
		}
		if !isRange && ptok.tok == ";" {
			parserNext() // consume ";"
			s1 = s2
			s2 = nil
			if ptok.tok != ";" {
				s2 = parseSimpleStmt(false)
			}
			parserExpectSemi(__func__)
			if ptok.tok != "{" {
				s3 = parseSimpleStmt(false)
			}
		}
	}

	parserExprLev = 0
	var body = parseBlockStmt()
	parserExpectSemi(__func__)

	var as *astAssignStmt
	var rangeX *astExpr
	if isRange {
		assert(s2.dtype == "*astAssignStmt", "type mismatch", __func__)
		as = s2.assignStmt
		logf(" [DEBUG] range as len lhs=%s\n", Itoa(len(as.Lhs)))
		var key *astExpr
		var value *astExpr
		switch len(as.Lhs) {
		case 0:
		case 1:
			key = as.Lhs[0]
		case 2:
			key = as.Lhs[0]
			value = as.Lhs[1]
		default:
			panic2(__func__, "Unexpected len of as.Lhs")
		}
		rangeX = as.Rhs[0].unaryExpr.X
		var rangeStmt = new(astRangeStmt)
		rangeStmt.Key = key
		rangeStmt.Value = value
		rangeStmt.X = rangeX
		rangeStmt.Body = body
		var r = new(astStmt)
		r.dtype = "*astRangeStmt"
		r.rangeStmt = rangeStmt
		closeScope()
		logf(" end %s\n", __func__)
		return r
	}
	var forStmt = new(astForStmt)
	forStmt.Init = s1
	forStmt.Cond = makeExpr(s2)
	forStmt.Post = s3
	forStmt.Body = body
	var r = new(astStmt)
	r.dtype = "*astForStmt"
	r.forStmt = forStmt
	closeScope()
	logf(" end %s\n", __func__)
	return r
}

func parseIfStmt() *astStmt {
	parserExpect("if", __func__)
	parserExprLev = -1
	var condStmt *astStmt = parseSimpleStmt(false)
	if condStmt.dtype != "*astExprStmt" {
		panic2(__func__, "unexpected dtype="+condStmt.dtype)
	}
	var cond = condStmt.exprStmt.X
	parserExprLev = 0
	var body = parseBlockStmt()
	var else_ *astStmt
	if ptok.tok == "else" {
		parserNext()
		if ptok.tok == "if" {
			else_ = parseIfStmt()
		} else {
			var elseblock = parseBlockStmt()
			parserExpectSemi(__func__)
			else_ = new(astStmt)
			else_.dtype = "*astBlockStmt"
			else_.blockStmt = elseblock
		}
	} else {
		parserExpectSemi(__func__)
	}
	var ifStmt = new(astIfStmt)
	ifStmt.Cond = cond
	ifStmt.Body = body
	ifStmt.Else = else_

	var r = new(astStmt)
	r.dtype = "*astIfStmt"
	r.ifStmt = ifStmt
	return r
}

func parseCaseClause() *astCaseClause {
	logf(" [%s] start\n", __func__)
	var list []*astExpr
	if ptok.tok == "case" {
		parserNext() // consume "case"
		list = parseRhsList()
	} else {
		parserExpect("default", __func__)
	}

	parserExpect(":", __func__)
	openScope()
	var body = parseStmtList()
	var r = new(astCaseClause)
	r.Body = body
	r.List = list
	closeScope()
	logf(" [%s] end\n", __func__)
	return r
}

func parseSwitchStmt() *astStmt {
	parserExpect("switch", __func__)
	openScope()

	var s2 *astStmt
	parserExprLev = -1
	s2 = parseSimpleStmt(false)
	parserExprLev = 0

	parserExpect("{", __func__)
	var list []*astStmt
	var cc *astCaseClause
	var ccs *astStmt
	for ptok.tok == "case" || ptok.tok == "default" {
		cc = parseCaseClause()
		ccs = new(astStmt)
		ccs.dtype = "*astCaseClause"
		ccs.caseClause = cc
		list = append(list, ccs)
	}
	parserExpect("}", __func__)
	parserExpectSemi(__func__)
	var body = new(astBlockStmt)
	body.List = list

	var switchStmt = new(astSwitchStmt)
	switchStmt.Body = body
	switchStmt.Tag = makeExpr(s2)
	var s = new(astStmt)
	s.dtype = "*astSwitchStmt"
	s.switchStmt = switchStmt
	closeScope()
	return s
}

func parseLhsList() []*astExpr {
	logf(" [%s] start\n", __func__)
	var list = parseExprList()
	logf(" end %s\n", __func__)
	return list
}

func parseSimpleStmt(isRangeOK bool) *astStmt {
	logf(" begin %s\n", __func__)
	var s = new(astStmt)
	var x = parseLhsList()
	var stok = ptok.tok
	var isRange = false
	var y *astExpr
	var rangeX *astExpr
	var rangeUnary *astUnaryExpr
	switch stok {
	case "=":
		parserNext() // consume =
		if isRangeOK && ptok.tok == "range" {
			parserNext() // consume "range"
			rangeX = parseRhs()
			rangeUnary = new(astUnaryExpr)
			rangeUnary.Op = "range"
			rangeUnary.X = rangeX
			y = new(astExpr)
			y.dtype = "*astUnaryExpr"
			y.unaryExpr = rangeUnary
			isRange = true
		} else {
			y = parseExpr() // rhs
		}
		var as = new(astAssignStmt)
		as.Tok = "="
		as.Lhs = x
		as.Rhs = make([]*astExpr, 1, 1)
		as.Rhs[0] = y
		s.dtype = "*astAssignStmt"
		s.assignStmt = as
		s.isRange = isRange
		logf(" end %s\n", __func__)
		return s
	case ";":
		s.dtype = "*astExprStmt"
		var exprStmt = new(astExprStmt)
		exprStmt.X = x[0]
		s.exprStmt = exprStmt
		logf(" end %s\n", __func__)
		return s
	}

	switch stok {
	case "++", "--":
		var s = new(astStmt)
		var sInc = new(astIncDecStmt)
		sInc.X = x[0]
		sInc.Tok = stok
		s.dtype = "*astIncDecStmt"
		s.incDecStmt = sInc
		parserNext() // consume "++" or "--"
		return s
	}
	var exprStmt = new(astExprStmt)
	exprStmt.X = x[0]
	var r = new(astStmt)
	r.dtype = "*astExprStmt"
	r.exprStmt = exprStmt
	logf(" end %s\n", __func__)
	return r
}

func parseStmt() *astStmt {
	logf("\n")
	logf(" = begin %s\n", __func__)
	var s *astStmt
	switch ptok.tok {
	case "var":
		var genDecl = parseDecl("var")
		s = new(astStmt)
		s.dtype = "*astDeclStmt"
		s.DeclStmt = new(astDeclStmt)
		var decl = new(astDecl)
		decl.dtype = "*astGenDecl"
		decl.genDecl = genDecl
		s.DeclStmt.Decl = decl
		logf(" = end parseStmt()\n")
	case "IDENT", "*":
		s = parseSimpleStmt(false)
		parserExpectSemi(__func__)
	case "return":
		s = parseReturnStmt()
	case "break", "continue":
		s = parseBranchStmt(ptok.tok)
	case "if":
		s = parseIfStmt()
	case "switch":
		s = parseSwitchStmt()
	case "for":
		s = parseForStmt()
	default:
		panic2(__func__, "TBI 3:"+ptok.tok)
	}
	logf(" = end parseStmt()\n")
	return s
}

func parseExprList() []*astExpr {
	logf(" [%s] start\n", __func__)
	var list []*astExpr
	var e = parseExpr()
	list = append(list, e)
	for ptok.tok == "," {
		parserNext() // consume ","
		e = parseExpr()
		list = append(list, e)
	}

	logf(" [%s] end\n", __func__)
	return list
}

func parseRhsList() []*astExpr {
	var list = parseExprList()
	return list
}

func parseBranchStmt(tok string) *astStmt {
	parserExpect(tok, __func__)

	parserExpectSemi(__func__)

	var branchStmt = new(astBranchStmt)
	branchStmt.Tok = tok
	var s = new(astStmt)
	s.dtype = "*astBranchStmt"
	s.branchStmt = branchStmt
	return s
}

func parseReturnStmt() *astStmt {
	parserExpect("return", __func__)
	var x []*astExpr
	if ptok.tok != ";" && ptok.tok != "}" {
		x = parseRhsList()
	}
	parserExpectSemi(__func__)
	var returnStmt = new(astReturnStmt)
	returnStmt.Results = x
	var r = new(astStmt)
	r.dtype = "*astReturnStmt"
	r.returnStmt = returnStmt
	return r
}

func parseStmtList() []*astStmt {
	var list []*astStmt
	for ptok.tok != "}" && ptok.tok != "EOF" && ptok.tok != "case" && ptok.tok != "default" {
		var stmt = parseStmt()
		list = append(list, stmt)
	}
	return list
}

func parseBody(scope *astScope) *astBlockStmt {
	parserExpect("{", __func__)
	parserTopScope = scope
	logf(" begin parseStmtList()\n")
	var list = parseStmtList()
	logf(" end parseStmtList()\n")

	closeScope()
	parserExpect("}", __func__)
	var r = new(astBlockStmt)
	r.List = list
	return r
}

func parseBlockStmt() *astBlockStmt {
	parserExpect("{", __func__)
	openScope()
	logf(" begin parseStmtList()\n")
	var list = parseStmtList()
	logf(" end parseStmtList()\n")
	closeScope()
	parserExpect("}", __func__)
	var r = new(astBlockStmt)
	r.List = list
	return r
}

func parseDecl(keyword string) *astGenDecl {
	var r *astGenDecl
	switch ptok.tok {
	case "var":
		parserExpect(keyword, __func__)
		var ident = parseIdent()
		var typ = parseType()
		var value *astExpr
		if ptok.tok == "=" {
			parserNext()
			value = parseExpr()
		}
		parserExpectSemi(__func__)
		var valSpec = new(astValueSpec)
		valSpec.Name = ident
		valSpec.Type = typ
		valSpec.Value = value
		var spec = new(astSpec)
		spec.dtype = "*astValueSpec"
		spec.valueSpec = valSpec
		var objDecl = new(ObjDecl)
		objDecl.dtype = "*astValueSpec"
		objDecl.valueSpec = valSpec
		declare(objDecl, parserTopScope, astVar, ident)
		r = new(astGenDecl)
		r.Spec = spec
		return r
	default:
		panic2(__func__, "TBI\n")
	}
	return r
}

func parserParseTypeSpec() *astSpec {
	logf(" [%s] start\n", __func__)
	parserExpect("type", __func__)
	var ident = parseIdent()
	logf(" decl type %s\n", ident.Name)

	var spec = new(astTypeSpec)
	spec.Name = ident
	var objDecl = new(ObjDecl)
	objDecl.dtype = "*astTypeSpec"
	objDecl.typeSpec = spec
	declare(objDecl, parserTopScope, astTyp, ident)
	var typ = parseType()
	parserExpectSemi(__func__)
	spec.Type = typ
	var r = new(astSpec)
	r.dtype = "*astTypeSpec"
	r.typeSpec = spec
	return r
}

func parserParseValueSpec(keyword string) *astSpec {
	logf(" [parserParseValueSpec] start\n")
	parserExpect(keyword, __func__)
	var ident = parseIdent()
	logf(" var = %s\n", ident.Name)
	var typ = parseType()
	var value *astExpr
	if ptok.tok == "=" {
		parserNext()
		value = parseExpr()
	}
	parserExpectSemi(__func__)
	var spec = new(astValueSpec)
	spec.Name = ident
	spec.Type = typ
	spec.Value = value
	var r = new(astSpec)
	r.dtype = "*astValueSpec"
	r.valueSpec = spec
	var objDecl = new(ObjDecl)
	objDecl.dtype = "*astValueSpec"
	objDecl.valueSpec = spec
	var kind = astCon
	if keyword == "var" {
		kind = astVar
	}
	declare(objDecl, parserTopScope, kind, ident)
	logf(" [parserParseValueSpec] end\n")
	return r
}

func parserParseFuncDecl() *astDecl {
	parserExpect("func", __func__)
	var scope = astNewScope(parserTopScope) // function scope

	var ident = parseIdent()
	var sig = parseSignature(scope)
	var params = sig.params
	var results = sig.results
	if results == nil {
		logf(" [parserParseFuncDecl] %s sig.results is nil\n", ident.Name)
	} else {
		logf(" [parserParseFuncDecl] %s sig.results.List = %s\n", ident.Name, Itoa(len(sig.results.List)))
	}
	var body *astBlockStmt
	if ptok.tok == "{" {
		logf(" begin parseBody()\n")
		body = parseBody(scope)
		logf(" end parseBody()\n")
		parserExpectSemi(__func__)
	} else {
		parserExpectSemi(__func__)
	}
	var decl = new(astDecl)
	decl.dtype = "*astFuncDecl"
	var funcDecl = new(astFuncDecl)
	decl.funcDecl = funcDecl
	decl.funcDecl.Name = ident
	decl.funcDecl.Type = new(astFuncType)
	decl.funcDecl.Type.params = params
	decl.funcDecl.Type.results = results
	decl.funcDecl.Body = body
	var objDecl = new(ObjDecl)
	objDecl.dtype = "*astFuncDecl"
	objDecl.funcDecl = funcDecl
	declare(objDecl, parserPkgScope, astFun, ident)
	return decl
}

func parserParseFile() *astFile {
	// expect "package" keyword
	parserExpect("package", __func__)
	parserUnresolved = nil
	var ident = parseIdent()
	var packageName = ident.Name
	parserExpectSemi(__func__)

	parserTopScope = new(astScope) // open scope
	parserPkgScope = parserTopScope

	for ptok.tok == "import" {
		parserParseImportDecl()
	}

	logf("\n")
	logf(" [parser] Parsing Top level decls\n")
	var decls []*astDecl
	var decl *astDecl

	for ptok.tok != "EOF" {
		switch ptok.tok {
		case "var", "const":
			var spec = parserParseValueSpec(ptok.tok)
			var genDecl = new(astGenDecl)
			genDecl.Spec = spec
			decl = new(astDecl)
			decl.dtype = "*astGenDecl"
			decl.genDecl = genDecl
		case "func":
			logf("\n\n")
			decl = parserParseFuncDecl()
			logf(" func decl parsed:%s\n", decl.funcDecl.Name.Name)
		case "type":
			var spec = parserParseTypeSpec()
			var genDecl = new(astGenDecl)
			genDecl.Spec = spec
			decl = new(astDecl)
			decl.dtype = "*astGenDecl"
			decl.genDecl = genDecl
			logf(" type parsed:%s\n", "")
		default:
			panic2(__func__, "TBI:"+ptok.tok)
		}
		decls = append(decls, decl)
	}

	parserTopScope = nil

	// dump parserPkgScope
	logf("[DEBUG] Dump objects in the package scope\n")
	var oe *objectEntry
	for _, oe = range parserPkgScope.Objects {
		logf("    object %s\n", oe.name)
	}

	var unresolved []*astIdent
	var idnt *astIdent
	logf(" [parserParseFile] resolving parserUnresolved (n=%s)\n", Itoa(len(parserUnresolved)))
	for _, idnt = range parserUnresolved {
		logf(" [parserParseFile] resolving ident %s ...\n", idnt.Name)
		var obj *astObject = scopeLookup(parserPkgScope, idnt.Name)
		if obj != nil {
			logf(" resolved \n")
			idnt.Obj = obj
		} else {
			logf(" unresolved \n")
			unresolved = append(unresolved, idnt)
		}
	}
	logf(" [parserParseFile] Unresolved (n=%s)\n", Itoa(len(unresolved)))

	var f = new(astFile)
	f.Name = packageName
	f.Decls = decls
	f.Unresolved = unresolved
	logf(" [%s] end\n", __func__)
	return f
}

func parseFile(filename string) *astFile {
	var text = readSource(filename)
	parserInit(text)
	return parserParseFile()
}