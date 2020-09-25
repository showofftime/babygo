package main

// --- walk ---
type sliteral struct {
	label  string
	strlen int
	value  string // raw value/pre/precompiler.go:2150
}

type stringLiteralsContainer struct {
	lit *astBasicLit
	sl  *sliteral
}

type Func struct {
	localvars []*string
	localarea int
	argsarea  int
	name      string
	Body      *astBlockStmt
}

type Variable struct {
	name         string
	isGlobal     bool
	globalSymbol string
	localOffset  int
}

//type localoffsetint int //@TODO

var stringLiterals []*stringLiteralsContainer
var stringIndex int
var localoffset int
var currentFuncDecl *astFuncDecl

func getStringLiteral(lit *astBasicLit) *sliteral {
	var container *stringLiteralsContainer
	for _, container = range stringLiterals {
		if container.lit == lit {
			return container.sl
		}
	}

	panic2(__func__, "string literal not found:"+lit.Value)
	var r *sliteral
	return r
}

func registerStringLiteral(lit *astBasicLit) {
	logf(" [registerStringLiteral] begin\n")

	if pkg.name == "" {
		panic2(__func__, "no pkgName")
	}

	var strlen int
	var c uint8
	var vl = []uint8(lit.Value)
	for _, c = range vl {
		if c != '\\' {
			strlen++
		}
	}

	var label = fmtSprintf(".%s.S%d", []string{pkg.name, Itoa(stringIndex)})
	stringIndex++

	var sl = new(sliteral)
	sl.label = label
	sl.strlen = strlen - 2
	sl.value = lit.Value
	logf(" [registerStringLiteral] label=%s, strlen=%s\n", sl.label, Itoa(sl.strlen))
	var cont = new(stringLiteralsContainer)
	cont.sl = sl
	cont.lit = lit
	stringLiterals = append(stringLiterals, cont)
}

func newGlobalVariable(name string) *Variable {
	var vr = new(Variable)
	vr.name = name
	vr.isGlobal = true
	vr.globalSymbol = name
	return vr
}

func newLocalVariable(name string, localoffset int) *Variable {
	var vr = new(Variable)
	vr.name = name
	vr.isGlobal = false
	vr.localOffset = localoffset
	return vr
}

