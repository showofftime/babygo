package main


// --- universe ---
var gNil *astObject
var identNil *astIdent
var eNil *astExpr
var gTrue *astObject
var gFalse *astObject
var gString *astObject
var gInt *astObject
var gUint8 *astObject
var gUint16 *astObject
var gUintptr *astObject
var gBool *astObject
var gNew *astObject
var gMake *astObject
var gAppend *astObject
var gLen *astObject
var gCap *astObject

func createUniverse() *astScope {
	var universe = new(astScope)

	scopeInsert(universe, gInt)
	scopeInsert(universe, gUint8)
	scopeInsert(universe, gUint16)
	scopeInsert(universe, gUintptr)
	scopeInsert(universe, gString)
	scopeInsert(universe, gBool)
	scopeInsert(universe, gNil)
	scopeInsert(universe, gTrue)
	scopeInsert(universe, gFalse)
	scopeInsert(universe, gNew)
	scopeInsert(universe, gMake)
	scopeInsert(universe, gAppend)
	scopeInsert(universe, gLen)
	scopeInsert(universe, gCap)

	logf(" [%s] scope insertion of predefined identifiers complete\n", __func__)

	// @FIXME package names should not be be in universe
	var pkgOs = new(astObject)
	pkgOs.Kind = "Pkg"
	pkgOs.Name = "os"
	scopeInsert(universe, pkgOs)

	var pkgSyscall = new(astObject)
	pkgSyscall.Kind = "Pkg"
	pkgSyscall.Name = "syscall"
	scopeInsert(universe, pkgSyscall)

	var pkgUnsafe = new(astObject)
	pkgUnsafe.Kind = "Pkg"
	pkgUnsafe.Name = "unsafe"
	scopeInsert(universe, pkgUnsafe)
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

func initGlobals() {
	gNil = new(astObject)
	gNil.Kind = astCon // is it Con ?
	gNil.Name = "nil"

	identNil = new(astIdent)
	identNil.Obj = gNil
	identNil.Name = "nil"
	eNil = new(astExpr)
	eNil.dtype = "*astIdent"
	eNil.ident = identNil

	gTrue = new(astObject)
	gTrue.Kind = astCon
	gTrue.Name = "true"

	gFalse = new(astObject)
	gFalse.Kind = astCon
	gFalse.Name = "false"

	gString = new(astObject)
	gString.Kind = astTyp
	gString.Name = "string"
	tString = new(Type)
	tString.e = new(astExpr)
	tString.e.dtype = "*astIdent"
	tString.e.ident = new(astIdent)
	tString.e.ident.Name = "string"
	tString.e.ident.Obj = gString

	gInt = new(astObject)
	gInt.Kind = astTyp
	gInt.Name = "int"
	tInt = new(Type)
	tInt.e = new(astExpr)
	tInt.e.dtype = "*astIdent"
	tInt.e.ident = new(astIdent)
	tInt.e.ident.Name = "int"
	tInt.e.ident.Obj = gInt

	gUint8 = new(astObject)
	gUint8.Kind = astTyp
	gUint8.Name = "uint8"
	tUint8 = new(Type)
	tUint8.e = new(astExpr)
	tUint8.e.dtype = "*astIdent"
	tUint8.e.ident = new(astIdent)
	tUint8.e.ident.Name = "uint8"
	tUint8.e.ident.Obj = gUint8

	gUint16 = new(astObject)
	gUint16.Kind = astTyp
	gUint16.Name = "uint16"
	tUint16 = new(Type)
	tUint16.e = new(astExpr)
	tUint16.e.dtype = "*astIdent"
	tUint16.e.ident = new(astIdent)
	tUint16.e.ident.Name = "uint16"
	tUint16.e.ident.Obj = gUint16

	gUintptr = new(astObject)
	gUintptr.Kind = astTyp
	gUintptr.Name = "uintptr"
	tUintptr = new(Type)
	tUintptr.e = new(astExpr)
	tUintptr.e.dtype = "*astIdent"
	tUintptr.e.ident = new(astIdent)
	tUintptr.e.ident.Name = "uintptr"
	tUintptr.e.ident.Obj = gUintptr

	gBool = new(astObject)
	gBool.Kind = astTyp
	gBool.Name = "bool"
	tBool = new(Type)
	tBool.e = new(astExpr)
	tBool.e.dtype = "*astIdent"
	tBool.e.ident = new(astIdent)
	tBool.e.ident.Name = "bool"
	tBool.e.ident.Obj = gBool

	gNew = new(astObject)
	gNew.Kind = astFun
	gNew.Name = "new"

	gMake = new(astObject)
	gMake.Kind = astFun
	gMake.Name = "make"

	gAppend = new(astObject)
	gAppend.Kind = astFun
	gAppend.Name = "append"

	gLen = new(astObject)
	gLen.Kind = astFun
	gLen.Name = "len"

	gCap = new(astObject)
	gCap.Kind = astFun
	gCap.Name = "cap"
}

var pkg *PkgContainer

type PkgContainer struct {
	name string
	vars []*astValueSpec
	funcs []*Func
}


