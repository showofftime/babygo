package main

var (
	gNil     *astObject
	identNil *astIdent
	exprNil  *astExpr
	gTrue    *astObject
	gFalse   *astObject
	gString  *astObject
	gInt     *astObject
	gUint8   *astObject
	gUint16  *astObject
	gUintptr *astObject
	gBool    *astObject
	gNew     *astObject
	gMake    *astObject
	gAppend  *astObject
	gLen     *astObject
	gCap     *astObject
)

// 初始化默认ast对象定义
func initGlobals() {

	gNil = new(astObject)
	gNil.Kind = astConst
	gNil.Name = "nil"

	identNil = new(astIdent)
	identNil.Obj = gNil
	identNil.Name = "nil"

	exprNil = new(astExpr)
	exprNil.dtype = "*astIdent"
	exprNil.ident = identNil

	gTrue = new(astObject)
	gTrue.Kind = astConst
	gTrue.Name = "true"

	gFalse = new(astObject)
	gFalse.Kind = astConst
	gFalse.Name = "false"

	gString = new(astObject)
	gString.Kind = astType
	gString.Name = "string"
	tString = new(Type)
	tString.e = new(astExpr)
	tString.e.dtype = "*astIdent"
	tString.e.ident = new(astIdent)
	tString.e.ident.Name = "string"
	tString.e.ident.Obj = gString

	gInt = new(astObject)
	gInt.Kind = astType
	gInt.Name = "int"
	tInt = new(Type)
	tInt.e = new(astExpr)
	tInt.e.dtype = "*astIdent"
	tInt.e.ident = new(astIdent)
	tInt.e.ident.Name = "int"
	tInt.e.ident.Obj = gInt

	gUint8 = new(astObject)
	gUint8.Kind = astType
	gUint8.Name = "uint8"
	tUint8 = new(Type)
	tUint8.e = new(astExpr)
	tUint8.e.dtype = "*astIdent"
	tUint8.e.ident = new(astIdent)
	tUint8.e.ident.Name = "uint8"
	tUint8.e.ident.Obj = gUint8

	gUint16 = new(astObject)
	gUint16.Kind = astType
	gUint16.Name = "uint16"
	tUint16 = new(Type)
	tUint16.e = new(astExpr)
	tUint16.e.dtype = "*astIdent"
	tUint16.e.ident = new(astIdent)
	tUint16.e.ident.Name = "uint16"
	tUint16.e.ident.Obj = gUint16

	gUintptr = new(astObject)
	gUintptr.Kind = astType
	gUintptr.Name = "uintptr"
	tUintptr = new(Type)
	tUintptr.e = new(astExpr)
	tUintptr.e.dtype = "*astIdent"
	tUintptr.e.ident = new(astIdent)
	tUintptr.e.ident.Name = "uintptr"
	tUintptr.e.ident.Obj = gUintptr

	gBool = new(astObject)
	gBool.Kind = astType
	gBool.Name = "bool"
	tBool = new(Type)
	tBool.e = new(astExpr)
	tBool.e.dtype = "*astIdent"
	tBool.e.ident = new(astIdent)
	tBool.e.ident.Name = "bool"
	tBool.e.ident.Obj = gBool

	gNew = new(astObject)
	gNew.Kind = astFunc
	gNew.Name = "new"

	gMake = new(astObject)
	gMake.Kind = astFunc
	gMake.Name = "make"

	gAppend = new(astObject)
	gAppend.Kind = astFunc
	gAppend.Name = "append"

	gLen = new(astObject)
	gLen.Kind = astFunc
	gLen.Name = "len"

	gCap = new(astObject)
	gCap.Kind = astFunc
	gCap.Name = "cap"
}

func createUniverse() *astScope {

	initGlobals()

	var universe = new(astScope)

	objs := []*astObject{
		gInt, gUint8, gUint16, gUintptr, gString, gBool, gNil,
		gTrue, gFalse, gNew, gMake, gAppend, gLen, gCap,
	}
	scopeInsert(universe, objs...)

	logf(" [%s] scope insertion of predefined identifiers complete\n", __func__)

	// @FIXME package names should not be be in universe
	// os, syscall, unsafe
	scopeInsert(universe, &astObject{Kind: "Pkg", Name: "os"})
	scopeInsert(universe, &astObject{Kind: "Pkg", Name: "syscall"})
	scopeInsert(universe, &astObject{Kind: "Pkg", Name: "unsafe"})

	logf(" [%s] scope insertion complete\n", __func__)
	return universe
}

func resolveUniverse(file *astFile, universe *astScope) {
	logf(" [%s] start\n", __func__)
	// create universe scope
	// inject predeclared identifers
	var unresolved []*astIdent
	var ident *astIdent
	logf(" [SEMA] resolving file.Unresolved (n=%s)\n", Itoa(len(file.Unresolved)))
	for _, ident = range file.Unresolved {
		logf(" [SEMA] resolving ident %s ... \n", ident.Name)
		var obj *astObject = scopeLookup(universe, ident.Name)
		if obj != nil {
			logf(" matched\n")
			ident.Obj = obj
		} else {
			panic2(__func__, "Unresolved : "+ident.Name)
			unresolved = append(unresolved, ident)
		}
	}
}

var pkg *PkgContainer

type PkgContainer struct {
	name  string
	vars  []*astValueSpec
	funcs []*Func
}