func walkStmt(stmt *astStmt) {
	logf(" [%s] begin dtype=%s\n", __func__, stmt.dtype)
	switch stmt.dtype {
	case "*astDeclStmt":
		logf(" [%s] *ast.DeclStmt\n", __func__)
		if stmt.DeclStmt == nil {
			panic2(__func__, "nil pointer exception\n")
		}
		var declStmt = stmt.DeclStmt
		if declStmt.Decl == nil {
			panic2(__func__, "ERROR\n")
		}
		var dcl = declStmt.Decl
		if dcl.dtype != "*astGenDecl" {
			panic2(__func__, "[dcl.dtype] internal error")
		}
		var valSpec = dcl.genDecl.Spec.valueSpec
		if valSpec.Type == nil {
			if valSpec.Value == nil {
				panic2(__func__, "type inference requires a value")
			}
			var _typ = getTypeOfExpr(valSpec.Value)
			if _typ != nil && _typ.e != nil {
				valSpec.Type = _typ.e
			} else {
				panic2(__func__, "type inference failed")
			}
		}
		var typ = valSpec.Type // Type can be nil
		logf(" [walkStmt] valSpec Name=%s, Type=%s\n",
			valSpec.Name.Name, typ.dtype)

		var t = e2t(typ)
		var sizeOfType = getSizeOfType(t)
		localoffset = localoffset - sizeOfType

		valSpec.Name.Obj.Variable = newLocalVariable(valSpec.Name.Name, localoffset)
		logf(" var %s offset = %d\n", valSpec.Name.Obj.Name,
			Itoa(valSpec.Name.Obj.Variable.localOffset))
		if valSpec.Value != nil {
			walkExpr(valSpec.Value)
		}
	case "*astAssignStmt":
		var rhs = stmt.assignStmt.Rhs
		var rhsE *astExpr
		for _, rhsE = range rhs {
			walkExpr(rhsE)
		}
	case "*astExprStmt":
		walkExpr(stmt.exprStmt.X)
	case "*astReturnStmt":
		var rt *astExpr
		for _, rt = range stmt.returnStmt.Results {
			walkExpr(rt)
		}
	case "*astIfStmt":
		if stmt.ifStmt.Init != nil {
			walkStmt(stmt.ifStmt.Init)
		}
		walkExpr(stmt.ifStmt.Cond)
		var s *astStmt
		for _, s = range stmt.ifStmt.Body.List {
			walkStmt(s)
		}
		if stmt.ifStmt.Else != nil {
			walkStmt(stmt.ifStmt.Else)
		}
	case "*astForStmt":
		stmt.forStmt.Outer = currentFor
		currentFor = stmt
		if stmt.forStmt.Init != nil {
			walkStmt(stmt.forStmt.Init)
		}
		if stmt.forStmt.Cond != nil {
			walkExpr(stmt.forStmt.Cond)
		}
		if stmt.forStmt.Post != nil {
			walkStmt(stmt.forStmt.Post)
		}
		var _s = new(astStmt)
		_s.dtype = "*astBlockStmt"
		_s.blockStmt = stmt.forStmt.Body
		walkStmt(_s)
		currentFor = stmt.forStmt.Outer
	case "*astRangeStmt":
		walkExpr(stmt.rangeStmt.X)
		stmt.rangeStmt.Outer = currentFor
		currentFor = stmt
		var _s = blockStmt2Stmt(stmt.rangeStmt.Body)
		walkStmt(_s)
		localoffset = localoffset - intSize
		var lenvar = newLocalVariable(".range.len", localoffset)
		localoffset = localoffset - intSize
		var indexvar = newLocalVariable(".range.index", localoffset)
		stmt.rangeStmt.lenvar = lenvar
		stmt.rangeStmt.indexvar = indexvar
		currentFor = stmt.rangeStmt.Outer
	case "*astIncDecStmt":
		walkExpr(stmt.incDecStmt.X)
	case "*astBlockStmt":
		var s *astStmt
		for _, s = range stmt.blockStmt.List {
			walkStmt(s)
		}
	case "*astBranchStmt":
		stmt.branchStmt.currentFor = currentFor
	case "*astSwitchStmt":
		if stmt.switchStmt.Tag != nil {
			walkExpr(stmt.switchStmt.Tag)
		}
		walkStmt(blockStmt2Stmt(stmt.switchStmt.Body))
	case "*astCaseClause":
		var e_ *astExpr
		var s_ *astStmt
		for _, e_ = range stmt.caseClause.List {
			walkExpr(e_)
		}
		for _, s_ = range stmt.caseClause.Body {
			walkStmt(s_)
		}
	default:
		panic2(__func__, "TBI: stmt.dtype="+stmt.dtype)
	}
}

var currentFor *astStmt

func walkExpr(expr *astExpr) {
	logf(" [walkExpr] dtype=%s\n", expr.dtype)
	switch expr.dtype {
	case "*astIdent":
		// what to do ?
	case "*astCallExpr":
		var arg *astExpr
		walkExpr(expr.callExpr.Fun)
		// Replace __func__ ident by a string literal
		var basicLit *astBasicLit
		var i int
		var newArg *astExpr
		for i, arg = range expr.callExpr.Args {
			if arg.dtype == "*astIdent" {
				var ident = arg.ident
				if ident.Name == "__func__" && ident.Obj.Kind == astVar {
					basicLit = new(astBasicLit)
					basicLit.Kind = "STRING"
					basicLit.Value = "\"" + currentFuncDecl.Name.Name + "\""
					newArg = new(astExpr)
					newArg.dtype = "*astBasicLit"
					newArg.basicLit = basicLit
					expr.callExpr.Args[i] = newArg
					arg = newArg
				}
			}
			walkExpr(arg)
		}
	case "*astBasicLit":
		switch expr.basicLit.Kind {
		case "STRING":
			registerStringLiteral(expr.basicLit)
		}
	case "*astCompositeLit":
		var v *astExpr
		for _, v = range expr.compositeLit.Elts {
			walkExpr(v)
		}
	case "*astUnaryExpr":
		walkExpr(expr.unaryExpr.X)
	case "*astBinaryExpr":
		walkExpr(expr.binaryExpr.X)
		walkExpr(expr.binaryExpr.Y)
	case "*astIndexExpr":
		walkExpr(expr.indexExpr.Index)
		walkExpr(expr.indexExpr.X)
	case "*astSliceExpr":
		if expr.sliceExpr.Low != nil {
			walkExpr(expr.sliceExpr.Low)
		}
		if expr.sliceExpr.High != nil {
			walkExpr(expr.sliceExpr.High)
		}
		if expr.sliceExpr.Max != nil {
			walkExpr(expr.sliceExpr.Max)
		}
		walkExpr(expr.sliceExpr.X)
	case "*astStarExpr":
		walkExpr(expr.starExpr.X)
	case "*astSelectorExpr":
		walkExpr(expr.selectorExpr.X)
	case "*astArrayType": // []T(e)
		// do nothing ?
	case "*astParenExpr":
		walkExpr(expr.parenExpr.X)
	default:
		panic2(__func__, "TBI:"+expr.dtype)
	}
}

func walk(pkgContainer *PkgContainer, file *astFile) {
	var decl *astDecl
	for _, decl = range file.Decls {
		switch decl.dtype {
		case "*astGenDecl":
			var genDecl = decl.genDecl
			switch genDecl.Spec.dtype {
			case "*astValueSpec":
				var valSpec = genDecl.Spec.valueSpec
				var nameIdent = valSpec.Name
				if nameIdent.Obj.Kind == astVar {
					nameIdent.Obj.Variable = newGlobalVariable(nameIdent.Obj.Name)
					pkgContainer.vars = append(pkgContainer.vars, valSpec)
				}
				if valSpec.Value != nil {
					walkExpr(valSpec.Value)
				}
			case "*astTypeSpec":
				// do nothing
				var typeSpec = genDecl.Spec.typeSpec
				switch kind(e2t(typeSpec.Type)) {
				case T_STRUCT:
					calcStructSizeAndSetFieldOffset(typeSpec)
				default:
					// do nothing
				}
			default:
				panic2(__func__, "Unexpected dtype="+genDecl.Spec.dtype)
			}
		case "*astFuncDecl":
			var funcDecl = decl.funcDecl
			currentFuncDecl = funcDecl
			logf(" [sema] == astFuncDecl %s ==\n", funcDecl.Name.Name)
			localoffset = 0
			var paramoffset = 16
			var field *astField
			for _, field = range funcDecl.Type.params.List {
				var obj = field.Name.Obj
				obj.Variable = newLocalVariable(obj.Name, paramoffset)
				var varSize = getSizeOfType(e2t(field.Type))
				paramoffset = paramoffset + varSize
				logf(" field.Name.Obj.Name=%s\n", obj.Name)
				//logf("   field.Type=%#v\n", field.Type)
			}
			if funcDecl.Body != nil {
				var stmt *astStmt
				for _, stmt = range funcDecl.Body.List {
					walkStmt(stmt)
				}
				var fnc = new(Func)
				fnc.name = funcDecl.Name.Name
				fnc.Body = funcDecl.Body
				fnc.localarea = localoffset
				fnc.argsarea = paramoffset

				pkgContainer.funcs = append(pkgContainer.funcs, fnc)
			}
		default:
			panic2(__func__, "TBI: "+decl.dtype)
		}
	}

	if len(stringLiterals) == 0 {
		panic2(__func__, "stringLiterals is empty\n")
	}
}
